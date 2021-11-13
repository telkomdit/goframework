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
    PlayExport(PlayExportSignature{"ctx_get": CtxGet, "ctx_set": CtxSet, "ctx_unset": CtxUnset,
        "ctx_code": CtxCode, "ctx_redirect": CtxRedirect, "ctx_header": CtxHeader,
        "ctx_content_type": CtxContentType})
}

func CtxGet(p *PlayContext, b *PlayBlock) PlayType {
    v := p.Visit(b.BlockValue()).String(p)
    if p.Cntx.Exists(v) {
        return PlayString(p.Cntx.Get(v))
    }
    return PlayNull
}

func CtxSet(p *PlayContext, b *PlayBlock) PlayType {
    p.Cntx.Set(p.Visit(b.BlockName()).String(p), p.Visit(b.BlockValue()).String(p))
    return PlayNull
}

func CtxUnset(p *PlayContext, b *PlayBlock) PlayType {
    v := p.Visit(b.BlockValue()).String(p)
    if p.Cntx.Exists(v) {
        p.Cntx.Unset(v)
    }
    return PlayNull
}

func CtxCode(p *PlayContext, b *PlayBlock) PlayType {
    code := int(p.Visit(b.BlockValue()).Int(p))
    p.Cntx.Code(code)
    return PlayNull
}

func CtxRedirect(p *PlayContext, b *PlayBlock) PlayType {
    p.Cntx.Redirect(p.Visit(b.BlockValue()).String(p))
    return PlayNull
}

func CtxHeader(p *PlayContext, b *PlayBlock) PlayType {
    p.Cntx.Header(p.Visit(b.BlockName()).String(p), p.Visit(b.BlockValue()).String(p))
    return PlayNull
}

func CtxContentType(p *PlayContext, b *PlayBlock) PlayType {
    v := ""
    n := p.Visit(b.BlockName()).String(p)
    switch n {
        case "TEXT":
            v = ContentTypeTEXT
        case "HTML":
            v = ContentTypeHTML
        case "JSON":
            v = ContentTypeJSON
        case "IMG":
            v = ContentTypeIMG
    }
    if v != "" { p.Cntx.ContentType(v) }
    return PlayNull
}