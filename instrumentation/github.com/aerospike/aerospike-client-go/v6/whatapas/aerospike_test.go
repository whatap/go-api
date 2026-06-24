package whatapas

import (
	"testing"

	aerospike "github.com/aerospike/aerospike-client-go/v6"
)

func TestDbhostFromHosts_Empty(t *testing.T) {
	got := DbhostFromHosts()
	if got != "aerospike" {
		t.Errorf("empty hosts: got %q, want %q", got, "aerospike")
	}
}

func TestDbhostFromHosts_Single(t *testing.T) {
	h := aerospike.NewHost("localhost", 3000)
	got := DbhostFromHosts(h)
	want := "aerospike://localhost:3000"
	if got != want {
		t.Errorf("single host: got %q, want %q", got, want)
	}
}

func TestDbhostFromHosts_Multiple(t *testing.T) {
	h1 := aerospike.NewHost("host1", 3000)
	h2 := aerospike.NewHost("host2", 3001)
	h3 := aerospike.NewHost("host3", 3002)
	got := DbhostFromHosts(h1, h2, h3)
	want := "aerospike://host1:3000,host2:3001,host3:3002"
	if got != want {
		t.Errorf("multiple hosts: got %q, want %q", got, want)
	}
}

func TestDbhostFromHosts_NilEntriesSkipped(t *testing.T) {
	h := aerospike.NewHost("localhost", 3000)
	got := DbhostFromHosts(nil, h, nil)
	want := "aerospike://localhost:3000"
	if got != want {
		t.Errorf("nil entries should be skipped: got %q, want %q", got, want)
	}
}

func TestDbhostFromHosts_AllNil(t *testing.T) {
	got := DbhostFromHosts(nil, nil)
	if got != "aerospike" {
		t.Errorf("all nil hosts: got %q, want %q", got, "aerospike")
	}
}

func TestDbhostFromHosts_Spread(t *testing.T) {
	hosts := []*aerospike.Host{
		aerospike.NewHost("h1", 3000),
		aerospike.NewHost("h2", 3000),
	}
	got := DbhostFromHosts(hosts...)
	want := "aerospike://h1:3000,h2:3000"
	if got != want {
		t.Errorf("spread call: got %q, want %q", got, want)
	}
}

// §236 — multi-element spread (5 hosts in a slice). Verifies the helper
// scales beyond the 2-element common case.
func TestDbhostFromHosts_SpreadMultiElement(t *testing.T) {
	hosts := []*aerospike.Host{
		aerospike.NewHost("h1", 3000),
		aerospike.NewHost("h2", 3001),
		aerospike.NewHost("h3", 3002),
		aerospike.NewHost("h4", 3003),
		aerospike.NewHost("h5", 3004),
	}
	got := DbhostFromHosts(hosts...)
	want := "aerospike://h1:3000,h2:3001,h3:3002,h4:3003,h5:3004"
	if got != want {
		t.Errorf("5-element spread: got %q, want %q", got, want)
	}
}

// §236 — individual args (5 host variables). The user can also write
// f(policy, h1, h2, h3, h4, h5) directly; the helper must handle it the same.
func TestDbhostFromHosts_IndividualMultiArgs(t *testing.T) {
	h1 := aerospike.NewHost("h1", 3000)
	h2 := aerospike.NewHost("h2", 3001)
	h3 := aerospike.NewHost("h3", 3002)
	h4 := aerospike.NewHost("h4", 3003)
	h5 := aerospike.NewHost("h5", 3004)
	got := DbhostFromHosts(h1, h2, h3, h4, h5)
	want := "aerospike://h1:3000,h2:3001,h3:3002,h4:3003,h5:3004"
	if got != want {
		t.Errorf("5 individual args: got %q, want %q", got, want)
	}
}

// §236 — multi-element slice with nils interleaved. Nil entries must be
// skipped, remaining entries joined in original order.
func TestDbhostFromHosts_MultiElementWithNils(t *testing.T) {
	hosts := []*aerospike.Host{
		aerospike.NewHost("h1", 3000),
		nil,
		aerospike.NewHost("h2", 3001),
		nil,
		aerospike.NewHost("h3", 3002),
	}
	got := DbhostFromHosts(hosts...)
	want := "aerospike://h1:3000,h2:3001,h3:3002"
	if got != want {
		t.Errorf("nils-interleaved spread: got %q, want %q", got, want)
	}
}
