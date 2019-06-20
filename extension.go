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

	// Hook will be called on each http request.
	Hook(sc *script.Script, w io.Writer, resp http.ResponseWriter, r *http.Request) error
}
