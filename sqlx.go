// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// Ide-nya adalah menyederhakan abstraksi database (CRUD) di golang kedalam
// abstraksi yang umum digunakan (PHP, java dll)
//
// Jadi setiap struct adalah extends dari struct bawaan (database/sql) dengan
// enhancement sesuai tujuan framework untuk aplikasi transaksi
package tlkm

import (
	"context"
    "database/sql"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"github.com/tlkm/buffer"
	"github.com/tlkm/to"
)

type (
    Driver  int

    // alias abstraksi database/sql bawaan golang
    DB = sql.DB
    Conn = sql.Conn
    Stmt = sql.Stmt
    Rows = sql.Rows
    Row = sql.Row
    RawBytes = sql.RawBytes
    Result = sql.Result

    // ResultSet mengenkapsulasi sql.Rows kedalam abstraksi untuk menghilangkan
    // kebutuhan *sql.Rows.Scan secara eksplisit
    ResultSet struct {
        scan  bool
        cols  int
        rows  *Rows
        name  []string
        vals  []RawBytes
        args  []interface{}
        indx  map[string]*RawBytes
        bind  map[string]interface{}
        argv  interface{}
    }

    // framework menggunakan terminologi Connection karena lebih jelas
    Connection struct {
        *DB
        driver  Driver
    }

    // Disarankan untuk mengambil object QueryBuilder dari sync.Pool via SQL.Builder()
    QueryBuilder struct {
        cols, from  string
        join    List
        groupBy, orderBy    string
        limit, offset   int
        where, value    *GMap
        rollup  bool
        union   *QueryBuilder
        unall   bool
        driver  Driver
    }

    // Buffer untuk memenuhi kebutuhan multi values insert: INSERT INTO TABLE_A (B, C) VALUES (D, E), (F, G) ...
    // sebagai alternative lebih praktis dan sederhana dibanding bulk insert dari
    // SQL.InsertQuery yang mensyaratkan mapping {column: value}
    BulkBuffer struct {
        iter    int
        size    int
        bfer    *buffer.ByteBuffer
    }

    // extends struct transaksi, sementara tanpa enhancement
    Tx struct {
        *sql.Tx
    }

    // fungsi utama dari public var (SQL):
    //   * menyediakan registrasi/mapping datasource kedalam koneksi
    //   * lookup (open koneksi) berdasarkan nama datasource
    //   * me-manage resource database (query, resulset dll) dalam sync.Pool
    sqlx struct {
        mutex   sync.RWMutex
        proto   map[string]*Connection
        rsset   sync.Pool
        query   sync.Pool
        bbfer   sync.Pool
    }

    RecordInterface interface {
        TableName() string
    }

    Record struct {
        //
    }

    FakeResult struct {
        ID  int64   // Last Insert ID
        AR  int64   // Affected Rows
    }
)

const (
    // interpolasi parameter yang modelnya cem-macem kita serahkan ke driver seperti
    // apa baiknya biar aman dari injection dll
    MYSQL Driver = iota // ?
    ORACL   // :column
    PGSQL   // $index
    MSSQL   // @column
)

var (
    // ** private **
    driversMap = map[Driver]string{MYSQL: "mysql", ORACL: "oracl", PGSQL: "pgsql", MSSQL: "mssql"}
    drivers = make(map[string]Driver)

    // ** GLOBAL **
    SQL = &sqlx{proto: make(map[string]*Connection)}
)

// TODOC
func init() {
	SQL.rsset.New = func() interface{} {
		return &ResultSet{}
	}
	SQL.query.New = func() interface{} {
		return &QueryBuilder{}
	}
	SQL.bbfer.New = func() interface{} {
		return &BulkBuffer{}
	}
    for k, v := range driversMap {
        drivers[v] = k
    }
}

func (self *FakeResult) LastInsertId() (int64, error) {
	return self.ID, nil
}

func (self *FakeResult) RowsAffected() (int64, error) {
	return self.AR, nil
}

// sql.Open akan dilakukan 1x pada saat skema koneksi didaftarkan (sebelum server berjalan)
// dengan mekanisme pooling diserahkan ke golang
//
// referensi: https://pkg.go.dev/database/sql#Open
//
// The returned DB is safe for concurrent use by multiple goroutines and maintains
// its own pool of idle connections. Thus, the Open function should be called just once.
// It is rarely necessary to close a DB.
func (self *sqlx) Register(k, v string) {
    self.mutex.Lock()
    defer self.mutex.Unlock()
    spos := strings.IndexRune(k, '.')
    name := k[:spos]
    indx := k[spos+1:]
    if conn, e := sql.Open(k[:spos], v); e == nil {
        if e = conn.Ping(); e != nil {  // untuk memastikan real koneksi berhasil
            panic(e.Error())
        }
        self.proto[indx] = &Connection{DB: conn, driver: drivers[name]}
    }
}

// TODOC
func (self *sqlx) Flush() {
    self.mutex.Lock()
    defer self.mutex.Unlock()
    self.proto = make(map[string]*Connection)
}

// TODOC
func (self *sqlx) Exists(name string) (e bool) {
    _, e = self.proto[name]
    return
}

// TODOC
func (self *sqlx) Lookup(name string) (c *Connection) {
    e := false
    if c, e = self.proto[name]; !e {
        panic(Sprintf("DSN %s not found", name))
    }
    return
}

// Return default connection (package/modul) utama
func (self *sqlx) Default() *Connection {
    return self.Lookup(PackageSystem)
}

// TODOC
func (self *sqlx) Builder(driver... Driver) (qb *QueryBuilder) {
	qb = self.query.Get().(*QueryBuilder)
    if len(driver) > 0 {
        qb.driver = driver[0]
    } else {
        qb.driver = MYSQL
    }
    return
}

// TODOC
func (self *sqlx) WhereQuery(driver Driver, stmt *buffer.ByteBuffer, argv *GMap, args *[]interface{}, namespace... string) {
    name := false
    if len(namespace) > 0 { name = true }
    ccat := false
    for column, value := range *argv {
        if ccat {
            stmt.WS(" AND ")
        } else {
            ccat = true
        }
        if column[0] == '@' {
            if name {
                stmt.WS(namespace[0]).WRune('.')
            }
            stmt.WS(column[1:]).WRune('=').WS(to.String(value))
            continue
        }
        idx := 0
        out:
        for key, val := range column {
            if val == '@' {
                idx = key
                break out
            }
        }
        if idx > 0 {
            stmt.WS(column[0:idx]).WRune(' ').WS(column[idx+1:]).WRune(' ')
            self.bind(driver, stmt, column, idx)
            *args = append(*args, value)
            continue
        }
        if name {
            stmt.WS(namespace[0]).WRune('.')
        }
        stmt.WS(column).WRune('=')
        if value == nil {
            stmt.WS("NULL")
        } else {
            self.bind(driver, stmt, column, idx)
            *args = append(*args, value)
        }
    }
}

// TODOC
func (self *sqlx) bind(driver Driver, stmt *buffer.ByteBuffer, column string, counter int) {
    switch driver {
        case MYSQL:
            stmt.WRune('?')
        case ORACL:
            stmt.WRune(':').WS(column)
        case PGSQL:
            stmt.WRune('$').WS(strconv.Itoa(counter))
        case MSSQL:
            stmt.WRune('@').WS(column)
    }
}

// TODOC
func (self *sqlx) insertQuery(driver Driver, intoTable string, ignore bool, vals... *GMap) (string, []interface{}) {
    argv := make([]interface{}, 0)
    stmt := buffer.Get()
    bulk := buffer.Get()
    defer stmt.Close()
    defer bulk.Close()
    stmt.WS("INSERT")
    if ignore { stmt.WS(" IGNORE") }
    stmt.WS(" INTO ")
    stmt.WS(intoTable).WRune(' ').WRune('(')
    top := false
    inn := false
    out := true
    cnt := 1
    for _, args := range vals {
        if top {
            bulk.WRune(')').WRune(',').WRune('(')
        } else {
            top = true
        }
        inn = false
        for column, value := range *args {
            if inn {
                bulk.WRune(',')
                if out { stmt.WRune(',') }
            } else {
                inn = true
            }
            if out { stmt.WS(column) }
            self.bind(driver, bulk, column, cnt)
            argv = append(argv, value)
            cnt+= 1
        }
        out = false
    }
    stmt.WS(") VALUES (").WS(bulk.String()).WRune(')')
    return stmt.String(), argv
}

// TODOC
func (self *sqlx) InsertQuery(driver Driver, intoTable string, vals... *GMap) (string, []interface{}) {
    return self.insertQuery(driver, intoTable, false, vals...)
}

// TODOC
func (self *sqlx) InsertIgnoreQuery(driver Driver, intoTable string, vals... *GMap) (string, []interface{}) {
    return self.insertQuery(driver, intoTable, true, vals...)
}

// TODOC
func (self *sqlx) InsertSelectQuery(intoTable string, qb *QueryBuilder) (stmt string, argv []interface{}) {
    return
}

// TODOC
func (self *sqlx) InsertIgnoreSelectQuery(intoTable string, qb *QueryBuilder) (stmt string, argv []interface{}) {
    return
}

// TODOC
func (self *sqlx) UpdateQuery(driver Driver, tableName string, set *GMap, where... *GMap) (string, []interface{}) {
    argv := make([]interface{}, 0)
    stmt := buffer.Get()
    defer stmt.Close()
    stmt.WS("UPDATE ").WS(tableName).WS(" SET ")
    ccat := false
    idx := 0
    for column, value := range *set {
        idx+= 1
        if ccat {
            stmt.WRune(',')
        } else {
            ccat = true
        }
        if column[0] == '@' {
            stmt.WS(column[1:]).WRune('=').WS(to.String(value))
            continue
        }
        stmt.WS(column).WRune('=')
        if value == nil {
            stmt.WS("NULL")
        } else {
            self.bind(driver, stmt, column, idx)
            argv = append(argv, value)
        }
    }
    if len(where) > 0 {
        stmt.WS(" WHERE ")
        self.WhereQuery(driver, stmt, where[0], &argv)
    }
    return stmt.String(), argv
}

// TODOC
func (self *sqlx) DeleteQuery(driver Driver, tableName string, where... *GMap) (string, []interface{}) {
    argv := make([]interface{}, 0)
    stmt := buffer.Get()
    defer stmt.Close()
    stmt.WS("DELETE FROM ").WS(tableName)
    if len(where) > 0 {
        stmt.WS(" WHERE ")
        self.WhereQuery(driver, stmt, where[0], &argv)
    }
    return stmt.String(), argv
}

// TODOC
func (self *sqlx) bulkInsert(intoTable string, cols List, ignore bool) *BulkBuffer {
	var b *BulkBuffer = self.bbfer.Get().(*BulkBuffer)
    b.bfer = buffer.Get()
    var u *buffer.ByteBuffer = b.bfer
    u.WS("INSERT")
    if ignore {
        u.WS(" IGNORE")
    }
    u.WS(" INTO ").WS(intoTable).WRune('(')
    c := false
    for _, n := range cols {
        if c {
            u.WRune(',')
        } else {
            c = true
        }
        u.WS(n)
    }
    u.WS(") VALUES ")
    return b
}

// TODOC
func (self *sqlx) BulkInsert(intoTable string, cols List) *BulkBuffer {
    return self.bulkInsert(intoTable, cols, false)
}

// TODOC
func (self *sqlx) BulkInsertIgnore(intoTable string, cols List) *BulkBuffer {
    return self.bulkInsert(intoTable, cols, true)
}

// Jadi idenya sederhana, dereference alamat memory *sql.RawBytes dilakukan oleh
// ResultSet berdasarkan index/nama
func (self *sqlx) ResultSet(rows *Rows, err error) *ResultSet {
    if err != nil { panic(err.Error()) }
    name, _ := rows.Columns()
    cols := len(name)
    vals := make([]RawBytes, cols) // reference *sql.Rows.Scan
    args := make([]interface{}, cols)
    indx := make(map[string]*RawBytes, cols)
    for i := range vals {
        args[i] = &vals[i]
        indx[name[i]] = &vals[i]
    }
	var rs *ResultSet = self.rsset.Get().(*ResultSet)
    rs.scan = false
    rs.cols = cols
    rs.rows = rows
    rs.name = name
    rs.vals = vals
    rs.args = args
    rs.indx = indx
    return rs
}

// TODOC
func (self *ResultSet) Close() {
    self.rows.Close()
    self.cols = 0
    self.rows = nil
    self.indx = nil
    self.args = nil
    self.vals = nil
    self.name = nil
    self.bind = nil
    self.argv = nil
    SQL.rsset.Put(self)
}

// TODOC
func (self *ResultSet) Next() (b bool) {
    if b = self.rows.Next(); b {
        if e := self.rows.Scan(self.args...); e != nil {
            panic(e.Error())
        }
    }
    return
}

// TODOC
func (self *ResultSet) Scan(argv interface{}) bool {
    vobj := reflect.ValueOf(argv).Elem()
    args := make([]interface{}, self.cols)
    bind := make(map[string]interface{}, self.cols)
    for indx := range args {
        name := self.name[indx]
        coln := vobj.FieldByName(name)
        var iptr interface{}
        if coln.IsValid() {
			iptr = coln.Addr().Interface()
        } else {
			iptr = &RawBytes{}
        }
        args[indx] = iptr
        bind[name] = iptr
    }
    self.scan = true
    self.args = args
    self.bind = bind
    self.argv = argv

    return self.Next()
}

// TODOC
func (self *ResultSet) Get() interface{} {
    return self.argv
}

// TODOC
func (self *ResultSet) SMap() SMap {
    r := make(SMap, len(self.name))
    if self.scan {
        for i, j := range self.bind {
            r[i] = to.String(j)
        }
    } else {
        for i, j := range self.indx {
            r[i] = string(*j)
        }
    }
    return r
}

// TODOC
func (self *ResultSet) KeyMap(index string, Key *List, Map *GMap) (string, SMap) {
    var (
        v string
        e bool
    )
    m := self.SMap()
    if v, e = m[index]; !e {
        panic(Sprintf("Column %s does not exist", index))
    }
    if Key != nil {
        *Key = append(*Key, v)
    }
    if Map != nil {
        (*Map)[v] = m
    }
    return v, m
}

// TODOC
func (self *ResultSet) Bytes(name string) RawBytes {
    if self.scan {
        panic("Scan does not support RawBytes")
    }
    var (
        r *RawBytes
        b bool
    )
    if r, b = self.indx[name]; !b {
        panic(Sprintf("Column %s does not exist", name))
    }
    return *r
}

// TODOC
func (self *ResultSet) Exists(name string) (b bool) {
    if self.scan {
        _, b = self.bind[name]
    } else {
        _, b = self.indx[name]
    }
    return
}

// TODOC
func (self *ResultSet) String(name string) string {
    if self.scan {
        var (
            v interface{}
            b bool
        )
        if v, b = self.bind[name]; !b {
            panic(Sprintf("Column %s does not exist", name))
        }
        return to.String(v)
    }
    return string(self.Bytes(name))
}

// TODOC
func (self *ResultSet) Int(name string) int {
    if self.scan {
        var (
            v interface{}
            b bool
        )
        if v, b = self.bind[name]; !b {
            panic(Sprintf("Column %s does not exist", name))
        }
        return to.Int(to.String(v))
    }
    if c := self.Bytes(name); c != nil {
        if n, e := strconv.Atoi(string(c)); e == nil {
            return n
        }
	}
	return 0
}

// TODOC
func (self *ResultSet) Bool(name string) (r bool) {
    if self.scan {
        var (
            v interface{}
            b bool
        )
        if v, b = self.bind[name]; !b {
            panic(Sprintf("Column %s does not exist", name))
        }
        r = true
        s := to.UpperCase(to.String(v))
        if s == "0" || s == "FALSE" {
            r = false
        }
        return
    }
    r = true
    v := to.UpperCase(string(self.Bytes(name)))
    if v == "0" || v == "FALSE" {
        r = false
    }
    return
}

// TODOC
func (self *BulkBuffer) Add(args ...[]interface{}) *BulkBuffer {
    var u *buffer.ByteBuffer = self.bfer
    if self.iter > 0 { u.WRune(',') }
    u.WRune('(')
    ccat := false
    coma := false
    for _, argv := range args {
        if len(argv) != self.size { continue } // skip if not match
        self.iter+= 1
        if ccat {
            u.WRune(')').WRune(',').WRune('(')
        } else {
            ccat = true
        }
        coma = false
        for _, value := range argv {
            if coma {
                u.WRune(',')
            } else {
                coma = true
            }
            u.WRune('\'')
            u.WS(to.Escape(to.String(value)))
            u.WRune('\'')
        }
    }
    u.WRune(')')

    return self
}

// TODOC
func (self *BulkBuffer) String() string {
    return self.bfer.String()
}

// TODOC
func (self *BulkBuffer) Close() {
    self.bfer.Close()
    self.bfer = nil
    self.iter = 0
    self.size = 0
    SQL.bbfer.Put(self)
}

// TODOC
func (self *Connection) HasOne(from, to RecordInterface, foreignKeys *GMap) {
    //
}

// TODOC
func (self *Connection) HasMany(from, to RecordInterface, foreignKeys *GMap) {
    //
}

// TODOC
func (self *Connection) Close() {
    //
    // opo jare umak rah wes kah
    //
}

// TODOC
func (self *Connection) Query(query string, args ...interface{}) *ResultSet {
	return SQL.ResultSet(self.QueryContext(context.Background(), query, args...))
}

// TODOC
func (self *Connection) Begin() *Tx {
	tx, err := self.BeginTx(context.Background(), nil)
    if err != nil {
        panic(err.Error())
    }
    return &Tx{tx}
}

// TODOC
func (self *Tx) Query(query string, args ...interface{}) *ResultSet {
	return SQL.ResultSet(self.QueryContext(context.Background(), query, args...))
}

// TODOC
func (self *Connection) Truncate(name string) (Result, error) {
    return self.Exec("TRUNCATE ?", name)
}

// TODOC
func (self *Connection) DropTable(name string) (Result, error) {
    return self.Exec("DROP TABLE ?", name)
}

// TODOC
func (self *Connection) DropTableIfExists(name string) (Result, error) {
    return self.Exec("DROP TABLE IF EXISTS ?", name)
}

// TODOC
func (self *Connection) DropView(name string) (Result, error) {
    return self.Exec("DROP VIEW ?", name)
}

// TODOC
func (self *Connection) DropViewIfExists(name string) (Result, error) {
    return self.Exec("DROP VIEW IF EXISTS ?", name)
}

// TODOC
func (self *Connection) recordTypeValue(r RecordInterface) (T reflect.Type, v reflect.Value) {
    T = reflect.TypeOf(r)
    v = reflect.ValueOf(r)
    if T.Kind() == reflect.Ptr {
        T = T.Elem()
        v = v.Elem()
    }
    return
}

// TODOC
func (self *Connection) recordMap(T reflect.Type, v reflect.Value) GMap {
    m := make(GMap)
    for i := 0; i < T.NumField(); i++ {
        j := T.Field(i)
        u := v.Field(i)
		if j.Anonymous || u.IsNil() || u.Kind() != reflect.Ptr { continue }
        m[j.Name] = u.Elem().Interface()
    }
    return m
}

// TODOC
func (self *Connection) recordMaps(T reflect.Type, v reflect.Value) (CL , PK , FK GMap, FR SMap, err error) {
    CL = make(GMap)
    PK = make(GMap)
    FK = make(GMap)
    FR = make(SMap)
    for i := 0; i < T.NumField(); i++ {
        j := T.Field(i)
        u := v.Field(i)
        t := j.Tag
        _, isPK := t.Lookup("PK")
        f, isFK := t.Lookup("FK")
        if isPK && u.IsNil() {
            return nil, nil, nil, nil, errors.New(Sprintf("Primary Key %s is nil", j.Name))
        }
		if j.Anonymous || u.IsNil() || u.Kind() != reflect.Ptr { continue }
        g := u.Elem().Interface()
        if isFK {
            FR[j.Name] = f
            FK[j.Name] = g
        }
        if isPK {
            PK[j.Name] = g
        } else {
            CL[j.Name] = g
        }
    }
    return
}

// TODOC
func (self *Connection) recordFK(T reflect.Type, v reflect.Value) (FK SMap) {
    FK = make(SMap)
    for i := 0; i < T.NumField(); i++ {
        j := T.Field(i)
        t := j.Tag
        if f, isFK := t.Lookup("FK"); isFK {
            FK[j.Name] = f
        }
    }
    return
}

// TODOC
func (self *Connection) LoadByPK(r RecordInterface) (e error) {
    _, PK, _, _, err := self.recordMaps(self.recordTypeValue(r))
    if err != nil {
        return err
    }
    argv := make([]interface{}, 0)
    stmt := buffer.Get()
    defer stmt.Close()
    stmt.WS("SELECT * FROM ").WS(r.TableName()).WS(" WHERE ")
    SQL.WhereQuery(self.driver, stmt, &PK, &argv)
    rs := self.Query(stmt.String(), argv...)
    defer rs.Close()
    if !rs.Scan(r) {
        e = errors.New("Not Found")
    }
    return
}

// TODOC
func (self *Connection) LoadByFK(fr, to RecordInterface) (e error) {
    _, frPK, _, _, err := self.recordMaps(self.recordTypeValue(fr))
    if err != nil { return err }
    toFR := self.recordFK(self.recordTypeValue(to))
    frTb, toTb := fr.TableName(), to.TableName()
    stmt := buffer.Get()
    defer stmt.Close()
    stmt.WS("SELECT ").WS(toTb).WS(".* FROM ").WS(toTb).WRune(',').WS(frTb).WS(" WHERE ")
    l := len(frTb)
    for local, foreign := range toFR {
        if foreign[0:l] == frTb {
            stmt.WS(toTb).WRune('.').WS(local).WRune('=').WS(foreign)
        }
    }
    stmt.WS(" AND ")
    argv := make([]interface{}, 0)
    SQL.WhereQuery(self.driver, stmt, &frPK, &argv, frTb)
    rs := self.Query(stmt.String(), argv...)
    defer rs.Close()
    if !rs.Scan(to) {
        e = errors.New("Not Found")
    }
    return
}

// TODOC
func (self *Connection) ExecInsert(intoTable string, vals... *GMap) (Result, error) {
    stmt, argv := SQL.InsertQuery(self.driver, intoTable, vals...)
    return self.Exec(stmt, argv...)
}

// TODOC
func (self *Connection) Save(r RecordInterface) (Result, error) {
    args := self.recordMap(self.recordTypeValue(r))
    stmt, argv := SQL.InsertQuery(self.driver, r.TableName(), &args)
    return self.Exec(stmt, argv...)
}

// TODOC
func (self *Connection) ExecInsertIgnore(intoTable string, vals... *GMap) (Result, error) {
    stmt, argv := SQL.InsertIgnoreQuery(self.driver, intoTable, vals...)
    return self.Exec(stmt, argv...)
}

// TODOC
func (self *Connection) SaveIgnore(r RecordInterface) (Result, error) {
    args := self.recordMap(self.recordTypeValue(r))
    stmt, argv := SQL.InsertIgnoreQuery(self.driver, r.TableName(), &args)
    return self.Exec(stmt, argv...)
}

// TODOC
func (self *Connection) ExecUpdate(tableName string, CL *GMap, where... *GMap) (Result, error) {
    stmt, argv := SQL.UpdateQuery(self.driver, tableName, CL, where...)
    return self.Exec(stmt, argv...)
}

// TODOC
func (self *Connection) Update(r RecordInterface) (Result, error) {
    CL, PK, _, _, err := self.recordMaps(self.recordTypeValue(r))
    if err != nil {
        return nil, err
    }
    stmt, argv := SQL.UpdateQuery(self.driver, r.TableName(), &CL, &PK)
    return self.Exec(stmt, argv...)
}

// TODOC
func (self *Connection) ExecDelete(tableName string, where... *GMap) (Result, error) {
    stmt, argv := SQL.DeleteQuery(self.driver, tableName, where...)
    return self.Exec(stmt, argv...)
}

func (self *Connection) Delete(r RecordInterface) (Result, error) {
    _, PK, _, _, err := self.recordMaps(self.recordTypeValue(r))
    if err != nil {
        return nil, err
    }
    stmt, argv := SQL.DeleteQuery(self.driver, r.TableName(), &PK)
    return self.Exec(stmt, argv...)
}

// TODOC
func (self *Connection) Select(qb *QueryBuilder) *ResultSet {
    stmt, argv := qb.SelectQuery()
	return SQL.ResultSet(self.QueryContext(context.Background(), stmt, argv...))
}

// TODOC
func (self *QueryBuilder) Close() {
    self.cols = "*"
    self.from = ""
    self.groupBy = ""
    self.orderBy = ""
    self.limit = 0
    self.offset = 0
    self.join = nil
    self.where = nil
    self.value = nil
    self.union = nil
    self.unall = false
    SQL.query.Put(self)
}

// TODOC
func (self *QueryBuilder) Select(cols ...string) *QueryBuilder {
    switch len(cols) {
    case 0:
        self.cols = "*"
    case 1:
        self.cols = cols[0]
    default:
        b := buffer.Get()
        defer b.Close()
        for i, j := range cols {
            if i > 0 { b.WRune(',') }
            b.WS(j)
        }
        self.cols = b.String()
    }
    return self
}

// TODOC
func (self *QueryBuilder) From(from string) *QueryBuilder {
    self.from = from
    return self
}

// TODOC
func (self *QueryBuilder) makeList() {
    if self.join == nil {
        self.join = make(List, 0)
    }
}

// TODOC
func (self *QueryBuilder) Join(from string, on string) *QueryBuilder {
    self.makeList()
    q := buffer.Get()
    defer q.Close()

    q.WS("     JOIN ").WS(from).WS(" ON (").WS(on).WS(")")

    self.join = append(self.join, q.String())

    return self
}

// TODOC
func (self *QueryBuilder) LeftJoin(from string, on string) *QueryBuilder {
    self.makeList()
    q := buffer.Get()
    defer q.Close()

    q.WS("LEFT JOIN ").WS(from).WS(" ON (").WS(on).WS(")")

    self.join = append(self.join, q.String())

    return self
}

// TODOC
func (self *QueryBuilder) InnerJoin(from string, on string) *QueryBuilder {
    self.makeList()
    q := buffer.Get()
    defer q.Close()

    q.WS("INNER JOIN ").WS(from).WS(" ON (").WS(on).WS(")")

    self.join = append(self.join, q.String())

    return self
}

// TODOC
func (self *QueryBuilder) Where(where *GMap) *QueryBuilder {
    self.where = where
    return self
}

// TODOC
func (self *QueryBuilder) Group(groupBy string) *QueryBuilder {
    self.groupBy = groupBy
    return self
}

// TODOC
func (self *QueryBuilder) WithRollup() *QueryBuilder {
    self.rollup = true
    return self
}

// TODOC
func (self *QueryBuilder) Order(orderBy string) *QueryBuilder {
    self.orderBy = orderBy
    return self
}

// TODOC
func (self *QueryBuilder) Limit(limit int, offset ...int) *QueryBuilder {
    self.limit = limit
    if len(offset) > 0 { self.offset = offset[0] }
    return self
}

// TODOC
func (self *QueryBuilder) Union(union *QueryBuilder, all ...bool) *QueryBuilder {
    if self == union { panic("Self Reference UNION") }
    self.union = union
    if len(all) > 0 { self.unall = all[0] }
    return self
}

// TODOC
func (self *QueryBuilder) SelectQuery() (string, []interface{}) {
    argv := make([]interface{}, 0)
    stmt := buffer.Get()
    defer stmt.Close()
    if self.cols == "" { self.cols = "*" }
    stmt.WS("   SELECT ").WS(self.cols).NL()
    stmt.WS("     FROM ").WS(self.from)
    if self.join != nil {
	    for _, v := range self.join {
            stmt.NL().WS(v)
        }
    }
    if self.where != nil {
        stmt.NL().WS("    WHERE ")
        SQL.WhereQuery(self.driver, stmt, self.where, &argv)
    }
    if self.groupBy != "" {
        stmt.NL().WS(" GROUP BY ").WS(self.groupBy)
    }
    if self.rollup {
        stmt.NL().WS("WITH ROLLUP")
    }
    if self.orderBy != "" {
        stmt.NL().WS(" ORDER BY ").WS(self.orderBy)
    }
    if self.limit > 0 {
        if self.driver == ORACL {
            _sql := stmt.String()
            stmt.Reset()
            stmt.WS("SELECT B.* FROM (SELECT A.*, rownum AS SQL_ROWNUM FROM (").NL().WS(_sql).NL().WS(") A) B WHERE ")
            if self.offset > 0 {
                stmt.WS("B.SQL_ROWNUM > ").WS(strconv.Itoa(self.offset))
                if self.limit > 0 {
                    stmt.WS(" AND B.SQL_ROWNUM <= ").WS(strconv.Itoa(self.offset + self.limit))
                }
            } else {
                stmt.WS(" B.SQL_ROWNUM <= ").WS(strconv.Itoa(self.limit))
            }
        } else {
            stmt.NL().WS("    LIMIT ")
            if self.offset > 0 {
                stmt.WS(strconv.Itoa(self.offset)).WRune(',')   // MySQL: LIMIT offset, limit
            }
            stmt.WS(strconv.Itoa(self.limit))
        }
    }
    if self.union != nil {
        stmt.NL().WS(" UNION")
        if self.unall { stmt.WS(" ALL") }
        slct, args := self.union.SelectQuery()
        argv = append(argv, args...)
        stmt.NL().WS(slct)
    }
    return stmt.String(), argv
}

// TODOC
func (self *Record) TableName() string {
    return to.SnakeCase(reflect.TypeOf(self).Elem().Name())
}
