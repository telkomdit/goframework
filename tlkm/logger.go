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

// Perlu diperhatikan bahwa Logger di desain spesifik untuk logging dalam proses
// request-response, bukan logging secara umum
package tlkm

// Karena alasan praktis, logs akan dikelompokkan melalui 2 informasi
//  1. namespace    path dari object/instance yang menulis log
//  2. timestamp    waktu database
type Logger struct {
    // ** private **
    // namespace akan diinject secara otomatis pada saat object diexport
    logNs   string

    // ** private **
    // copy global log level untuk menghindari lookup setiap kali dibutuhkan
    logLv   int
}

// Framework di desain sebagai web-server. Yang perlu diperhatikan adalah log
// level FRAUD. Semua kondisi yang (menurut developer) tidak mungkin terjadi tapi
// bisa saja terjadi, harus di-log untuk evaluasi security/bug etc
//
// 4 level selain FRAUD cukup jelas sesuai namanya. Untuk kebutuhan tracing, developer
// bisa menggunakan level ERROR/FRAUD
const (
    FRAUD   = 5
    ERROR   = 4
    WARN    = 3
    INFO    = 2
    DEBUG   = 1
)

// ** WARNING **
// By default logging dilakukan secara async tanpa mempedulikan return dari database.
// Tidak ada jaminan bahwa log berhasil disimpan. Jika dibutuhkan informasi berhasil/tidak
// logging dilakukan, gunakan method LogSync dengan return LGID (Log ID)
//
// Informasi tambahan setelah message (sesuai urutan): URI, USR, ADDR
func (self *Logger) Log(logLv int, message string, args ...string) {
    if logLv >= self.logLv { // hanya jika lebih besar (atau sama dengan) threshold
        go self.LogSync(logLv, message, args ...) // tidak ada kebutuhan untuk sync
    }
}

// Gunakan jika dibutuhkan (apapun alasannya) informasi berhasil/tidak logging dilakukan
// untuk mendapatkan return LGID
func (self *Logger) LogSync(logLv int, message string, args ...string) (ID int64, OK bool) {
    conn := SQL.Default()
    defer conn.Close()
    argv := GMap{   // basic info
        "LGLV": logLv,
        "NAMESPACE": self.logNs,
        "MSG": message,
        "@LOGT": "CURRENT_TIMESTAMP",
    }
    arln := len(args)
    if arln > 0 { argv["URI"] = args[0] }
    if arln > 1 { argv["USR"] = args[1] }
    if arln > 2 { argv["ADDR"] = args[2] }
    OK = false
    if r, e := conn.ExecInsertIgnore("st_logs", &argv); e == nil {
        ID, _ = r.LastInsertId()
        OK = true
    }
    return
}
