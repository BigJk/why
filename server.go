package why

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/d5/tengo/script"

	"github.com/d5/tengo/objects"

	"go.uber.org/atomic"

	"github.com/pkg/errors"

	"github.com/d5/tengo/stdlib"
)

var globalVariables = []string{"http", "PUB_DIR"}
var requestedAbort = errors.New("requested abort")

// Server represents a instance of the why server.
type Server struct {
	conf       *Config
	running    *atomic.Bool
	serv       *http.Server
	extensions []Extension
	stdModules *objects.ModuleMap
	bufferPool *sync.Pool
	cache      *scriptCache
}

// New creates a new why server.
func New(conf *Config) *Server {
	s := &Server{
		running: atomic.NewBool(false),
		conf:    conf,
		bufferPool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
		stdModules: stdlib.GetModuleMap(stdlib.AllModuleNames()...),
	}

	// Create a script cache that will cache compiled scripts.
	s.cache = newCache(func(sc *script.Script) {
		sc.EnableFileImport(true)
		sc.SetImports(s.stdModules)

		for i := range globalVariables {
			_ = sc.Add(globalVariables[i], "")
		}

		for i := range s.extensions {
			for j := range s.extensions[i].Vars() {
				_ = sc.Add(s.extensions[i].Vars()[j], "")
			}
		}
	})

	return s
}

// AddExtension adds a new extension to the server.
// This function can only be called before when the server
// is not running.
func (s *Server) AddExtension(e Extension) error {
	if s.running.Load() {
		return errors.New("can't add extension while running")
	}
	s.extensions = append(s.extensions, e)
	return nil
}

// Start starts the server and binds it to the
// given address.
func (s *Server) Start(address string) error {
	s.running.Store(true)
	defer func() {
		s.running.Store(false)
	}()

	for i := range s.extensions {
		if err := s.extensions[i].Init(); err != nil {
			return errors.Wrapf(err, "error while init of '%s'", s.extensions[i].Name())
		}
	}

	s.serv = &http.Server{Addr: address}

	http.HandleFunc("/", s.handle)

	return s.serv.ListenAndServe()
}

// Shutdown will try to shut the server down.
func (s *Server) Shutdown() error {
	defer func() {
		for i := range s.extensions {
			_ = s.extensions[i].Shutdown()
		}
	}()

	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	return s.serv.Shutdown(ctx)
}

func (s *Server) error(w http.ResponseWriter, err error, code int) {
	if !s.conf.EnableError {
		http.Error(w, "error", code)
	} else {
		http.Error(w, err.Error(), code)
	}
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	// If the path contains '..' a attacker could traverse upper directories
	// and access files that could contain sensitive information. If a '..'
	// appears in the path we will return a error.
	path := r.URL.Path
	if strings.Contains(path, "..") {
		s.error(w, errors.New("upper traversal of directory forbidden"), http.StatusInternalServerError)
		return
	}

	// If the given path has no extension assume that a .tengo file
	// is meant.
	if len(filepath.Ext(path)) == 0 {
		path += ".tengo"
	}

	// Read the target file.
	file, err := os.OpenFile(filepath.Join(s.conf.PublicDir, path), os.O_RDONLY, 0666)
	if err != nil {
		s.error(w, err, http.StatusNotFound)
		return
	}

	// If it it's not a .tengo script we just return the content of the file.
	if !strings.HasSuffix(path, ".tengo") {
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, file)
		return
	}

	// transpile html containing tengo scripts to a complete tengo script.
	transpiled := s.bufferPool.Get().(*bytes.Buffer)
	defer func() {
		transpiled.Reset()
		s.bufferPool.Put(transpiled)
	}()

	if err := Transpile(file, transpiled); err != nil {
		s.error(w, err, http.StatusInternalServerError)
		return
	}

	// Parse POST form.
	_ = r.ParseForm()

	// Create final buffer where the html will be written to before
	// writing to the response.
	buf := s.bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		s.bufferPool.Put(buf)
	}()

	// Compile the script or get a instance from cache.
	back, sc, err := s.cache.get(transpiled.Bytes())
	if err != nil {
		s.error(w, err, http.StatusInternalServerError)
		return
	}

	defer func() {
		s.cache.put(back, sc)
	}()

	// The final status code.
	statusCode := http.StatusOK

	// Contains data about the request. Besides writing to the buffer
	// and the responseWriter, the script may set the status code.
	si := &scriptInstance{
		script:     sc,
		buf:        buf,
		req:        r,
		statusCode: &statusCode,
		respWriter: w,
	}

	// Replace all the variables with the correct ones for this request.
	_ = sc.Set("PUB_DIR", s.conf.PublicDir)
	err = addHTTP(si)
	if err != nil {
		s.error(w, err, http.StatusNotFound)
		return
	}

	// Call all extension hooks.
	for i := range s.extensions {
		if err := s.extensions[i].Hook(sc, buf, w, r); err != nil {
			s.error(w, err, http.StatusInternalServerError)
			return
		}
	}

	// Run the script.
	if err := sc.Run(); err != nil && !strings.Contains(err.Error(), requestedAbort.Error()) {
		s.error(w, err, http.StatusInternalServerError)
		return
	}

	// Write the response.
	w.WriteHeader(statusCode)
	_, _ = w.Write(buf.Bytes())
}
