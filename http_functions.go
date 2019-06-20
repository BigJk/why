package why

import (
	"errors"
	"html"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/d5/tengo/objects"
	"github.com/d5/tengo/script"
)

func valuesToObject(v url.Values) (*objects.Array, error) {
	var keys []objects.Object
	for key := range v {
		keyObj, err := objects.FromInterface(key)
		if err != nil {
			return nil, err
		}
		keys = append(keys, keyObj)
	}
	return &objects.Array{Value: keys}, nil
}

func writeHTML(w io.Writer) objects.CallableFunc {
	return func(interop objects.Interop, args ...objects.Object) (ret objects.Object, err error) {
		if len(args) == 0 {
			return nil, objects.ErrWrongNumArguments
		}

		for i := range args {
			if html, ok := objects.ToString(args[i]); ok {
				if _, err := w.Write([]byte(html)); err != nil {
					return nil, err
				}
				continue
			}
			if _, err := w.Write([]byte(args[i].String())); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
}

func escapeHTML(interop objects.Interop, args ...objects.Object) (ret objects.Object, err error) {
	if len(args) == 0 {
		return nil, objects.ErrWrongNumArguments
	}

	var res string
	for i := range args {
		if e, ok := objects.ToString(args[i]); ok {
			res += html.EscapeString(e)
			continue
		}
		res += html.EscapeString(args[i].String())
	}

	return &objects.String{
		Value: res,
	}, nil
}

func getPostParam(r *http.Request) objects.CallableFunc {
	return func(interop objects.Interop, args ...objects.Object) (ret objects.Object, err error) {
		if len(args) != 1 {
			return nil, objects.ErrWrongNumArguments
		}

		key, ok := objects.ToString(args[0])
		if !ok {
			return nil, errors.New("not a string")
		}

		return &objects.String{Value: r.FormValue(key)}, nil
	}
}

func getPostKeys(r *http.Request) objects.CallableFunc {
	return func(interop objects.Interop, args ...objects.Object) (ret objects.Object, err error) {
		if len(args) != 0 {
			return nil, objects.ErrWrongNumArguments
		}
		return valuesToObject(r.PostForm)
	}
}

func getGetParam(r *http.Request) objects.CallableFunc {
	return func(interop objects.Interop, args ...objects.Object) (ret objects.Object, err error) {
		if len(args) != 1 {
			return nil, objects.ErrWrongNumArguments
		}

		key, ok := objects.ToString(args[0])
		if !ok {
			return nil, errors.New("not a string")
		}

		return &objects.String{Value: r.URL.Query().Get(key)}, nil
	}
}

func getGetKeys(r *http.Request) objects.CallableFunc {
	return func(interop objects.Interop, args ...objects.Object) (ret objects.Object, err error) {
		if len(args) != 0 {
			return nil, objects.ErrWrongNumArguments
		}
		return valuesToObject(r.URL.Query())
	}
}

func getHeader(r *http.Request) objects.CallableFunc {
	return func(interop objects.Interop, args ...objects.Object) (ret objects.Object, err error) {
		if len(args) != 1 {
			return nil, objects.ErrWrongNumArguments
		}

		key, ok := objects.ToString(args[0])
		if !ok {
			return nil, errors.New("not a string")
		}

		return &objects.String{Value: r.Header.Get(key)}, nil
	}
}

func getHeaderKeys(r *http.Request) objects.CallableFunc {
	return func(interop objects.Interop, args ...objects.Object) (ret objects.Object, err error) {
		if len(args) != 0 {
			return nil, objects.ErrWrongNumArguments
		}

		var keys []objects.Object
		for key := range r.Header {
			keyObj, err := objects.FromInterface(key)
			if err != nil {
				return nil, err
			}
			keys = append(keys, keyObj)
		}

		return &objects.Array{Value: keys}, nil
	}
}

func setHeader(w http.ResponseWriter) objects.CallableFunc {
	return func(interop objects.Interop, args ...objects.Object) (ret objects.Object, err error) {
		if len(args) != 2 {
			return nil, objects.ErrWrongNumArguments
		}

		key, ok := objects.ToString(args[0])
		if !ok {
			return nil, errors.New("not a string")
		}

		if value, ok := objects.ToString(args[1]); ok {
			w.Header().Set(key, value)
		} else {
			w.Header().Set(key, args[1].String())
		}

		return nil, nil
	}
}

func getBody(r *http.Request) objects.CallableFunc {
	return func(interop objects.Interop, args ...objects.Object) (ret objects.Object, err error) {
		if len(args) != 0 {
			return nil, objects.ErrWrongNumArguments
		}
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		return &objects.Bytes{Value: data}, nil
	}
}

func getCookies(r *http.Request) objects.CallableFunc {
	return func(interop objects.Interop, args ...objects.Object) (ret objects.Object, err error) {
		if len(args) != 0 {
			return nil, objects.ErrWrongNumArguments
		}

		var arr objects.Array
		for i := range r.Cookies() {
			arr.Value = append(arr.Value, &objects.ImmutableMap{
				Value: map[string]objects.Object{
					"value":   &objects.String{Value: r.Cookies()[i].Value},
					"name":    &objects.String{Value: r.Cookies()[i].Name},
					"domain":  &objects.String{Value: r.Cookies()[i].Domain},
					"max_age": &objects.Int{Value: int64(r.Cookies()[i].MaxAge)},
					"expires": &objects.Time{Value: r.Cookies()[i].Expires},
				},
			})
		}

		return &arr, nil
	}
}

func getCookie(r *http.Request) objects.CallableFunc {
	return func(interop objects.Interop, args ...objects.Object) (ret objects.Object, err error) {
		if len(args) != 1 {
			return nil, objects.ErrWrongNumArguments
		}

		key, ok := objects.ToString(args[0])
		if !ok {
			return nil, errors.New("not a string")
		}

		cookie, err := r.Cookie(key)
		if err != nil {
			return nil, err
		}

		return &objects.ImmutableMap{
			Value: map[string]objects.Object{
				"value":   &objects.String{Value: cookie.Value},
				"name":    &objects.String{Value: cookie.Name},
				"path":    &objects.String{Value: cookie.Path},
				"domain":  &objects.String{Value: cookie.Domain},
				"max_age": &objects.Int{Value: int64(cookie.MaxAge)},
				"expires": &objects.Time{Value: cookie.Expires},
			},
		}, nil
	}
}

func setCookie(resp http.ResponseWriter) objects.CallableFunc {
	return func(interop objects.Interop, args ...objects.Object) (ret objects.Object, err error) {
		if len(args) != 1 {
			return nil, objects.ErrWrongNumArguments
		}

		m := objects.ToInterface(args[0])
		cookieMap, ok := m.(map[string]interface{})
		if !ok {
			return nil, errors.New("not a cookie")
		}

		cookie := new(http.Cookie)
		if name, ok := cookieMap["name"].(string); ok {
			cookie.Name = name
		}

		if value, ok := cookieMap["value"].(string); ok {
			cookie.Value = value
		}

		if path, ok := cookieMap["path"].(string); ok {
			cookie.Path = path
		}

		if maxAge, ok := cookieMap["max_age"].(int64); ok {
			cookie.MaxAge = int(maxAge)
		}

		if expires, ok := cookieMap["expires"].(time.Time); ok {
			cookie.Expires = expires
		}

		http.SetCookie(resp, cookie)
		return nil, nil
	}
}

func addHTTP(sc *script.Script, w io.Writer, resp http.ResponseWriter, r *http.Request) error {
	return sc.Add("http", &objects.ImmutableMap{
		Value: map[string]objects.Object{
			"method": &objects.String{
				Value: r.Method,
			},
			"full_uri": &objects.String{
				Value: r.RequestURI,
			},
			"path": &objects.String{
				Value: r.URL.Path,
			},
			"scheme": &objects.String{
				Value: r.URL.Scheme,
			},
			"host": &objects.String{
				Value: r.URL.Host,
			},
			"ip": &objects.String{
				Value: r.RemoteAddr,
			},
			"proto": &objects.String{
				Value: r.Proto,
			},
			"write": &objects.UserFunction{
				Value: writeHTML(w),
			},
			"escape": &objects.UserFunction{
				Value: escapeHTML,
			},
			"body": &objects.UserFunction{
				Value: getBody(r),
			},
			"GET": &objects.ImmutableMap{
				Value: map[string]objects.Object{
					"keys": &objects.UserFunction{
						Value: getGetKeys(r),
					},
					"param": &objects.UserFunction{
						Value: getGetParam(r),
					},
				},
			},
			"POST": &objects.ImmutableMap{
				Value: map[string]objects.Object{
					"keys": &objects.UserFunction{
						Value: getPostKeys(r),
					},
					"param": &objects.UserFunction{
						Value: getPostParam(r),
					},
				},
			},
			"HEADER": &objects.ImmutableMap{
				Value: map[string]objects.Object{
					"keys": &objects.UserFunction{
						Value: getHeaderKeys(r),
					},
					"param": &objects.UserFunction{
						Value: getHeader(r),
					},
					"set": &objects.UserFunction{
						Value: setHeader(resp),
					},
				},
			},
			"COOKIES": &objects.ImmutableMap{
				Value: map[string]objects.Object{
					"all": &objects.UserFunction{
						Value: getCookies(r),
					},
					"param": &objects.UserFunction{
						Value: getCookie(r),
					},
					"set": &objects.UserFunction{
						Value: setCookie(resp),
					},
				},
			},
		},
	})
}
