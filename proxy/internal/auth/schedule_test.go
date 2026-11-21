package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netbirdio/netbird/shared/schedule"
)

func scheduledMiddleware(t *testing.T, s *schedule.Schedule, now time.Time) *Middleware {
	t.Helper()
	mw := NewMiddleware(log.StandardLogger(), nil, nil)
	mw.now = func() time.Time { return now }

	config, err := NewDomainConfig("example.com", nil, "", time.Hour, "", "", nil, false, nil, WithSchedule(s))
	require.NoError(t, err)
	mw.AddDomainConfig("example.com", config)
	return mw
}

func serve(mw *Middleware) (*httptest.ResponseRecorder, bool) {
	var reached bool
	handler := mw.Protect(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		_, _ = w.Write([]byte("backend"))
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://example.com/", nil))
	return rec, reached
}

func officeHours() *schedule.Schedule {
	return &schedule.Schedule{
		Timezone: "Europe/Madrid",
		Windows: []schedule.Window{{
			Days:  []string{"mon", "tue", "wed", "thu", "fri"},
			Start: "08:00",
			End:   "17:00",
		}},
	}
}

func madrid(t *testing.T, value string) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Madrid")
	require.NoError(t, err)
	ts, err := time.ParseInLocation("2006-01-02 15:04", value, loc)
	require.NoError(t, err)
	return ts
}

func TestProtect_Schedule_InsideWindowReachesBackend(t *testing.T) {
	mw := scheduledMiddleware(t, officeHours(), madrid(t, "2026-11-23 10:00"))

	rec, reached := serve(mw)

	assert.True(t, reached)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "backend", rec.Body.String())
}

func TestProtect_Schedule_OutsideWindowServesUnavailable(t *testing.T) {
	tests := []struct {
		name string
		when string
	}{
		{"after hours", "2026-11-23 18:00"},
		{"weekend", "2026-11-21 10:00"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mw := scheduledMiddleware(t, officeHours(), madrid(t, tc.when))

			rec, reached := serve(mw)

			assert.False(t, reached, "backend must not be reached outside the schedule")
			assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
			assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
			assert.NotContains(t, rec.Body.String(), "backend")
		})
	}
}

func TestProtect_Schedule_NilIsAlwaysOn(t *testing.T) {
	mw := scheduledMiddleware(t, nil, madrid(t, "2026-11-21 03:00"))

	_, reached := serve(mw)

	assert.True(t, reached)
}

// A schedule the proxy cannot evaluate must close the service rather than
// silently exposing it.
func TestProtect_Schedule_InvalidFailsClosed(t *testing.T) {
	mw := scheduledMiddleware(t, &schedule.Schedule{Timezone: "Not/AZone"}, madrid(t, "2026-11-23 10:00"))

	rec, reached := serve(mw)

	assert.False(t, reached)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestWithSchedule_CopiesInput(t *testing.T) {
	s := officeHours()
	config, err := NewDomainConfig("example.com", nil, "", time.Hour, "", "", nil, false, nil, WithSchedule(s))
	require.NoError(t, err)

	s.Windows[0].Days[0] = "sun"

	assert.Equal(t, "mon", config.Schedule.Windows[0].Days[0], "later edits to the input must not leak into the snapshot")
}
