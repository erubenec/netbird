package proxy

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/netbirdio/netbird/shared/errorpage"
	"github.com/netbirdio/netbird/shared/management/proto"
)

func TestErrorPageFromProto(t *testing.T) {
	assert.Nil(t, errorPageFromProto(nil), "no page keeps the built-in one")

	page := errorPageFromProto(&proto.ErrorPage{Html: "<h1>x</h1>"})
	if assert.NotNil(t, page) {
		assert.Equal(t, "<h1>x</h1>", page.HTML)
	}
}

// A page the proxy cannot trust is dropped, so visitors still get a working page.
func TestErrorPageFromProto_DropsInvalidPage(t *testing.T) {
	assert.Nil(t, errorPageFromProto(&proto.ErrorPage{Html: ""}))
	assert.Nil(t, errorPageFromProto(&proto.ErrorPage{Html: strings.Repeat("a", errorpage.MaxHTMLBytes+1)}))
	assert.Nil(t, errorPageFromProto(&proto.ErrorPage{Html: "<p>\xff</p>"}))
}
