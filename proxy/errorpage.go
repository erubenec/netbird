package proxy

import (
	log "github.com/sirupsen/logrus"

	"github.com/netbirdio/netbird/shared/errorpage"
	"github.com/netbirdio/netbird/shared/management/proto"
)

// errorPageFromProto converts a mapping's custom error page. An invalid page is
// dropped so the built-in page is served instead of something unusable.
func errorPageFromProto(p *proto.ErrorPage) *errorpage.Page {
	if p == nil {
		return nil
	}
	page := &errorpage.Page{HTML: p.GetHtml()}
	if err := page.Validate(); err != nil {
		log.Warnf("ignoring invalid custom error page: %v", err)
		return nil
	}
	return page
}
