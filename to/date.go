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

package to

import (
    "time"
    "github.com/tlkm/buffer"
)

// TODO: mapping format golang ke format standar Y-m-d
func Date(format string, v... int64) string {
    t := time.Now().Unix()
    if len(v) > 0 {
        t = v[0]
    }
    return time.Unix(t, 0).Format(format)
}

// oldDate adalah format tanggal sesuai input dengan padding jika m/d 1 digit
// newDate adalah format Y-m-d
func DateSplit(n string) (Y, m, d, oldDate, newDate string) { // Y: Tahun, m: bulan, d: hari
    var r [5]string
    var z []byte
    i := 0
    b := buffer.Get()
    defer b.Close()

    var separator rune
    var Ymd = true

    // Jadi idenya adalah menggunakan non-numeric sebagai separator (-/.) untuk memecah tanggal
    // menjadi 3 bagian (Y, m, d). Padding dilakukan jika ditemukan 1 digit (seharusnya mm/dd)
    for _, c := range n {
        if i > 2 { break }  // ambil hanya 3 bagian awal, informasi H:i:s dll akan diskip
	    if c >= '0' && c <= '9' {
            b.WRune(c)  // simpan di buffer selama numeric
        } else {
            separator = c   // separator terakhir mereplace sebelumnya. ex: 2021.01-02, separator: -
            z = b.Bytes()
            if len(z) == 1 {
                z = append([]byte{'0'}, z...)   // padding jika 1 digit
            }
            r[i] = string(z)    // simpan di array sesuai index
            b.Reset()
            i+= 1
        }
    }
    z = b.Bytes()
    b.Reset()
    if len(z) == 1 {
        z = append([]byte{'0'}, z...)
    }
    r[i] = string(z)
    if len(r[2]) == 4 { // Asumsi index pertama Y. Jika sebaliknya, swap index pertama dan terakhir
        r[0], r[2] = r[2], r[0]
        Ymd = false
    }
    if Ymd {
        b.WS(r[0]).WRune(separator).WS(r[1]).WRune(separator).WS(r[2])
    } else {
        b.WS(r[2]).WRune(separator).WS(r[1]).WRune(separator).WS(r[1])
    }
    r[3] = b.String()
    b.Reset()
    b.WS(r[0]).WRune(separator).WS(r[1]).WRune(separator).WS(r[2])
    r[4] = b.String()
    b.Reset()
    return r[0], r[1], r[2], r[3], r[4]
}

func Time(format, v string) (int64, error) {
    t, err := time.Parse(format, v)
    if err != nil {
        return 0, err
    }
    return t.Unix(), nil
}
