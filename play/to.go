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
    "errors"
    "github.com/telkomdit/goframework/to"
  . "github.com/telkomdit/goframework/tlkm"
)

func init() {
    PlayExport(PlayExportSignature{"utils_to": To})
}

// Utilities untuk play script, unifikasi fungsi2 transformasi dengan single parameter
// kedalam satu block: to
func To(p *PlayContext, b *PlayBlock) PlayType {
    f := b.FieldName()
    if f == nil { return PlayFieldException("NAME") }
    v := p.Visit(b.BlockValue()).String(p)
    var (
        r string
        e error
    )
    switch f.Value {
        case "String":
            r = to.String(v)
        case "Digit":
            r = to.Digit(v)
        case "Numeric":
            r = to.Numeric(v)
        case "Date":
            r = to.Date(v)
        case "CamelCase":
            r = to.CamelCase(v)
        case "SnakeCase":
            r = to.SnakeCase(v)
        case "Escape":
            r = to.Escape(v)
        case "Namespace":
            r = to.Namespace(v)
        case "Reverse":
            r = to.Reverse(v)
        case "SlashForward":
            r = to.SlashForward(v)
        case "Shuffle":
            r = to.Shuffle(v)
        case "LowerCase":
            r = to.LowerCase(v)
        case "LowerFirst":
            r = to.LowerFirst(v)
        case "UpperWords":
            r = to.UpperWords(v)
        case "URLDecode":
            r = to.URLDecode(v)
        case "URLEncode":
            r = to.URLEncode(v)
        case "MD5":
            r = to.MD5(v)
        case "MD5File":
            r, e = to.MD5File(v)
        case "Sha1":
            r = to.Sha1(v)
        case "Sha1File":
            r, e = to.Sha1File(v)
        case "Base64":
            r = to.Base64(v)
        case "Crc32":
            r = to.String(to.Crc32(v))
        default:
            e = errors.New("Opo jare umak rah wes kah")
    }
    if e != nil {
        return PlayFuncException(f.Value)
    }
    return PlayString(r)
}