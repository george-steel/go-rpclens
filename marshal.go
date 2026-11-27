package rpclens

import (
	"encoding/json/v2"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

// Types which can be returned over the wire as an HTTP response.
type HTTPResponse interface {
	WriteHTTPResponse(w http.ResponseWriter) error // the error should be ignorable if you do not care about dropped connections
}

// Response with a JSON body.
// Includes fields for status and extra headers.
type JSONBody[T any] struct {
	Status       int
	ExtraHeaders map[string]string
	Body         T
}

// Simple JSONBody constructor for a 200 OK response.
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

// Simple 204 No Content response with no extra headers.
// Pointers to this may be nil and still serialize.
type NoContent struct{}

// nil is a valid value here, as NoContent has no content
func (r *NoContent) WriteHTTPResponse(w http.ResponseWriter) error {
	h := w.Header()
	h.Set("Content-Type", "application/json")
	h.Set("Content-Length", "0")

	w.WriteHeader(204)
	return nil
}

// Empty response with a custom headers and status
type HeadersOnly struct {
	Status  int
	Headers map[string]string
}

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

// Attempts to read a JSON body from a http.Request.
// Configured with JSONOptions and AllowBlankContentType.
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
				Accepted: []string{"application/json"},
				Received: rawContentType,
				WantUTF8: true,
			}
		}
	}

	err := json.UnmarshalRead(r.Body, &out, JSONOptions)
	if err != nil {
		return out, Problemf(InvalidJSONProblemType, "Error decoding JSON body: %w", err)
	}
	return out, nil
}

var InvalidJSONProblemType = ProblemType{
	Status: 400,
	Title:  "Invalid JSON Body",
	URI:    "https://www.json.org/json-en.html",
}

// Error for unsupported media type in request body.
// Sets Accept header.
type UnsupportedMediaTypeError struct {
	Accepted []string
	Received string
	WantUTF8 bool
}

func (e *UnsupportedMediaTypeError) Error() string {
	return fmt.Sprintf("Unsupported Media Type: expecting %s, received %s", strings.Join(e.Accepted, ", "), e.Received)
}

func (e *UnsupportedMediaTypeError) ProblemType() ProblemType {
	return ProblemTypeStatus(http.StatusUnsupportedMediaType)
}

func (e *UnsupportedMediaTypeError) ResponseMessage() string {
	return fmt.Sprintf("Accepts %s, received %s", strings.Join(e.Accepted, ", "), e.Received)
}

func (e *UnsupportedMediaTypeError) ProblemPermalink() string {
	return ""
}

func (e *UnsupportedMediaTypeError) ResponseData() map[string]any {
	return map[string]any{"accepted_types": e.Accepted, "received_type": e.Received}
}

func (e *UnsupportedMediaTypeError) LogData() map[string]any {
	return nil
}

func (e *UnsupportedMediaTypeError) SetResponseHeaders(headers_out http.Header) {
	headers_out.Set("Accept", strings.Join(e.Accepted, ","))
	if e.WantUTF8 {
		headers_out.Set("Accept-Charset", "utf-8")
	}
}
