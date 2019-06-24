// bbolt represents a bbolt extension for the why server.
// bbolt is a key-value storage that can be used to persistently
// store and access data in why.
//
// Set data:
//   bbolt.set("users", "test@test.com", { name: "John Do", age: 23 })
//
// Get data:
//   user := bbolt.get("users", "test@test.com")
//
//   name := user.name
//   age  := user.age
//
// Iterate over data:
//   bbolt.iterate("users", func(key, value) {
//      email := key
//      name  := value.name
//      age   := value.age
//
//      return true // return false if you want to stop the iteration
//   })
package bbolt

import (
	"io"
	"net/http"

	"github.com/BigJk/why"

	"github.com/d5/tengo/objects"
	"github.com/d5/tengo/script"
	"github.com/d5/tengo/stdlib/json"
	"github.com/pkg/errors"

	"go.etcd.io/bbolt"
)

// Extension represents the bbolt extension
// for the why server.
type Extension struct {
	conn *bbolt.DB
}

// New creates a new bbolt extension instance.
func New(file string, options *bbolt.Options) (*Extension, error) {
	db, err := bbolt.Open(file, 0666, options)
	if err != nil {
		return nil, err
	}
	return &Extension{
		conn: db,
	}, nil
}

// Bolt returns the connection to bbolt. This can
// be useful if you setup your own server and add the
// extension manually. This way you can access bolt
// directly and insert data from within your backend.
func (e *Extension) Bolt() *bbolt.DB {
	return e.conn
}

// Name returns the name of the bbolt extension.
func (e *Extension) Name() string {
	return "BBolt Storage"
}

// Init inits the bbolt extension.
func (e *Extension) Init() error {
	return nil
}

// Shutdown closes the bbolt extension.
func (e *Extension) Shutdown() error {
	return e.conn.Close()
}

// Vars returns all the variables bbolt will globally create
func (e *Extension) Vars() []string {
	return []string{"bbolt"}
}

// Hook will add a variable called 'bbolt' to the
// script runtime that contains functions to access
// bbolt.
func (e *Extension) Hook(sc *script.Compiled, w io.Writer, resp http.ResponseWriter, r *http.Request) error {
	return sc.Set("bbolt", &objects.ImmutableMap{
		Value: map[string]objects.Object{
			"set": &objects.UserFunction{
				Value: func(interop objects.Interop, args ...objects.Object) (ret objects.Object, err error) {
					if len(args) != 3 {
						return nil, objects.ErrWrongNumArguments
					}

					bucket, ok := objects.ToString(args[0])
					if !ok {
						return nil, errors.New("first argument is not a string")
					}

					key, ok := objects.ToString(args[1])
					if !ok {
						return nil, errors.New("second argument is not a string")
					}

					data, err := json.Encode(args[2])
					if err != nil {
						return nil, err
					}

					return nil, e.conn.Update(func(tx *bbolt.Tx) error {
						b, err := tx.CreateBucketIfNotExists([]byte(bucket))
						if err != nil {
							return err
						}
						return b.Put([]byte(key), data)
					})
				},
			},
			"get": &objects.UserFunction{
				Value: func(interop objects.Interop, args ...objects.Object) (ret objects.Object, err error) {
					if len(args) != 2 {
						return nil, objects.ErrWrongNumArguments
					}

					bucket, ok := objects.ToString(args[0])
					if !ok {
						return nil, errors.New("first argument is not a string")
					}

					key, ok := objects.ToString(args[1])
					if !ok {
						return nil, errors.New("second argument is not a string")
					}

					var data []byte
					if err := e.conn.View(func(tx *bbolt.Tx) error {
						b := tx.Bucket([]byte(bucket))
						if b == nil {
							return errors.New("bucket not found")
						}

						data = b.Get([]byte(key))
						return nil
					}); err != nil {
						return why.ToError(err), nil
					}

					if len(data) == 0 {
						return why.ToError(errors.New("not found")), nil
					}
					return json.Decode(data)
				},
			},
			"iterate": &objects.UserFunction{
				Value: func(interop objects.Interop, args ...objects.Object) (ret objects.Object, err error) {
					if len(args) != 2 {
						return nil, objects.ErrWrongNumArguments
					}

					bucket, ok := objects.ToString(args[0])
					if !ok {
						return nil, errors.New("first argument is not a string")
					}

					return why.ToError(e.conn.View(func(tx *bbolt.Tx) error {
						b := tx.Bucket([]byte(bucket))
						if b == nil {
							return errors.New("bucket not found")
						}

						c := b.Cursor()

						for k, v := c.First(); k != nil; k, v = c.Next() {
							key := &objects.String{
								Value: string(k),
							}

							obj, err := json.Decode(v)
							if err != nil {
								return err
							}

							ret, err := interop.InteropCall(args[1], key, obj)
							if err != nil {
								return err
							}

							keepRunning, ok := objects.ToBool(ret)
							if ok && !keepRunning {
								break
							}
						}

						return nil
					})), nil
				},
			},
		},
	})
}
