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

package tlkm

import (
    "testing"
)

func TestUtilsMerge(t *testing.T) {
    v := []interface{}{"utas", "aud"}
    u := []interface{}{"agit"}
    r := Merge_interface(&v, &u)

    t.Log(r)
}

func TestUtilsCombineToSMap(t *testing.T) {
    v := []string{"utas", "aud"}
    u := []string{"agit", "papat", "amil"}
    r := CombineToSMap(&u, &v)

    t.Log(r)
}

func TestUtilsCombineToIMap(t *testing.T) {
    v := []string{"utas", "aud"}
    u := []int{1, 2, 3}
    r := CombineToIMap(&v, &u)

    t.Log(r)
}

func TestUtilsShift(t *testing.T) {
    v := []string{"utas", "aud"}

    r := Shift(&v)

    t.Log(r)
    t.Log(v)
}

func TestUtilsUnshift(t *testing.T) {
    v := []int{1, 2}

    Unshift_int(&v, 3)

    t.Log(v)
}

func TestUtilsPop(t *testing.T) {
    v := []string{"utas", "aud"}

    r := Pop(&v)

    t.Log(r)
    t.Log(v)
}

func TestUtilsSMapKeys(t *testing.T) {
    v := IMap{"utas": 1, "aud": 2}

    m := IMapKeys(&v)

    t.Log(m)
}
