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
    "strconv"
  . "github.com/telkomdit/goframework/tlkm"
)

func init() {
    PlayExport(PlayExportSignature{"math_number": Number,
        "text": String, "logic_boolean": Boolean, "variables_set": VarSet,
        "variables_get": VarGet, "text_print": TextPrint, "text_println": TextPrintln})
}

func Number(p *PlayContext, b *PlayBlock) PlayType {
    if f := b.FieldNmbr(); f != nil {
        v, e := strconv.ParseFloat(f.Value, 64)
        if e != nil {
            return PlayExit(e.Error())
        }
        return PlayNumber(v)
    }
    return PlayFieldException("NUM")
}

func Boolean(p *PlayContext, b *PlayBlock) PlayType {
    if f, b := b.FieldBool(), false; f != nil {
        if f.Value == "TRUE" { b = true }
        return PlayBool(b)
    }
    return PlayFieldException("BOOL")
}

func String(p *PlayContext, b *PlayBlock) PlayType {
    if f := b.FieldText(); f != nil {
        return PlayString(f.Value)
    }
    return PlayFieldException("TEXT")
}

func VarSet(p *PlayContext, b *PlayBlock) PlayType {
    if f := b.FieldVar(); f != nil {
        p.Argv[f.Value] = p.Visit(b.BlockValue())
        return PlayNull
    }
    return PlayFieldException("VAR")
}

func VarGet(p *PlayContext, b *PlayBlock) PlayType {
    if f := b.FieldVar(); f != nil {
        if v, e := p.Argv[f.Value]; e {
            return v
        }
        return PlayNull
    }
    return PlayFieldException("VAR")
}

func TextPrint(p *PlayContext, b *PlayBlock) PlayType {
    p.Cntx.Echo(p.Visit(b.BlockText()).String(p))
    return PlayNull
}

func TextPrintln(p *PlayContext, b *PlayBlock) PlayType {
    p.Cntx.Echo(p.Visit(b.BlockText()).String(p))
    p.Cntx.Echo("\r\n")
    return PlayNull
}