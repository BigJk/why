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

	"github.com/d5/tengo/objects"

	"go.uber.org/atomic"

	"github.com/pkg/errors"

	"github.com/d5/tengo/script"
	"github.com/d5/tengo/stdlib"
)

// Server represents a instance of the why server.
type Server struct {
	serv       *http.Server
	extensions []Extension
	running    *atomic.Bool
	conf       *Config
	bufferPool *sync.Pool
	stdModules *objects.ModuleMap
}

// New creates a new why server.
func New(conf *Config) *Server {
	return &Server{
		running: atomic.NewBool(false),
		conf:    conf,
		bufferPool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
		stdModules: stdlib.GetModuleMap(stdlib.AllModuleNames()...),
	}
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

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	// Trim '.' and '/' from the path to stop traversal of higher
	// folders.
	path := strings.TrimLeft(r.URL.Path, "./")

	// If the given path has no extension assume that a .tengo file
	// is meant.
	if len(filepath.Ext(path)) == 0 {
		path += ".tengo"
	}

	// Read the target file.
	file, err := os.OpenFile(filepath.Join(s.conf.PublicDir, path), os.O_RDONLY, 0666)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse POST form
	_ = r.ParseForm()

	// Create final buffer where the html will be written to before
	// writing to the response.
	buf := s.bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		s.bufferPool.Put(buf)
	}()

	// Create script and bind all the custom functions and variables.
	sc := script.New(transpiled.Bytes())
	sc.EnableFileImport(true)
	sc.SetImports(s.stdModules)

	_ = sc.Add("PUB_DIR", s.conf.PublicDir)

	err = addHTTP(sc, buf, w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Call all extension hooks.
	for i := range s.extensions {
		if err := s.extensions[i].Hook(sc, buf, w, r); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Run the script.
	if _, err := sc.Run(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write the response.
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}
