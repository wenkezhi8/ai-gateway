package bootstrap

import (
	"os"
	"testing"
)

func TestMetricsListenAddr_DefaultsToLocalhost(t *testing.T) {
	prevPort, hadPort := os.LookupEnv("METRICS_PORT")
	prevHost, hadHost := os.LookupEnv("METRICS_HOST")
	_ = os.Unsetenv("METRICS_PORT")
	_ = os.Unsetenv("METRICS_HOST")
	defer func() {
		if hadPort {
			_ = os.Setenv("METRICS_PORT", prevPort)
		} else {
			_ = os.Unsetenv("METRICS_PORT")
		}
		if hadHost {
			_ = os.Setenv("METRICS_HOST", prevHost)
		} else {
			_ = os.Unsetenv("METRICS_HOST")
		}
	}()

	if got := metricsListenAddr(); got != "127.0.0.1:9090" {
		t.Fatalf("metrics listen addr = %q, want %q", got, "127.0.0.1:9090")
	}
}

func TestMetricsListenAddr_UsesConfiguredHostAndPort(t *testing.T) {
	prevPort, hadPort := os.LookupEnv("METRICS_PORT")
	prevHost, hadHost := os.LookupEnv("METRICS_HOST")
	_ = os.Setenv("METRICS_PORT", "9191")
	_ = os.Setenv("METRICS_HOST", "0.0.0.0")
	defer func() {
		if hadPort {
			_ = os.Setenv("METRICS_PORT", prevPort)
		} else {
			_ = os.Unsetenv("METRICS_PORT")
		}
		if hadHost {
			_ = os.Setenv("METRICS_HOST", prevHost)
		} else {
			_ = os.Unsetenv("METRICS_HOST")
		}
	}()

	if got := metricsListenAddr(); got != "0.0.0.0:9191" {
		t.Fatalf("metrics listen addr = %q, want %q", got, "0.0.0.0:9191")
	}
}
