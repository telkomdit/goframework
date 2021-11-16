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

// Semua parameter yang dibutuhkan oleh handler akan dienkapsulasi ke dalam
// Context: current session, payload (parsed) dll
package tlkm

import (
    "encoding/json"
    "errors"
    "io"
    "io/ioutil"
    "mime/multipart"
    "net/http"
    "net/url"
    "os"
    "time"
    "github.com/satori/go.uuid"
    "github.com/golang-jwt/jwt"
    "github.com/telkomdit/goframework/to"
)

type (
    // Enkapsulasi seluruh environment request/response yang dibutuhkan rule/handler:
    //  * parameter via query atau payload
    //  * session & cookie
    //  * json response
    //  * etc
    //
    // Karena struktur Context cukup banyak (field + method) dan harus dibuat secara
    // spesifik per hit/request, mekanismenya akan diatur oleh controller via sync.Pool
    // untuk mengurangi latency GC
    Context struct {
        Request    *http.Request
        Response    http.ResponseWriter

        SID, newSID     string  // session ID
        PID, HID, GID   string  // invoked handler
        Values      url.Values  // parameter dari (GET/POST/PUT/DELETE)
        Files       map[string]*multipart.FileHeader    // parsed multipart

        // ** private **
        sesMap      GMap    // session variables

        // sinkronisasi antara cache dan database dilakukan jika dan hanya jika
        // ada perubahan Map (melalui set/unset)
        sesCreate, sesUpdate   bool

        json        GMap    // default property untuk response json

        // service tidak dibatasi pada abstraksi CRUD (GET, POST, PUT, DELETE)
        // **khusus method GET**, untuk alasan praktis, method2 non transaksional
        // bisa dipanggil sebagai prosedur biasa dengan menambahkan path pada handler
        call        string
        method      httpMethod

        // flag ini akan di cek oleh controller untuk memastikan apakah ada data
        // yang sudah ditulis/tidak
        //
        // jika tidak ada data yang ditulis, controller akan melihat apakah ada
        // struktur json yang akan dikirim
        sent        bool
        
        // data tidak akan diteruskan ke http.Response jika flag exit true:
        //   1. memanggil method Code dengan parameter tambahan true (translate status dlm body)
        //   2. push raw json melalui method JSON
        exit        bool
        
        code        int     // Http Status Code
    }
)

const (
    SAFSID = "SAFSID"   // Session ID
)

func (self *Context) HttpMethod() httpMethod {
    return self.method
}

func (self *Context) MethodName() string {
    return self.call
}

// Set cookie (manual) selain yang digunakan untuk session. Pastikan set-cookie dilakukan
// sebelum ada data yang ditulis/dikirim
func (self *Context) SetCookie(name string, value string, maxAge int,
                               domain string, secure bool) {
    cookie := &http.Cookie{}
    cookie.Name = name
    cookie.Value = value
    if maxAge != 0 {
        cookie.MaxAge = maxAge  // by default sampai logout atau browser ditutup
    }
    cookie.Path = FileSeparator
    cookie.HttpOnly = true      // asumsi client (secara umum) disini adalah browser
    if len(domain) > 0 {
        cookie.Domain = domain
    }
    if secure {
        cookie.Secure = secure
    }
    cookie.SameSite = http.SameSiteStrictMode
    http.SetCookie(self.Response, cookie)
}

// Ambil data cookie yang dikirim client
func (self *Context) Cookie(name string) string {
    cookie, e := self.Request.Cookie(name)
    if e != nil {
        return ""
    }
    return cookie.Value
}

// Method ini akan dipanggil oleh controller tepat pada saat context diambil dari
// sync.Pool
//
// SID (Session ID) dikirim melalui dua cara:
//   1. otomatis oleh browser (jika client adalah browser), curl, http-based client
//   2. http header Authorization: Bearer JWT (jika client adalah user API)
func (self *Context) sessionStart(conn *Connection) (e error) {
    SID := ""
    if cookie, e := self.Request.Cookie(SAFSID); e == nil {
        SID = cookie.Value
    } else {
        if HDR := self.Request.Header.Get("Authorization"); HDR != "" {
            mc := jwt.MapClaims{}   // lebih praktis karena isi claim hanya 1 field

            // (Authorization)[7:] karena selain Bearer tidak ada rencana untuk implementasi
            // auth method yang lain (Basic, Digest etc)
            //
            // Kedepan, paling mungkin switch HDR[:6] antara Bearer atau Digest
            _, e = jwt.ParseWithClaims(HDR[7:], mc, func(T *jwt.Token) (interface{}, error) {
                if _, b := T.Method.(*jwt.SigningMethodHMAC); !b {
                    return nil, errors.New("SigningMethodException: " + to.String(T.Header["alg"]))
                }
                return jwtSecret, nil
            })
            if e == nil {
                SID = to.String(mc[SAFSID])  // claim hanya 1 field yaitu session yang sama yang digunakan browser
            }
        }
    }
    // Dengan semua proses yang dilakukan pada saat login, harga yang harus dibayar
    // akan mahal jika JWT diperlakukan secara stateless. Proses yang mau tidak mau
    // harus dilakukan akan dilakukan setiap hit/request. Dengan alasan ini, JWT
    // diperlakukan sama seperti session-based
    //
    // Jadi session-based atau JWT keduanya stateful
    if SID == "" {
        self.sesMap = GMap{}    // atau sebaiknya nil?
    } else {
        if v, b := Cache.GMap(SID); b {
            self.sesMap = v
            self.SID = SID
        } else {
            // Tidak ditemukan di cache belum tentu benar2 expired, ada kemungkinan
            // karena flush Cache. Jadi selama session di DB ditemukan (dan valid),
            // maka cache harus push ulang
            ssot, _ := Cache.Int("SSO_SSN_EXP")
            rows := conn.Query("SELECT MSGT FROM st_sessions WHERE SID=? AND UTS>(UNIX_TIMESTAMP()-?)", SID, ssot)
            if rows.Next() {
                var v GMap
                if e := json.Unmarshal(rows.Bytes("MSGT"), &v); e == nil {
                    ssox, _ := Cache.Int("SSO_SSN_EXP")
                    self.sesMap = v
                    self.remap("GID", "ACL")
                    Cache.Set(SID, self.sesMap, time.Duration(ssox))
                }
            } else {
                return errors.New("SessionNotFoundException: " + SID)
            }
        }
    }
    return
}

// memastikan session (yang dimanage framework) sesuai tipe data yang diinginkan
// untuk menghindari assert dalam setiap request/response
//
// beberapa kasus, hasil assert json.Unmarshal SMap menjadi GMap
func (self *Context) remap(index... string) {
    for _, j := range index {
        if k, v := self.sesMap[j]; v {
            g := k.(map[string]interface{})
            m := make(map[string]string)
            for o, p := range g {
                m[o] = p.(string)
            }
            self.sesMap[j] = m
        }
    }
}

// Akan dipanggil pada saat write data pertama kali ke http.Response
func (self *Context) sessionClose() {
    USR, e := self.SessionUser()
    if !e || self.sent { return }
    ssox, _ := Cache.Int("SSO_SSN_EXP")
    if !self.sesUpdate {
        Cache.Extend(self.SID, time.Duration(ssox)) // jika tidak ada perubahan, extends lifetime
        return
    }
    conn := SQL.Default()
    defer conn.Close()
    _new := false
    if self.SID == "" { _new = true }
    // sessMap private hanya untuk package tlkm, tidak ada alasan lain untuk mutable
    // kecuali perubahannya krn internal
    if _new {
        for _, l := range sessMap {
            l.OnCreate(conn, self)  // 1x pada saat client berhasil login
        }
    } else { // Dipanggil setiap ada perubahan. Belum ada mekanisme subscribe untuk package spesifik (misal)
        for _, l := range sessMap {
            l.OnUpdate(conn, self)
        }
    }
    if j, e := json.Marshal(self.sesMap); e == nil { // serialize/marshal map untuk disimpan di DB
        jso := string(j)
        if _new {
            self.SID = self.newSID
            self.sessionCreate(0)   // valid sampai browser ditutup atau expired dari sisi server
            conn.Exec(
                "INSERT INTO st_sessions(SID,USR,ADDR,UTS,LOGT,MSGT) VALUES (?,?,?,UNIX_TIMESTAMP(),CURRENT_TIMESTAMP,?)",
                self.SID, USR, self.ClientIP(),
            jso)
        } else {
            conn.Exec("UPDATE st_sessions SET UTS=UNIX_TIMESTAMP(),MSGT=? WHERE SID=?", jso, self.SID)
        }
        Cache.Set(self.SID, self.sesMap, time.Duration(ssox))
    }
}

// Hanya untuk membedakan session cookie dibuat atau sudah expired
func (self *Context) sessionCreate(maxAge int) *Context {
    cookie := &http.Cookie{}
    cookie.Name = SAFSID
    cookie.Value = self.SID
    if maxAge != 0 {
        cookie.MaxAge = maxAge
    }
    cookie.Path = FileSeparator
    cookie.HttpOnly = true
    cookie.SameSite = http.SameSiteStrictMode
    http.SetCookie(self.Response, cookie)
    self.sesCreate = true
    return self
}

// Dua hal yang dilakukan jika session expired karena client (dengan sengaja) melakukan
// signout/logout:
//   1. Memindahkan data session ke archive untuk menjaga performansi tabel session dan
//      untuk kebutuhan audit
//   2. Hapus cache yang digunakan untuk lookup data
func (self *Context) SessionDestroy(conn *Connection) (e error) {
    tx := conn.Begin()
    (&Go{
        Try: func() {
            tx.Exec(`INSERT IGNORE INTO st_session_archive(PERIOD,SID,USR,ADDR,UTS,LOGT,MSGT)
                     SELECT DATE_FORMAT(LOGT, "%Y%m") PERIOD,SID,USR,ADDR,UTS,LOGT,MSGT
                       FROM st_sessions
                      WHERE SID=? FOR UPDATE`, self.SID)
            tx.Exec("LOCK TABLE st_sessions WRITE")
            tx.Exec("DELETE FROM st_sessions WHERE SID=?", self.SID)    // hapus dari tabel operasional
            tx.Exec("UNLOCK TABLES")
            tx.Commit()
            Cache.Delete(self.SID)
            self.sessionCreate(-1)
        },
        Catch: func(ex Exception) {
            tx.Rollback()
            e = errors.New(to.String(ex))
        },
    }).Run()
    return e
}

// Ambil data session tanpa casting, lakukan assert hasil return
func (self *Context) Session(name string) (k interface{}, v bool) {
    k, v = self.sesMap[name]
    return
}

// Informasi SID yang aktif
func (self *Context) SessionID() (k string, v bool) {
    if self.SID == "" && self.newSID == "" {
        v = false
    } else {
        v = true
        if self.SID == "" {
            k = self.newSID
        } else {
            k = self.SID
        }
    }
    return
}

// Method untuk kebutuhan serbaguna dengan data session tipe string
func (self *Context) SessionUser(name ...string) (k string, v bool) {
    index := "USR"
    if len(name) > 0 {
        index = name[0]
    }
    v = false
    if i, j := self.sesMap[index]; j {
        k = i.(string)
        v = true
    }
    return
}

// Mengikuti perilaku $_SESSION di PHP, tanpa inisialisasi secara eksplisit, session
// akan dibentuk secara otomatis pada saat pertama kali data session dibuat
func (self *Context) SessionSet(k string, v interface{}) *Context {
    self.sesUpdate = true
    if self.SID == "" && self.newSID == "" {
        newV4, _ := uuid.NewV4()
        self.newSID = newV4.String()  // dipastikan hanya 1x
    }
    self.sesMap[k] = v
    return self
}

// Hapus data session
func (self *Context) SessionUnset(name string) *Context {
    self.sesUpdate = true
    if _, v := self.sesMap[name]; v {   // hanya jika ditemukan
        delete(self.sesMap, name)
    }
    return self
}

// Token yang (mungkin) akan digunakan oleh client non-browser (akses via API)
//
// Pembentukan token tidak di expose ke modul untuk menjaga jwtSecret tetap private
// (available) hanya di package tlkm
func (self *Context) JWT() (v string, e error) {
    if SID, b := self.SessionID(); b {
        m := jwt.MapClaims{}
        m[SAFSID] = SID
        j := jwt.NewWithClaims(jwt.SigningMethodHS256, m)
        v, e = j.SignedString(jwtSecret)
    }
    return
}

// check (apapun datanya) k ditemukan di v
func (self *Context) keyExists(k, v string) bool {
    if i, j := self.sesMap[k]; j {
        if _, j := i.(map[string]string)[v]; j {
            return true
        }
    }
    return false
}

// Karena sejak awal framework sudah coupling dengan database (frameworknya sendiri)
// beberapa fungsi checking akan langsung disediakan oleh framework
func (self *Context) HasRole(GID string) bool {
    return self.keyExists("GID", GID)
}

// Sama seperti fungsi diatas
func (self *Context) HasHandler(GID, PID, HID string) bool {
    return self.keyExists("ACL", GID + PID + HID)
}

// Set http header secara langsung
func (self *Context) Header(k, v string) *Context {
    self.Response.Header().Set(k, v)

    return self
}

// Shortcut ctx.Header karena fungsi ini bisa kita anggap sudah umum digunakan
func (self *Context) ContentType(v string) *Context {
    return self.Header("Content-Type", v)
}

// By default, redirect akan mengikirim http status 302 (status found) tapi memungkinkan
// untuk tidak mengikuti standar, mengirim status selain 302. dibutuhkan???
//
// referensi: https://id.wikipedia.org/wiki/HTTP_302
func (self *Context) Redirect(location string, status ...int) {
    self.sessionClose()
    self.sent = true
    self.Header("Location", location)
    if len(status) > 0 {
        http.Redirect(self.Response, self.Request, location, status[0])
    } else {
        http.Redirect(self.Response, self.Request, location, StatusFound)
    }
}

// Informasi Client Addr diambil dari Request.RemoteAddr, kecuali ada informasi spesifik
// yang dikirim via http header
func (self *Context) ClientIP() string {
    cip := self.Request.RemoteAddr
    chk := List{"X-Real-IP", "X-Forwarded-For", "Forwarded-For", "X-Forwarded", "X-Cluster-Client-IP", "Client-IP"}
    for _, v := range chk {
        xip := self.Request.Header.Get(v)
        if len(xip) > 0 {
            cip = xip
            break
        }
    }
    return cip
}

// Write hanya menerima array of byte
func (self *Context) Write(data []byte) *Context {
    if !self.sent {
        self.sessionClose()
        self.Response.WriteHeader(self.code)
        self.sent = true
    }
    if !self.exit {
        self.Response.Write(data)
    }
    return self
}

// Alias dari method Write dengan parameter string
func (self *Context) Echo(data string) *Context {
    return self.Write([]byte(data))
}

// Check parameter pada payload. Perlu diperhatikan hanya parameter yang dikirim
// via query (?a=b&c=d...) atau payload (url-encoded atau json) akan di merger
// kedalam satu map
//
// Jika parameter yang sama dikirim via query dan payload, framework akan menyimpan
// parameter dalam bentuk list
func (self *Context) Exists(n string) bool {
    _, b := self.Values[n]
    return b
}

// Check nama file pada payload. Transfer file mengikuti format standar multipart/form-data
func (self *Context) FileExists(n string) bool {
    _, b := self.Files[n]
    return b
}

// Ambil parameter (non file) dalam bentuk map[string]string
func (self *Context) Map() SMap {
    m := make(SMap)
    for k := range self.Values {
        m[k] = self.Values.Get(k)
    }
    return m
}

// return pointer session map, data session umumnya konstan (sedikit perubahan)
func (self *Context) SessionMap() *GMap {
    return &self.sesMap
}

func (self *Context) RequestPath() string {
    return self.Request.URL.Path
}

func (self *Context) RawRequest() (r []byte, e error) {
    r, e = ioutil.ReadAll(self.Request.Body)
    defer self.Request.Body.Close()
    return
}

// Untuk parameter list, kemungkinan index pertama yang akan dikembalikan
func (self *Context) Get(name string) string {
    return self.Values.Get(name)
}

// Solusi skenario data harus dikirim dalam format url-encoded, dengan hasil akhir
// map[string]string, tapi kita tau bahwa beberapa parameter dalam bentuk json (encoded)
//
// Itulah alasan method ini dibuat, untuk memudahkan proses decode skenario diatas
func (self *Context) Unmarshal(name string, v interface{}) error {
    if e := json.Unmarshal([]byte(self.Get(name)), &v); e != nil {
        return e
    }
    return nil
}

// Ambil parameter dalam bentuk list (umumnya parameter checkbox)
func (self *Context) GetList(n string) (v List) {
    if self.Values == nil {
        return
    }
    v = self.Values[n]
    return
}

// Set?? Kenapa dalam Context??
//
// ** CATATAN **
// Secara desain framework, bisnis rule yang dieksekusi sebelum method handler
// tidak hanya berfungsi sebagai constraint, bisa juga sebagai penyedia fakta yang
// akan digunakan oleh handler, atau bahkan rule lain setelahnya (rule punya urutan)
//
// Karena komunikasi antar rules atau antara rules dan handler tidak bisa dilakukan
// melalui parameter, Context digunakan untuk meneruskan data/fakta
func (self *Context) Set(n string, v string) *Context {
    self.Values.Set(n, v)
    return self
}

// Hapus parameter
func (self *Context) Unset(n string) *Context {
    self.Values.Del(n)
    return self
}

// Versi awal, user diharapkan memproses sendiri parsed-file wkwk
func (self *Context) File(n string) *multipart.FileHeader {
    return self.Files[n]
}

// TODO: file ditangani oleh framework, copy ke lokal, ke s3 dll
func (self *Context) FileMove(n, to string) (e error) {
    if src, e := self.Files[n].Open(); e == nil {
        defer src.Close()
        if out, e := os.Create(to); e == nil {
            defer out.Close()
            _, e = io.Copy(out, src)
        }
    }
    return
}

// return []byte yang bisa langsung ditulis kedalam file
func (self *Context) FileBytes(n string) ([]byte, error) {
    if f, b := self.Files[n]; b {
        r, e := f.Open()
        if e != nil {
            return ioutil.ReadAll(r)
        }
        return nil, e
    }
    return nil, nil
}

func (self *Context) FileUnlink(n string) *Context {
    delete(self.Files, n)
    return self
}

// Http status yang akan dikirim. Parameter kedua menginstruksikan framework agar
// menerjemahkan status sebagai text di body
//
// Perlu diperhatikan bahwa jika parameter kedua true, output (apapun) setelah
// method ini dipanggil akan didrop
func (self *Context) Code(code int, text ...bool) *Context {
    self.code = code
    if len(text) > 0 {
        if text[0] {
            self.Echo(StatusText(code))
            self.exit = true
        }
    }
    return self
}

// Replace default response untuk tipe data json. Lebih jelasnya adalah keterangan
// untuk fungsi dibawah fungsi JSON
func (self *Context) JSON(json GMap) {
    if !self.exit {
        self.json = json
        self.exit = true
    }
}

// ** CATATAN **
//
// Semua methods dibawah ini adalah porting struktur (default) request-response
// dari SIMPKBL. Tidak harus digunakan untuk membuat aplikasi secara umum,
// dimasukkan dalam Context karena tujuan/desain framework adalah untuk memenuhi
// semua kebutuhan transaksi dari sisi backend dan frontend
//
// Frontend dari framework ini sendiri 100% akan menggunakan struktur json dari
// semua methods dibawah ini

// Set key-value pair default response untuk tipe data json
func (self *Context) set(name string, data interface{}) *Context {
    if self.json == nil {
        self.ContentType(ContentTypeJSON)
        self.json = make(GMap)
    }
    self.json[name] = data

    return self
}

// Informasi tambahan kepada client untuk melakukan iterasi data/map sesuai
// urutan list/key
func (self *Context) KeyMap(Key List, Map GMap) *Context {
    self.set("list", Key).set("data", Map)
    return self
}

// multi-purpose field, []string
func (self *Context) List(list List) *Context {
    return self.set("list", list)
}

// multi-purpose field, tidak harus berupa json, bisa string etc
func (self *Context) Data(data interface{}) *Context {
    return self.set("data", data)
}

// Informasi default control sebuah handler, bisa kita anggap sebagai manual/petunjuk
// bagaimana client berinteraksi dengan handler
//
// Default control umumnya dikirim bersama dengan Form/UI dan akan dimodifikasi/update
// melalui method GET pada saat lookup data
func (self *Context) Ctrl(C, R, U, D bool) *Context {
    return self.set("ctrl", GMap{"C": C, "R": R, "U": U, "D": D})
}

// Digunakan oleh GRID handler
func (self *Context) Cols(cols interface{}) *Context {
    return self.set("cols", cols)
}

// Digunakan oleh GRID handler
func (self *Context) Rows(rows interface{}) *Context {
    return self.set("rows", rows)
}

// Umumnya untuk informasi reload non-GRID (options, groups dll)
func (self *Context) Call(call ...string) *Context {
    return self.set("call", call)
}

// Jika satu (atau lebih) yang harus direload oleh client adalah GRID
func (self *Context) CallGRID(call ...string) *Context {
    return self.set("call", append(List{"GRID"}, call...))
}

// Informasi (umum) kepada client
func (self *Context) Info(info string) *Context {
    return self.set("info", info)
}

// Informasi (pesan) kepada client
func (self *Context) Message(message string) *Context {
    return self.set("message", message)
}

// Informasi (peringatan) kepada client
func (self *Context) Warn(warn string) *Context {
    return self.set("warn", warn)
}
