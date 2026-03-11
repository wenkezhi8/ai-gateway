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

func TestMetricsListenAddr_RejectsNonLocalHostAndKeepsConfiguredPort(t *testing.T) {
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

	if got := metricsListenAddr(); got != "127.0.0.1:9191" {
		t.Fatalf("metrics listen addr = %q, want %q", got, "127.0.0.1:9191")
	}
}

func TestMetricsListenAddr_AllowsExplicitLocalhostHosts(t *testing.T) {
	prevPort, hadPort := os.LookupEnv("METRICS_PORT")
	prevHost, hadHost := os.LookupEnv("METRICS_HOST")
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

	cases := []struct {
		name string
		host string
		want string
	}{
		{name: "localhost", host: "localhost", want: "localhost:9090"},
		{name: "ipv4 loopback", host: "127.0.0.1", want: "127.0.0.1:9090"},
		{name: "ipv6 loopback", host: "::1", want: "[::1]:9090"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Setenv("METRICS_PORT", "9090")
			_ = os.Setenv("METRICS_HOST", tc.host)
			if got := metricsListenAddr(); got != tc.want {
				t.Fatalf("metrics listen addr = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMetricsListenAddr_TrimSpaceAndFallbackDefaults(t *testing.T) {
	prevPort, hadPort := os.LookupEnv("METRICS_PORT")
	prevHost, hadHost := os.LookupEnv("METRICS_HOST")
	_ = os.Setenv("METRICS_PORT", "   ")
	_ = os.Setenv("METRICS_HOST", "   ")
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
