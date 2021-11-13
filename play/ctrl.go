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

type (
    signal struct {
        K   int
        v   PlayType
    }
)

const (
    SigStop = iota + 2
    SigCont
    SigRtrn
)

func init() {
    PlayExport(PlayExportSignature{"controls_if": If,
        "controls_whileUntil": While, "controls_for": For, "controls_flow_statements": Break,
        "procedures_ifreturn": IfReturn})
}

func isBreak(v *bool) {
    if r := recover(); r != nil {
        if e, b := r.(signal); b && e.K != SigRtrn {
            *v = e.K == SigCont
        } else {
            panic(r)
        }
    }
}

func isReturn(v *PlayType) {
    if r := recover(); r != nil {
        if e, b := r.(signal); b && e.K == SigRtrn {
            *v = e.v
        } else {
            panic(r)
        }
    }
}

func If(p *PlayContext, b *PlayBlock) PlayType {
    var j, e int
    if b.Mutation != nil {
        j = b.Mutation.Elif
        e = b.Mutation.Else
    }
    for i := 0; i <= j; i++ {
        f := b.Block(Sprintf("IF%d", i))
        d := b.Stat(Sprintf("DO%d", i))
        if d != nil {
            if p.Visit(f).Boolean(p) {
                return p.Visit(d)
            }
        }
    }
    if e > 0 {
        if v := b.Stat("ELSE"); v != nil {
            return p.Visit(v)
        }
    }
    return PlayNull
}

func For(p *PlayContext, b *PlayBlock) PlayType {
    n := b.FieldVar()
    if n == nil {
        return PlayFieldException("VAR")
    }
    i := p.Visit(b.BlockFrom()).Float(p)
    j := p.Visit(b.BlockTo()).Float(p)
    k := math.Abs(p.Visit(b.BlockInc()).Float(p))
    u := b.Stat("DO")
    r := true
    var v PlayType = PlayNull
    var c func(a, b float64) bool
    if i <= j {
        c = func(a, b float64) bool {
            return a <= b
        }
    } else {
        c = func(a, b float64) bool {
            return a >= b
        }
        k = -k
    }
    for ; c(i, j) && r; i += k {
        p.Argv[n.Value] = PlayNumber(i)
        func() {
            defer isBreak(&r)
            v = p.Visit(u)
        }()
    }
    return v
}

func While(p *PlayContext, b *PlayBlock) PlayType {
    m := b.FieldMode()
    if m == nil {
        return PlayFieldException("MODE")
    }
    var v PlayType = PlayNull
    c := b.BlockBool()
    u := b.Stat("DO")
    r := true
    for r {
        switch m.Value {
        case "WHILE":
            r = p.Visit(c).Boolean(p)
        case "UNTIL":
            r = !(p.Visit(c).Boolean(p))
        }
        if r {
            func() {
                defer isBreak(&r)
                v = p.Visit(u)
            }()
        }
    }
    return v
}

func Break(p *PlayContext, b *PlayBlock) PlayType {
    if f := b.FieldFlow(); f != nil {
        switch f.Value {
        case "BREAK":
            panic(signal{K: SigStop})
        case "CONTINUE":
            panic(signal{K: SigCont})
        default:
            return PlayNull
        }
    }
    return PlayFieldException("FLOW")
}

func IfReturn(p *PlayContext, b *PlayBlock) PlayType {
    /*if p.Visit(b.BlockCondition()).Boolean(p) {
        panic(signal{K: SigRtrn, v: p.Visit(&b.BlockValue().Tree[0])})
    }*/
    return PlayNull
}