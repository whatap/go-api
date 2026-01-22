//go:build !goid_nopeter

package gid

import "github.com/petermattis/goid"

// peterProvider uses petermattis/goid library.
// This is the fastest and most reliable method.
type peterProvider struct{}

func (p *peterProvider) Get() int64 {
	return goid.Get()
}

func (p *peterProvider) Name() string {
	return "petermattis/goid"
}

func (p *peterProvider) Available() bool {
	return true
}

func getPeterProvider() Provider {
	return &peterProvider{}
}
