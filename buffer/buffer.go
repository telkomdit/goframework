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

// Masuk sebagai core untuk memenuhi kebutuhan manipulasi (byte) buffer. Digunakan
// di H2G untuk membentuk source-code go dari HTML. Akan dipertahankan seminimal
// dan secompact mungkin selama memenuhi kebutuhan framework
//
// referensi: https://segmentfault.com/a/1190000039969499/en
package buffer

import (
	"io"
	"sort"
	"sync"
	"sync/atomic"
)

const (
    // Kapasitas buffer akan bervariasi, sesuai kebutuhan pada saat digunakan.
    // Karena ekspansi kapasitas (alokasi memory) akan memakan waktu, kalibrasi
    // kapasitas awal buffer dibutuhkan untuk seminimal mungkin mengeleminasi
    // re-alokasi (grow pada saat digunakan)
    //
    // Kapasitas buffer akan dibagai menjadi 20 interval dari 2^6 sd 2^25 dengan
    // nilai interval berdasarkan statistik kapasitas paling banyak dalam pool
	bitSize = 6     // CPU cache umumnya akan mengambil 64 bytes (2^6) dalam setiap lookup atau simpan
    steps   = 20    // dalam hal ini, slice diusahakan inline dengan bare metal wkwk

    // Kapasitas buffer dari 64 sd maxSize
    minSize = 1 << bitSize
	maxSize = 1 << (bitSize + steps - 1)
)

type (
    ByteBuffer struct {
        B []byte
    }

    // ** private **
    buffer struct {
        calls       [steps]uint64
        calibrating uint64
        defaultSize uint64
        maxSize     uint64
        pool        sync.Pool
    }
    callSize struct {
        calls uint64
        size  uint64
    }
    callSizes []callSize
)

var obj buffer

func Get() *ByteBuffer {
    return obj.Get()
}

func (self *ByteBuffer) Close() {
    obj.Put(self)
}

func (self *ByteBuffer) Len() int {
	return len(self.B)
}

func (self *ByteBuffer) Reset() {
	self.B = self.B[:0]
}

func (self *ByteBuffer) Read(r io.Reader) (int64, error) {
	p := self.B
	s := int64(len(p))
	x := int64(cap(p))
	n := s
	if x == 0 {
		x = 64
		p = make([]byte, x)
	} else {
		p = p[:x]
	}
	for {
		if n == x {
			x *= 2
			n := make([]byte, x)
			copy(n, p)
			p = n
		}
		i, e := r.Read(p[n:])
		n += int64(i)
		if e != nil {
			self.B = p[:n]
			n -= s
			if e == io.EOF {
				return n, nil
			}
			return n, e
		}
	}
}

func (self *ByteBuffer) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(self.B)
    return int64(n), err
}

func (self *ByteBuffer) Bytes() []byte {
	return self.B
}

// Write Byte
func (self *ByteBuffer) WB(c byte) *ByteBuffer {
	self.B = append(self.B, c)
    return self
}

// Write Rune
func (self *ByteBuffer) WRune(c rune) *ByteBuffer {
    return self.WB(byte(c))
}

// Write array of byte
func (self *ByteBuffer) WR(b []byte) *ByteBuffer {
	self.B = append(self.B, b...)
    return self
}

// Write String
func (self *ByteBuffer) WS(s string) *ByteBuffer {
	self.B = append(self.B, s...)
    return self
}

// Write Line: string + CRLF
func (self *ByteBuffer) WL(s string) *ByteBuffer {
    return self.WS(s).NL()
}

// New Line: Carriage Return + Line Feed
func (self *ByteBuffer) NL() *ByteBuffer {
	self.B = append(self.B, '\r', '\n')
    return self
}

func (self *ByteBuffer) Set(b []byte) {
	self.B = append(self.B[:0], b...)
}

func (self *ByteBuffer) SetString(s string) {
	self.B = append(self.B[:0], s...)
}

func (self *ByteBuffer) String() string {
	return string(self.B)
}

// Method akan memotong byte array 1 byte jika hanya ada line feed (\n) atau 2 byte
// jika posisi sebelumnya carriage return
//
// Sejauh ini hanya digunakan di H2G karena adanya method untuk memenuhi kebutuhan H2G
func (self *ByteBuffer) RemLine() {
    l := self.Len()
    if l > 0 {
        l-= 1
        if self.B[l] == '\n' { self.B = self.B[:l] }
        l-= 1
        if self.B[l] == '\r' { self.B = self.B[:l] }
    }
}

// ** buffer impl **
//
// Kapasitas buffer akan dikalibrasi ulang pada saat object dikembalikan ke pool
//
// referensi: https://segmentfault.com/a/1190000039969499/en
//
func (self *buffer) Get() *ByteBuffer {
	v := self.pool.Get()
	if v != nil {
		return v.(*ByteBuffer)
	}
    return &ByteBuffer{
		B: make([]byte, 0, atomic.LoadUint64(&self.defaultSize)),
	}
}

func (self *buffer) Put(b *ByteBuffer) {
    n := len(b.B) - 1
	n >>= bitSize
	idx := 0
	for n > 0 {
		n >>= 1
		idx++
	}
	if idx >= steps { idx = steps - 1 }
	if atomic.AddUint64(&self.calls[idx], 1) > 42000 { self.calibrate() }
	maxSize := int(atomic.LoadUint64(&self.maxSize))
	if maxSize == 0 || cap(b.B) <= maxSize {
		b.Reset()
		self.pool.Put(b)
	}
}

func (self *buffer) calibrate() {
	if !atomic.CompareAndSwapUint64(&self.calibrating, 0, 1) { return }
	a := make(callSizes, 0, steps)
	var callsSum uint64
	for i := uint64(0); i < steps; i++ {
		calls := atomic.SwapUint64(&self.calls[i], 0)
		callsSum += calls
		a = append(a, callSize{
			calls: calls,
			size:  minSize << i,
		})
	}
	sort.Sort(a)
	defaultSize := a[0].size
	maxSize := defaultSize
	maxSum := uint64(float64(callsSum) * 0.95)
	callsSum = 0
	for i := 0; i < steps; i++ {
		if callsSum > maxSum { break }
		callsSum += a[i].calls
		size := a[i].size
		if size > maxSize { maxSize = size }
	}
	atomic.StoreUint64(&self.defaultSize, defaultSize)
	atomic.StoreUint64(&self.maxSize, maxSize)
	atomic.StoreUint64(&self.calibrating, 0)
}

// ** sort interface **
func (self callSizes) Len() int {
	return len(self)
}
func (self callSizes) Less(i, j int) bool {
	return self[i].calls > self[j].calls
}
func (self callSizes) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}
