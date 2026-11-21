package auth

import (
	"net/http"
	"time"

	"github.com/netbirdio/netbird/proxy/internal/proxy"
	"github.com/netbirdio/netbird/proxy/web"
	"github.com/netbirdio/netbird/shared/schedule"
)

// WithSchedule limits the hours during which the domain is reachable. A nil
// schedule means always reachable.
func WithSchedule(s *schedule.Schedule) DomainOption {
	return func(config *DomainConfig) error {
		config.Schedule = s.Copy()
		return nil
	}
}

// checkSchedule serves the service unavailable page and returns false when the
// request arrives outside the domain's schedule. An unevaluable schedule counts
// as closed.
func (mw *Middleware) checkSchedule(w http.ResponseWriter, r *http.Request, config DomainConfig) bool {
	if config.Schedule == nil || config.Schedule.Active(mw.clock()) {
		return true
	}

	var requestID string
	if cd := proxy.CapturedDataFromContext(r.Context()); cd != nil {
		cd.SetOrigin(proxy.OriginProxyError)
		cd.SetMetadata("schedule", "closed")
		requestID = cd.GetRequestID()
	}
	w.Header().Set("Cache-Control", "no-store")
	web.ServeErrorPage(w, r, http.StatusServiceUnavailable, "Service Unavailable",
		"This service is not available at this time. Please try again during its scheduled hours.",
		requestID, web.ErrorStatus{Proxy: true, Destination: false})
	return false
}

func (mw *Middleware) clock() time.Time {
	if mw.now != nil {
		return mw.now()
	}
	return time.Now()
}
