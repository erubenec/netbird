package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netbirdio/netbird/shared/errorpage"
)

const marker = "CUSTOM-PAGE-MARKER"

func customPage() *errorpage.Page {
	return &errorpage.Page{HTML: "<!doctype html><h1>" + marker + " {{status}}</h1><p>{{title}}: {{message}} ({{request_id}})</p>"}
}

func request(t *testing.T, page *errorpage.Page) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "http://svc.example/", nil)
	return r.WithContext(WithErrorPage(r.Context(), page))
}

func TestServeErrorPage_CustomPage(t *testing.T) {
	rec := httptest.NewRecorder()

	ServeErrorPage(rec, request(t, customPage()), http.StatusBadGateway, "Bad <Gateway>", "it broke", "req-1", ErrorStatus{Proxy: true})

	assert.Equal(t, http.StatusBadGateway, rec.Code, "the original status code must be kept")
	body := rec.Body.String()
	assert.Contains(t, body, marker+" 502")
	assert.Contains(t, body, "Bad &lt;Gateway&gt;: it broke (req-1)", "values must be HTML-escaped")
	assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
}

// The page is administrator-supplied HTML served from the service's own origin,
// so scripts must be blocked by policy even if the markup contains them.
func TestServeErrorPage_CustomPageIsLockedDown(t *testing.T) {
	rec := httptest.NewRecorder()
	page := &errorpage.Page{HTML: "<h1>x</h1><script>alert(document.cookie)</script>"}

	ServeErrorPage(rec, request(t, page), http.StatusServiceUnavailable, "t", "m", "id", ErrorStatus{})

	csp := rec.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "default-src 'none'")
	assert.Contains(t, csp, "sandbox", "the sandbox directive drops the origin, and with it the cookies")
	assert.NotContains(t, csp, "script-src")
	assert.NotContains(t, csp, "'unsafe-eval'")
	assert.Contains(t, csp, "frame-ancestors 'none'")
	assert.Contains(t, csp, "form-action 'none'")
}

func TestServeErrorPage_WithoutCustomPageUsesBuiltIn(t *testing.T) {
	rec := httptest.NewRecorder()

	ServeErrorPage(rec, request(t, nil), http.StatusServiceUnavailable, "Service Unavailable", "m", "id", ErrorStatus{Proxy: true})

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.NotContains(t, rec.Body.String(), marker)
	assert.Empty(t, rec.Header().Get("Content-Security-Policy"), "the built-in page keeps its own behaviour")
}

// Access denied is part of the authentication flow, so it is never replaced.
func TestServeAccessDeniedPage_IgnoresCustomPage(t *testing.T) {
	rec := httptest.NewRecorder()

	ServeAccessDeniedPage(rec, request(t, customPage()), http.StatusForbidden, "Access Denied", "no", "id")

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.NotContains(t, rec.Body.String(), marker)
}

func TestWithErrorPage_NilLeavesContextUnchanged(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	require.Equal(t, r.Context(), WithErrorPage(r.Context(), nil))
	assert.Nil(t, errorPageFrom(r.Context()))
}
