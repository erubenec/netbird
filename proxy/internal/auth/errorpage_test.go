package auth

import (
	"net/http"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netbirdio/netbird/shared/errorpage"
	"github.com/netbirdio/netbird/shared/schedule"
)

const customMarker = "CUSTOM-PAGE-MARKER"

func middlewareWithPage(t *testing.T, page *errorpage.Page, s *schedule.Schedule) *Middleware {
	t.Helper()
	mw := NewMiddleware(log.StandardLogger(), nil, nil)
	mw.now = func() time.Time { return madrid(t, "2026-11-21 03:00") }

	config, err := NewDomainConfig("example.com", nil, "", time.Hour, "", "", nil, false, nil,
		WithSchedule(s), WithErrorPage(page))
	require.NoError(t, err)
	mw.AddDomainConfig("example.com", config)
	return mw
}

func closedSchedule() *schedule.Schedule {
	return &schedule.Schedule{
		Timezone: "Europe/Madrid",
		Windows:  []schedule.Window{{Days: []string{"mon"}, Start: "08:00", End: "09:00"}},
	}
}

func TestProtect_ErrorPage_ServedWhenScheduleCloses(t *testing.T) {
	page := &errorpage.Page{HTML: "<h1>" + customMarker + " {{status}}</h1><p>{{title}}</p>"}
	mw := middlewareWithPage(t, page, closedSchedule())

	rec, reached := serve(mw)

	assert.False(t, reached, "the backend must not be reached outside the schedule")
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), customMarker+" 503")
	assert.Contains(t, rec.Body.String(), "Service Unavailable")
	assert.Contains(t, rec.Header().Get("Content-Security-Policy"), "sandbox")
}

func TestProtect_ErrorPage_AbsentKeepsBuiltInPage(t *testing.T) {
	mw := middlewareWithPage(t, nil, closedSchedule())

	rec, reached := serve(mw)

	assert.False(t, reached)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.NotContains(t, rec.Body.String(), customMarker)
	assert.Empty(t, rec.Header().Get("Content-Security-Policy"))
}

func TestProtect_ErrorPage_NotUsedWhileServiceIsOpen(t *testing.T) {
	page := &errorpage.Page{HTML: "<h1>" + customMarker + "</h1>"}
	mw := middlewareWithPage(t, page, nil)

	rec, reached := serve(mw)

	assert.True(t, reached, "a custom error page must not affect healthy requests")
	assert.Equal(t, "backend", rec.Body.String())
}

func TestWithErrorPage_CopiesInput(t *testing.T) {
	page := &errorpage.Page{HTML: "a"}
	config, err := NewDomainConfig("example.com", nil, "", time.Hour, "", "", nil, false, nil, WithErrorPage(page))
	require.NoError(t, err)

	page.HTML = "changed"

	assert.Equal(t, "a", config.ErrorPage.HTML)
}
