//go:build !goid_unsafe

package gid

// unsafeProvider stub when goid_unsafe tag is NOT set.
// Returns nil to indicate unsafe provider is disabled.
// This is the default - unsafe must be explicitly opted in.

func getUnsafeProvider() Provider {
	return nil
}
