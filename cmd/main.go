package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"reflect"

	"github.com/BigJk/why/extensions/jwt"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	"github.com/BigJk/why"
	"github.com/BigJk/why/extensions/bbolt"
)

var config = struct {
	why.Config

	BindAddress string
	Extensions  map[string][]interface{}
}{}

var extensions = map[string]interface{}{
	"bbolt": bbolt.New,
	"jwt":   jwt.New,
}

func tryCreate(fn interface{}, args []interface{}) (why.Extension, error) {
	fnType := reflect.TypeOf(fn)
	fnValue := reflect.ValueOf(fn)

	if fnType.Kind() != reflect.Func {
		return nil, errors.New("fn wasn't a function")
	}

	if fnType.NumIn() != len(args) {
		return nil, errors.Errorf("arguments doesn't match. got=%d expected=%d", len(args), fnType.NumIn())
	}

	if fnType.Out(1).Kind() != reflect.Interface || !fnType.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return nil, errors.New("fn doesn't return a error as second value")
	}

	var callValues []reflect.Value
	for i := range args {
		argType := reflect.TypeOf(args[i])

		if argType == nil {
			switch fnType.In(i).Kind() {
			case reflect.Ptr:
				fallthrough
			case reflect.Uintptr:
				fallthrough
			case reflect.Map:
				fallthrough
			case reflect.Slice:
				callValues = append(callValues, reflect.New(fnType.In(i)).Elem())
				continue
			}

			return nil, errors.Errorf("argument %d can't be null", i+1)
		}

		if fnType.In(i).Kind() == reflect.Struct && argType.Kind() == reflect.Map {
			s := reflect.New(fnType.In(i))
			if err := mapstructure.Decode(args[i], s.Interface()); err != nil {
				return nil, err
			}

			callValues = append(callValues, s.Elem())
			continue
		}

		if fnType.In(i).Kind() != argType.Kind() {
			if argType.Kind() == reflect.Float64 {
				switch fnType.In(i).Kind() {
				case reflect.Int:
					fallthrough
				case reflect.Int8:
					fallthrough
				case reflect.Int16:
					fallthrough
				case reflect.Int32:
					fallthrough
				case reflect.Int64:
					fallthrough
				case reflect.Uint8:
					fallthrough
				case reflect.Uint16:
					fallthrough
				case reflect.Uint32:
					fallthrough
				case reflect.Uint64:
					fallthrough
				case reflect.Float32:
					callValues = append(callValues, reflect.ValueOf(args[i]).Convert(fnType.In(i)))
					continue
				}
			}

			return nil, errors.Errorf("mismatching argument type of %d. argument. got=%s expected=%s", i+1, argType.Kind().String(), fnType.In(i).Kind().String())
		}

		callValues = append(callValues, reflect.ValueOf(args[i]))
	}

	res := fnValue.Call(callValues)
	if res[1].Interface() != nil {
		err := res[1].Interface().(error)
		if err != nil {
			return nil, err
		}
	}

	return res[0].Interface().(why.Extension), nil
}

func main() {
	// --config flag to define the config file.
	configFile := flag.String("config", "./config.json", "config for the instance")
	flag.Parse()

	// Read and unmarshal the config file.
	data, err := ioutil.ReadFile(*configFile)
	if err != nil {
		panic(err)
	}

	if err := json.Unmarshal(data, &config); err != nil {
		panic(err)
	}

	// Create the why server instance.
	server := why.New(&config.Config)

	// Dynamically load the extensions.
	for name, conf := range config.Extensions {
		if ctor, ok := extensions[name]; ok {
			ext, err := tryCreate(ctor, conf)
			if err != nil {
				log.Fatalf("Error while building extension: %v\n", err.Error())
			} else {
				if err := server.AddExtension(ext); err != nil {
					log.Fatalf("Error while adding extension: %v\n", err.Error())
				}
			}
		} else {
			log.Fatalf("Extension '%s' not found\n", name)
		}
	}

	// Start the why server.
	go server.Start(config.BindAddress)

	// Wait for interrupt.
	quit := make(chan os.Signal, 10)
	signal.Notify(quit, os.Interrupt)
	<-quit

	// Shut down server.
	if err := server.Shutdown(); err != nil {
		log.Printf("Error while shutting down server: %v\n", err)
	}
}
