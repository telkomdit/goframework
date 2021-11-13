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

package to

import (
    "encoding/base64"
    "errors"
    "net/url"
)

func Base64(v string) string {
    return base64.StdEncoding.EncodeToString([]byte(v))
}

func HttpQuery(v interface{}, prefix... string) (string, error) {
    p := ""
    if len(prefix) > 0 { p = prefix[0] }
    q := ""
    switch u := v.(type) {
    case []interface{}:
        for i, v := range u {
            c, e := HttpQuery(v, p + "[]")
            if e != nil {
                return "", e
            }
            q+= c
            if i < len(u)-1 { q+= "&" }
        }
    case map[string]interface{}:
        length := len(u)
        for k, v := range u {
            m := ""
            if p != "" {
                m = p + "[" + url.QueryEscape(k) + "]"
            } else {
                m = url.QueryEscape(k)
            }
            c, e := HttpQuery(v, m)
            if e != nil {
                return "", e
            }
            q+= c
            length -= 1
            if length > 0 {
                q+= "&"
            }
        }
    case string:
        if p == "" {
            return "", errors.New("value must be a map[string]interface{}")
        }
        q+= p + "=" + url.QueryEscape(u)
    default:
        q+= p
    }
    return q, nil
}

func URLDecode(v string) string {
    u, _ := url.QueryUnescape(v)
    return u
}

func URLEncode(v string) string {
    return url.QueryEscape(v)
}
