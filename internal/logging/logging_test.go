package logging

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
	"time"
)

func Test_CustomFormatter(t *testing.T) {
	entry := &logrus.Entry{
		Logger:  logrus.New(),
		Time:    time.Now(),
		Message: "test message",
		Level:   logrus.InfoLevel,
	}
	customFormatter := &CustomFormatter{
		CustomFormatter: &logrus.TextFormatter{},
		TraceID:         "trace123",
	}

	bytes, err := customFormatter.Format(entry)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	formatted := string(bytes)
	// Now check if formatted string contains all expected components
	// like file, line number, TraceID, and original message.
	// For example:
	assert.Contains(t, formatted, "trace123")
}

func Test_GenerateTraceID(t *testing.T) {
	// Test for length
	traceID := GenerateTraceID()
	if len(traceID) != 12 { // 6 bytes = 12 hexadecimal characters
		t.Errorf("Expected length 12, got %d", len(traceID))
	}

	// Test for uniqueness
	traceIDSet := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		traceID := GenerateTraceID()
		if traceIDSet[traceID] {
			t.Errorf("Duplicate trace ID generated: %s", traceID)
		}
		traceIDSet[traceID] = true
	}

	// Test for valid hexadecimal characters
	matched, err := regexp.MatchString("^[a-fA-F0-9]+$", traceID)
	if err != nil || !matched {
		t.Errorf("Invalid characters in trace ID: %s", traceID)
	}
}
