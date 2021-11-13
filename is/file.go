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
    "os"
    "syscall"
)

func Folder(v string) bool {
    if f, e := os.Stat(v); e == nil { return f.Mode().IsDir() }
    return false
}

func File(v string) bool {
    if _, e := os.Stat(v); e != nil && os.IsNotExist(e) { return false }
    return true
}

func FileReadable(v string) bool {
    if _, e := syscall.Open(v, syscall.O_RDONLY, 0); e != nil { return false }
    return true
}

func FileWriteable(v string) bool {
    if _, e := syscall.Open(v, syscall.O_WRONLY, 0); e != nil { return false }
    return true
}
