//go:build goid_nopeter

package gid

// peterProvider stub when goid_nopeter tag is set.
// Returns nil to disable petermattis/goid provider.

func getPeterProvider() Provider {
	return nil
}
