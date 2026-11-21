package auth

import "github.com/netbirdio/netbird/shared/errorpage"

// WithErrorPage sets the page served in place of the built-in error pages. A
// nil page keeps the built-in ones.
func WithErrorPage(p *errorpage.Page) DomainOption {
	return func(config *DomainConfig) error {
		config.ErrorPage = p.Copy()
		return nil
	}
}
