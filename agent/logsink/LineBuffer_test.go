package logsink

import (
	"fmt"
	"strings"
	"testing"
	// "github.com/stretchr/testify/assert"
)

func TestAppendLine(t *testing.T) {
	bf := NewLineBuffer()
	ss := []string{"abc1", "abc2", "\tdef3", "\tdef4", "abc5", "abc6"}
	for i, s := range ss {
		v := bf.AppendLine(s)
		for j, tmp := range v {
			fmt.Println(i, "-", j, ", ", s, "=>", tmp)
		}
	}
}

// TestStackTraceScenario tests the stack trace line splitting issue (§29)
// Expected: Error message and stack trace should be grouped together
func TestStackTraceScenario(t *testing.T) {
	fmt.Println("\n=== Stack Trace Scenario (Current Behavior) ===")
	bf := NewLineBuffer()

	// Simulated stack trace like log.Println would produce
	// Each line is a separate call (simulating separate Println calls)
	lines := []string{
		"Error: something went wrong",
		"\tat github.com/example/app.handler()",
		"\tat github.com/example/app.middleware()",
		"\tat main.main()",
		"Next log message",
	}

	fmt.Println("Input lines:")
	for i, line := range lines {
		fmt.Printf("  [%d] %q\n", i, line)
	}

	fmt.Println("\nOutput (each AppendLine result):")
	for i, line := range lines {
		results := bf.AppendLine(line)
		if len(results) > 0 {
			for j, r := range results {
				// Show what's returned - separate log entries
				fmt.Printf("  [%d-%d] %q\n", i, j, r)
			}
		}
	}

	// Flush remaining
	if remaining := bf.Flush(); remaining != "" {
		fmt.Printf("  [flush] %q\n", remaining)
	}

	fmt.Println("\n=== Analysis ===")
	fmt.Println("Current: 'Error:' line is returned BEFORE the tab-prefixed stack lines")
	fmt.Println("Issue: Error message and stack trace are in separate log entries")
}

// TestStackTraceGrouping tests if we can modify behavior to group stack traces
func TestStackTraceGrouping(t *testing.T) {
	fmt.Println("\n=== Stack Trace Grouping Test ===")
	bf := NewLineBuffer()

	// Real runtime.Stack() output pattern (single Write with multiple lines)
	stackTrace := `goroutine 1 [running]:
	main.handleStack(...)
		/app/main.go:42 +0x1a3
	net/http.HandlerFunc.ServeHTTP(...)
		/go/src/net/http/server.go:2294 +0x29`

	fmt.Println("Input (single multi-line string):")
	fmt.Println(stackTrace)

	fmt.Println("\nProcessing line by line (simulating bufio.Scanner):")
	for i, line := range strings.Split(stackTrace, "\n") {
		results := bf.AppendLine(line)
		if len(results) > 0 {
			for j, r := range results {
				fmt.Printf("  [%d-%d] %q\n", i, j, r)
			}
		}
	}

	if remaining := bf.Flush(); remaining != "" {
		fmt.Printf("  [flush] %q\n", remaining)
	}
}

func TestAppend(t *testing.T) {
	bf := NewLineBuffer()
	ss := []string{"abc1", "abc2", "\tdef3", "\tdef4", "abc5", "abc6\n"}
	for i, s := range ss {
		v := bf.Append(s)
		for j, tmp := range v {
			fmt.Println(i, "-", j, ", ", s, "=>", tmp)
		}
	}

	fmt.Println(bf.Flush())
}
