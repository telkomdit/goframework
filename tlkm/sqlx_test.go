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

package tlkm

import (
	"testing"

	_ "github.com/mysql"
)

func init() {
	SQL.Register("mysql.syst", "root:test@tcp(127.0.0.1:3306)/go")
}

func TestInsertQuery(t *testing.T) {
	stmt, argv := SQL.InsertQuery(MYSQL, "st_test", &GMap{"ID": 1, "MSG": "Value1"}, &GMap{"ID": 2, "MSG": "Value2"}, &GMap{"ID": 3, "MSG": "Value3"})
	t.Log(stmt)
	t.Log(argv)
}

func TestInsertIgnoreQuery(t *testing.T) {
	stmt, argv := SQL.InsertIgnoreQuery(MYSQL, "st_test", &GMap{"ID": 1, "MSG": "Value1"}, &GMap{"ID": 2, "MSG": "Value2"}, &GMap{"ID": 3, "MSG": "Value3"})
	t.Log(stmt)
	t.Log(argv)
}

func TestUpdateQuery(t *testing.T) {
	stmt, argv := SQL.UpdateQuery(MYSQL, "st_test", &GMap{"MSG": nil, "@BEGDA": "CURRENT_DATE"}, &GMap{"ID": "3", "MSG": "test"})
	t.Log(stmt)
	t.Log(argv)
}

func TestDeleteQuery(t *testing.T) {
	stmt, argv := SQL.DeleteQuery(MYSQL, "st_test", &GMap{"ID": "7", "MSG@LIKE": "%data%"})
	t.Log(stmt)
	t.Log(argv)
}

func TestBulkInsertQuery(t *testing.T) {
	bulk := SQL.BulkInsert("st_test", List{"ID", "MSG"})
	defer bulk.Close()

	bulk.Add(GList{1, "MSG1"}, GList{2, "MSG2"})
	bulk.Add(GList{3, "MSG3"}, GList{4, "MSG4"})
	t.Log(bulk.String())
}

func TestBulkInsertIgnoreQuery(t *testing.T) {
	bulk := SQL.BulkInsertIgnore("st_test", List{"ID", "MSG"})
	defer bulk.Close()

	bulk.Add(GList{1, "MSG1"}, GList{2, "MSG2"})
	t.Log(bulk.String())
}

// Hanya melihat hasil SQL yang dibentuk, tanpa melihat valid/tidak
func TestSQLBuilder(t *testing.T) {
	_sql := SQL.Builder()
	defer _sql.Close()
	_uni := SQL.Builder()
	defer _uni.Close()

	_sql.Select("A.USR", "A.NAME").From("TBLA A")
	_sql.Join("TBLB B", "B.KEY=A.KEY")
	_sql.Join("TBLC C", "C.KEY=A.KEY")
	_sql.Join("TBLD D", "D.KEY=A.KEY")
	_sql.LeftJoin("TBLE E", "E.KEY=A.KEY")
	_sql.Where(&GMap{"A.USR@LIKE": "test%", "B.USR": nil})
	_sql.Limit(10, 3)
	_sql.Group("A.USR").Order("A.USR")

	_uni.Select("X.USR", "X.NAME").From("TBLX X").Where(&GMap{"X.USR": "not-exist", "X.NAME": nil})

	_sql.Union(_uni, true)

	stmt, argv := _sql.SelectQuery()

	t.Log(stmt)
	t.Log(argv)
}

func TestTransaction(t *testing.T) {
	conn := SQL.Default()
	defer conn.Close()
	tx := conn.Begin()
	(&Go{
		Try: func() {
			tx.Exec("UPDATE st_users SET SHR=SHR+? WHERE USR=?", 1, "test")
			tx.Commit()
		},
		Catch: func(e Exception) {
			tx.Rollback()
			t.Error(e)
		},
	}).Run()
}

func TestQuery(t *testing.T) {
	conn := SQL.Default()
	defer conn.Close()
	rows := conn.Query("SELECT * FROM st_test WHERE ID>?", 3)
	defer rows.Close()
	for rows.Next() {
		t.Log(rows.String("ID"), rows.String("MSG"))
	}
}

type (
	Test struct {
		*Record
		ID  *int `PK:"true"`
		MSG *string
	}
	TestOne struct {
		*Record
		ID   *int `PK:"true" FK:"st_test.ID"`
		ATTR *string
	}
	TestMany struct {
		*Record
		ID    *int `PK:"true" FK:"st_test.ID"`
		ID_AT *int `PK:"true"`
		ATTR1 *string
		ATTR2 *string
		ATTR3 *string
	}
)

func (self *Test) TableName() string { return "st_test" }
func (self *Test) SetID(ID int)      { self.ID = &ID }
func (self *Test) GetID() int        { return *self.ID }
func (self *Test) SetMSG(MSG string) { self.MSG = &MSG }
func (self *Test) GetMSG() string    { return *self.MSG }

func (self *TestOne) TableName() string  { return "st_test_one" }
func (self *TestMany) TableName() string { return "st_test_many" }

func TestARMap(t *testing.T) {
	conn := SQL.Default()
	defer conn.Close()
	dt := make([]*Test, 0)
	rs := conn.Query("SELECT * FROM st_test WHERE ID=?", 3)
	defer rs.Close()
	for rs.Scan(&Test{}) {
		dt = append(dt, rs.Get().(*Test))
	}
	for st := range dt {
		t.Log(dt[st].GetMSG())
	}
}

func TestARLoadByPK(t *testing.T) {
	conn := SQL.Default()
	defer conn.Close()

	test := new(Test)
	test.SetID(3)
	if e := conn.LoadByPK(test); e == nil {
		t.Log(*test.MSG)
	}
}

func TestARLoadByFK(t *testing.T) {
	conn := SQL.Default()
	defer conn.Close()

	fr := new(Test)
	fr.SetID(4)

	to := new(TestOne)
	if e := conn.LoadByFK(fr, to); e == nil {
		t.Log(*to.ATTR)
	} else {
		t.Log(e.Error())
	}
}

/*func TestARFetchByFK(t *testing.T) {
    conn := SQL.Default()
    defer conn.Close()
    fr := new(Test)
    fr.SetID(4)
    to := new(TestMany)
    dt := make([]*TestMany, 0)
    rs := conn.FetchByFK(fr, to)
    defer rs.Close()
    for rs.Scan(&TestMany{}) {
        dt = append(dt, rs.Get().(*TestMany))
    }
    for st := range dt {
        t.Log(dt[st].GetMSG())
    }
}*/

func TestARSave(t *testing.T) {
	conn := SQL.Default()
	defer conn.Close()

	test := new(Test)
	test.SetMSG("Test Message")

	rs, _ := conn.Save(test)
	if affected, _ := rs.RowsAffected(); affected > 0 {
		t.Log("MSG: " + *test.MSG)
	}
}

func TestARUpdate(t *testing.T) {
	conn := SQL.Default()
	defer conn.Close()

	test := new(Test)
	test.SetID(12)
	test.SetMSG("New Message")
	if rs, er := conn.Update(test); er != nil {
		t.Log(er.Error())
	} else {
		if affected, _ := rs.RowsAffected(); affected > 0 {
			t.Log("Berhasil Update")
		}
	}
}

func TestARDelete(t *testing.T) {
	conn := SQL.Default()
	defer conn.Close()

	test := new(Test)
	test.SetID(12)
	if rs, er := conn.Delete(test); er != nil {
		t.Log(er.Error())
	} else {
		if affected, _ := rs.RowsAffected(); affected > 0 {
			t.Log("Berhasil Delete")
		}
	}
}
