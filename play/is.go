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
package play

import (
    "tlkm/is"
  . "github.com/tlkm"
)

func init() {
    PlayExport(PlayExportSignature{"utils_is": Is})
}

// Utilities untuk play script, unifikasi fungsi2 pengecekan dengan single parameter
// kedalam satu block: is
func Is(p *PlayContext, b *PlayBlock) PlayType {
    f := b.FieldName()
    if f == nil { return PlayFieldException("NAME") }
    v := p.Visit(b.BlockValue()).String(p)
    r := false
    switch f.Value {
        case "Alnum":
            r = is.Alnum(v)
        case "Alpha":
            r = is.Alpha(v)
        case "Date":
            r = is.Date(v)
        case "Digit":
            r = is.Digit(v)
        case "Email":
            r = is.Email(v)
        case "Latitude":
            r = is.Latitude(v)
        case "Longitude":
            r = is.Longitude(v)
        case "LowerCase":
            r = is.LowerCase(v)
        case "Numeric":
            r = is.Numeric(v)
        case "UpperCase":
            r = is.UpperCase(v)
        case "Boolean":
            r = is.Boolean(v)
        case "Empty":
            r = is.Empty(v)
        case "Float":
            r = is.Float(v)
        case "File":
            r = is.File(v)
        case "Folder":
            r = is.Folder(v)
        case "FileReadable":
            r = is.FileReadable(v)
        case "FileWriteable":
            r = is.FileWriteable(v)
    }
    return PlayBool(r)
}