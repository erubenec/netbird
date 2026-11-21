// Package errorpage describes the custom HTML an administrator can serve in
// place of the proxy's built-in error page for a service.
package errorpage

import (
	"errors"
	"fmt"
	"html"
	"strconv"
	"strings"
	"unicode/utf8"
)

// MaxHTMLBytes bounds the stored page so a mapping stays small.
const MaxHTMLBytes = 64 * 1024

// Page is an optional replacement for the proxy's error page. A nil *Page
// means the built-in page is used.
type Page struct {
	// HTML is a complete HTML document. The placeholders {{status}}, {{title}},
	// {{message}} and {{request_id}} are replaced with HTML-escaped values.
	HTML string `json:"html"`
}

// Data holds the values substituted into the placeholders.
type Data struct {
	Status    int
	Title     string
	Message   string
	RequestID string
}

// Copy returns a copy of the page, or nil for a nil receiver.
func (p *Page) Copy() *Page {
	if p == nil {
		return nil
	}
	out := *p
	return &out
}

// Validate checks that the page can be stored and served.
func (p *Page) Validate() error {
	if p == nil {
		return nil
	}
	if strings.TrimSpace(p.HTML) == "" {
		return errors.New("error page html is empty")
	}
	if len(p.HTML) > MaxHTMLBytes {
		return fmt.Errorf("error page html is %d bytes, maximum is %d", len(p.HTML), MaxHTMLBytes)
	}
	if !utf8.ValidString(p.HTML) {
		return errors.New("error page html is not valid UTF-8")
	}
	if strings.ContainsRune(p.HTML, 0) {
		return errors.New("error page html contains a NUL byte")
	}
	return nil
}

// Render fills the placeholders. Every substituted value is HTML-escaped, so
// request data can never inject markup; unknown placeholders are left as is.
func (p *Page) Render(d Data) []byte {
	if p == nil {
		return nil
	}
	return []byte(strings.NewReplacer(
		"{{status}}", strconv.Itoa(d.Status),
		"{{title}}", html.EscapeString(d.Title),
		"{{message}}", html.EscapeString(d.Message),
		"{{request_id}}", html.EscapeString(d.RequestID),
	).Replace(p.HTML))
}
