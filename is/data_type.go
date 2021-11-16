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

package is

import (
    "reflect"
    "strings"
)

// Konvensi umum jika integer digunakan untuk boolean: selain 0 dianggap true
func Boolean(v interface{}) bool {
    switch b := v.(type) {
    case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
        if b != 0 { return true }
    case bool:
        return b
    case string:
        r := strings.ToLower(v.(string))
        if r != "false" && r != "0" { return true }
    case complex64, complex128:
        if b != complex128(0) { return true }
    case float32, float64:
        if b != float64(0) { return true }
    default:
        return b == nil
    }
    return false
}

// Penyatuan pengecekan kondisi variabel mengikuti perilaku bahasa scripting,
// bernilai true untuk:
//
// string/array dengan panjang 0, integer 0, nil pointer, boolean false
//
// @params interface{}  printable vars
func Empty(e interface{}) bool {
    if e == nil { return true }     // kemungkinan nil pointer to variable
    v := reflect.ValueOf(e)
    switch v.Kind() {
        case reflect.String, reflect.Array:
            return v.Len() == 0
        case reflect.Map, reflect.Slice:
            return v.Len() == 0 || v.IsNil()
        case reflect.Bool:
            return !v.Bool()
        case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
            return v.Int() == 0
        case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
            return v.Uint() == 0
        case reflect.Float32, reflect.Float64:
            return v.Float() == 0
        case reflect.Interface, reflect.Ptr:    // pointer hanya mengenal nil atau addr (utk dereference)
            return v.IsNil()
    }
    return reflect.DeepEqual(e, reflect.Zero(v.Type()).Interface()) // seharusnya tidak perlu sampai titik ini
}

// Check apakah string merepresentasikan floating point
func Float(v string) bool {
    b := false  // char eval
    i := false  // decimal separator
    e := false  // exp
    for j, c := range v {
        if b = ('0' <= c && c <= '9') || (c == '-' && j == 0); !b {
            if c == '.' {
                if i { return false }   // memastikan hanya muncul 1x
                i = true
                continue
            }
            if c == 'e' || c == 'E' {
                if e { return false }   // memastikan hanya muncul 1x
                e = true
                continue
            }
            return false
        }
    }
    return true
}

// ** List and Map **
func KeyExists(v interface{}, listOrMap interface{}) bool {
    r := reflect.ValueOf(listOrMap)
    switch r.Kind() {
    case reflect.Slice, reflect.Array:
        for i := 0; i < r.Len(); i++ {
            if reflect.DeepEqual(v, r.Index(i).Interface()) { return true }
        }
    case reflect.Map:
        for _, k := range r.MapKeys() {
            if reflect.DeepEqual(v, r.MapIndex(k).Interface()) { return true }
        }
    }
    return false
}
