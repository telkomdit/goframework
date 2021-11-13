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
    PlayExport(PlayExportSignature{"list_create": PlayListCreate,
        "list_size": PlayListSize, "list_get": PlayListGet, "list_exists": PlayListExists,
        "list_push": PlayListPush, "list_pop": PlayListPop, "list_shift": PlayListShift,
        "list_unshift": PlayListUnshift})
}

func PlayListCreate(p *PlayContext, b *PlayBlock) PlayType {
	T := make([]PlayType, 0)
    return PlayList{T: &T}
}

func PlayListSize(p *PlayContext, b *PlayBlock) PlayType {
    if f := b.FieldVar(); f != nil {
        if v, e := p.Argv[f.Value]; e {
            l := v.List(p)
            return PlayNumber(len(*l.T))
        }
        return PlayNull
    }
    return PlayFieldException("VAR")
}

func PlayListGet(p *PlayContext, b *PlayBlock) PlayType {
    if f := b.FieldVar(); f != nil {
        if v, e := p.Argv[f.Value]; e {
            l := v.List(p)
            i := p.Visit(b.BlockValue()).Int(p)
            return (*l.T)[i]
        }
        return PlayNull
    }
    return PlayFieldException("VAR")
}

func PlayListExists(p *PlayContext, b *PlayBlock) PlayType {
    if f := b.FieldVar(); f != nil {
        if v, e := p.Argv[f.Value]; e {
            l := v.List(p)
            T := *l.T
            o := p.Visit(b.BlockValue())
            for _, j := range T {
                if o == j {
                    return PlayBool(true)
                }
            }
        }
        return PlayBool(false)
    }
    return PlayFieldException("VAR")
}

func PlayListPush(p *PlayContext, b *PlayBlock) PlayType {
    if f := b.FieldVar(); f != nil {
        if v, e := p.Argv[f.Value]; e {
            l := v.List(p)
            *l.T = append(*l.T, p.Visit(b.BlockValue()))
        }
        return PlayNull
    }
    return PlayFieldException("VAR")
}

func PlayListPop(p *PlayContext, b *PlayBlock) (pt PlayType) {
    if f := b.FieldVar(); f != nil {
        if v, e := p.Argv[f.Value]; e {
            l := v.List(p)
            if ln := len(*l.T); ln > 0 {
                of := ln - 1
                pt = (*l.T)[of]
                *l.T = (*l.T)[:of]
                return
            }
        }
        return PlayNull
    }
    return PlayFieldException("VAR")
}

func PlayListShift(p *PlayContext, b *PlayBlock) PlayType {
    if f := b.FieldVar(); f != nil {
        if v, e := p.Argv[f.Value]; e {
            l := v.List(p)
            if len(*l.T) > 0 {
                *l.T = (*l.T)[1:]
            }
        }
        return PlayNull
    }
    return PlayFieldException("VAR")
}

func PlayListUnshift(p *PlayContext, b *PlayBlock) PlayType {
    if f := b.FieldVar(); f != nil {
        if v, e := p.Argv[f.Value]; e {
            l := v.List(p)
            T := make([]PlayType, 0)
            T = append(T, p.Visit(b.BlockValue()))
            *l.T = append(T, *l.T...)
        }
        return PlayNull
    }
    return PlayFieldException("VAR")
}

func PlayListDelete(p *PlayContext, b *PlayBlock) PlayType {
    if f := b.FieldVar(); f != nil {
        if v, e := p.Argv[f.Value]; e {
            l := v.List(p)
            i := int(p.Visit(b.BlockValue()).Int(p))
            s := len(*l.T)
            if i < s {
                (*l.T)[i] = (*l.T)[s-1]
                (*l.T)[s-1] = PlayNull
                *l.T = (*l.T)[:s-1]
            }
        }
        return PlayNull
    }
    return PlayFieldException("VAR")
}
