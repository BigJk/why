// jwt represents a JSON Web Token extension for the
// why server. You can store any arbitrary data object
// inside the token and later extract it again. The
// token will be signed with HMAC-SHA512. Keep in mind that
// a token is just signed and not encrypted. Only store data
// that any attacker can have without negative security
// implications.
//
// Generate a jwt token:
//   token := jwt.generate({ name: "username", age: 23 })
//
// Extract the data from a jwt token:
//   extracted := jwt.extract(token)
//   if(is_error(extracted)) {
//      // token was invalid
//   }
//   else {
//      name := extracted.name
//      age := extracted.age
//   }
package jwt

import (
	"encoding/base64"
	"io"
	"net/http"

	"github.com/BigJk/why"

	"github.com/pkg/errors"

	"github.com/d5/tengo/objects"
	"github.com/d5/tengo/script"
	"github.com/d5/tengo/stdlib/json"

	"github.com/dgrijalva/jwt-go"
)

// Extension represents the jwt extension
// for the why server.
type Extension struct {
	secret string
}

// New creates a new jwt extension instance with
// a given secret.
func New(secret string) (*Extension, error) {
	return &Extension{
		secret: secret,
	}, nil
}

// Name returns the name of the jwt extension.
func (e *Extension) Name() string {
	return "JSON Web Tokens"
}

// Init inits the jwt extension.
func (e *Extension) Init() error {
	return nil
}

// Shutdown closes the jwt extension.
func (e *Extension) Shutdown() error {
	return nil
}

// Vars returns all the variables jwt will globally create
func (e *Extension) Vars() []string {
	return []string{"jwt"}
}

// Hook will add a variable called 'jwt' to the
// script runtime that contains functions to generate
// jwt tokens and extract the token data.
func (e *Extension) Hook(sc *script.Compiled, w io.Writer, resp http.ResponseWriter, r *http.Request) error {
	return sc.Set("jwt", &objects.ImmutableMap{
		Value: map[string]objects.Object{
			"generate": &objects.UserFunction{
				Value: func(interop objects.Interop, args ...objects.Object) (ret objects.Object, err error) {
					if len(args) != 1 {
						return nil, objects.ErrWrongNumArguments
					}

					data, err := json.Encode(args[0])
					if err != nil {
						return nil, err
					}

					token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
						"data": data,
					})

					tokenString, err := token.SignedString([]byte(e.secret))
					if err != nil {
						return nil, err
					}

					return &objects.String{
						Value: tokenString,
					}, nil
				},
			},
			"extract": &objects.UserFunction{
				Value: func(interop objects.Interop, args ...objects.Object) (ret objects.Object, err error) {
					if len(args) != 1 {
						return nil, objects.ErrWrongNumArguments
					}

					tokenString, ok := objects.ToString(args[0])
					if !ok {
						return nil, errors.New("first argument was not a string")
					}

					token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
						if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
							return nil, errors.Errorf("Unexpected signing method: %v", token.Header["alg"])
						}

						return []byte(e.secret), nil
					})

					if err != nil {
						return &objects.Error{
							Value: &objects.String{
								Value: err.Error(),
							},
						}, nil
					}

					if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
						if data, ok := claims["data"]; ok {
							if base, ok := data.(string); ok {
								decoded, err := base64.StdEncoding.DecodeString(base)
								if err == nil {
									return json.Decode(decoded)
								}
							}
						}
					}

					return why.ToError(errors.New("invalid token")), nil
				},
			},
		},
	})
}
