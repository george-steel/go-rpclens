package rpclens

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
)

var JSONOptions json.Options = json.JoinOptions(
	jsontext.WithIndent("\t"),
	jsontext.Multiline(true),
	jsontext.CanonicalizeRawInts(false),
	json.FormatNilSliceAsNull(false), // no more gojson
	json.FormatNilMapAsNull(false),
)
