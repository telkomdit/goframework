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
    "fmt"
    "math/rand"
    "reflect"
    "strconv"
    "strings"
    "time"
    "unicode"
    "unicode/utf8"
    "github.com/tlkm/buffer"
)

func CamelCase(v string) string {
    b := buffer.Get()
    defer b.Close()
    r := true
    for _, j := range v {
	    if ('0' <= j && j <= '9') || ('a' <= j && j <= 'z') || ('A' <= j && j <= 'Z') {
            if r {
                j = unicode.ToUpper(j)
                r = false
            } else {
                j = unicode.ToLower(j)
            }
            b.WRune(j)
        } else {
            r = true
        }
    }
    return b.String()
}

func Chr(v int) string {
    return string(v)
}

func Digit(v string) string {
    b := buffer.Get()
    defer b.Close()
    for _, c := range v {
	    if '0' <= c && c <= '9' {
            b.WRune(c)
        }
    }
    return b.String()
}

func Escape(v string) string {
    b := buffer.Get()
    defer b.Close()
    for _, c := range v {
        switch c {
        case '\'', '"', '\\', '\a', '\b', '\f', '\n', '\r', '\t':
            b.WRune('\\')
        }
        b.WRune(c)
    }
    return b.String()
}

func Explode(s, v string, limit... int) []string {
    r := strings.Split(v, s)
    if len(limit) > 0 {
        if limit[0] <= len(r) {
            r = r[0:limit[0]-1]
        }
    }
    return r
}

func Implode(s string, v []string) string {
    return strings.Join(v[:], s)
}

func LowerCase(v string) string {
    return strings.ToLower(v)
}

func LowerFirst(v string) string {
    for _, r := range v {
        u := string(unicode.ToLower(r))
        return u + v[len(u):]
    }
    return ""
}

func Namespace(v string) string {
    b := buffer.Get()
    defer b.Close()
    for _, c := range v {
        switch c {
            case '/':
                b.WRune('.')
            break
            default:
                b.WRune(c)
        }
    }
    return b.String()
}

func Numeric(v string) string {
    b := buffer.Get()
    defer b.Close()
    i := true
    for j, c := range v {
	    if '0' <= c && c <= '9' {
            b.WRune(c)
        } else {
            if j == 0 && c == '-' {
                b.WRune(c)  // sign/unsign kita anggap bagian dari numeric untuk membedakan dengan to.Digit
            } else {
                if i && c == '.' {
                    b.WRune(c)
                    i = false   // hanya 1x separator decimal
                }
            }
        }
    }
    return b.String()
}

func Ord(v string) int {
    r, _ := utf8.DecodeRune([]byte(v))
    return int(r)
}

func Reverse(v string) string {
    r := []rune(v)
    for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
        r[i], r[j] = r[j], r[i]
    }
    return string(r)
}

func SlashForward(p string) string {
    b := buffer.Get()
    defer b.Close()
    for _, c := range p {
        switch c {
        case '\\':
            b.WRune('/')
        default:
            b.WRune(c)
        }
    }
    return b.String()
}

// Untuk memenuhi kebutuhan framework mentransformasikan nama struct pada active record
// dalam format camel case ke penamaan tabel snake case
func SnakeCase(v string) string {
    b := buffer.Get()
    defer b.Close()
    upper := true
    lower := true
    digit := true
    snake := false
    for i, j := range v {
        upper = ('A' <= j && j <= 'Z')
        lower = ('a' <= j && j <= 'z')
        digit = ('0' <= j && j <= '9')
	    if digit || lower || upper {
            if i > 0 && upper { snake = true }
            if i > 0 && snake {
                b.WRune('_')
                snake = false
            }
            b.WRune(unicode.ToLower(j))
        } else {
            snake = true
        }
    }
    return b.String()
}

func String(v interface{}) string {
    switch u := v.(type) {
    case string:
        return u
    case *string:
        return *u
    case time.Time, *time.Time:
        t, ok := u.(time.Time)
        if !ok {
            p, _ := u.(*time.Time)
            t = *p
        }
        if t.Hour() + t.Minute() + t.Second() + t.Nanosecond() == 0 {
            return t.Format("2006-01-02")
        }
        return t.Format("2006-01-02 15:04:05")
    case byte:
        return string(u)
    case rune:
        return string(u)
    case []byte:
        return string(u)
    case bool:
        return strconv.FormatBool(u)
    case *bool:
        return strconv.FormatBool(*u)
    case int:
        return strconv.Itoa(u)
    case *int:
        return strconv.Itoa(*u)
    case *int8, *int16, *int32, *int64, *uint, *uint16, *uint32, *uint64, *uintptr, *float32, *float64, *complex64, *complex128:
        return fmt.Sprintf("%v", reflect.ValueOf(u).Elem())
    default:
        return fmt.Sprintf("%v", u)
    }
}

func Shuffle(v string) string {
    u := []rune(v)
    r := rand.New(rand.NewSource(time.Now().UnixNano()))
    s := make([]rune, len(u))
    for i, v := range r.Perm(len(u)) {
        s[i] = u[v]
    }
    return string(s)
}

func UpperCase(v string) string {
    return strings.ToUpper(v)
}

func UpperFirst(v string) string {
    for _, r := range v {
        u := string(unicode.ToUpper(r))
        return u + v[len(u):]
    }
    return ""
}

func UpperWords(v string) string {
    return strings.Title(v)
}

func WordWrap(v string, ll int, sp string, cut bool) string {
    sl := len(v)
    if sl == 0 { return "" }
    bl := len(sp)
    if bl == 0 { sp = "\r\n" }
    if ll <= 0 { ll = 80 }
    cp, lp, ls := 0, 0, 0
    var ns []byte
    for cp = 0; cp < sl; cp++ {
        if v[cp] == sp[0] && cp + bl < sl && v[cp:cp+bl] == sp {
            ns = append(ns, v[lp:cp+bl]...)
            cp += bl - 1
            ls = cp + 1
            lp = ls
        } else if v[cp] == ' ' {
            if cp - lp >= ll {
                ns = append(ns, v[lp:cp]...)
                ns = append(ns, sp[:]...)
                lp = cp + 1
            }
            ls = cp
        } else if cp - lp >= ll && cut && lp >= ls {
            ns = append(ns, v[lp:cp]...)
            ns = append(ns, sp[:]...)
            lp = cp
            ls = cp
        } else if cp - lp >= ll && lp < ls {
            ns = append(ns, v[lp:ls]...)
            ns = append(ns, sp[:]...)
            ls++
            lp = ls
        }
    }
    if lp != cp {
        ns = append(ns, v[lp:cp]...)
    }
    return string(ns)
}
