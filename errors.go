package rpclens

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
)

// Error classification type from RFC 9457.
// This extends an HTTP status code with an optional URI for further specificity.
type ProblemType struct {
	Title    string     `json:"title"`          // Human readable error type, should be consistent within a given type (status and URI).
	Status   int        `json:"status"`         // HTTP status code
	URI      string     `json:"type,omitempty"` // More specific categories within a status code. Leave blank to refer to the entire status.
	LogLevel slog.Level `json:"-"`              // Not returned, used for logging.
}

// ProbemType for an HTTP status code with no further specificity.
func ProblemTypeStatus(status int) ProblemType {
	var logLevel slog.Level = slog.LevelInfo
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

// A structured error interface that can be returned from an HTTP API following RFC 9457.
// This is highly generalized and used for inplementing custom errors.
// It supports classification with a ProblemType (allowing specific cases in frontend code),
// separate messages for the log and response (this can be necessary to prevent secret leaks),
// additional data fields (for error handling code in clients) and HTTP headers.
// BasicProblem and its constructor Problemf provide a simple concrete implementation without extra fields.
type Problem interface {
	error                                       // Allows this to be returned as an error as well as providing a log message through the Error() method
	ProblemType() ProblemType                   // Type of error including the status code
	ProblemPermalink() string                   // Optional URI pointing to a persistent record of this occurrence of the problem. The RFC calls this "instance".
	ResponseMessage() string                    // Human-readable error message to be returned to the client. The RFC calls this "detail".
	SetResponseHeaders(headers_out http.Header) // Set any headers required for the response to the client.
	ResponseData() map[string]any               // additional fields to be sent to the response, MUST be json/v2 serializable
	LogData() map[string]any                    // additional data to be included in log
}

type problemJSONBody struct {
	ProblemType
	Permalink      string         `json:"instance,omitempty"`
	Message        string         `json:"detail"`
	AdditionalData map[string]any `json:",embed"`
}

// Wrapper for Problem that can be returned as an HTTPResponse.
// Follows RFC 9457 serialization and uses the application/problem+json MIME type.
// Note that this serialization uses "detail" for the error message field and "instance" for the permalink.
type ProblemResponse struct {
	Prob Problem
}

// Serializes the Problem as JSON according to RFC 9457.
func (p *ProblemResponse) RawBody() []byte {
	bodyobj := problemJSONBody{
		ProblemType:    p.Prob.ProblemType(),
		Message:        p.Prob.ResponseMessage(),
		AdditionalData: p.Prob.ResponseData(),
	}
	body, err := json.Marshal(&bodyobj, JSONOptions)
	if err != nil {
		panic(err) // can only happen due to a type error in AditionalData
	}
	return body
}

// Writes a Problem as an HTTP response.
// Returns the errror from the Write call itself, which can be ignored if you don't care about dropped sockets.
// Panics if ProblemData returns fields with non-serializable types.
func (p *ProblemResponse) WriteHTTPResponse(w http.ResponseWriter) error {
	body := p.RawBody()

	p.Prob.SetResponseHeaders(w.Header())
	w.Header().Set("Content-Type", "application/problem+json")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(p.Prob.ProblemType().Status)

	_, err := w.Write(body)
	return err
}

// Logs a Problem to a logger, including extra log fields and the problem type.
func LogProblem(ctx context.Context, log *slog.Logger, p Problem) {
	ptype := p.ProblemType()
	var attrs []slog.Attr = []slog.Attr{
		slog.String("problem_type", ptype.Title),
		slog.Int("status", ptype.Status),
	}
	for k, v := range p.LogData() {
		attrs = append(attrs, slog.Any(k, v))
	}
	log.LogAttrs(ctx, ptype.LogLevel, p.Error(), attrs...)
}

// A simple implementation of Problem without any extra fields.
type BasicProblem struct {
	PType        ProblemType
	Message      string
	LogMessage   string
	WrappedError error
}

func (p *BasicProblem) Error() string {
	return p.LogMessage
}

func (p *BasicProblem) Unwrap() error {
	return p.WrappedError
}

func (p *BasicProblem) ProblemType() ProblemType {
	return p.PType
}

func (p *BasicProblem) ProblemPermalink() string {
	return ""
}

func (p *BasicProblem) ResponseMessage() string {
	return p.Message
}

func (p *BasicProblem) SetResponseHeaders(h http.Header) {}

func (p *BasicProblem) ResponseData() map[string]any {
	return nil
}

func (p *BasicProblem) LogData() map[string]any {
	return nil
}

// A version of fmt.Errorf which returns a Problem providing the same error for both server log and client response.
// Supports error wrapping like Errorf does using the %w parameter.
func Problemf(ptype ProblemType, format string, args ...any) *BasicProblem {
	wrapper := fmt.Errorf(format, args...)
	wrapped := errors.Unwrap(wrapper)
	message := wrapper.Error()

	return &BasicProblem{
		PType:        ptype,
		Message:      message,
		LogMessage:   message,
		WrappedError: wrapped,
	}
}

// Like Problemf, but takes a status code instead of a ProblemType.
func StatusProblemf(status int, format string, args ...any) *BasicProblem {
	return Problemf(ProblemTypeStatus(status), format, args...)
}

// If err is already a Problem, return it.
// Otherwise, return a fallback error which includes the original error in the log message only (and not the client message).
// This is useful when expecting a Problem from an inner function (such as the action in a database transaction).
func ProblemOrFallback(err error, fallbackType ProblemType, fallbackDetail string) Problem {
	var p Problem
	if errors.As(err, &p) {
		return p
	} else {
		logMessage := fallbackDetail + ": " + err.Error()
		return &BasicProblem{
			PType:        fallbackType,
			Message:      fallbackDetail,
			LogMessage:   logMessage,
			WrappedError: err,
		}
	}
}
