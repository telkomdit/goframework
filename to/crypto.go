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
    "crypto/md5"
    "crypto/sha1"
    "encoding/hex"
    "hash/crc32"
    "io/ioutil"
)

func MD5(v string) string {
    m := md5.New()
    m.Write([]byte(v))
    return hex.EncodeToString(m.Sum(nil))
}

func MD5File(v string) (string, error) {
    r, e := ioutil.ReadFile(v)
    if e != nil {
        return "", e
    }
    m := md5.New()
    m.Write([]byte(r))
    return hex.EncodeToString(m.Sum(nil)), nil
}

func Sha1(v string) string {
    m := sha1.New()
    m.Write([]byte(v))
    return hex.EncodeToString(m.Sum(nil))
}

func Sha1File(v string) (string, error) {
    r, e := ioutil.ReadFile(v)
    if e != nil {
        return "", e
    }
    m := sha1.New()
    m.Write([]byte(r))
    return hex.EncodeToString(m.Sum(nil)), nil
}

func Crc32(v string) uint32 {
    return crc32.ChecksumIEEE([]byte(v))
}