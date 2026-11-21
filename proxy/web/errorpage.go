package web

import (
	"context"
	"net/http"

	"github.com/netbirdio/netbird/shared/errorpage"
)

// customPageCSP lets an administrator's page style itself and show https or
// inline images, and nothing else: no scripts, no forms, no framing, and the
// sandbox directive drops the page into an opaque origin so it cannot reach
// the service's cookies even if a script slipped through.
const customPageCSP = "default-src 'none'; style-src 'unsafe-inline'; img-src https: data:; " +
	"font-src https: data:; form-action 'none'; base-uri 'none'; frame-ancestors 'none'; sandbox"

type errorPageKey struct{}

// WithErrorPage returns a context that makes ServeErrorPage serve the service's
// custom page instead of the built-in one. A nil page returns ctx unchanged.
func WithErrorPage(ctx context.Context, page *errorpage.Page) context.Context {
	if page == nil {
		return ctx
	}
	return context.WithValue(ctx, errorPageKey{}, page)
}

func errorPageFrom(ctx context.Context) *errorpage.Page {
	page, _ := ctx.Value(errorPageKey{}).(*errorpage.Page)
	return page
}

// serveCustomErrorPage writes the administrator's page with the original
// status code and the restrictive headers above.
func serveCustomErrorPage(w http.ResponseWriter, page *errorpage.Page, code int, title, message, requestID string) {
	body := page.Render(errorpage.Data{Status: code, Title: title, Message: message, RequestID: requestID})

	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Content-Security-Policy", customPageCSP)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	w.Write(body) //nolint:errcheck
}
