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

// Untuk memenuhi kebutuhan transparansi dalam pengembangan rule/handler, framework
// menggunakan pendekatan scratch programming: https://id.wikipedia.org/wiki/Scratch_(bahasa_pemrograman)
// sebagai prototype rule/handler
//
// Tujuan akhir interpreter adalah menggenerate source-code go untuk bisa dicompile
// sebagai native handler mereplace namespace yang sebelumnya ditangani oleh interpreter.
// Idealnya, iterasi dari scratch (prototype) ke go (native) tidak melibatkan developer
// kecuali untuk pengembangan komponen/fungsi spesifik
//
// Jika performansi bukan issue, disarankan untuk menggunakan interpreter sebagai
// handler karena framework akan fokus kepada pengembangan interpreter untuk otomatisasi
// pattern2 transaksi
package tlkm

import (
    "encoding/xml"
    "errors"
    "strconv"
    "strings"
	"sync"
    "github.com/telkomdit/goframework/buffer"
    "github.com/telkomdit/goframework/to"
)

type (
    // definisi variables, kecuali kedepan ada kebutuhan untuk strict, sementara akan
    // kita skip, map untuk global variable kita create per-context
    PlayVars struct {
        Argv    []string    `xml:"variable"`
    }

    // untuk block input yang sifatnya fixed (tanpa traverse)
    PlayField struct {
        Name    string  `xml:"name,attr"`
        Value   string  `xml:",chardata"`
    }

    // generic input yang harus traverse
    PlayValue struct {
        Name    string  `xml:"name,attr"`
        Tree    []PlayBlock `xml:"block"`
    }

    // block statement
    PlayStat struct {
        Name    string  `xml:"name,attr"`
        Tree    []PlayBlock `xml:"block"`
    }

    // block mutation>arg
    PlayArgv struct {
        Name    string  `xml:"name,attr"`
    }

    PlayMutation struct {
        At0     bool    `xml:"at,attr"`
        At1     bool    `xml:"at1,attr"`
        At2     bool    `xml:"at2,attr"`
        Elif    int     `xml:"elseif,attr"`
        Else    int     `xml:"else,attr"`
        Items   int     `xml:"items,attr"`
        Mode    string  `xml:"mode,attr"`
        Stat    bool    `xml:"statement,attr"`
        Stats   bool    `xml:"statements,attr"`
        Name    string  `xml:"name,attr"`
        Argv    []PlayArgv  `xml:"arg"`
    }

    // ast node
    PlayBlock struct {
        Type    string      `xml:"type,attr"`
        Fields  []PlayField `xml:"field"`
        Values  []PlayValue `xml:"value"`
        Stats  []PlayStat  `xml:"statement"`
        Mutation    *PlayMutation    `xml:"mutation"`
        Next    *PlayBlock  `xml:"next>block"`
    }

    // root
    PlayXML struct {
        Vars    PlayVars    `xml:"variables"`
        Tree    []PlayBlock     `xml:"block"`
    }

    // seperti umumnya bahasa script, setiap variable (selama memungkinkan) bisa
    // kita konversi kesemua tipe yang disupport. assert diserahkan ke handler
    PlayType interface {
        String(*PlayContext) string
        Float(*PlayContext) float64
        Int(*PlayContext) int64
        Boolean(*PlayContext) bool
        List(*PlayContext) PlayList
        Map(*PlayContext) PlayMap
        EQ(*PlayContext, PlayType) bool
        LT(*PlayContext, PlayType) bool
    }

    // struktur data untuk function/procedure, dalam hal ini block statement (STACK)
    PlayFunc struct {
        Argv    []string
        Func    *PlayBlock
        Rtrn    *PlayBlock
    }

    // Enkapsulasi service interface CRUD(*Connection, *Context). Karena context diambil
    // dari pool, cara paling efektif untuk mengakses *Connection dan *Context
    // dengan membuat reference
    PlayContext struct {
        Conn    *Connection
        Cntx    *Context
        Argv    map[string]PlayType
        Func    map[string]PlayFunc
    }

    // versi interpreter dari native service
    PlayService struct {
        Func    map[string]PlayFunc
    }

    // signature handlers
    PlaySignature func(*PlayContext, *PlayBlock) PlayType
    PlayExportSignature map[string]PlaySignature

    // Interpreter types. List dan Map hanya bisa digunakan jika ada reference ke
    // global variable untuk memudahkan manipulasi oleh handler
    PlayBool bool
    PlayNumber float64
    PlayString string
    playNull struct {}

    PlayList struct {
	    T   *[]PlayType
    }

    // Ini yang paling ribet karena pada saat iterasi map, block tidak bisa menyimpan
    // pointer key. Yang paling mungkin dilakukan adalah menyimpan key dalam list
    // dan memperlakukan iterasi sama seperti list
    PlayMap struct {
	    T   *map[PlayType]PlayType
	    K   *[]PlayType     // key reference
        P   *int    // TODO: reset counter pada saat pertama kali iterasi
    }

    Player struct {
        pool  sync.Pool
    }
)

var (
    playMap = make(map[string]PlayService)
    playHandlers = make(PlayExportSignature)
    PlayNull playNull
    player = &Player{}
)

func init() {
	player.pool.New = func() interface{} {
		return &PlayContext{}
	}
}

func PlayExport(m PlayExportSignature) {
    for i, j := range m {
        playHandlers[i] = j
    }
}

func PlayExit(message ...interface{}) PlayType {
    m := ""
    l := len(message)
    if l > 0 {
        b := buffer.Get()
        defer b.Close()
        for i, j := range message {
            if i > 0 {
                b.WS(" ")
            }
            b.WS(to.String(j))
        }
        m = b.String()
    }
    panic(m)
    return PlayNull
}

func PlayFieldException(name string) PlayType { return PlayExit("FieldNotFoundException: ", name) }
func PlayBlockException(name string) PlayType { return PlayExit("BlockNotFoundException: ", name) }
func PlayFuncException(name string) PlayType { return PlayExit("FuncNotFoundException: ", name) }

func (self *PlayBlock) FieldVar() *PlayField { return self.Field("VAR") }
func (self *PlayBlock) FieldOper() *PlayField { return self.Field("OP") }
func (self *PlayBlock) FieldName() *PlayField { return self.Field("NAME") }
func (self *PlayBlock) FieldBool() *PlayField { return self.Field("BOOL") }
func (self *PlayBlock) FieldText() *PlayField { return self.Field("TEXT") }
func (self *PlayBlock) FieldNmbr() *PlayField { return self.Field("NUM") }
func (self *PlayBlock) FieldMode() *PlayField { return self.Field("MODE") }
func (self *PlayBlock) FieldFlow() *PlayField { return self.Field("FLOW") }
func (self *PlayBlock) Field(name string) *PlayField {
    for _, v := range self.Fields {
        if v.Name == name {
            return &v
        }
    }
    return nil
}

func (self *PlayBlock) ValueExists(name string) (v bool) {
    v = false
    if self.Value(name) != nil { v = true }
    return
}
func (self *PlayBlock) Value(name string) *PlayValue {
    for _, v := range self.Values {
        if v.Name == name {
            return &v
        }
    }
    return nil
}

func (self *PlayBlock) BlockText() *PlayBlock { return self.Block("TEXT") }
func (self *PlayBlock) BlockFrom() *PlayBlock { return self.Block("FROM") }
func (self *PlayBlock) BlockTo() *PlayBlock { return self.Block("TO") }
func (self *PlayBlock) BlockInc() *PlayBlock { return self.Block("BY") }
func (self *PlayBlock) BlockA() *PlayBlock { return self.Block("A") }
func (self *PlayBlock) BlockB() *PlayBlock { return self.Block("B") }
func (self *PlayBlock) BlockBool() *PlayBlock { return self.Block("BOOL") }
func (self *PlayBlock) BlockName() *PlayBlock { return self.Block("NAME") }
func (self *PlayBlock) BlockKey() *PlayBlock { return self.Block("KEY") }
func (self *PlayBlock) BlockValue() *PlayBlock { return self.Block("VALUE") }
func (self *PlayBlock) BlockIf() *PlayBlock { return self.Block("IF") }
func (self *PlayBlock) BlockThen() *PlayBlock { return self.Block("THEN") }
func (self *PlayBlock) BlockElse() *PlayBlock { return self.Block("ELSE") }
func (self *PlayBlock) BlockCondition() *PlayBlock { return self.Block("CONDITION") }
func (self *PlayBlock) Block(name string) *PlayBlock {
    v := self.Value(name)
    if v == nil {
        Exit("BlockNameException: ", name)
        return nil
    }
    if len(v.Tree) != 1 {
        Exit("SingleBlockException: ", name)
        return nil
    }
    return &v.Tree[0]
}

func (self *PlayBlock) Stack() *PlayBlock { return self.Stat("STACK") }
func (self *PlayBlock) Stat(name string) *PlayBlock {
    for _, v := range self.Stats {
        if v.Name == name {
            if len(v.Tree) != 1 {
                Exit("SingleBlockException:", name)
                return nil
            }
            return &v.Tree[0]
        }
    }
    return nil
}

func (self playNull) NullException() { Exit("NullAssertException") }
func (self playNull) String(p *PlayContext) (r string) { self.NullException(); return }
func (self playNull) Float(p *PlayContext) (r float64) { self.NullException(); return }
func (self playNull) Int(p *PlayContext) (r int64) { self.NullException(); return }
func (self playNull) Boolean(p *PlayContext) (r bool) { self.NullException(); return }
func (self playNull) List(p *PlayContext) (r PlayList) { self.NullException(); return }
func (self playNull) Map(p *PlayContext) (r PlayMap) { self.NullException(); return }
func (self playNull) EQ(p *PlayContext, t PlayType) (r bool) { _, r = t.(playNull); return }
func (self playNull) LT(p *PlayContext, t PlayType) (r bool) { return !self.EQ(p, t) }

func (self PlayNumber) String(p *PlayContext) string { return to.String(self) }
func (self PlayNumber) Float(p *PlayContext) float64 { return float64(self) }
func (self PlayNumber) Int(p *PlayContext) int64 { return int64(self) }
func (self PlayNumber) Boolean(p *PlayContext) bool { return float64(self) != 0 }
func (self PlayNumber) List(p *PlayContext) (r PlayList) { Exit("NumberAssertException"); return }
func (self PlayNumber) Map(p *PlayContext) (r PlayMap) { Exit("NumberAssertException"); return }
func (self PlayNumber) EQ(p *PlayContext, r PlayType) bool { return self == r }
func (self PlayNumber) LT(p *PlayContext, r PlayType) bool { return float64(self) < r.Float(p) }

func (self PlayString) String(p *PlayContext) string { return string(self) }
func (self PlayString) Float(p *PlayContext) (r float64) {
    f, e := strconv.ParseFloat(string(self), 64)
    if e != nil {
        Exit(e.Error())
        return
    }
    return f
}

func (self PlayString) Int(p *PlayContext) (r int64) {
    f, e := strconv.ParseInt(string(self), 64, 64)
    if e != nil {
        Exit(e.Error())
        return
    }
    return f
}
func (self PlayString) Boolean(p *PlayContext) bool { return string(self) != "" }
func (self PlayString) List(p *PlayContext) (r PlayList) { Exit("StringAssertException"); return }
func (self PlayString) Map(p *PlayContext) (r PlayMap) { Exit("StringAssertException"); return }
func (self PlayString) EQ(p *PlayContext, r PlayType) bool { return self == r }
func (self PlayString) LT(p *PlayContext, r PlayType) bool { return string(self) < r.String(p) }

func (self PlayBool) BoolException() { Exit("BoolAssertException") }
func (self PlayBool) String(p *PlayContext) string {
    if bool(self) {
        return "true"
    } else {
        return "false"
    }
}
func (self PlayBool) Float(p *PlayContext) (r float64) { self.BoolException(); return }
func (self PlayBool) Int(p *PlayContext) (r int64) { self.BoolException(); return }
func (self PlayBool) Boolean(p *PlayContext) bool { return bool(self) }
func (self PlayBool) List(p *PlayContext) (r PlayList) { self.BoolException(); return }
func (self PlayBool) Map(p *PlayContext) (r PlayMap) { self.BoolException(); return }
func (self PlayBool) EQ(p *PlayContext, r PlayType) bool { return self == r }
func (self PlayBool) LT(p *PlayContext, r PlayType) bool {
    if bool(self) {
        return false
    } else {
        return r.Boolean(p)
    }
}

func (self PlayList) ListException() { Exit("ListAssertException") }
func (self PlayList) String(p *PlayContext) (r string) {
	v := *self.T
	a := make([]string, len(v))
	for i, j := range v {
		a[i] = j.String(p)
	}
	return strings.Join(a, ",")
}
func (self PlayList) Float(p *PlayContext) (r float64) { self.ListException(); return }
func (self PlayList) Int(p *PlayContext) (r int64) { self.ListException(); return }
func (self PlayList) Boolean(p *PlayContext) (r bool) { self.ListException(); return }
func (self PlayList) List(p *PlayContext) PlayList { return self }
func (self PlayList) Map(p *PlayContext) (r PlayMap) { self.ListException(); return }
func (self PlayList) EQ(p *PlayContext, t PlayType) (r bool) {
    r = false
	if b, v := t.(PlayList); v {
        a := *self.T
        b := *b.T
        if len(a) == len(b) {
            r = true
            for i, j := range a {
                if j != b[i] {
                    r = false
                    break
                }
            }
        }
    }
	return
}
func (self PlayList) LT(p *PlayContext, t PlayType) (r bool) { return !self.EQ(p, t) }

func (self PlayMap) MapException() { Exit("MapAssertException") }
func (self PlayMap) String(p *PlayContext) (r string) { self.MapException(); return }
func (self PlayMap) Float(p *PlayContext) (r float64) { self.MapException(); return }
func (self PlayMap) Int(p *PlayContext) (r int64) { self.MapException(); return }
func (self PlayMap) Boolean(p *PlayContext) (r bool) { self.MapException(); return }
func (self PlayMap) List(p *PlayContext) (r PlayList) { self.MapException(); return }
func (self PlayMap) Map(p *PlayContext) PlayMap { return self }
func (self PlayMap) EQ(p *PlayContext, t PlayType) (r bool) {
    r = false
	if b, v := t.(PlayMap); v {
        a := *self.T
        b := *b.T
        if len(a) == len(b) {
            r = true
            for i, j := range a {
                k, l := b[i]
                if !l || k != j {
                    r = false
                    break
                }
            }
        }
    }
	return
}
func (self PlayMap) LT(p *PlayContext, t PlayType) (r bool) { return !self.EQ(p, t) }

// Sesuai struktur (xml:next>block) visit hanya menerima satu block dibawah sibling
// block yang dijalankan sebelumnya
func (self *PlayContext) Visit(b *PlayBlock) (r PlayType) {
    if m := playHandlers[b.Type]; m != nil {
        r = m(self, b)
        if b.Next != nil {
            r = self.Visit(b.Next)
        }
    } else {
        r = PlayExit("HandlerNotFound: ", b.Type)
    }
    return
}

// Entry poin Visit, dipanggil dari context pertama atau cross context
func (self *PlayContext) callFunc(m string) (r PlayType) {
    if v, e := self.Func[m]; e {
        r = self.Visit(v.Func)
    }
    return
}

// Eksekusi AST lain diluar current context. Database connection dan http context
// akan di delegasikan/copy dari context sebelumnya
//
// PlayContext (baru) dibuat untuk menghindari tumpang tindih (global) variable,
// method sebelumnya dengan method baru yang bisa berakibat tidak konsisten
// perilaku ketika dieksekusi secara mandiri dengan ketika dieksekusi dari namespace
// yang berbeda
//
// Diluar map (Argv) untuk menampung variable global, environment yang sama akan
// diteruskan dari context sebelumnya
func (self *PlayContext) Execute(namespace, methodName string) error {
    b, v := playMap[namespace]
    if !v {
        return errors.New("ASTNotFoundException: " + namespace)
    }
    p := player.getContext()
    defer p.close()
    p.Conn = self.Conn
    p.Cntx = self.Cntx
    p.Func = b.Func
    p.callFunc(methodName)

    return nil
}

// Jika eksekusi berhenti karena panic, output akan diteruskan ke http context
// sebelum play context dikembalikan ke sync.Pool
//
// Dua referensi service interface (*Connection, *Context) harus nil
func (self *PlayContext) close() {
    if r := recover(); r != nil { self.Cntx.Echo(to.String(r)) }
    self.Conn = nil
    self.Cntx = nil
    self.Argv = nil
    self.Func = nil
    player.pool.Put(self)
}

// Dengan asumsi semua field sudah nil pada saat kembali ke pool, Argv akan dibuat
// pada kesempatan pertama context diambil dari pool
func (self *Player) getContext() (v *PlayContext) {
	v = self.pool.Get().(*PlayContext)
    v.Argv = make(map[string]PlayType)    // global var
    return
}

// Map untuk http method diasumsikan tidak ada perubahan, hanya Map untuk global variables
// yang akan dibentuk ulang pada saat diambil dari pool
func (self *Player) execute(namespace string, conn *Connection, cntx *Context) error {
    b, v := playMap[namespace]
    if !v {
        return errors.New("ASTNotFoundException: " + namespace)
    }
    p := self.getContext()
    defer p.close()
    p.Conn = conn
    p.Cntx = cntx
    p.Func = b.Func
    p.callFunc(cntx.MethodName())

    return nil
}

func saveXML(conn *Connection, cntx *Context, namespace, methodName string, XML []byte) {
    PID, HID := ShortURL(namespace)
    USR, _ := cntx.SessionUser()
    GID := cntx.GID
    data := string(XML)
    rows := conn.Query("SELECT PID FROM st_plays WHERE PID=? AND HID=?", PID, HID)
    if !rows.Next() {
        _sql := "INSERT INTO st_plays(PID, HID, SRC, CREATED_AT, CREATED_BY, CREATED) VALUES (?, ?, ?, CURRENT_TIMESTAMP, ?, ?)"
        conn.Exec(_sql, PID, HID, namespace, USR, GID)
    }
    rows.Close()
    rows = conn.Query("SELECT REV FROM st_play_repos WHERE PID=? AND HID=? AND API=? ORDER BY REV DESC LIMIT 1", PID, HID, methodName)
    _rev := 1
    if rows.Next() {
        _rev = rows.Int("REV") + 1
    } else {
        conn.Exec("INSERT INTO st_play_methods(PID, HID, API, REV, XML) VALUES (?, ?, ?, ?, ?)", PID, HID, methodName, _rev, data)
    }
    rows.Close()
    _sql := "INSERT INTO st_play_repos(PID, HID, API, REV, XML, UPDATED_AT, CREATED_BY, CREATED) VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?, ?)"
    conn.Exec(_sql, PID, HID, methodName, _rev, data, USR, GID)
    if _rev > 1 {
        conn.Exec("UPDATE st_play_methods SET REV=?, XML=? WHERE PID=? AND HID=? AND API=?", _rev, data, PID, HID, methodName)
    }
}

// Umumnya hanya dipanggil 1x, atau jika diperlukan untuk mereplace AST sebelumnya
// pada saat development
func PlayParse(conn *Connection, cntx *Context, namespace string, XML []byte, repo... bool) (e error) {
	var (
        m PlayXML
        x PlayService
    )
    if e = xml.Unmarshal(XML, &m); e != nil { return }
    b := false
    x, b = playMap[namespace]
    if !b {
        x.Func = make(map[string]PlayFunc)
    }
    save := false
    if len(repo) > 0 { save = repo[0] }
    for _, v := range m.Tree {
        if v.Type != "procedures_defnoreturn" && v.Type != "procedures_defreturn" { continue }
        if f := v.Field("NAME"); f != nil {
            methodName := f.Value
            u := PlayFunc{}
            if v.Mutation != nil {
                for _, o := range v.Mutation.Argv {
                    u.Argv = append(u.Argv, o.Name)
                }
            }
            u.Func = v.Stack()
            r := v.Value("RETURN")
            if r != nil && len(r.Tree) > 0 { u.Rtrn = &r.Tree[0] }
            x.Func[methodName] = u
            if save {
                saveXML(conn, cntx, namespace, methodName, XML)
            }
        }
    }
    playMap[namespace] = x
    return
}

func PlayExecute(conn *Connection, cntx *Context, namespace string) error {
    return player.execute(namespace, conn, cntx)
}