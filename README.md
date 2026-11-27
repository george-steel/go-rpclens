# rpclens

A minimal HTTP API framework in modern go where endpoints are written as ordinary functions taking and returning structured data.
Heavily inspired by [Servant](https://www.servant.dev/) and [FastAPI](https://fastapi.tiangolo.com/).

Core functionality is roughly divided into two parts:
a system for returning structured, serializable errors (the `Problem` interface and its `Problemf` constructor),
and a set of lens functions for used to convert endpoint functions into `http.Handler` objects.

## Error Handling

While go's text-based `error` interface is simple,
it is not quite sufficient for dealing with structured errors that need to be returned from HTTP endpoints.
To be able to return structured errors, we define the `Problem` interface, which extends error with additional data.

Problem follows the semantics of the [RFC 9457 *Problem Details for HTTP APIs*](https://www.rfc-editor.org/info/rfc9457/) specification.
In addition to the human-readable error message (with separate methods for log message and the response, so they can differ if necessary),
a Problem also contains a classification type (represented here as a `ProblemType` object),
consisting of an HTTP status code, an optional URI for further specificity,
and a human-readable title (this should be consistent for each status/URI pair).
A Problem can also define HTTP headers, additional fields for the response body, and additional log fields.

Although Problems can be created as fully-custom objects implementing the interface (see `UnsupportedMediaTypeError` for such an example),
simple cases (lacking headers and additional fields) can be handled through the `Problemf` constructor.
This acts as `fmt.Errorf` does, taking a ProblemType as its first parameter to provide classification, and returning a Problem.
`StatusProblemf` does the same, but takes an integer status code instead of a ProblemType.
By using the `%w` formatter, Problemf can wrap existing `error` objects,
including their message and making the original error accessable through Unwrap,
allowing it to be matched by the function in go's `errors` package even when wrapped.

When returning an error response (in the library's default configuration),
a non-nil Problem returned by an endpoint function is serialized as a JSON object,
using the standard fields defined in RFC 9457.
This can be overriden by setting `MakeProblemResponse`,
the default value of which constructs a `ProblemResponseJSON` wrapping the Problem.
For APIs which require errors to be sent as `text/plain`,
constructing a `ProblemResponseText` instead will achieve this.

## Response Types

To allow HTTP responses to be returned from endpoints (instead of having to write to a `ResponseWriter` manually),
we define the `HTTPResponse` interface, along with some common implementations.
This consists of a single method `WriteHTTPResponse`, which sets the headers and status code and writes the response.

Most of the time, you will likely use `*JSONBody[T any]` to return structured data as JSON,
along with a status code and any additional headers.
This serializes using `json/v2` which, unlike its predecessor, can correctly handle the `nil` lists returned by many library functions as empty.
For `200 OK` responses, the `AsJSONBody` constructor can keep things simple.

Additionally, endpoints that do not need to return a response body can use
`*NoContent` (which is allowed to be nil and still be used) or
`*HeadersOnly` (which can not be nil in a non-error case).

## Endpoint Lenses

To avoid a need for excessive reflection when creating a `http.Handler` from an endpoint function,
this library takes inspiration from the lens pattern,
using several generic lens functions for different cases of transforming endpoints into Handlers.

For those unfamiliar with the pattern, a lens is a function mapping operations on a part of something to operations on the whole.
They are often used as a form of composable field accessor,
as they are isomorphic to a getter/setter pair if we restrict the lens to call the operation exactly once,
but if we remove this restriction, they can become much more general, such as providing access to a specific case.

In this library, we provide a number of generic lenses to transform various shapes of endpoint functions into raw `http.Handler`s,
usable by `http.ServeMux` or any other routing system.

The simplest of these is `HandleEmptyRequest`, which takes an endpoint of the form
`func(r *http.Request, log *slog.Logger) (T, Problem)`,
where T is a generic parameter for the success-case return type implementing `HTTPResponse`.

For endpoints requiring a request body, we have `HandleJSONRequest`, which takes an endpoint of type
`func(r *http.Request, body S, log *slog.Logger) (T, Problem)`,
where S is any JSON-serializable type and T implements `HTTPResponse`.
This will automatically construct a zero `var body S` and unmarshal the HTTP request into it using `json/v2` before calling the endpoint.
An error an unmarshaling will cause the handler to send a 400 error to the client without calling the endpoint.


## Configuration
