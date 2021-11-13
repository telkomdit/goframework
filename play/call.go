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
    PlayExport(PlayExportSignature{"procedures_defnoreturn": FuncCall, "procedures_defreturn": FuncCall})
}

func copyArgv(p *PlayContext, m map[string]PlayType) map[string]PlayType {
    r := make(map[string]PlayType)
    for k, v := range m {
        if o, e := p.Argv[k]; e {
            r[k] = o
        }
        p.Argv[k] = v
    }
    return r
}

func restore(p *PlayContext, m map[string]PlayType, o map[string]PlayType) {
    for k, _ := range m {
        delete(p.Argv, k)
        if v, e := o[k]; e {
            p.Argv[k] = v
        }
    }
}

func FuncCall(p *PlayContext, b *PlayBlock) PlayType {
    if b.Mutation == nil {
        return PlayBlockException("Mutation")
    }
    u, e := p.Func[b.Mutation.Name]
    if !e {
        return PlayFuncException(b.Mutation.Name)
    }
    v := make(map[string]PlayType)
    for i, j := range b.Mutation.Argv {
        v[j.Name] = p.Visit(b.Block(Sprintf("ARG%d", i)))
    }
    c := copyArgv(p, v)
    defer restore(p, v, c)
    var r PlayType
    if u.Func != nil {
        func() {
            defer isReturn(&r)
            p.Visit(u.Func)
        }()
    }
    if r == nil {
        r = PlayNull
        if u.Rtrn != nil {
            r = p.Visit(u.Rtrn)
        }
    }
    return r
}