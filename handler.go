package rpclens

import (
	"log/slog"
	"net/http"
	"reflect"
	"runtime"
)

// Returns the name of a function from its pointer.
func GetFuncName(f any) string {
	fv := reflect.ValueOf(f)
	if fv.Kind() != reflect.Func {
		return ""
	}
	rf := runtime.FuncForPC(fv.Pointer())
	if rf == nil {
		return ""
	}
	return rf.Name()
}

type jsonBodyHandler[S any, T HTTPResponse] struct {
	Name         string
	EndpointFunc func(r *http.Request, body S, log *slog.Logger) (T, Problem)
}

func (h *jsonBodyHandler[S, T]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: add X-Request-Id handling to logger
	log := slog.Default().With("endpoint", h.Name)

	body, prob := GetJSONBody[S](r)
	if prob != nil {
		LogProblem(r.Context(), log, prob)
		MakeProblemResponse(prob).WriteHTTPResponse(w)
		return
	}

	resp, prob := h.EndpointFunc(r, body, log)
	if prob != nil {
		LogProblem(r.Context(), log, prob)
		MakeProblemResponse(prob).WriteHTTPResponse(w)
		return
	}

	log.DebugContext(r.Context(), "Finished Calling "+h.Name)
	resp.WriteHTTPResponse(w)
}

func HandleJSONRequest[S any, T HTTPResponse](endpoint func(r *http.Request, body S, log *slog.Logger) (T, Problem)) http.Handler {
	name := GetFuncName(endpoint)
	return &jsonBodyHandler[S, T]{
		Name:         name,
		EndpointFunc: endpoint,
	}
}

type emptyBodyHandler[T HTTPResponse] struct {
	Name         string
	EndpointFunc func(r *http.Request, log *slog.Logger) (T, Problem)
}

func (h *emptyBodyHandler[T]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := slog.Default().With("endpoint", h.Name)

	resp, prob := h.EndpointFunc(r, log)
	if prob != nil {
		LogProblem(r.Context(), log, prob)
		MakeProblemResponse(prob).WriteHTTPResponse(w)
		return
	}

	log.DebugContext(r.Context(), "Finished Calling "+h.Name)
	resp.WriteHTTPResponse(w)
}

func HandleEmptyRequest[T HTTPResponse](endpoint func(r *http.Request, log *slog.Logger) (T, Problem)) http.Handler {
	name := GetFuncName(endpoint)
	return &emptyBodyHandler[T]{
		Name:         name,
		EndpointFunc: endpoint,
	}
}
