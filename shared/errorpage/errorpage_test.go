package errorpage

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPage_Nil(t *testing.T) {
	var p *Page
	assert.NoError(t, p.Validate())
	assert.Nil(t, p.Copy())
	assert.Nil(t, p.Render(Data{Status: 503}))
}

func TestPage_Validate(t *testing.T) {
	tests := []struct {
		name    string
		page    Page
		wantErr string
	}{
		{"ok", Page{HTML: "<h1>{{title}}</h1>"}, ""},
		{"empty", Page{HTML: ""}, "empty"},
		{"blank", Page{HTML: " \n\t"}, "empty"},
		{"at the limit", Page{HTML: strings.Repeat("a", MaxHTMLBytes)}, ""},
		{"over the limit", Page{HTML: strings.Repeat("a", MaxHTMLBytes+1)}, "maximum"},
		{"invalid utf8", Page{HTML: "<p>\xff\xfe</p>"}, "UTF-8"},
		{"nul byte", Page{HTML: "<p>a\x00b</p>"}, "NUL"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.page.Validate()
			if tc.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestPage_Render_SubstitutesPlaceholders(t *testing.T) {
	p := &Page{HTML: "<h1>{{status}} {{title}}</h1><p>{{message}}</p><i>{{request_id}}</i>"}

	out := string(p.Render(Data{Status: 503, Title: "Service Unavailable", Message: "Try later", RequestID: "abc123"}))

	assert.Equal(t, "<h1>503 Service Unavailable</h1><p>Try later</p><i>abc123</i>", out)
}

// The values come from the proxy and from request headers, so they must never
// be able to add markup to the administrator's page.
func TestPage_Render_EscapesValues(t *testing.T) {
	p := &Page{HTML: "<p>{{message}}</p><i>{{request_id}}</i><b>{{title}}</b>"}

	out := string(p.Render(Data{
		Title:     `<img src=x onerror=alert(1)>`,
		Message:   `"><script>alert(1)</script>`,
		RequestID: `</i><script>x</script>`,
	}))

	assert.NotContains(t, out, "<script>")
	assert.NotContains(t, out, "<img")
	assert.Contains(t, out, "&lt;script&gt;")
}

func TestPage_Render_LeavesUnknownPlaceholders(t *testing.T) {
	p := &Page{HTML: "{{status}} {{unknown}} {{ title }}"}
	assert.Equal(t, "404 {{unknown}} {{ title }}", string(p.Render(Data{Status: 404, Title: "x"})))
}

func TestPage_Copy_IsIndependent(t *testing.T) {
	orig := &Page{HTML: "a"}
	cp := orig.Copy()
	cp.HTML = "b"
	assert.Equal(t, "a", orig.HTML)
}
