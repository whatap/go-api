// Package gid provides goroutine ID retrieval with multiple provider implementations.
// It automatically selects the best available provider at runtime.
//
// Provider priority: petermattis/goid > unsafe (getg) > slow (runtime.Stack)
//
// Build tags:
//   - goid_nopeter: Disable petermattis/goid provider
//   - goid_unsafe: Enable unsafe provider (opt-in)
//   - goid_noslow: Disable slow provider (not recommended)
package gid

import (
	"fmt"
	"os"
	"sync"

	"github.com/whatap/go-api/agent/agent/config"
)

// Provider defines the interface for goroutine ID providers.
type Provider interface {
	// Get returns the current goroutine ID.
	Get() int64
	// Name returns the provider name for logging/debugging.
	Name() string
	// Available returns true if the provider is available.
	Available() bool
}

var (
	// activeProvider is the currently active GID provider.
	activeProvider Provider
	// providerOnce ensures provider selection happens only once.
	providerOnce sync.Once
)

// selectProvider selects the best available GID provider.
// Priority: petermattis > unsafe > slow
func selectProvider() {
	providers := []Provider{
		getPeterProvider(),
		getUnsafeProvider(),
		getSlowProvider(),
	}

	for _, p := range providers {
		if p == nil || !p.Available() {
			continue
		}

		// Self-check: verify the provider returns correct GID
		if !selfCheck(p) {
			if config.GetConfig().Debug {
				fmt.Fprintf(os.Stderr, "[WA-GID] Provider %s failed self-check, skipping\n", p.Name())
			}
			continue
		}

		activeProvider = p
		if config.GetConfig().Debug {
			fmt.Fprintf(os.Stderr, "[WA-GID] Using provider: %s\n", p.Name())
		}
		return
	}

	// This should never happen since slow is always available
	panic("No GID provider available")
}

// selfCheck verifies a provider returns the correct goroutine ID.
// Uses slow provider as the reference (always correct).
func selfCheck(p Provider) bool {
	slow := getSlowProvider()
	if slow == nil || !slow.Available() {
		// If slow is not available (goid_noslow tag), skip self-check
		return true
	}

	// Run multiple checks to catch potential race conditions
	for i := 0; i < 3; i++ {
		expected := slow.Get()
		actual := p.Get()
		if expected != actual {
			if config.GetConfig().Debug {
				fmt.Fprintf(os.Stderr, "[WA-GID] Self-check failed: expected=%d, actual=%d (provider=%s)\n",
					expected, actual, p.Name())
			}
			return false
		}
	}
	return true
}

// GetGID returns the current goroutine ID.
// This is the main API that should be used by all callers.
func GetGID() int64 {
	providerOnce.Do(selectProvider)
	return activeProvider.Get()
}

// GetProviderName returns the name of the active provider.
// Useful for debugging and logging.
func GetProviderName() string {
	providerOnce.Do(selectProvider)
	return activeProvider.Name()
}
