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

// *** Abstraksi Framework ***
//
// Implementasi (dan porting) design pattern Front Controller yang digunakan
// di SIMPKBL PT. Telkom unit CDC (versi PHP5) kedalam golang
//
// Framework versi PHP5 sendiri masih digunakan (production) sampai versi awal
// porting ini ditulis
package tlkm

import (
    "context"
    "encoding/json"
    "github.com/judwhite/go-svc"
    "github.com/telkomdit/goframework/buffer"
    "github.com/telkomdit/goframework/to"
    "net/http"
    "os"
    "path"
    "reflect"
    "strconv"
    "sync"
    "time"
    "unicode"
)

type (
    httpMethod int // http method index

    // Wrapper yang digunakan di semua modules untuk menyederhanakan syntax,
    // asumsi wrapper digunakan untuk mapping http-request (by default string)
    //
    // List: by default list of string
    // SMap: map string to string
    // GMap: map string to object
    List []string
    SMap map[string]string
    GMap map[string]interface{}
    BMap map[string]bool
    IMap map[string]int

    SList []string // aka List
    GList []interface{}
    BList []bool
    IList []int

    Exception interface{}
    // Block transaksi database dengan begin sebagai outer, commit pada block
    // Try dan rollback pada block Catch
    //
    // Selain variabel yang bisa diakses dalam scope Try/Catch/Finally bisa
    // diakses via closure
    Go struct {
        Try     func()          // gunakan untuk transaksi
        Catch   func(Exception) // rollback
        Finally func()          // optional
    }

    // Default interface pada setiap handler untuk menghilangkan kebutuhan
    // reflection (alasan performansi) untuk method yang umum digunakan
    //
    // Dengan demikian setiap handler sekaligus berfungsi sebagai API (REST)
    // yang bisa dipanggil dengan atau tanpa default form/interface
    //
    // Mekanisme eksekusi handler tidak menggunakan sync.Pool atau copy (return),
    // untuk alasan thread safety, tidak disarankan handler memiliki property/field
    // pada struct (kecuali read-only)
    //
    // Framework lain (umumnya) hanya mensyaratkan COntext, tapi menurut pengalaman,
    // koneksi database hampir selalu dibutuhkan. Konvensinya adalah jika sebuah
    // package/module memiliki koneksi spesifik di syst.env maka koneksi akan dipassing,
    // jika tidak maka default koneksi yang akan digunakan
    Service interface {
        //
        // Method (REST API) berikut ini akan mengikuti HTTP Method
        //
        GET(*Connection, *Context)
        POST(*Connection, *Context)
        PUT(*Connection, *Context)
        DELETE(*Connection, *Context)

        // Default method yang (terpaksa) dimasukkan kedalam interface karena hampir
        // selalu ada di setiap handler
        //
        // Catatan: semua method selain HTTP method (GET/POST/PUT/DELETE) hanya
        // bisa dipanggil via GET
        GRID(*Connection, *Context)
        HTML(*Connection, *Context)
        JSON(*Connection, *Context)
        TEXT(*Connection, *Context)
        FILE(*Connection, *Context)
    }

    // Karena handlers hanya mensyaratkan function interface, kebutuhan untuk property
    // handler (secure, package|handler ID) menggunakan map lain sebagai referensi
    ServiceProperty struct {
        SEC      bool   // true: handler hanya bisa diakses kalau sudah login
        PID, HID string // Package ID, Handler ID
    }

    // Menghilangkan kebutuhan concat setiap kali lookup cache
    cacheKey struct {
        GET, POST, PUT, DELETE, GRID, HTML, JSON, TEXT, FILE string
    }
    serviceKey struct {
        rule, argv cacheKey
    }

    // ServiceRule bisa kita terjemahkan sebagai hook handler yang akan dipanggil.
    // Umumnya berupa constraint berkaitan dengan data/transaksi.
    //
    // Tujuannya adalah memisahkan antara transaksi/bispro dengan constraint (rule)
    // agar setiap proses transparan
    //
    // Rules akan dipanggil sebelum method handler dipanggil. Exception akan dibangkitkan
    // jika hasil satu diantara rules yang dieksekusi tidak sesuai ekspektasi
    ServiceRule interface {
        Execute(*Connection, *Context, bool) error
    }

    // Framework didesain multi package/module. SessionCallback disediakan untuk
    // memenuhi kebutuhan setting session attributes yang spesifik untuk masing2
    // package/module
    //
    // OnCreate dipanggil saat pertama kali user berhasil login
    // OnUpdate dipanggil setiap ada perubahan set/unset attributes
    //
    // contoh:
    //
    //      func init() {
    //          ExportSessionListener(new(mySessionListener))
    //      }
    //
    // Kedepan mungkin perlu enhancement agar session listener bisa subscribe/bind
    // method OnUpdate hanya untuk perubahan pada package tertentu, kalo diperlukan
    SessionCallback interface {
        OnCreate(*Connection, *Context)
        OnUpdate(*Connection, *Context)
    }

    // Interface untuk object yang akan dipublish/export sebagai cron. Parameter
    // cron ada di database dan bisa dimanage via UI
    CronService interface {
        Execute(*Connection, int64)
    }

    // ** private **
    // Enkapsulasi cron (object) untuk kebutuhan eksekusi
    cronjob struct {
        M, D, H, I, S int         // month, day, hour, minute, second
        B             bool        // disabled/enabled flag
        N             string      // namespace
        F             CronService // cron impl
        PID, HID      string      //package + handler ID
    }

    // ** private **
    // Enkapsulasi object yang dibutuhkan oleh github.com/svc untuk menjalankan
    // aplikasi sebagai service
    win32svc struct {
        srv *http.Server
        swg *sync.WaitGroup
    }
)

const (
    doPOST httpMethod = iota
    doGET
    doPUT
    doDELETE
    doGRID
    doHTML
    doJSON
    doTEXT
    doFILE
    doPATH
    dOPTIONS

    PackageSystem = "syst"
    FileSeparator = "/"

    CronAny = -1

    ContentTypeHTML = "text/html; charset=UTF-8"
    ContentTypeJPEG = "image/jpeg"
    ContentTypeJSON = "application/json; charset=UTF-8"
    ContentTypeTEXT = "text/plain; charset=UTF-8"
    ContentTypeXLSX = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
    ContentTypeXML  = "text/html; charset=UTF-8"
    ContentTypeDEF  = "application/octet-stream"
    ContentTypePDF  = "application/pdf"
    ContentTypeGIF  = "image/gif"
    ContentTypeJPG  = "image/jpg"
    ContentTypePNG  = "image/png"
    ContentTypeIMG  = "image/*"

    // Http status alias. Environment (minimal) yang dibutuhkan akan di expose
    // via package tlkm (database connection, context dll) termasuk http status
    StatusContinue                      = http.StatusContinue                      // 100
    StatusSwitchingProtocols            = http.StatusSwitchingProtocols            // 101
    StatusProcessing                    = http.StatusProcessing                    // 102
    StatusEarlyHints                    = http.StatusEarlyHints                    // 103
    StatusOK                            = http.StatusOK                            // 200
    StatusCreated                       = http.StatusCreated                       // 201
    StatusAccepted                      = http.StatusAccepted                      // 202
    StatusNonAuthoritativeInfo          = http.StatusNonAuthoritativeInfo          // 203
    StatusNoContent                     = http.StatusNoContent                     // 204
    StatusResetContent                  = http.StatusResetContent                  // 205
    StatusPartialContent                = http.StatusPartialContent                // 206
    StatusMultiStatus                   = http.StatusMultiStatus                   // 207
    StatusAlreadyReported               = http.StatusAlreadyReported               // 208
    StatusIMUsed                        = http.StatusIMUsed                        // 226
    StatusMultipleChoices               = http.StatusMultipleChoices               // 300
    StatusMovedPermanently              = http.StatusMovedPermanently              // 301
    StatusFound                         = http.StatusFound                         // 302
    StatusSeeOther                      = http.StatusSeeOther                      // 303
    StatusNotModified                   = http.StatusNotModified                   // 304
    StatusUseProxy                      = http.StatusUseProxy                      // 305
    StatusTemporaryRedirect             = http.StatusTemporaryRedirect             // 307
    StatusPermanentRedirect             = http.StatusPermanentRedirect             // 308
    StatusBadRequest                    = http.StatusBadRequest                    // 400
    StatusUnauthorized                  = http.StatusUnauthorized                  // 401
    StatusPaymentRequired               = http.StatusPaymentRequired               // 402
    StatusForbidden                     = http.StatusForbidden                     // 403
    StatusNotFound                      = http.StatusNotFound                      // 404
    StatusMethodNotAllowed              = http.StatusMethodNotAllowed              // 405
    StatusNotAcceptable                 = http.StatusNotAcceptable                 // 406
    StatusProxyAuthRequired             = http.StatusProxyAuthRequired             // 407
    StatusRequestTimeout                = http.StatusRequestTimeout                // 408
    StatusConflict                      = http.StatusConflict                      // 409
    StatusGone                          = http.StatusGone                          // 410
    StatusLengthRequired                = http.StatusLengthRequired                // 411
    StatusPreconditionFailed            = http.StatusPreconditionFailed            // 412
    StatusRequestEntityTooLarge         = http.StatusRequestEntityTooLarge         // 413
    StatusRequestURITooLong             = http.StatusRequestURITooLong             // 414
    StatusUnsupportedMediaType          = http.StatusUnsupportedMediaType          // 415
    StatusRequestedRangeNotSatisfiable  = http.StatusRequestedRangeNotSatisfiable  // 416
    StatusExpectationFailed             = http.StatusExpectationFailed             // 417
    StatusTeapot                        = http.StatusTeapot                        // 418
    StatusMisdirectedRequest            = http.StatusMisdirectedRequest            // 421
    StatusUnprocessableEntity           = http.StatusUnprocessableEntity           // 422
    StatusLocked                        = http.StatusLocked                        // 423
    StatusFailedDependency              = http.StatusFailedDependency              // 424
    StatusTooEarly                      = http.StatusTooEarly                      // 425
    StatusUpgradeRequired               = http.StatusUpgradeRequired               // 426
    StatusPreconditionRequired          = http.StatusPreconditionRequired          // 428
    StatusTooManyRequests               = http.StatusTooManyRequests               // 429
    StatusRequestHeaderFieldsTooLarge   = http.StatusRequestHeaderFieldsTooLarge   // 431
    StatusUnavailableForLegalReasons    = http.StatusUnavailableForLegalReasons    // 451
    StatusInternalServerError           = http.StatusInternalServerError           // 500
    StatusNotImplemented                = http.StatusNotImplemented                // 501
    StatusBadGateway                    = http.StatusBadGateway                    // 502
    StatusServiceUnavailable            = http.StatusServiceUnavailable            // 503
    StatusGatewayTimeout                = http.StatusGatewayTimeout                // 504
    StatusHTTPVersionNotSupported       = http.StatusHTTPVersionNotSupported       // 505
    StatusVariantAlsoNegotiates         = http.StatusVariantAlsoNegotiates         // 506
    StatusInsufficientStorage           = http.StatusInsufficientStorage           // 507
    StatusLoopDetected                  = http.StatusLoopDetected                  // 508
    StatusNotExtended                   = http.StatusNotExtended                   // 510
    StatusNetworkAuthenticationRequired = http.StatusNetworkAuthenticationRequired // 511
)

var (
    doMap = map[httpMethod]string{doGET: "GET", doPOST: "POST",
        doPUT: "PUT", doDELETE: "DELETE", doGRID: "GRID", doHTML: "HTML",
        doJSON: "JSON", doTEXT: "TEXT",
        doFILE: "FILE", doPATH: "PATH", dOPTIONS: "OPTIONS"}
    doKey = make(map[string]httpMethod)

    // ** private **
    // secret key bisa ditaruh di environment variable (akan diakses via os.Getenv)
    // atau langsung didefinisikan di source-code
    //
    // variable ini digunakan http context pada saat membentuk token
    jwtSecretKey = "JWT_SECRET"              // env variable SALT
    jwtSecret    = []byte("JWT_SECRET_TEXT") // jika SALT ingin ditanam di kode

    StatusText = http.StatusText

    // ** private **
    // ** WARNING **
    // Karena tidak menggunakan sync.Pool, ini adalah object yang sama yang akan
    // dieksekusi secara multi-thread. Jangan pernah mendefinisikan field (kecuali read-only)
    // pada handler
    //
    // Alasan kenapa handler tidak menggunakan sync.Pool karena afaik, selama sebuah
    // fungsi independen (ex: receiver tanpa akses ke field struct) aman diasumsikan
    // thread-safe
    //
    // Note: method akan diubah kalau kedepan ternyata asumsinya salah wkwkwk
    servMap = make(map[string]Service)
    servRef = make(map[string]ServiceProperty)
    servKey = make(map[string]serviceKey)
    ruleMap = make(map[string]ServiceRule)
    sessMap = make(map[string]SessionCallback)
    ruleRef = make(map[string]string)
    configs = make(map[string]GMap)

    // ** private **
    service *win32svc // pointer receiver service interface, impl github.com/svc

    // ** private **
    // cronjob properties
    chant     chan struct{}
    crons     []cronjob
    cronState int

    loglv = 2

    ctrl *controller
)

// Semua proses berkaitan dengan framework yang harus dilakukan sebelum http server
// dijalankan akan dipanggil dalam fungsi ini
func init() {
    secret := os.Getenv(jwtSecretKey) // jika (dan hanya jika) JWT_SECRET ditemukan di environment variable, maka
    if secret != "" {
        jwtSecret = []byte(secret) // kita gunakan yang ada di ENV
    }
    for k, v := range doMap {
        doKey[v] = k
    }
}

// Simulasi Try Catch Finally pada environment OOP. Struktur yang digunakan
// terdiri dari tiga block:
//
//      (&Go{
//          Try: func() {},
//          Catch: func(e Exception) {},
//          Finally: func() {},
//      }).Run()
//
func (self *Go) Run() {
    if self.Finally != nil {
        defer self.Finally() // defer di golang menggunakan metode LIFO
    }
    if self.Catch != nil {
        defer func() {
            if r := recover(); r != nil {
                self.Catch(r) // panggil jika (dan hanya) terjadi panic: recover() != nil
            }
        }()
    }
    if self.Try != nil { // Try juga bisa nil
        self.Try()
    }
}

// Wrapper panic dengan pesan berupa concat arguments dengan space separator
//
// @params interface{}  printable vars
func Exit(message ...interface{}) {
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
}

// Fungsi sementara melakukan injeksi Tag namespace dan Logger
func injectProperties(namespace string, reflectType reflect.Type, reflectValue reflect.Value) {
    for i := 0; i < reflectType.NumField(); i++ {
        j := reflectType.Field(i)
        if string(j.Tag) == "namespace" { // untuk lookup cache agar konsisten secara namespace
            p := reflectValue.FieldByName(j.Name)
            if p.IsValid() {
                if p.CanSet() {
                    p.SetString(namespace + "::" + j.Name)
                }
            }
            continue
        }
        if j.Name == "Logger" {
            p := reflectValue.FieldByName(j.Name)
            if p.IsValid() {
                if p.CanSet() {
                    p.Set(reflect.ValueOf(&Logger{logNs: namespace, logLv: loglv}))
                }
            }
        }
    }
}

// Memetakan object (handler/cron) kedalam Path dan ID
//      IDX: Path (sesuai namespace pada modules)
//      PID: Package ID
//      HID: Handler ID
//
// @params interface{}  object (pointer receiver)
func getIndexes(object interface{}) (IDX, PID, HID string) {
    typ := reflect.TypeOf(object)
    val := reflect.ValueOf(object)
    if typ.Kind() == reflect.Ptr {
        typ = typ.Elem()
        val = val.Elem()
    }
    namespace := typ.PkgPath()
    className := typ.Name()
    IDX = path.Join(FileSeparator, namespace, className)
    PID, HID = ShortURL(IDX)
    b := buffer.Get()
    defer b.Close()
    for _, c := range namespace {
        if c == '/' {
            b.WRune('.')
        } else {
            b.WRune(c)
        }
    }
    b.WRune('.').WS(className)
    name := b.String()
    injectProperties(name, typ, val)
    return
}

func getKey(prefix string) cacheKey {
    K := cacheKey{}
    T := reflect.TypeOf(&K).Elem()
    V := reflect.ValueOf(&K).Elem()
    for i := 0; i < T.NumField(); i++ {
        j := T.Field(i).Name
        p := V.FieldByName(j)
        if p.IsValid() && p.CanSet() {
            p.SetString(prefix + j)
        }
    }
    return K
}

// Mapping object handler ke servMap (object) dan servRef (property). Contoh object
// yang di export dengan secure flag false adalah /syst/api/sso API untuk authentifikasi.
//
// @params Service      service object
// @params bool         secure flag, by default true (secure)
func Export(object Service, secure ...bool) (PID, HID string) {
    IDX, PID, HID := getIndexes(object)
    servMap[IDX] = object
    servKey[IDX] = serviceKey{rule: getKey(Sprintf("rule:%s.", IDX)), argv: getKey(Sprintf("argv:%s.", IDX))}
    property := ServiceProperty{SEC: true, PID: PID, HID: HID} // default exported object adalah secure service
    if len(secure) > 0 {
        property.SEC = secure[0] // kecuali didefinisikan sebaliknya (ex: API login/sso)
    }
    servRef[IDX] = property

    return
}

// Mapping rule object ke ruleMap
//
// @params ServiceRule      service object
func ExportRule(object ServiceRule) (PID, HID string) {
    IDX, PID, HID := getIndexes(object)
    ruleMap[IDX] = object
    return
}

// Add cron object ke crons list
func ExportCron(object CronService, j ...int) {
    m := CronAny
    d := CronAny
    H := CronAny
    i := CronAny
    s := CronAny
    l := len(j)
    if l > 0 {
        s = j[0]
    }
    if l > 1 {
        i = j[1]
    }
    if l > 2 {
        H = j[2]
    }
    if l > 3 {
        d = j[3]
    }
    if l > 4 {
        m = j[4]
    }
    IDX, PID, HID := getIndexes(object)
    crons = append(crons, cronjob{m, d, H, i, s, false, IDX, object, PID, HID})
}

// inject (read-only) Tag
func ExportDBF(object interface{}) {
    getIndexes(object)
}

// Mapping object SessionCallback kedalam sessMap
func ExportSessionListener(object SessionCallback) (PID, HID string) {
    IDX, PID, HID := getIndexes(object)
    sessMap[IDX] = object
    return
}

// Digunakan untuk mendapatkan Package ID dan Handler ID berdasarkan namespace handler
//
// ex: <first-folder-as-PID>/<rest-as-HID>
//
func ShortURL(namespace string) (PID, HID string) {
    var r rune
    i := 0
    a := false
    b := buffer.Get()
    d := buffer.Get()
    defer b.Close()
    defer d.Close()
    for _, c := range namespace {
        r = unicode.ToUpper(c)
        if a && i >= 2 {
            b.WRune(r)
        }
        if !a && i >= 3 && c != '/' && b.Len() <= 5 {
            b.WRune(r)
        }
        if i >= 1 && i < 2 && c != '/' && d.Len() <= 3 {
            d.WRune(r)
        }
        if a {
            a = false
        }
        if c == '/' || c == '.' {
            a = true
            i += 1
        }
    }
    PID = d.String()
    l := 4 - len(PID)
    if l <= 0 {
        PID = PID[0:4]
    } else {
        for l > 0 {
            d.WRune('0')
            l -= 1
        }
        PID = d.String()
    }
    HID = b.String()
    l = 5 - len(HID)
    if l < 0 {
        HID = HID[0:5]
    }
    if l > 0 {
        for l > 0 {
            b.WRune('0')
            l -= 1
        }
        HID = b.String()
    }
    return
}

func Sprintf(format string, args ...interface{}) string {
    num := len(args)
    if num == 0 {
        return format
    }
    b := buffer.Get()
    defer b.Close()
    w := false
    n := 0
    for _, c := range format {
        if w {
            w = false
            if n < num {
                s := to.String(args[n])
                switch c {
                case 's':
                    b.WS(to.Escape(s))
                case 'n':
                    b.WS(to.Numeric(s))
                case 'd':
                    b.WS(to.Digit(s))
                }
                n += 1
            } else {
                b.WRune('%')
                b.WRune(c)
            }
            continue
        }
        if c == '%' {
            w = true
            continue
        }
        b.WRune(c)
    }
    return b.String()
}

// Content tabel st_handlers akan dibentuk otomatis pertama kali handlers di export,
// dan akan diupdate sesuai waktu up server
//
// Handlers yang tidak ter-mapping (dihapus dll) akan di delete
func updateHandlers(conn *Connection, now string) {
    chk := "SELECT * FROM st_handlers WHERE PID=? AND HID=?"
    sql := "INSERT INTO st_handlers(PID, HID, SRC, CREATED_AT, UPDATED_AT) VALUES (?, ?, ?, ?, ?)"
    for SRC, _ := range servMap {
        if SRC == FileSeparator {
            continue
        }
        PID, HID := ShortURL(SRC)
        rows := conn.Query(chk, PID, HID)
        if rows.Next() {
            conn.Exec("UPDATE st_handlers SET UPDATED_AT=? WHERE PID=? AND HID=?", now, PID, HID)
        } else {
            conn.Exec(sql, PID, HID, SRC, now, now)
        }
        rows.Close()
    }

    conn.Exec("DELETE FROM st_handlers WHERE UPDATED_AT<?", now)
    conn.Exec("DELETE A FROM st_handler_arguments A LEFT JOIN st_handlers B ON (B.PID=A.PID AND B.HID=A.HID) WHERE B.PID IS NULL")
    conn.Exec("DELETE A FROM st_handler_rules A LEFT JOIN st_handlers B ON (B.PID=A.PID AND B.HID=A.HID) WHERE B.PID IS NULL")

    chk = "SELECT * FROM st_rules WHERE PID=? AND RID=?"
    sql = "INSERT INTO st_rules(PID, RID, SRC, CREATED_AT, UPDATED_AT) VALUES (?, ?, ?, ?, ?)"
    for SRC, _ := range ruleMap {
        PID, HID := ShortURL(SRC)
        rows := conn.Query(chk, PID, HID)
        if rows.Next() {
            conn.Exec("UPDATE st_rules SET UPDATED_AT=? WHERE PID=? AND RID=?", now, PID, HID)
        } else {
            conn.Exec(sql, PID, HID, SRC, now, now)
        }
        rows.Close()
    }

    conn.Exec("DELETE FROM st_rules WHERE UPDATED_AT<?", now)
    conn.Exec("DELETE A FROM st_rule_arguments A LEFT JOIN st_rules B ON (B.PID=A.PID AND B.RID=A.RID) WHERE B.PID IS NULL")
}

// Proses yang sama seperti handlers dilakukan untuk crons
func updateCron(conn *Connection, now string) {
    if chant != nil {
        close(chant)
        for cronState > 0 {
            time.Sleep(time.Second)
        }
        chant = nil
    }

    conn.Exec("DELETE FROM st_cronjob WHERE UPDATED_AT<?", now)

    x := "INSERT INTO st_cronjob (PID, SRC, T_MON, T_DAY, T_HOU, T_MIN, T_SEC, CREATED_AT, UPDATED_AT) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)"
    for i, _ := range crons {
        v := &crons[i]
        v.B = false
        rows := conn.Query("SELECT * FROM st_cronjob WHERE PID=? AND SRC=?", v.PID, v.N)
        if rows.Next() {
            if e := rows.Int("ENABLED"); e == 1 {
                v.B = true
                v.M = rows.Int("T_MON")
                v.D = rows.Int("T_DAY")
                v.H = rows.Int("T_HOU")
                v.I = rows.Int("T_MIN")
                v.S = rows.Int("T_SEC")
            }
            conn.Exec("UPDATE st_cronjob SET UPDATED_AT=? WHERE PID=? AND SRC=?", now, v.PID, v.N)
        } else {
            conn.Exec(x, v.PID, v.N, v.M, v.D, v.H, v.I, v.S, now, now)
        }
        rows.Close()
    }
    chant = make(chan struct{})
    go func() {
        cronState = 1
        for {
            select {
            case <-chant:
                cronState = 0
                return
            default:
                unow := time.Now()
                unix := unow.Unix()
                for _, j := range crons {
                    if j.B && (j.M == CronAny || j.M == int(unow.Month())) && (j.D == CronAny || j.D == int(unow.Day())) && (j.H == CronAny || j.H == int(unow.Hour())) && (j.I == CronAny || j.I == int(unow.Minute())) && (j.S == CronAny || j.S == int(unow.Second())) {
                        go func() {
                            conn := SQL.Default()
                            defer conn.Close()
                            j.F.Execute(conn, unix)
                        }()
                    }
                }
            }
            time.Sleep(time.Second)
        }
    }()
}

// Kondisi dimana server harus restart, sessions yang terbentuk (dan masih valid)
// akan di push ulang ke cache
func updateSession(conn *Connection, now time.Time) {
    ssot, _ := Cache.Int("SSO_SSN_EXP")
    ssof := strconv.FormatInt(now.Add(time.Duration(-ssot)*time.Minute).Unix(), 10)
    rows := conn.Query("SELECT SID, MSGT FROM st_sessions WHERE UTS>? FOR UPDATE", ssof)
    defer rows.Close()
    ssox, _ := Cache.Int("SSO_SSN_EXP")
    for rows.Next() {
        SID := rows.String("SID")
        var m GMap
        if e := json.Unmarshal(rows.Bytes("MSGT"), &m); e == nil {
            g := remap(m, "GID", "ACL")
            Cache.Set(SID, g, time.Duration(ssox))
        }
    }
}

func remap(sesMap GMap, index ...string) GMap {
    for _, j := range index {
        if k, v := sesMap[j]; v {
            g := k.(map[string]interface{})
            m := make(map[string]string)
            for o, p := range g {
                m[o] = p.(string)
            }
            sesMap[j] = m
        }
    }
    return sesMap
}

func LoadConfig(conn *Connection) {
    rows := conn.Query("SELECT PID,CFT,CFK,CFV FROM st_configs WHERE CHK='1' AND BEGDA<=CURRENT_DATE AND BEGDA IS NOT NULL AND (ENDDA>=CURRENT_DATE OR ENDDA IS NULL)")
    defer rows.Close()
    for rows.Next() {
        PID := rows.String("PID")
        if _, v := configs[PID]; !v {
            configs[PID] = make(GMap)
        }
        CFK := rows.String("CFK")
        switch rows.Int("CFT") {
        case 0:
            configs[PID][CFK] = rows.String("CFV")
        case 1:
            configs[PID][CFK] = rows.Int("CFV")
        case 2:
            configs[PID][CFK] = rows.Bool("CFV")
        }
        if PID == "SYST" {
            Cache.Set(CFK, configs[PID][CFK])
        }
    }
}

func Config(PID ...string) (g GMap, b bool) {
    if len(PID) > 0 {
        g, b = configs[PID[0]]
    } else {
        g, b = configs["SYST"]
    }
    return
}

// *** startup ***
//
// Init, Start, Stop implementasi method yang dibutuhkan github.com/svc untuk menjalankan
// aplikasi sebagai service
func (self *win32svc) Init(e svc.Environment) error {
    return nil
}

func (self *win32svc) setup() {
    conn := SQL.Default()
    defer conn.Close()
    now := time.Now()
    str := now.Format("2006-01-02 15:04:05")
    updateHandlers(conn, str)
    updateCron(conn, str)
    updateSession(conn, now)
}

func (self *win32svc) Start() error {
    self.setup()
    defer self.swg.Done()
    return self.srv.ListenAndServe()
}

func (self *win32svc) Stop() error {
    if chant != nil {
        close(chant)
    }
    if er := self.srv.Shutdown(context.TODO()); er != nil {
        panic(er)
    }
    self.swg.Wait()
    return nil
}

func FrontController(www string, dev bool, loglv int) *controller {
    if ctrl == nil {
        ctrl = &controller{Logger: &Logger{logNs: "HTTP", logLv: loglv}}
        ctrl.init(www, dev)
    }
    return ctrl
}

// Tricky part: memetakan salah satu handler ke namespace / karena semua handler
// ada dalam package/modul
//
// By default, yang digunakan sebagai home handler adalah /syst/api/index (secure flag false)
func Win32Service(object Service, https bool, www string, dev bool) *win32svc {
    if service != nil {
        return service
    }
    conn := SQL.Default()
    LoadConfig(conn)
    conn.Close()
    port, _ := Cache.String("HTTPD_PORT")
    loglv, _ = Cache.Int("LOG_LEVEL")
    _, PID, HID := getIndexes(object)
    servMap[FileSeparator] = object
    servRef[FileSeparator] = ServiceProperty{SEC: false, PID: PID, HID: HID}
    fc := FrontController(www, dev, loglv)
    service = &win32svc{
        srv: &http.Server{Addr: ":" + port, Handler: fc},
    }
    return service
}
