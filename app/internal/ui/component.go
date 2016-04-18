package ui

import (
	"crypto/sha256"
	"encoding/json"
	"html/template"
	"net/http"

	"gopkg.in/inconshreveable/log15.v2"

	"golang.org/x/net/context"
	"sourcegraph.com/sourcegraph/appdash"
	"sourcegraph.com/sourcegraph/sourcegraph/app/jscontext"
	"sourcegraph.com/sourcegraph/sourcegraph/util/traceutil"
)

// RenderResult is the "HTTP response"-like data returned by the
// JavaScript server-side rendering operation.
type RenderResult struct {
	Body             string          // HTTP response body
	Error            string          // internal error message (should only be shown to admins, may contain secret info)
	Stores           json.RawMessage // contents of stores after prerendering (for client bootstrapping)
	StatusCode       int             // HTTP status code for response
	ContentType      string          // HTTP Content-Type response header
	RedirectLocation string          // HTTP Location header

	// Head is the contents of Helmet after rewind (for server rendering <head>).
	Head Head
}

// Head is data for the HTML <head> element.
type Head struct {
	HTMLAttributes template.HTML
	Title          template.HTML
	Base           template.HTML
	Meta           template.HTML
	Link           template.HTML
	Script         template.HTML
}

type renderState struct {
	JSContext  jscontext.JSContext    `json:"jsContext"`
	Location   string                 `json:"location"`
	Deadline   int64                  `json:"deadline"` // milliseconds since epoch, like Date.now()
	ExtraProps map[string]interface{} `json:"extraProps"`
}

// RenderRouter calls into JavaScript (using jsserver) to render the
// page for the given HTTP request.
var RenderRouter = func(ctx context.Context, req *http.Request, extraProps map[string]interface{}) (*RenderResult, error) {
	// Trace operations.
	ctx = traceutil.NewContext(ctx, appdash.NewSpanID(traceutil.SpanIDFromContext(ctx)))

	jsctx, err := jscontext.NewJSContextFromRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	deadline, _ := ctx.Deadline()

	return renderRouterState(ctx, &renderState{
		JSContext:  jsctx,
		Location:   req.URL.String(),
		Deadline:   deadline.UnixNano() / (1000 * 1000),
		ExtraProps: extraProps,
	})
}

func renderRouterState(ctx context.Context, state *renderState) (*RenderResult, error) {
	if ctx == nil || !shouldPrerenderReact(ctx) {
		return nil, nil
	}

	arg, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}

	r, err := getRenderer(ctx)
	if err != nil {
		return nil, err
	}

	// Construct cache key.
	//
	// To increase cache hit rate, omit items from the cache key that
	// do not alter the behavior or privileges of the request, such as
	// the Appdash span ID.
	var cacheKey string
	{
		key := *state
		key.JSContext.CurrentSpanID = ""
		key.JSContext.CacheControl = ""
		key.JSContext.CSRFToken = "" // CSRFToken never used for auth, and this returns data, doesn't perform actions
		key.Deadline = 0
		keyJSON, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		keyArray := sha256.Sum256(keyJSON)
		cacheKey = string(keyArray[:])
	}

	data, err := r.Call(ctx, cacheKey, arg)
	if err != nil {
		log15.Warn("Error rendering React component on the server (falling back to client-side rendering)", "err", err, "arg", truncateArg(arg))
		return nil, err
	}

	var res *RenderResult
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func truncateArg(arg []byte) string {
	if max := 300; len(arg) > max {
		arg = arg[:max]
	}
	return string(arg)
}
