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
  . "github.com/telkomdit/goframework/tlkm"
)

func init() {
    PlayExport(PlayExportSignature{"logic_compare": BoolCompare,
        "logic_ternary": BoolTernary, "logic_operation": BoolOperation,
        "logic_negate": BoolNegate})
}

func BoolCompare(p *PlayContext, b *PlayBlock) PlayType {
    A := p.Visit(b.BlockA())
    B := p.Visit(b.BlockB())
    o := b.FieldOper().Value
    var v bool
    switch o {
    case "EQ":
        v = A.EQ(p, B)
    case "NEQ":
        v = !A.EQ(p, B)
    case "LT":
        v = A.LT(p, B)
    case "LTE":
        v = A.LT(p, B) || A.EQ(p, B)
    case "GT":
        v = !A.LT(p, B) && !A.EQ(p, B)
    case "GTE":
        v = !A.LT(p, B)
    default:
        return PlayExit("BoolException: ", o)
    }
    return PlayBool(v)
}

func BoolTernary(p *PlayContext, b *PlayBlock) PlayType {
    v := p.Visit(b.BlockIf()).Boolean(p)
    if v {
        return p.Visit(b.BlockThen())
    } else {
        return p.Visit(b.BlockElse())
    }
}

func BoolOperation(p *PlayContext, b *PlayBlock) PlayType {
    A := p.Visit(b.BlockA()).Boolean(p)
    B := p.Visit(b.BlockB()).Boolean(p)
    o := b.FieldOper().Value
    var v bool
    switch o {
    case "AND":
        v = A && B
    case "OR":
        v = A || B
    case "XOR":
        v = (A || B) && !(A && B)
    default:
        return PlayExit("BoolException: ", o)
    }
    return PlayBool(v)
}

func BoolNegate(p *PlayContext, b *PlayBlock) PlayType {
    return PlayBool(!p.Visit(b.BlockBool()).Boolean(p))
}