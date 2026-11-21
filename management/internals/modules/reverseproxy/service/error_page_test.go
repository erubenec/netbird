package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netbirdio/netbird/management/internals/modules/reverseproxy/proxy"
	"github.com/netbirdio/netbird/shared/errorpage"
	"github.com/netbirdio/netbird/shared/management/http/api"
)

func customPage() *errorpage.Page {
	return &errorpage.Page{HTML: "<!doctype html><h1>{{status}}</h1><p>{{message}}</p>"}
}

func TestErrorPage_APIRoundTrip(t *testing.T) {
	svc := validProxy()
	svc.ErrorPage = customPage()

	resp := svc.ToAPIResponse()
	require.NotNil(t, resp.ErrorPage)
	assert.Equal(t, svc.ErrorPage.HTML, resp.ErrorPage.Html)

	req := &api.ServiceRequest{
		Name:      svc.Name,
		Domain:    svc.Domain,
		Enabled:   true,
		Targets:   &resp.Targets,
		ErrorPage: resp.ErrorPage,
	}
	restored := &Service{}
	require.NoError(t, restored.FromAPIRequest(req, "acc-1"))

	assert.Equal(t, svc.ErrorPage, restored.ErrorPage)
}

func TestErrorPage_FromAPIRequest_AbsentClearsPage(t *testing.T) {
	svc := validProxy()
	svc.ErrorPage = customPage()

	req := &api.ServiceRequest{Name: svc.Name, Domain: svc.Domain, Enabled: true}
	require.NoError(t, svc.FromAPIRequest(req, "acc-1"))

	assert.Nil(t, svc.ErrorPage, "a PUT without an error page must remove it")
}

func TestErrorPage_ToProtoMapping(t *testing.T) {
	svc := validProxy()
	svc.ErrorPage = customPage()

	mapping := svc.ToProtoMapping(Create, "token", proxy.OIDCValidationConfig{})

	require.NotNil(t, mapping.ErrorPage)
	assert.Equal(t, svc.ErrorPage.HTML, mapping.ErrorPage.Html)
}

func TestErrorPage_ToProtoMapping_NoneByDefault(t *testing.T) {
	mapping := validProxy().ToProtoMapping(Create, "token", proxy.OIDCValidationConfig{})
	assert.Nil(t, mapping.ErrorPage)
}

func TestErrorPage_Validate(t *testing.T) {
	t.Run("valid HTTP service", func(t *testing.T) {
		svc := validProxy()
		svc.ErrorPage = customPage()
		assert.NoError(t, svc.Validate())
	})
	t.Run("empty html", func(t *testing.T) {
		svc := validProxy()
		svc.ErrorPage = &errorpage.Page{HTML: "  "}
		assert.ErrorContains(t, svc.Validate(), "empty")
	})
	t.Run("too large", func(t *testing.T) {
		svc := validProxy()
		svc.ErrorPage = &errorpage.Page{HTML: strings.Repeat("a", errorpage.MaxHTMLBytes+1)}
		assert.ErrorContains(t, svc.Validate(), "maximum")
	})
	t.Run("not supported for L4", func(t *testing.T) {
		svc := validProxy()
		svc.Mode = ModeTCP
		svc.ListenPort = 9000
		svc.ErrorPage = customPage()
		assert.ErrorContains(t, svc.Validate(), "only supported for HTTP")
	})
}

func TestErrorPage_Copy_IsIndependent(t *testing.T) {
	svc := validProxy()
	svc.ErrorPage = customPage()

	cp := svc.Copy()
	cp.ErrorPage.HTML = "changed"

	assert.NotEqual(t, "changed", svc.ErrorPage.HTML)
}
