package rpclens

import (
	"fmt"
	"testing"
)

func TestProblemJSON(t *testing.T) {
	badRequest := ProblemStatus(400)
	innerErr := fmt.Errorf("sample parse error foo")
	sampleProblem := Problemf(badRequest, "https://example.com/foo", "invalid request: %w", innerErr)

	jsonProblem := ProblemJSON{Problem: sampleProblem}
	var _ HTTPResponse = &jsonProblem
	rawbody := jsonProblem.RawBody()
	t.Log(string(rawbody))
}
