package rpclens

import (
	"encoding/json/v2"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

type HTTPResponse interface {
	WriteHTTPResponse(w http.ResponseWriter) error // the error should be ignorable if you do not care about dropped connections
}

type JSONBody[T any] struct {
	Status       int
	ExtraHeaders map[string]string
	Body         T
}

func AsJSONBody[T any](body T) *JSONBody[T] {
	return &JSONBody[T]{
		Status:       200,
		ExtraHeaders: map[string]string{},
		Body:         body,
	}
}

func (r *JSONBody[T]) WriteHTTPResponse(w http.ResponseWriter) error {
	rawbody, err := json.Marshal(r.Body, JSONOptions)
	if err != nil {
		panic(fmt.Errorf("error marshalling JSON response: %w", err))
	}

	h := w.Header()
	for k, v := range r.ExtraHeaders {
		h.Set(k, v)
	}
	h.Set("Content-Type", "application/json")
	h.Set("Content-Length", strconv.Itoa(len(rawbody)))

	w.WriteHeader(r.Status)
	_, err = w.Write(rawbody)
	return err
}

type NoContent struct{}

// nil is a valid value here, as NoContent has no content
func (r *NoContent) WriteHTTPResponse(w http.ResponseWriter) error {
	h := w.Header()
	h.Set("Content-Type", "application/json")
	h.Set("Content-Length", "0")

	w.WriteHeader(204)
	return nil
}

type HeadersOnly struct {
	Status  int
	Headers map[string]string
}

// nil is a valid value here, as NoContent has no content
func (r *HeadersOnly) WriteHTTPResponse(w http.ResponseWriter) error {
	h := w.Header()
	for k, v := range r.Headers {
		h.Set(k, v)
	}
	h.Set("Content-Type", "application/json")
	h.Set("Content-Length", "0")

	w.WriteHeader(r.Status)
	return nil
}

func GetJSONBody[T any](r *http.Request) (T, Problem) {
	var out T

	rawContentType := r.Header.Get("Content-Type")
	if (rawContentType != "") || !AllowBlankContentType {
		baseContentType, ctParams, err := mime.ParseMediaType(rawContentType)
		if err != nil {
			return out, &UnsupportedMediaTypeError{
				Accepted: []string{"application/json"},
				Received: rawContentType,
				WantUTF8: true,
			}
		}
		if baseContentType != "application/json" {
			return out, &UnsupportedMediaTypeError{
				Accepted: []string{"application/json"},
				Received: baseContentType,
				WantUTF8: true,
			}
		}
		charset := ctParams["charset"]
		if charset != "" && strings.EqualFold(charset, "utf-8") {
			return out, &UnsupportedMediaTypeError{
				Accepted: []string{"application/json; charset=utf-8"},
				Received: rawContentType,
				WantUTF8: true,
			}
		}
	}

	err := json.UnmarshalRead(r.Body, &out, JSONOptions)
	if err != nil {
		return out, Problemf(ProblemStatus(400), "", "Error decoding JSON body: %w", err)
	}
	return out, nil
}

type UnsupportedMediaTypeError struct {
	Accepted []string
	Received string
	WantUTF8 bool
}

func (e *UnsupportedMediaTypeError) Error() string {
	return fmt.Sprintf("Unsupported Media Type: expecting %s, received %s", strings.Join(e.Accepted, ", "), e.Received)
}

func (e *UnsupportedMediaTypeError) ProblemType() ProblemType {
	return ProblemStatus(http.StatusUnsupportedMediaType)
}

func (e *UnsupportedMediaTypeError) ProblemDetail() string {
	return fmt.Sprintf("Accepts %s, received %s", strings.Join(e.Accepted, ", "), e.Received)
}

func (e *UnsupportedMediaTypeError) ProblemInstance() string {
	return ""
}

func (e *UnsupportedMediaTypeError) ProblemData() map[string]any {
	return map[string]any{"accepted_types": e.Accepted, "received_type": e.Received}
}

func (e *UnsupportedMediaTypeError) ErrorData() map[string]any {
	return nil
}

func (e *UnsupportedMediaTypeError) SetProblemHeaders(headers_out http.Header) {
	headers_out.Set("Accept", strings.Join(e.Accepted, ","))
	if e.WantUTF8 {
		headers_out.Set("Accept-Charset", "utf-8")
	}
}
