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

// controller mengatur flow eksekusi Service, memastikan Service yang dieksekusi
// memenuhi semua requirements dan constraints terkait payload, rules dan ACL
package tlkm

import (
    "encoding/json"
    "errors"
    "fmt"
    "io/ioutil"
    "mime"
    "net/http"
	"net/url"
    "reflect"
	"strconv"
    "strings"
    "sync"
    "time"
    "github.com/tlkm/is"
    "github.com/tlkm/to"
)

type (
    // ** private **
    // controller private hanya untuk package tlkm (sebelumnya public)
    controller struct {
        *Logger
        fileHandler http.Handler    // semua request non-handler akan dianggap request ke static resources
        syncPool    sync.Pool       // context pool
    }

    // ** private **
    // mapping parameter -> constraint
    Argument struct {
        Required, Logged    bool
        Minl, Maxl    int
        Minv, Maxv    int
        Enum    string
        Coerce  string
        Type    string
    }
)

// Inisialisasi static (resource) handler dan context pool
func (self *controller) init(fp string, dev bool) {

    // create instance http.FileServer untuk melayani resources /www/* (non-handlers)
    self.fileHandler = http.FileServer(http.Dir(fp))

    // Context menggunakan sync.Pool karena property (dan proses) yang dilakukan,
    // jika setiap request/hit membuat instance baru, hanya akan menambah latency GC
	self.syncPool.New = func() interface{} {
		return &Context{}
	}
}

// struct tidak support indexing
func (self *controller) getCacheKey(cache cacheKey, method httpMethod) string {
    switch method {
    case doGET:
        return cache.GET
    case doPOST:
        return cache.POST
    case doPUT:
        return cache.PUT
    case doDELETE:
        return cache.DELETE
    case doGRID:
        return cache.GRID
    case doHTML:
        return cache.HTML
    case doJSON:
        return cache.JSON
    case doFILE:
        return cache.FILE
    case doTEXT:
        return cache.TEXT
    }
    return ""
}

// Membentuk struktur data BMap (map[string]bool) rule dan return yang diharapkan.
// Key hanya helper untuk lookup Map agar eksekusi dijalankan sesuai urutan yang diharapkan
//
// Hasil akhir akan disimpan dalam cache untuk memastikan fungsi lookup rules untuk
// path dan call yang sama hanya dijalankan 1x
func (self *controller) handlerRules(conn *Connection, path, call string, method httpMethod) (Key List, Map BMap) {
    nv, _ := servKey[path]
    ns := self.getCacheKey(nv.rule, method)
    if ns == "" { return }

    // return dari cache jika path + call yang sama sudah dilakukan. Ada/tidak rules, proses
    // dipastikan hanya dilakukan 1x. Artinya jika sebuah handler tidak memiliki rules, maka
    // Key dan Map bernilai nil
    e := false
    if Key, Map, e = Cache.KeyBMap(ns); e { return }

    // Karena output proses ini akan masuk cache, pertama yang harus dipastikan adalah
    // handler punya rules atau tidak: inisialisasi Key dan Map
    stmt := `SELECT COUNT(*) T
       FROM st_handlers a
       JOIN st_handler_rules b ON (a.PID=b.PID AND a.HID=b.HID AND b.MID=?)
       JOIN st_rules c ON (b.PID=c.PID AND b.RID=c.RID)
      WHERE a.SRC=? AND b.USED='1' AND c.USED='1'`
    rchk := conn.Query(stmt, call, path)
    defer rchk.Close()

    if rchk.Next() {    // jika handler tidak memiliki rules, Key dan Map nil
        if cnt := rchk.Int("T"); cnt > 0 {
            stmt = `SELECT c.SRC, f.TBL, f.COL
       FROM st_handlers a
       JOIN st_handler_rules b ON (b.PID=a.PID AND b.HID=a.HID AND b.MID=?)
       JOIN st_rules c ON (c.PID=b.PID AND c.RID=b.RID)
       JOIN st_rule_arguments d ON (d.PID=b.PID AND d.RID=b.RID)
  LEFT JOIN st_handler_arguments e ON (e.PID=a.PID AND e.HID=a.HID AND e.TID=d.TID AND e.CID=d.CID AND e.MID=b.MID)
       JOIN st_metadata f ON (f.TID=d.TID AND f.CID=d.CID)
      WHERE a.SRC=? AND b.USED='1' AND c.USED='1' AND d.REQUIRED='1' AND (e.REQUIRED='0' OR e.CID IS NULL)`
            rows := conn.Query(stmt, call, path)
            for rows.Next() {
                SRC := rows.String("SRC")
                ruleRef[SRC] = Sprintf(" required %s (%s) IS NOT NULL", rows.String("TBL"), rows.String("COL"))
            }
            rows.Close()

            Key = make(List, 0)
            Map = make(BMap)

            stmt = `SELECT c.SRC, b.EXPR
               FROM st_handlers a
               JOIN st_handler_rules b ON (a.PID=b.PID AND a.HID=b.HID AND b.MID=?)
               JOIN st_rules c ON (b.PID=c.PID AND b.RID=c.RID)
              WHERE a.SRC=? AND b.USED='1' AND c.USED='1'
           ORDER BY b.SEQ`
            rows = conn.Query(stmt, call, path)
            for rows.Next() {
                SRC := rows.String("SRC")
                Key = append(Key, SRC)
                Map[SRC] = false
                if EXPR := rows.Int("EXPR"); EXPR == 1 {
                    Map[SRC] = true
                }
            }
            rows.Close()
        }
    }
    Cache.Set(ns, KeyBMap{Key: Key, Map: Map}) // simpan hasil dalam cache untuk lookup
    return
}

// Membentuk struktur data Map (map[string]Argument) parameter sesuai tabel st_handler_arguments
//
// Tidak ada kebutuhan untuk memproses arguments/parameter sesuai urutan, dengan asumsi bahwa
// payload hanya mensyaratkan constraint masing2 field, bukan urutan
func (self *controller) handlerArguments(conn *Connection, path, call string, method httpMethod) (list map[string]Argument) {
    nv, _ := servKey[path]
    ns := self.getCacheKey(nv.argv, method)
    if ns == "" { return }

    // return dari cache jika path + call yang sama sudah dilakukan. Ada/tidak arguments, proses
    // dipastikan hanya dilakukan 1x. Artinya jika sebuah handler tidak memiliki arguments, maka
    // list adalah empty map
    if object, e := Cache.Get(ns); e {
        list = object.(map[string]Argument)
        return
    }

    list = make(map[string]Argument)

    stmt := `SELECT c.COL COLN, c.COLT, c.MINL, c.MAXL, c.MINV, c.MAXV, c.ENUM, b.COERCED, b.REQUIRED, b.LOGGED
       FROM st_handlers a
       JOIN st_handler_arguments b ON (a.PID=b.PID AND a.HID=b.HID AND b.MID=?)
       JOIN st_metadata c ON (b.TID=c.TID AND b.CID=c.CID)
      WHERE a.SRC=?`
    rows := conn.Query(stmt, call, path)
    defer rows.Close()
    for rows.Next() {
        COLN := rows.String("COLN")
        COLT := rows.String("COLT")
        COERCED := rows.String("COERCED")
        ENUM := rows.String("ENUM")
        data := Argument{Required: false, Logged: false, Minl: -1, Maxl: -1, Minv: -1, Maxv: -1, Enum: "",}
        data.Type = COERCED
        if i := rows.Int("REQUIRED"); i == 1 { data.Required = true }
        if i := rows.Int("LOGGED"); i == 1 { data.Logged = true }
        if i := rows.Int("MINL"); i != 0 { data.Minl = i }
        if i := rows.Int("MAXL"); i != 0 { data.Maxl = i }
        if i := rows.Int("MINV"); i != 0 { data.Minv = i }
        if i := rows.Int("MAXV"); i != 0 { data.Maxv = i }
        if ENUM != "" { data.Enum = ENUM }
        if COERCED != "" {
            data.Coerce = COERCED
        } else {
            if COLT == "DATE" { data.Coerce = COLT }
        }
        list[COLN] = data
    }
    Cache.Set(ns, list) // simpan hasil dalam cache untuk lookup
    return
}

// Idenya adalah daripada mengirim response error karena data tidak memenuhi constraint,
// kita paksa parameter memenuhi constraint dan menerima apapun hasil coerce sebagai
// data final parameter
//
// ex: client mengirim 123x45 untuk constraint digit, hasil akhir coerce: 12345
func (self *controller) coerce(ctx *Context, enum, name, value string) error {
    l := len(value)
    switch enum {
    case "NUMERIC":
        if !is.Numeric(value) { ctx.Set(name, to.Numeric(value)) }
    case "DIGIT":
        if !is.Digit(value) { ctx.Set(name, to.Digit(value)) }
    case "DATE":
        if l < 8 || l > 10 {
            return errors.New(Sprintf("expected (%s) date length 8|10. received %s", name, strconv.Itoa(l)))
        }
        exp := errors.New(Sprintf("expected (%s) date. received %s", name, value))
        var d string
        switch l {
            case 10:
                if is.Digit(value) { return exp }
                _, _, _, _, d = to.DateSplit(value)
            case 9,8:
                if l == 8 && is.Digit(value) { // diijinkan untuk dikirim tanpa separator dalam format Ymd
                    d = Sprintf("%s-%s-%s", value[:4], value[4:6], value[6:])
                    if _, err := time.Parse("2006-01-02", d); err != nil {
                        d = Sprintf("%s-%s-%s", value[4:], value[2:4], value[:2])
                    }
                } else {
                    _, _, _, _, d = to.DateSplit(value) // asumsi m/d hanya 1 digit
                }
        }
        if _, e := time.Parse("2006-01-02", d); e != nil { return e }
        ctx.Set(name, d)
    }
    return nil
}

// Validasi payload berdasarkan mapping constraint ke argument. Yang akan diproses hanya
// parameter yang memiliki constraint (didefinisikan di st_handler_arguments)
func (self *controller) validate(args map[string]Argument, ctx *Context, log bool, USR string) (e error) {
    var m SMap
    if log {
        m = make(SMap)
        // Untuk memastikan Logging akan dieksekusi. Yang perlu diperhatikan adalah
        // (saat ini) yang akan di log hanya parameter yang masuk sebagai argument (ditentukan constraint nya)
        // bukan payload secara keseluruhan
        defer func() {
            if len(m) > 0 && self.logLv >= INFO {
                if j, err := json.Marshal(m); err == nil {
                    self.Log(INFO, string(j), ctx.Request.URL.Path, USR, ctx.ClientIP())
                }
            }
        }()
    }
    z := make([]string, 0)
    for n, r := range args {
        if !ctx.Exists(n) {
            if r.Required { // rule 1: seperti apapun datanya, mandatory tidak boleh null
                z = append(z, Sprintf("expected (%s) is not null", n))
            }
            continue // hanya disyaratkan memenuhi contraint selama tidak null
        }
        v := ctx.Get(n)
        if v == "" && r.Required {  // rule 2: argument mandatory, tidak null tapi juga tidak boleh kosong
            z = append(z, Sprintf("expected (%s) is not empty", n))
        }
        if log && r.Logged { m[n] = v }

        // constraint awal paling sederhana: panjang minimal/maksimal data parameter
        l := len(v)
        if r.Minl > 0 && l < r.Minl {
            z = append(z, Sprintf("expected (%s) min-length %s. received %s (%s)", n, strconv.Itoa(r.Minl), v, strconv.Itoa(l)))
        }
        if r.Maxl > 0 && l > r.Maxl {
            z = append(z, Sprintf("expected (%s) max-length %s. received %s (%s)", n, strconv.Itoa(r.Maxl), v, strconv.Itoa(l)))
        }
        switch r.Type {
        case "TINYINT","SMALLINT","MEDIUMINT","INT","BIGINT","DECIMAL","NUMERIC":
            if !is.Numeric(v) {
                z = append(z, Sprintf("expected (%s) type of %s. received %s", n, r.Type, v))
            }
        case "YEAR","DIGIT":
            if !is.Digit(v) {
                z = append(z, Sprintf("expected (%s) type of %s. received %s", n, r.Type, v))
            }
        case "ENUM":
            if r.Enum != "" {
                if strings.Index(r.Enum, v) == -1 {
                    z = append(z, Sprintf("expected (%s) enum of %s. received %s", n, r.Enum, v))
                }
            }
        }
        // constraint (integer/currency) range minimal dan maksimal
        if r.Minv != -1 || r.Maxv != -1 {
            u, err := strconv.Atoi(v)
            if err != nil {
                z = append(z, Sprintf("expected (%s) as int, strconv.Atoi(%s) error: %s", n, v, err.Error()))
            }
            if r.Minv >= 0 && u < r.Minv {
                z = append(z, Sprintf("expected (%s) min %s. received %s", n, strconv.Itoa(r.Minv), v))
            }
            if r.Maxv >= 0 && l > r.Maxv {
                z = append(z, Sprintf("expected (%s) max %s. received %s", n, strconv.Itoa(r.Maxv), v))
            }
        }
        // coerce digit/numeric
        if r.Coerce != "" {
            if err := self.coerce(ctx, r.Coerce, n, v); err != nil {
                z = append(z, Sprintf("coerce error: %s", err.Error()))
            }
        }
    }
    if len(z) > 0 {
        e = errors.New(strings.Join(z, "\r\n"))
    }
    return
}

// Tujuan utamanya adalah mengenkapsulasi payload (url-encoded ataupun json) kedalam map
//
// Sesuai spesifikasi standar HTTP, payload hanya berlaku untuk http-method POST/PUT. Payload untuk
// http-method selain POST/PUT akan di-drop oleh browser (default client)
func (self *controller) NewContext(w http.ResponseWriter, r *http.Request, conn *Connection, methodName string) (*Context, error) {
	ctx := self.syncPool.Get().(*Context)
    // inisialisasi awal diperlukan karena context akan dikembalikan ke sync.Pool
    // untuk digunakan oleh request yang lain
	ctx.Response = w
	ctx.Request = r
    ctx.SID = ""
    ctx.newSID = ""
    ctx.Values = url.Values{}
    ctx.Files = nil
    ctx.sesMap = nil
    ctx.sesCreate = false
    ctx.sesUpdate = false
    ctx.json = nil
    ctx.call = ""
    ctx.method = doPATH
    ctx.sent = false
    ctx.exit = false
    ctx.code = StatusNoContent
    if e := ctx.sessionStart(conn); e != nil { return ctx, e }  // JWT
    for k, v := range r.URL.Query() {   // tidak dibedakan parameter dikirim via query atau body
        for _, j := range v {
            ctx.Values.Add(k, j)
        }
    }
    invoke := r.Method
    if method, v := doKey[invoke]; v {
        switch method {
        case doGET:
            if k, v := doKey[methodName]; (v && k > doDELETE) || (!v && methodName != "") { invoke = methodName }
        case doPOST,doPUT:
            contentType := r.Header.Get("Content-Type")
            if contentType == "" {
                contentType = "application/octet-stream"
            }
            contentType, _, _ = mime.ParseMediaType(contentType)
            switch contentType {
            case "application/json":
                b, err := ioutil.ReadAll(r.Body)
                defer r.Body.Close()
                if err != nil { return ctx, err }
                if len(b) > 0 {
                    var j interface{}
                    if err := json.Unmarshal(b, &j); err != nil { return ctx, err }
                    argv := j.(GMap)
                    for k, v := range argv {
                        switch v.(type) {
                        case string:
                            ctx.Values.Add(k, strings.TrimSpace(v.(string)))
                        case bool, float64:
                            ctx.Values.Add(k, fmt.Sprint(v))
                        case nil:
                            ctx.Values.Add(k, "")
                        case GMap, []interface{}:
                            val, err := json.Marshal(v)
                            if err == nil {
                                ctx.Values.Add(k, string(val))
                            }
                        }
                    }
                }
            case "application/x-www-form-urlencoded":
                r.ParseForm()
                for key, vals := range r.PostForm {
                    if strings.Index(key, "[") <= 0 {
                        for _, val := range vals {
                            ctx.Values.Add(key, strings.TrimSpace(val))
                        }
                    } else {
                        // TODO: parseMap
                        self.parseMap(ctx, key, vals)
                    }
                }
            case "multipart/form-data":
                r.ParseMultipartForm(1024 * 1024 * 16)
                for key, vals := range r.MultipartForm.Value {
                    if strings.Index(key, "[") <= 0 {
                        for _, val := range vals {
                            ctx.Values.Add(key, strings.TrimSpace(val))
                        }
                    } else {
                        // TODO: parseMap
                        self.parseMap(ctx, key, vals)
                    }
                }
                for key, files := range r.MultipartForm.File {
                    if len(files) != 0 {
                        ctx.Files[key] = files[0]
                    }
                }
            }
        }
    }
    if ctx.Exists("GID") {
        ctx.GID = ctx.Get("GID")
    } else {
        GID := r.Header.Get("GID")
        if GID != "" {
            ctx.GID = GID
        }
    }
    ctx.call = invoke
    if method, v := doKey[ctx.call]; v {    // selain service interface, semua method masuk doPATH
        ctx.method = method
    }
    USR, logged := ctx.SessionUser()
    var e error
    if r.URL.Path != FileSeparator {
        // TODO: doPATH bisa dienhance mapping group ke path
        if ctx.method != doPATH {
            if argv := self.handlerArguments(conn, r.URL.Path, ctx.call, ctx.method); argv != nil {
                e = self.validate(argv, ctx, logged, USR)
            }
        }
    }

    return ctx, e
}

func (self *controller) parseMap(ctx *Context, key string, vals []string) {
    // TODO:
}

// Kembalikan Context ke sync.Pool, termasuk recovery untuk kondisi panic yang tidak
// tertangani oleh handler
func (self *controller) Recover(w http.ResponseWriter, ctx *Context) {
    self.syncPool.Put(ctx)
    if r := recover(); r != nil {
        m := ""
        switch v := r.(type) {
        case string:
            m = v
        default:
            m = fmt.Sprintf("%+v", v)
        }
        if m != "" {
            if is.Numeric(m) {
                if code, err := strconv.Atoi(m); err == nil {
                    w.WriteHeader(code)
                }
            } else {
                w.Write([]byte(m))
            }
        }
    }
}

// helper pada proses controller::ServeHTTP
func (self *controller) sendError(w http.ResponseWriter, c int, e string) {
    w.WriteHeader(c)
    w.Write([]byte(e))
}

// *** request-response dimulai dari sini ***
func (self *controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if r.Method == doMap[dOPTIONS] {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "POST,PUT,GET,OPTIONS,DELETE")
        w.Header().Set("Access-Control-Allow-Headers", "*")
        w.Header().Set("Access-Control-Allow-Credentials", "true")
        w.Header().Set("Access-Control-Expose-Headers", "Authorization")
        return
    }
    methodName := ""
    handler, v := servMap[r.URL.Path]
    if !v { // semua request yang tidak memiliki default handler *harus* GET dan akan ditangani fileHandler
        if r.Method != doMap[doGET] {
            if r.Method == doMap[dOPTIONS] {
                w.Write([]byte("allowed"))
                return
            }
            self.sendError(w, StatusMethodNotAllowed, StatusText(StatusMethodNotAllowed))
            return
        }
        // Asumsi awal adalah static resources. Karena method GET memiliki fleksibilitas
        // bisa dimapping langsung kedalam path, jika ini yang terjadi, handler harus
        // dicari dengan menghilangkan nama path setelah handler. Jika ditemukan, skip asumsi awal
        resFile := true
        if indx := strings.LastIndex(r.URL.Path, FileSeparator); indx > 0 {
            obj := r.URL.Path[:indx]
            if handler, v = servMap[obj]; v {
                methodName = r.URL.Path[indx+1:] // abaikan dulu methodName bisa dipanggil/tidak
                r.URL.Path = obj    // update URL.Path
                resFile = false
            }
        }
        if resFile {
            // asumsi (saat ini) semua resource di bawah www/* adalah public
            //
            // TODO: enhance mekanisme akses berdasarkan ACL, mungkin mapping package/group ke resources??
            self.fileHandler.ServeHTTP(w, r)
            return
        }
    }

    // Parameter pertama (*Connection) akan dilookup berdasarkan nama package/modul
    packageName := PackageSystem
    if indx := strings.Index(r.URL.Path[1:], FileSeparator) + 1; indx > 0 {
        packageName = r.URL.Path[1:indx]
    }

    // default connection: system
    conn := SQL.Default()
    defer conn.Close()

    // error atau tidak, Context diambil dari sync.Pool dan harus dikembalikan
    ctx, err := self.NewContext(w, r, conn, methodName)
    defer self.Recover(w, ctx) // oleh karena itu, defer setelahnya
	if err != nil { // sebelum return kalau ada error
        self.sendError(w, StatusPreconditionFailed, err.Error())
        return
    }

    // ** Check secure flag **
    //
    // Proses ini dilakukan sebagai screening tahap awal sebuah handler secure
    // (client harus login) atau tidak
    //
    // Hanya memastikan bahwa jika handler secure, client harus teridentifikasi
    // sebagai user
    isUser := false
    secure := false
    if b, v := servRef[r.URL.Path]; v { // informasi dari ServiceProperty
        ctx.PID = b.PID
        ctx.HID = b.HID
        _, e := ctx.SessionUser()
        isUser = e
        secure = b.SEC
        if secure && !isUser {
            self.sendError(w, StatusUnauthorized, StatusText(StatusUnauthorized))
            return
        }
    }

    // ** Check Role dan ACL **
    //
    // Proses (sebenarnya) jika sudah dipastikan bahwa handler yang akan dipanggil adalah
    // secure handler (dari tahab sebelumnya) maka framework harus memastikan bahwa
    // user memiliki hak akses terhadap method handler
    //
    // Hak akses pada framework dipetakan kedalam 2 map (session):
    //  1. GID  berisi user groups
    //  2. ACL  flat map yang berisi default ACL setiap group
    //
    // Mengingat user bisa memiliki beberapa group yang boleh jadi menunjuk pada
    // handler yang sama dengan ACL berbeda, user diwajibkan mengirim GID yang akan
    // digunakan untuk melakukan transaksi
    if isUser && secure && r.URL.Path != FileSeparator {
        switch ctx.method {
        case doGET,doPOST,doPUT,doDELETE:
            g, v := ctx.Session("GID")
            if ctx.GID == "" || !v {
                self.sendError(w, StatusUnauthorized, "EmptyGIDException: expected parameter GID")
                return
            }

            // Group/Role yang dikirim harus ada dalam list user groups
            GID := g.(map[string]string)
            if _, v := GID[ctx.GID]; !v {
                self.sendError(w, StatusBadRequest, "InvalidGIDException: " + ctx.GID)
                return
            }

            // ACL berlaku (hanya) jika resource diatur/termapping kedalam group/role, karena
            // mewajibkan semua resource (harus) termapping, selain tidak efektif juga akan
            // meribetkan administrator dan proses audit
            IDX := ctx.GID + ctx.PID + ctx.HID
            if g, v := ctx.Session("ACL"); v {
                ACL := g.(map[string]string)
                if m, v := ACL[IDX]; v {
                    if i, v := doKey[ctx.call]; v && i <= doDELETE {
                        if m[i] == '0' {
                            self.sendError(w, StatusUnauthorized, Sprintf("GID (%s) Does Not Have (%s) ACL", ctx.GID, ctx.call))
                            return
                        }
                    }
                }
                // TODO: regex untuk class method (bebas) yang dipanggil via HTTP GET
                //
                // Tidak terlalu urgent karena GET (dengan path bebas) secara
                // umum ditujukan untuk return data parameter
            }
        }
    }

    // ** Eksekusi ServiceRule **
    //
    // Jika sebuah handler memiliki rules, akan dieksekusi tepat sebelum method handler
    // dieksekusi. Rule harus memastikan bahwa return error akan dikembalikan jika (dan hanya jika)
    // output tidak sesuai dengan expected-return
    if rkey, rmap := self.handlerRules(conn, r.URL.Path, ctx.call, ctx.method); rkey != nil && rmap != nil {
        for ridx, _ := range rkey {
            name := rkey[ridx]  // rkey[ridx] untuk memastikan lookup pada ruleMap sesuai urutan rule di database
            if rref, v := ruleRef[name]; v {
                self.sendError(w, StatusFailedDependency, name + rref)
                return
            }
            robj, v := ruleMap[name]
            if !v {
                self.sendError(w, StatusFailedDependency, "RuleNotFoundException: " + name)
                return
            }
            // rule yang sama bisa memiliki expected-return berbeda tergantung kebutuhan
            // diposisi mana rule dipanggil dalam workflow/proses
            EXPR, _ := rmap[name]
            if e := robj.Execute(conn, ctx, EXPR); e != nil {
                self.sendError(w, StatusExpectationFailed, e.Error())
                return
            }
        }
    }

    // jika ditemukan datasource sesuai nama package, database connection akan disesuaikan
    if packageName != PackageSystem && SQL.Exists(packageName) {
        conn.Close()
        conn = SQL.Lookup(packageName)
        defer conn.Close()
    }

    // ** Eksekusi Service/Handler **
    //
    // Karena performansi reflect tidak akan sebagus direct call, reflect dilakukan jika
    // (dan hanya jika) method yang akan di-invoke diluar standar interface Service
    // dan dipanggil via http method GET (umumnya digunakan untuk handler2 non-transaksi)
    switch ctx.method {
    case doGET:
        handler.GET(conn, ctx)
    case doPOST:
        handler.POST(conn, ctx)
    case doPUT:
        handler.PUT(conn, ctx)
    case doDELETE:
        handler.DELETE(conn, ctx)
    case doGRID:
        handler.GRID(conn, ctx)
    case doHTML:
        handler.HTML(conn, ctx)
    case doJSON:
        handler.JSON(conn, ctx)
    case doFILE:
        handler.FILE(conn, ctx)
    case doTEXT:
        handler.TEXT(conn, ctx)
    default:
        v := reflect.ValueOf(handler).MethodByName(ctx.call)
        if v.IsValid() {
            i := make([]reflect.Value, 2)
            i[0] = reflect.ValueOf(conn)
            i[1] = reflect.ValueOf(ctx)
            v.Call(i)
        } else {
            http.NotFound(w, r)
        }
    }

    // ** Check Buffer **
    //
    // Jika sebuah handler tidak melakukan operasi Write/Echo, maka framework akan
    // mengasumsikan response yang akan dikirim adalah format json
    //
    // Struktur default yang digunakan oleh framework bisa di-replace melalui method
    // Context.JSON()
    if !ctx.sent {
        if ctx.json != nil {
            b, e := json.Marshal(ctx.json)
            if e == nil {
                ctx.ContentType(ContentTypeJSON).Write(b)
            } else {
                ctx.ContentType(ContentTypeTEXT)
                self.sendError(w, StatusInternalServerError, e.Error())
            }
        } else {
            w.Write([]byte(StatusText(ctx.code)))
        }
    }
}