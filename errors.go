package rpclens

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
)

// Error type from RFC7807.
// This extends an HTTP status code with an optional URI for further specificity.
// Title is descriptive text but should be consistent for a given problem type.
type ProblemType struct {
	Title    string     `json:"title"`
	Status   int        `json:"status"`
	URI      string     `json:"type,omitempty"`
	LogLevel slog.Level `json:"-"`
}

// ProbemType for an HTTP status code with no further specificity.
func ProblemStatus(status int) ProblemType {
	var logLevel slog.Level = slog.LevelDebug
	if status == 502 {
		logLevel = slog.LevelWarn
	} else if status == 403 {
		logLevel = slog.LevelWarn
	} else if status >= 500 {
		logLevel = slog.LevelError
	}

	return ProblemType{
		URI:      "",
		Title:    http.StatusText(status),
		Status:   status,
		LogLevel: logLevel,
	}
}

// Am error message that can be returned from an HTTPP API.
// Follows RFC7807 structure.
type Problem interface {
	error                                      // allows this to be returned as an error as well as providing a log message through the Error() method
	ProblemType() ProblemType                  // type of error including the status code
	ProblemInstance() string                   // Optional URI to identify a specific instance of this problem, may be blank
	ProblemDetail() string                     // Human-readable error message to be returned to the client
	SetProblemHeaders(headers_out http.Header) // set any headers required for the response
	ProblemData() map[string]any               // additional fields to be sent to the response, MUST be json/v2 serializable
	ErrorData() map[string]any                 // additional data to be included in log
}

type problemJSONBody struct {
	ProblemType
	Instance       string         `json:"instance,omitempty"`
	Detail         string         `json:"detail"`
	AdditionalData map[string]any `json:",inline"`
}

type ProblemJSON struct {
	Problem
}

func (p *ProblemJSON) RawBody() []byte {
	bodyobj := problemJSONBody{
		ProblemType:    p.ProblemType(),
		Detail:         p.ProblemDetail(),
		AdditionalData: p.ProblemData(),
	}
	body, err := json.Marshal(&bodyobj, JSONOptions)
	if err != nil {
		panic(err) // can only happen due to a type error in AditionalData
	}
	return body
}

// Writes a Problem as an HTTP response.
// Returns the rrror from the Write call itself, which can be ignored if you don't care about dropped sockets.
// Panics if ProblemData returns fields with non-serializable types.
func (p *ProblemJSON) WriteHTTPResponse(w http.ResponseWriter) error {
	body := p.RawBody()

	p.SetProblemHeaders(w.Header())
	w.Header().Set("Content-Type", "application/problem+json")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(p.ProblemType().Status)

	_, err := w.Write(body)
	return err
}

type BasicProblem struct {
	PType           ProblemType
	Instance        string
	Detail          string
	InternalMessage string
	WrappedError    error
}

func (p *BasicProblem) Error() string {
	return p.InternalMessage
}

func (p *BasicProblem) Unwrap() error {
	return p.WrappedError
}

func (p *BasicProblem) ProblemType() ProblemType {
	return p.PType
}

func (p *BasicProblem) ProblemInstance() string {
	return p.Instance
}

func (p *BasicProblem) ProblemDetail() string {
	return p.Detail
}

func (p *BasicProblem) SetProblemHeaders(h http.Header) {}

func (p *BasicProblem) ProblemData() map[string]any {
	return nil
}

func (p *BasicProblem) ErrorData() map[string]any {
	return nil
}

func Problemf(ptype ProblemType, instance string, format string, args ...any) Problem {
	wrapper := fmt.Errorf(format, args...)
	wrapped := errors.Unwrap(wrapper)
	message := wrapper.Error()

	return &BasicProblem{
		PType:           ptype,
		Instance:        instance,
		Detail:          message,
		InternalMessage: message,
		WrappedError:    wrapped,
	}
}

func ProblemOrFallback(err error, fallbackType ProblemType, fallbackInstance string, fallbackDetail string) Problem {
	var p Problem
	if errors.As(err, &p) {
		return p
	} else {
		logMessage := fallbackDetail + ": " + err.Error()
		return &BasicProblem{
			PType:           fallbackType,
			Instance:        fallbackInstance,
			Detail:          fallbackDetail,
			InternalMessage: logMessage,
			WrappedError:    err,
		}
	}
}
