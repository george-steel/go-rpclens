# go-rpclens

An experimental framework for creating HTTP API endpoints using the lens pattern.

A lens is a function mapping operations on a part to operations on the whole.
If the operations are total (with no error pathway), such a function acts like a bidirectional, logical property accessor, and is a common pattern in functional programming.
If we generalize the concept to allow operations to fail, instead of providing access to a field, the lens provides access to a specific case, with the lens itself handling the others.

In this library, we provide generalized lenses to handle some of the common cases of HTTP API calls.
In all of these, the outer operation is a `http.Handler` (which can be used by ServeMux or any compatible framework), accepting a request and writing out a response.
The inner operation takes (optionally, depending on the lens used) a parsed response body
and returns either a structured response (such as NoContent or JSONBody) or a structured error (implementing Problem).
If a body is required but cannot be parsed, an error will be returned immediately and the inner operation will not be called.
This allows the operation to be decoupled from the HTTP protocol logic and use standard go control flow,
taking parameters and returning values and errors.
