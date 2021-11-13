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
    "strconv"
)

func Base(v string, from, to int) (string, error) {
    i, e := strconv.ParseInt(v, from, 0)
    if e != nil {
        return "", e
    }
    return strconv.FormatInt(i, to), nil
}

func Binary(v int64) string {
    return strconv.FormatInt(v, 2)
}

func Octa(v int64) string {
    return strconv.FormatInt(v, 8)
}

func Hexa(v int64) string {
    return strconv.FormatInt(v, 16)
}

func Int(v string) int {
    u := Numeric(v)
    if u == "" { u = "0" }
    i, _ := strconv.Atoi(u)

    return i
}
