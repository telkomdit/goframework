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
    PlayExport(PlayExportSignature{"map_create": PlayMapCreate,
        "map_size": PlayMapSize, "map_get": PlayMapGet, "map_exists": PlayMapExists,
        "map_put": PlayMapPut, "map_delete": PlayMapDelete})
}

func PlayMapCreate(p *PlayContext, b *PlayBlock) PlayType {
    T := make(map[PlayType]PlayType, 0)
    K := make([]PlayType, 0)
    P := 0
    return PlayMap{T: &T, K: &K, P: &P}
}

func PlayMapSize(p *PlayContext, b *PlayBlock) PlayType {
    if f := b.FieldVar(); f != nil {
        if v, e := p.Argv[f.Value]; e {
            l := v.Map(p)
            return PlayNumber(len(*l.T))
        }
        return PlayNull
    }
    return PlayFieldException("VAR")
}

func PlayMapGet(p *PlayContext, b *PlayBlock) PlayType {
    if f := b.FieldVar(); f != nil {
        if v, e := p.Argv[f.Value]; e {
            l := v.Map(p)
            i := p.Visit(b.BlockKey())
            return (*l.T)[i]
        }
        return PlayNull
    }
    return PlayFieldException("VAR")
}

func PlayMapExists(p *PlayContext, b *PlayBlock) PlayType {
    if f := b.FieldVar(); f != nil {
        if v, e := p.Argv[f.Value]; e {
            l := v.Map(p)
            T := *l.T
            o := p.Visit(b.BlockKey())
            if _, j := T[o]; j {
                return PlayBool(true)
            }
        }
        return PlayBool(false)
    }
    return PlayFieldException("VAR")
}

func PlayMapPut(p *PlayContext, b *PlayBlock) PlayType {
    if f := b.FieldVar(); f != nil {
        if v, e := p.Argv[f.Value]; e {
            l := v.Map(p)
            (*l.T)[p.Visit(b.BlockKey())] = p.Visit(b.BlockValue())
        }
        return PlayNull
    }
    return PlayFieldException("VAR")
}

func PlayMapDelete(p *PlayContext, b *PlayBlock) PlayType {
    if f := b.FieldVar(); f != nil {
        if v, e := p.Argv[f.Value]; e {
            l := v.Map(p)
            i := p.Visit(b.BlockKey())
            delete(*l.T, i)
        }
        return PlayNull
    }
    return PlayFieldException("VAR")
}
