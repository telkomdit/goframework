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

// Character Type Checking: impl <ctype.h> plus pengecekan tambahan
package is

import (
    "regexp"
    "unicode"
    "strconv"
    "github.com/tlkm/to"
)

var (
	regexEmail        = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$`)
	regexMacAddress   = regexp.MustCompile(`^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`)
	regexLatitude     = regexp.MustCompile(`^(\\+|-)?(?:90(?:(?:\\.0{1,6})?)|(?:[0-9]|[1-8][0-9])(?:(?:\\.[0-9]{1,6})?))$`)
	regexLongitude    = regexp.MustCompile(`^(\\+|-)?(?:180(?:(?:\\.0{1,6})?)|(?:[0-9]|[1-9][0-9]|1[0-7][0-9])(?:(?:\\.[0-9]{1,6})?))$`)
	regexURL          = regexp.MustCompile(`^(?:http(s)?:\\/\\/)?[\\w.-]+(?:\\.[\\w\\.-]+)+[\\w\\-\\._~:/?#[\\]@!\\$&'\\(\\)\\*\\+,;=.]+$`)
)

func lowerRune(v rune) bool {
    return ('a' <= v && v <= 'z')
}

func upperRune(v rune) bool {
    return ('A' <= v && v <= 'Z')
}

func digitRune(v rune) bool {
    return ('0' <= v && v <= '9')
}

// range: 0-9, a-z, A-Z
func Alnum(v string) bool {
    b := false
    for _, c := range v {
	    b = digitRune(c) || lowerRune(c) || upperRune(c)
        if !b { return false }
    }
    return true
}

// range: a-z, A-Z
func Alpha(v string) bool {
    b := false
    for _, c := range v {
	    b = lowerRune(c) || upperRune(c)
        if !b { return false }
    }
    return true
}

// range: a-z, A-Z plus _- sudah lupa kenapa dulu butuh fungsi wkwk
func AlphaDash(v string) bool {
    b := false
    for _, c := range v {
	    b = lowerRune(c) || upperRune(c) || (c == '_') || (c == '-')
        if !b { return false }
    }
    return true
}

// AlphaDash + space
func AlphaSpace(v string) bool {
    b := false
    for _, c := range v {
	    b = lowerRune(c) || upperRune(c) || (c == '_') || (c == '-') || (c == ' ')
        if !b { return false }
    }
    return true
}

// Memastikan tanggal valid sampe tahun 32767
func Date(v string) bool {
    var(
        Y, m, d int
        e error
    )
    T, b, h, _, _ := to.DateSplit(v)
    if Y, e = strconv.Atoi(T); e != nil { return false }
    if m, e = strconv.Atoi(b); e != nil { return false }
    if d, e = strconv.Atoi(h); e != nil { return false }

    // 32767: dari manual fungsi PHP datecheck
    if m < 1 || m > 12 || d < 1 || d > 31 || Y < 1 || Y > 32767 { return false }
    switch m {
    case 4, 6, 9, 11:
        if d > 30 { return false }
    case 2:
        if Y%4 == 0 && (Y%100 != 0 || Y%400 == 0) {
            if d > 29 { return false }
        } else if d > 28 {
            return false
        }
    }
    return true
}

// range: 0-9
func Digit(v string) bool {
    for _, c := range v {
        if !digitRune(c) { return false }
    }
    return true
}

func Email(v string) bool {
    return regexEmail.MatchString(v)
}

func Latitude(v string) bool {
    return regexLatitude.MatchString(v)
}

func Longitude(v string) bool {
    return regexLongitude.MatchString(v)
}

func LowerCase(s string) bool {
    for _, r := range s {
        if !unicode.IsLower(r) && unicode.IsLetter(r) {
            return false
        }
    }
    return true
}

func MacAddress(v string) bool {
    return regexMacAddress.MatchString(v)
}

func Numeric(v string) bool {
    b := false
    i := false
    for j, c := range v {
	    b = digitRune(c) || (c == '-' && j == 0)    // digit plus sign/unsign plus decimal
        if c == '.' {
            if i { return false }
            i = true
        }
        if !b { return false }
    }
    return true
}

func UpperCase(s string) bool {
    for _, r := range s {
        if !unicode.IsUpper(r) && unicode.IsLetter(r) {
            return false
        }
    }
    return true
}

func URL(v string) bool {
    return regexURL.MatchString(v)
}
