package why

import (
	"io"
	"net/http"

	"github.com/d5/tengo/script"
)

// Extension defines a why plugin. This can be used
// to add functionality like database access from
// inside the scripts.
type Extension interface {
	// Name should return the name of the plugin.
	Name() string

	// Init will be called one time before starting the server.
	Init() error

	// Shutdown will be called after server shutdown.
	Shutdown() error

	// Vars will return a array of global variables that the extension will insert.
	// If you try to set a variable inside the Hook function that isn't specified here
	// a error will occur. The names are needed so compiled scripts can be cached.
	Vars() []string

	// Hook will be called on each http request.
	Hook(sc *script.Compiled, w io.Writer, resp http.ResponseWriter, r *http.Request) error
}
