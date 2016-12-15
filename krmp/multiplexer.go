package krmp

import "os"
import "io"
import "fmt"
import "log"
import "net/http"

type Multiplexer struct {
	routes     []Route
	middleware []Middleware
}

func (mux *Multiplexer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	logger := log.New(os.Stdout, "krmp", log.LstdFlags)
	results := make(chan Result)
	errors := make(chan error)

	runtime := RequestRuntime{logger, request, results, errors}
	url := request.URL

	handler := func(runtime *RequestRuntime) {
		runtime.Error(fmt.Errorf("NOT_FOUND"))
	}

	path := url.EscapedPath()
	for _, element := range mux.routes {
		if match := element.Path.MatchString(path) && element.Method == request.Method; match != true {
			continue
		}

		handler = element.Handler
		break
	}

	for _, ware := range mux.middleware {
		handler = ware(handler)
	}

	go handler(&runtime)

	select {
	case err := <-errors:
		logger.Printf("failed: %s", err.Error())
		writer.WriteHeader(422)
		writer.Write([]byte(err.Error()))
		return
	case result := <-results:
		header := writer.Header()
		header.Set("Content-Type", result.ContentType)
		writer.WriteHeader(200)
		io.Copy(writer, result)
	}

	close(errors)
	close(results)
}

func (mux *Multiplexer) Use(routes []Route, middleware []Middleware) {
	mux.middleware = middleware
	mux.routes = routes
}