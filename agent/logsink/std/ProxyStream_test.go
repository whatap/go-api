package std

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/whatap/go-api/agent/logsink"
	"github.com/whatap/go-api/agent/util/logutil"
)

// Mock implementation of ISdSender for testing purposes
type MockSender struct {
	logs []logsink.LineLog
}

func (m *MockSender) Add(log *logsink.LineLog) {

	m.logs = append(m.logs, *log)
}

func TestNewProxyStreamStdout(t *testing.T) {

	logutil.GetLogger()

	sender := &MockSender{}
	origin := os.Stdout

	ps := NewProxyStream("TestCategory", os.Stdout, sender)
	os.Stdout = ps.GetWriter()
	ps.SetEnabled(true)

	time.Sleep(1 * time.Second)

	fmt.Println("Test Print")
	log.Println("Test Log Print")

	time.Sleep(1 * time.Second)

	fmt.Fprintln(origin, "sender.logs = ", sender.logs)
	assert.Equal(t, 1, len(sender.logs))

	os.Stdout = origin
	ps.Shutdown()
}

func TestLogFlushStdout(t *testing.T) {
	logutil.GetLogger()

	sender := &MockSender{}
	origin := os.Stdout

	ps := NewProxyStream("TestCategory", os.Stdout, sender)
	os.Stdout = ps.GetWriter()
	ps.SetEnabled(true)

	time.Sleep(1 * time.Second)

	fmt.Println("Test Print")
	fmt.Println("\t Tab1")
	fmt.Println("\t Tab2")
	fmt.Println("\t Tab3 ")
	fmt.Println("\t Tab4 ")
	fmt.Println("End Tab ")

	log.Println("Test Print")
	log.Println("\t Tab1")
	log.Println("\t Tab2")
	log.Println("\t Tab3 ")
	log.Println("\t Tab4 ")
	log.Println("End Tab ")

	time.Sleep(1 * time.Second)

	lineLog := ps.LogFlush()
	fmt.Fprintln(origin, "sender.flush logs = ", lineLog)
	assert.NotNil(t, lineLog)

	time.Sleep(1 * time.Second)

	os.Stdout = origin
	ps.Shutdown()
}

func TestNewProxyStreamStderr(t *testing.T) {

	logutil.GetLogger()

	sender := &MockSender{}
	origin := os.Stderr

	ps := NewProxyStream("TestCategory", os.Stderr, sender)
	os.Stderr = ps.GetWriter()
	ps.SetEnabled(true)

	time.Sleep(1 * time.Second)

	fmt.Println("Test Print")
	fmt.Fprintln(os.Stderr, "Test Error Print")
	log.Println("Test Log Print")

	// fmt.Fprintln(os.Stderr, "Test Log Print1")

	time.Sleep(1 * time.Second)

	fmt.Fprintln(origin, "sender.logs = ", sender.logs)
	assert.Equal(t, 1, len(sender.logs))

	lineLog := ps.LogFlush()
	fmt.Fprintln(origin, "sender.flush logs = ", lineLog)

	os.Stderr = origin
	ps.Shutdown()
}

func TestLogFlushStderr(t *testing.T) {
	logutil.GetLogger()

	sender := &MockSender{}
	origin := os.Stderr

	ps := NewProxyStream("TestCategory", os.Stderr, sender)
	os.Stderr = ps.GetWriter()
	ps.SetEnabled(true)

	time.Sleep(1 * time.Second)

	fmt.Println("Test Print")
	fmt.Println("\t Tab1")
	fmt.Println("\t Tab2")
	fmt.Println("\t Tab3 ")
	fmt.Println("\t Tab4 ")
	fmt.Println("End Tab ")

	log.Println("Test log Print")
	log.Println("\t log Tab1")
	log.Println("\t log Tab2")
	log.Println("\t log Tab3 ")
	log.Println("\t log Tab4 ")
	log.Println("End log Tab ")

	fmt.Fprintln(os.Stderr, "Test err Print")
	fmt.Fprintln(os.Stderr, "\t err Tab1")
	fmt.Fprintln(os.Stderr, "\t err Tab2")
	fmt.Fprintln(os.Stderr, "\t err Tab3 ")
	fmt.Fprintln(os.Stderr, "\t err Tab4 ")
	fmt.Fprintln(os.Stderr, "End err Tab ")

	time.Sleep(1 * time.Second)

	lineLog := ps.LogFlush()
	fmt.Fprintln(origin, "sender.flush logs = ", lineLog)
	assert.NotNil(t, lineLog)

	time.Sleep(1 * time.Second)

	os.Stderr = origin
	ps.Shutdown()
}
