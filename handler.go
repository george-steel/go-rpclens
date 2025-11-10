package rpclens

import (
	"log/slog"
	"net/http"
	"reflect"
	"runtime"
)

func getFuncName(f any) string {
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

type JSONBodyHandler[S any, T HTTPResponse] struct {
	Name         string
	EndpointFunc func(r *http.Request, body S, log *slog.Logger) (T, Problem)
}

func (h *JSONBodyHandler[S, T]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := slog.Default().With("endpoint", h.Name)

	body, prob := GetJSONBody[S](r)
	if prob != nil {
		LogProblem(r.Context(), log, prob)
		pj := &ProblemJSON{Problem: prob}
		pj.WriteHTTPResponse(w)
		return
	}

	resp, prob := h.EndpointFunc(r, body, log)
	if prob != nil {
		LogProblem(r.Context(), log, prob)
		pj := &ProblemJSON{Problem: prob}
		pj.WriteHTTPResponse(w)
		return
	}

	log.DebugContext(r.Context(), "Finished Calling "+h.Name)
	resp.WriteHTTPResponse(w)
}

func HandleJSONRequest[S any, T HTTPResponse](endpoint func(r *http.Request, body S, log *slog.Logger) (T, Problem)) http.Handler {
	name := getFuncName(endpoint)
	return &JSONBodyHandler[S, T]{
		Name:         name,
		EndpointFunc: endpoint,
	}
}

type BlankBodyHandler[S any, T HTTPResponse] struct {
	Name         string
	EndpointFunc func(r *http.Request, log *slog.Logger) (T, Problem)
}

func (h *BlankBodyHandler[S, T]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := slog.Default().With("endpoint", h.Name)

	resp, prob := h.EndpointFunc(r, log)
	if prob != nil {
		LogProblem(r.Context(), log, prob)
		pj := &ProblemJSON{Problem: prob}
		pj.WriteHTTPResponse(w)
		return
	}

	log.DebugContext(r.Context(), "Finished Calling "+h.Name)
	resp.WriteHTTPResponse(w)
}

func HandleBlankRequest[S any, T HTTPResponse](endpoint func(r *http.Request, body S, log *slog.Logger) (T, Problem)) http.Handler {
	name := getFuncName(endpoint)
	return &JSONBodyHandler[S, T]{
		Name:         name,
		EndpointFunc: endpoint,
	}
}
