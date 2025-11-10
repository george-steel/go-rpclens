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

// Incorrect Content-Type will always be rejected
// (required to prevent XSRF which can send JSON as text/plain bypassing CORS),
// but the header can be made optional.
var AllowBlankContentType = false
