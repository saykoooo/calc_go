package application

import (
	"os"
	"testing"
	"time"
)

func TestConfigFromEnv(t *testing.T) {
	os.Setenv("TIME_ADDITION_MS", "500")
	os.Setenv("PORT", "9090")
	defer os.Clearenv()

	config := ConfigFromEnv()

	if config.Addr != "9090" {
		t.Errorf("Expected Addr '9090', got '%s'", config.Addr)
	}

	if config.TimeAddition != 500*time.Millisecond {
		t.Errorf("Expected TimeAddition 500ms, got %v", config.TimeAddition)
	}

	if config.TimeSubtraction != 1000*time.Millisecond {
		t.Errorf("Expected TimeSubtraction 1000ms, got %v", config.TimeSubtraction)
	}
}

func TestGetEnvDuration(t *testing.T) {
	os.Setenv("TEST_DURATION", "invalid")
	defer os.Unsetenv("TEST_DURATION")

	duration := getEnvDuration("TEST_DURATION", 300)
	if duration != 300*time.Millisecond {
		t.Errorf("Expected duration 300ms, got %v", duration)
	}
}
