package rpclens

import (
	"encoding/json/v2"
	"net/http"
)

// Error type from RFC7807.
// This extends an HTTP status code with an optional URI for further specificity.
// Title is descriptive text but should be consistent for a given problem type.
type ProblemType struct {
	Title  string `json:"title"`
	Status int    `json:"status"`
	URI    string `json:"type,omitempty"`
}

// ProbemType for an HTTP status code with no further specificity.
func ProblemStatus(status int) ProblemType {
	return ProblemType{
		URI:    "",
		Title:  http.StatusText(status),
		Status: status,
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

type problemJSONResponse struct {
	ProblemType
	Instance       string         `json:"instance,omitempty`
	Detail         string         `json:"detail"`
	AdditionalData map[string]any `json:",inline"`
}

// Writes a Problem as an HTTP response.
// Returns the rrror from the Write call itself, which can be ignored if you don't care about dropped sockets.
// Panics if ProblemData returns fields with non-serializable types.
func WriteProblemResponse(w http.ResponseWriter, p Problem) error {
	bodyobj := problemJSONResponse{
		ProblemType:    p.ProblemType(),
		Detail:         p.ProblemDetail(),
		AdditionalData: p.ProblemData(),
	}
	body, err := json.Marshal(&bodyobj)
	if err != nil {
		panic(err) // can only happen due to a type error in AditionalData
	}

	p.SetProblemHeaders(w.Header())
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.ProblemType().Status)

	_, err = w.Write(body)
	return err
}
