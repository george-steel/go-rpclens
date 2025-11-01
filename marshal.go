package rpclens

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"strconv"
)

type HTTPResponse interface {
	WriteHTTPResponse(w http.ResponseWriter) error // the error should be ignorable if you do not care about dropped connections
}

type JSON[T any] struct {
	Status int
	Body   T
}

func AsJSON[T any](body T) *JSON[T] {
	return &JSON[T]{
		Status: 200,
		Body:   body,
	}
}

func (r *JSON[T]) WriteHTTPResponse(w http.ResponseWriter) error {
	rawbody, err := json.Marshal(r.Body, JSONOptions)
	if err != nil {
		panic(fmt.Errorf("error marshalling JSON response: %w", err))
	}

	h := w.Header()
	h.Set("Content-Type", "application/json")
	h.Set("Content-Length", strconv.Itoa(len(rawbody)))

	w.WriteHeader(r.Status)
	_, err = w.Write(rawbody)
	return err
}

type NoContent struct{}

func (r *NoContent) WriteHTTPResponse(w http.ResponseWriter) error {
	h := w.Header()
	h.Set("Content-Type", "application/json")
	h.Set("Content-Length", "0")

	w.WriteHeader(204)
	return nil
}
