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
    "math"
  . "github.com/telkomdit/goframework/tlkm"
)

func init() {
    PlayExport(PlayExportSignature{"math_arithmetic": MathArithmetic})
}

func MathArithmetic(p *PlayContext, b *PlayBlock) PlayType {
    A := p.Visit(b.BlockA()).Float(p)
    B := p.Visit(b.BlockB()).Float(p)
    o := b.FieldOper().Value
    var v float64
    switch o {
    case "ADD":
        v = A + B
    case "MIN":
        v = A - B
    case "MUL":
        v = A * B
    case "DIV":
        v = A / B
    case "POW":
        v = math.Pow(A, B)
    default:
        return PlayExit("MathArithmeticException: ", o)
    }
    return PlayNumber(v)
}
