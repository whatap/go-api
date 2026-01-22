//go:build goid_noslow

package gid

// slowProvider stub when goid_noslow tag is set.
// Returns nil to disable slow provider.
// WARNING: Not recommended as slow is the fallback and used for self-check.

func getSlowProvider() Provider {
	return nil
}
