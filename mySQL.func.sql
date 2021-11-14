DROP PROCEDURE IF EXISTS SYST_META_BUILD;
DROP FUNCTION IF EXISTS FUNC_ENUM_BUILD;
DROP FUNCTION IF EXISTS FUNC_META_BUILD;
DROP FUNCTION IF EXISTS FUNC_HAVE_DIGIT;
DROP FUNCTION IF EXISTS FUNC_DIGIT;

CREATE FUNCTION FUNC_HAVE_DIGIT(V VARCHAR(255)) RETURNS BOOLEAN
BEGIN
   DECLARE R INT DEFAULT 0;
   DECLARE I INT DEFAULT 1;
   CHK:
   WHILE I < (LENGTH(V) + 1) DO
      IF SUBSTRING(V, I, 1) IN ( '0', '1', '2', '3', '4', '5', '6', '7', '8', '9' ) THEN
         SET R = 1;
         LEAVE CHK;
      END IF;
      SET I = I + 1;
   END WHILE CHK;
   RETURN R;
END;

CREATE FUNCTION FUNC_DIGIT(V VARCHAR(255)) RETURNS VARCHAR(255) CHARSET utf8
BEGIN
   DECLARE R VARCHAR(255) DEFAULT '';
   DECLARE I INT          DEFAULT 1;
   WHILE I < (LENGTH(V) + 1) DO
      IF SUBSTRING(V, I, 1) IN ( '0', '1', '2', '3', '4', '5', '6', '7', '8', '9' ) THEN
         SET R = CONCAT(R, SUBSTRING(V, I, 1));
      END IF;
      SET I = I + 1;
   END WHILE;
   RETURN R;
END;

CREATE FUNCTION FUNC_ENUM_BUILD(V VARCHAR(255)) RETURNS VARCHAR(255) CHARSET utf8
BEGIN
   DECLARE R VARCHAR(255) DEFAULT '';
   DECLARE I INT          DEFAULT 1;
   WHILE I < (LENGTH(V) + 1) DO
      IF SUBSTRING(V, I, 1) NOT IN ('(', ')', '\'') THEN
         SET R = CONCAT(R, SUBSTRING(V, I, 1));
      END IF;
      SET I = I + 1;
   END WHILE;
   RETURN R;
END;

CREATE FUNCTION FUNC_META_BUILD(V VARCHAR(128)) RETURNS varchar(128) CHARSET utf8
    NO SQL
BEGIN
DECLARE F VARCHAR(128) DEFAULT NULL;
DECLARE L INT DEFAULT NULL;
DECLARE X INT DEFAULT NULL;
DECLARE Z VARCHAR(4) DEFAULT NULL;
SET Z = LEFT(V, 1);
SET X = 0;
REPEAT
    IF LENGTH(TRIM(V)) = 0 OR V IS NULL THEN
        SET Z = CONCAT(Z, SUBSTRING(F, X, 1));
    ELSE
        SET F = SUBSTRING_INDEX(V, '_', 1);
        SET L = LENGTH(F);
        SET V = TRIM(INSERT(V, 1, L + 1, ''));
        SET Z = CONCAT(Z, LEFT(V, 1));
    END IF;
SET X = X + 1;
UNTIL X>=4 END REPEAT;
SET X = 4 - LENGTH(Z);
IF X > 0 THEN
    REPEAT
        SET Z = CONCAT(Z, '0');
        SET X = X - 1;
    UNTIL X=0 END REPEAT;
END IF;
RETURN (Z);
END;

CREATE PROCEDURE SYST_META_BUILD(IN go VARCHAR(255))
BEGIN
TRUNCATE st_metadata_copy;
INSERT INTO st_metadata_copy(TID, CID, TBL, COL, COLT, COLP, UNSIGNED, MINL, MAXL, ENUM, SSN) SELECT TID, CID, TBL, COL, COLT, COLP, UNSIGNED, MINL, MAXL, ENUM, SSN FROM st_metadata;
TRUNCATE st_metadata;
INSERT INTO st_metadata(TID, CID, TBL, COL, COLT, COLP, MINL, MAXL, ENUM)
SELECT UPPER(FUNC_META_BUILD(TABLE_NAME)) TID,
       ORDINAL_POSITION CID,
       TABLE_NAME TBL,
       COLUMN_NAME COL,
       UPPER(DATA_TYPE) COLT,
       IF(COLUMN_KEY='PRI', '1', '0') COLP,
       IF(COLUMN_TYPE LIKE '%unsigned', '1', '0') as UNSIGNED,
       (CASE DATA_TYPE
           WHEN 'char' THEN CHARACTER_MAXIMUM_LENGTH
           WHEN 'date' THEN 10
           WHEN 'datetime' THEN 19
           WHEN 'timestamp' THEN 19
       END) MINL,
       IF (CHARACTER_MAXIMUM_LENGTH IS NOT NULL, CHARACTER_MAXIMUM_LENGTH,
           IF (NUMERIC_PRECISION IS NOT NULL, NUMERIC_PRECISION,
               (CASE DATA_TYPE
                   WHEN 'date' THEN 10
                   WHEN 'datetime' THEN 19
                   WHEN 'timestamp' THEN 19
               END)
           )
       ) MAXL,
       (CASE DATA_TYPE
           WHEN 'tinyint' THEN IF(COLUMN_TYPE LIKE '%unsigned', 0, -128)
           WHEN 'smallint' THEN IF(COLUMN_TYPE LIKE '%unsigned', 0, -32768)
           WHEN 'mediumint' THEN IF(COLUMN_TYPE LIKE '%unsigned', 0, -8388608)
           WHEN 'int' THEN IF(COLUMN_TYPE LIKE '%unsigned', 0, -2147483648)
       END) MINV,
       (CASE DATA_TYPE
           WHEN 'tinyint' THEN IF(COLUMN_TYPE LIKE '%unsigned', 255, 127)
           WHEN 'smallint' THEN IF(COLUMN_TYPE LIKE '%unsigned', 65535, 32767)
           WHEN 'mediumint' THEN IF(COLUMN_TYPE LIKE '%unsigned', 16777215, 8388607)
           WHEN 'int' THEN IF(COLUMN_TYPE LIKE '%unsigned', 4294967295, 2147483647)
       END) MAXV,
       IF (DATA_TYPE='enum', SUBSTRING(FUNC_ENUM_BUILD(COLUMN_TYPE), 5), NULL) ENUM
  FROM information_schema.COLUMNS
 WHERE TABLE_SCHEMA=go
 ORDER BY TABLE_NAME,ORDINAL_POSITION;
UPDATE st_metadata A, st_metadata_copy B
   SET A.MINL=B.MINL, A.MAXL=B.MAXL, A.MINV=B.MINV, A.MAXV=B.MAXV, A.SSN=B.SSN
 WHERE A.TID=B.TID AND A.CID=B.CID;
END;
