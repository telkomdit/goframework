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

// Karena framework di desain coupling dengan database, karena alasan performansi,
// seluruh informasi (normalnya yang sifatnya parameter/konfigurasi/session) yang
// ada di database akan di dump ke cache
//
// Bad practice? tapi harga yang harus dibayar sesuai dengan tujuan sebagai
// framework untuk memenuhi kebutuhan transaksional
//
// Public method yang disediakan (sementara) memenuhi semua kebutuhan tipe data
// dan casting semua type wrapper yang digunakan oleh framework
package tlkm

import (
    "runtime"
    "strconv"
    "sync"
    "time"
)

type (
    // Map (di golang) tidak menjamin hasil dump sesuai dengan urutan pada saat
    // item dimasukkan. Key(GMap|SMap|BMap|IMap) dibuat untuk menyimpan urutan
    // item kedalam []string (key/index) sebagai referensi
    //
    // Gunakan struktur Key*Map jika map membutuhkan urutan item untuk iterasi
    // atau untuk di dump. Jadi triknya adalah iterasi field Key ([]string) dan
    // gunakan valuenya untuk lookup Map
    KeyMap struct {
        Key List
        Map GMap
    }

    KeySMap struct {
        Key List
        Map SMap
    }

    KeyBMap struct {
        Key List
        Map BMap
    }

    KeyIMap struct {
        Key List
        Map IMap
    }

    // ** private **
    value struct {
        Object interface{}
        expire int64
    }

    // ** private **
    cache struct {
        items  *sync.Map
        start  *check
    }

    // ** private **
    check struct {
        du time.Duration
        ch chan bool
    }
)

// ** GLOBAL **
var (
    Cache *cache
)

// Single instance Cache, mekanisme access dll kita percayakan kepada sync.Map
func init() {
    Cache = &cache{
        items: &sync.Map{},
        start: &check{
            du: 10 * time.Minute,
            ch: make(chan bool),
        },
    }
    go Cache.start.run(Cache)
    runtime.SetFinalizer(Cache, finalize)
}

func finalize(c *cache) {
    c.start.ch <- true
}

func (self *check) run(c *cache) {
    t := time.NewTicker(self.du)
    for {
        select {
        case <-t.C:
            c.DeleteExpired()
        case <-self.ch:
            t.Stop()

            return
        }
    }
}

func (self *cache) Extend(k string, d time.Duration) {
    var e int64 = time.Now().Add(d * time.Second).Unix()
    if v, x := self.items.Load(k); x {
        o := v.(value)
        o.expire = e
        self.items.Store(k, o)
    }
}

func (self *cache) Set(k string, x interface{}, d ...time.Duration) {
    var e int64 = 0
    if len(d) > 0 {
        e = time.Now().Add(d[0] * time.Second).Unix()
    }
    self.items.Store(k, value{
        Object: x,
        expire: e,
    })
}

func (self *cache) Exists(k string) bool {
    _, e := self.items.Load(k)

    return e
}

// By default, cache return adalah interface{}
func (self *cache) Get(k string) (interface{}, bool) {
    v, e := self.items.Load(k)
    if !e {
        return nil, false
    }
    o := v.(value)
    if o.expire > 0 {
        if time.Now().Unix() > o.expire {
            return nil, false
        }
    }

    return o.Object, true
}

// WARNING: hanya jika yakin yang kita ambil adalah string/int
func (self *cache) String(k string) (i string, j bool) {
    j = false
    if k, v := self.Get(k); v {
        j = true
        switch v := k.(type) {
        case string:
            i = v
        case int:
            i = strconv.Itoa(v)
        default:
            j = false
        }
    }

    return
}

// WARNING: hanya jika yakin yang kita ambil adalah string/int
func (self *cache) Int(k string) (i int, j bool) {
    j = false
    if k, v := self.Get(k); v {
        j = true
        switch v := k.(type) {
        case int:
            j = true
            i = v
        case string:
            j = false
            if n, e := strconv.Atoi(k.(string)); e == nil {
                j = true
                i = n
            }
        }
    }

    return
}

// WARNING: hanya jika yakin yang kita ambil adalah string/int/bool
func (self *cache) Bool(k string) (i bool, j bool) {
    j = false
    if k, v := self.Get(k); v {
        j = true
        switch v := k.(type) {
        case bool:
            i = v
        case string:
            j = false
            if b, e := strconv.ParseBool(v); e == nil {
                i = b
            }
        default:
            j = false
        }
    }

    return
}

// casting GMap
func (self *cache) GMap(k string) (i GMap, j bool) {
    j = false
    if k, v := self.Get(k); v {
        i = k.(GMap)
        j = true
    }

    return
}

// casting KeyMap
func (self *cache) KeyMap(k string) (Key List, Map GMap, j bool) {
    j = false
    if k, v := self.Get(k); v {
        i := k.(KeyMap)
        Key = i.Key
        Map = i.Map
        j = true
    }

    return
}

// casting KeySMap
func (self *cache) KeySMap(k string) (Key List, Map SMap, j bool) {
    j = false
    if k, v := self.Get(k); v {
        i := k.(KeySMap)
        Key = i.Key
        Map = i.Map
        j = true
    }

    return
}

// casting KeyBMap
func (self *cache) KeyBMap(k string) (Key List, Map BMap, j bool) {
    j = false
    if k, v := self.Get(k); v {
        i := k.(KeyBMap)
        Key = i.Key
        Map = i.Map
        j = true
    }

    return
}

// casting KeyIMap
func (self *cache) KeyIMap(k string) (Key List, Map IMap, j bool) {
    j = false
    if k, v := self.Get(k); v {
        i := k.(KeyIMap)
        Key = i.Key
        Map = i.Map
        j = true
    }

    return
}

// casting SMap
func (self *cache) SMap(k string) (i SMap, j bool) {
    j = false
    if k, v := self.Get(k); v {
        i = k.(SMap)
        j = true
    }

    return
}

// casting BMap
func (self *cache) BMap(k string) (i BMap, j bool) {
    j = false
    if k, v := self.Get(k); v {
        i = k.(BMap)
        j = true
    }

    return
}

// casting BMap
func (self *cache) IMap(k string) (i IMap, j bool) {
    j = false
    if k, v := self.Get(k); v {
        i = k.(IMap)
        j = true
    }

    return
}

func (self *cache) Delete(k string) {
    self.items.Delete(k)
}

func (self *cache) DeleteExpired() {
    now := time.Now().Unix()
    self.items.Range(func(k, v interface{}) bool {
        o := v.(value)
        if o.expire > 0 && o.expire < now {
            self.items.Delete(k)
        }

        return true
    })
}

func (self *cache) GetExpired(k string, d int64) (int64, bool) {
    if v, e := self.items.Load(k); e {
        o := v.(value)
        if o.expire > 0 && o.expire > d {
            return o.expire, true
        }
    }

    return 0, false
}

// Akan dipanggil melalui menu admin (modules/syst) untuk clear cache secara manual
// pada skenario ada update parameter/config database, belum ada menu/UI tapi
// harus masuk cache. wtf?! wkwk
func (self *cache) Flush() {
    self.items = &sync.Map{}
}
