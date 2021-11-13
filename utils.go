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

// Beberapa fungsi (dengan block yang sama berbeda signature) diulang karena
// golang tidak mengenal generic (https://go.dev/blog/why-generics)
//
// utils.go masuk sebagai core karena hanya shortcut untuk operasi2 primitive
// yang bisa digunakan/tidak oleh modules
package tlkm

import (
    "io/ioutil"
)

func Merge_interface(v... *[]interface{}) []interface{} {
    n := 0
    for _, a := range v { n+= len(*a) }
    m := make([]interface{}, 0, n)
    for _, a := range v {
        m = append(m, *a...)
    }
    return m
}

func Merge_bool(v... *[]bool) []bool {
    n := 0
    for _, a := range v { n+= len(*a) }
    m := make([]bool, 0, n)
    for _, a := range v {
        m = append(m, *a...)
    }
    return m
}

func Merge_int(v... *[]int) []int {
    n := 0
    for _, a := range v { n+= len(*a) }
    m := make([]int, 0, n)
    for _, a := range v {
        m = append(m, *a...)
    }
    return m
}

func Merge_string(v... *[]string) []string {
    n := 0
    for _, a := range v { n+= len(*a) }
    m := make([]string, 0, n)
    for _, a := range v {
        m = append(m, *a...)
    }
    return m
}

func CombineToGMap(k *[]string, v *[]interface{}) GMap {
    i, j := len(*k), len(*v)
    l := i
    if j < i { l = j}
    m := make(GMap, l)
    l-= 1
    for i, j := range *k {
        if i > l { break }
        m[j] = (*v)[i]
    }
    return m
}

func CombineToSMap(k, v *[]string) SMap {
    i, j := len(*k), len(*v)
    l := i
    if j < i { l = j}
    m := make(SMap, l)
    l-= 1
    for i, j := range *k {
        if i > l { break }
        m[j] = (*v)[i]
    }
    return m
}

func CombineToBMap(k *[]string, v *[]bool) BMap {
    i, j := len(*k), len(*v)
    l := i
    if j < i { l = j}
    m := make(BMap, l)
    l-= 1
    for i, j := range *k {
        if i > l { break }
        m[j] = (*v)[i]
    }
    return m
}

func CombineToIMap(k *[]string, v *[]int) IMap {
    i, j := len(*k), len(*v)
    l := i
    if j < i { l = j}
    m := make(IMap, l)
    l-= 1
    for i, j := range *k {
        if i > l { break }
        m[j] = (*v)[i]
    }
    return m
}

func Shift_interface(v *[]interface{}) (rv interface{}) {
    if len(*v) > 0 {
        rv = (*v)[0]
        *v = (*v)[1:]
    }
    return
}

func Shift_bool(v *[]bool) (rv bool) {
    if len(*v) > 0 {
        rv = (*v)[0]
        *v = (*v)[1:]
    }
    return
}

func Shift_int(v *[]int) (rv int) {
    if len(*v) > 0 {
        rv = (*v)[0]
        *v = (*v)[1:]
    }
    return
}

func Shift_string(v *[]string) (rv string) {
    if len(*v) > 0 {
        rv = (*v)[0]
        *v = (*v)[1:]
    }
    return
}

func Shift(v interface{}) (rv interface{}) {
    switch u := v.(type) {
    case *[]interface{}:
        return Shift_interface(u)
    case *[]bool:
        return Shift_bool(u)
    case *[]int:
        return Shift_int(u)
    case *[]string:
        return Shift_string(u)
    }
    return
}

func Unshift(v *[]interface{}, e... interface{}) {
    *v = append(e, *v...)
}

func Unshift_bool(v *[]bool, e... bool) {
    *v = append(e, *v...)
}

func Unshift_int(v *[]int, e... int) {
    *v = append(e, *v...)
}

func Unshift_string(v *[]string, e... string) {
    *v = append(e, *v...)
}

func Pop_interface(v *[]interface{}) (rv interface{}) {
    if ln := len(*v); ln > 0 {
        of := ln - 1
        rv = (*v)[of]
        *v = (*v)[:of]
    }
    return
}

func Pop_bool(v *[]bool) (rv bool) {
    if ln := len(*v); ln > 0 {
        of := ln - 1
        rv = (*v)[of]
        *v = (*v)[:of]
    }
    return
}

func Pop_int(v *[]int) (rv int) {
    if ln := len(*v); ln > 0 {
        of := ln - 1
        rv = (*v)[of]
        *v = (*v)[:of]
    }
    return
}

func Pop_string(v *[]string) (rv string) {
    if ln := len(*v); ln > 0 {
        of := ln - 1
        rv = (*v)[of]
        *v = (*v)[:of]
    }
    return
}

func Pop(v interface{}) (rv interface{}) {
    switch u := v.(type) {
    case *[]interface{}:
        return Pop_interface(u)
    case *[]bool:
        return Pop_bool(u)
    case *[]int:
        return Pop_int(u)
    case *[]string:
        return Pop_string(u)
    }
    return
}

func SMapKeys(v *SMap) List {
    i, a := 0, make(List, len(*v))
    for e, _ := range *v {
        a[i] = e
        i++
    }
    return a
}

func GMapKeys(v *GMap) List {
    i, a := 0, make(List, len(*v))
    for e, _ := range *v {
        a[i] = e
        i++
    }
    return a
}

func BMapKeys(v *BMap) List {
    i, a := 0, make(List, len(*v))
    for e, _ := range *v {
        a[i] = e
        i++
    }
    return a
}

func IMapKeys(v *IMap) List {
    i, a := 0, make(List, len(*v))
    for e, _ := range *v {
        a[i] = e
        i++
    }
    return a
}

func MapKeys(v interface{}) List {
    switch u := v.(type) {
    case *GMap:
        return GMapKeys(u)
    case *SMap:
        return SMapKeys(u)
    case *BMap:
        return BMapKeys(u)
    case *IMap:
        return IMapKeys(u)
    }
    return nil
}

func SMapValues(v *SMap) List {
    i, a := 0, make(List, len(*v))
    for _, e := range *v {
        a[i] = e
        i++
    }
    return a
}

func GMapValues(v *GMap) []interface{} {
    i, a := 0, make([]interface{}, len(*v))
    for _, e := range *v {
        a[i] = e
        i++
    }
    return a
}

func BMapValues(v *BMap) []bool {
    i, a := 0, make([]bool, len(*v))
    for _, e := range *v {
        a[i] = e
        i++
    }
    return a
}

func IMapValues(v *IMap) []int {
    i, a := 0, make([]int, len(*v))
    for _, e := range *v {
        a[i] = e
        i++
    }
    return a
}

func SMapFlip(v *SMap) {
    n := make(SMap)
    for i, j := range *v {
        n[j] = i
    }
    *v = n
}

func GMapFlip(v *GMap) (m map[interface{}]string) {
    m = make(map[interface{}]string)
    for i, j := range *v {
        m[j] = i
    }
    return
}

func IMapFlip(v *IMap) (m map[int]string) {
    m = make(map[int]string)
    for i, j := range *v {
        m[j] = i
    }
    return
}

func ListFolder(root string) (v []string, e error) {
    r, e := ioutil.ReadDir(root)
    if e != nil {
        return v, e
    }
    for _, f := range r {
        v = append(v, f.Name())
    }
    return v, nil
}