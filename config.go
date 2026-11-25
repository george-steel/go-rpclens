package rpclens

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
)

// Options to use when processing and emitting JSON bodies
var JSONOptions json.Options = json.JoinOptions(
	jsontext.WithIndent("\t"),
	jsontext.Multiline(true),
	jsontext.CanonicalizeRawInts(false),
	json.FormatNilSliceAsNull(false), // no more gojson
	json.FormatNilMapAsNull(false),
)

// If set tu true, will make the Content-Type header optional in GetJSONBody
// Incorrect Content-Type will always be rejected
// (required to prevent XSRF which can send JSON as text/plain bypassing CORS),
// but the header can be made optional.
var AllowBlankContentType = false

// Overrideable function to turn a returned Problem into an HTTPResponse.
// The default value returns a ProblemResponseJSON which serializes as JSON using RFC 9457 conventions.
var MakeProblemResponse func(p Problem) HTTPResponse = func(p Problem) HTTPResponse {
	return &ProblemResponseJSON{Prob: p}
}
