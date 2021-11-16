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

package play

import (
    "fmt"
    "testing"
    "io/ioutil"
  _ "github.com/mysql"
    "net/http/httptest"
  . "tlkm"
)

func init() {
    SQL.Register("mysql.syst", "root:test@tcp(127.0.0.1:3306)/go")
}

func doTest(xmlname, methodName, namespace, params string, t *testing.T) {
    xmldata, e := ioutil.ReadFile("testdata/" + xmlname)
    if e != nil {
        t.Error(e)
        return
    }
    conn := SQL.Default()
    defer conn.Close()

    u := "http://127.0.0.1" + namespace + "?" + params
    r := httptest.NewRequest(methodName, u, nil)
    w := httptest.NewRecorder()
    cntx, _ := FrontController("D:/gdk", true, 4).NewContext(w, r, conn, methodName)

    if e := PlayParse(conn, cntx, namespace, xmldata); e != nil {
        t.Log(e.Error())
        t.Fail()
        return
    }
    e = PlayExecute(conn, cntx, namespace)
    if e != nil {
        t.Log(e.Error())
        t.Fail()
        return
    }
    z := w.Result()
    o, _ := ioutil.ReadAll(z.Body)
    fmt.Println("----------")
    fmt.Println("code:", z.StatusCode)
    fmt.Println("mime:", z.Header.Get("Content-Type"))
    fmt.Println("body:")
    fmt.Println("----------")
    fmt.Println(string(o))
}
