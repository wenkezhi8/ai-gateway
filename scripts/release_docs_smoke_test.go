package scripts_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestREADME_DescribesEditionBaselinesAndConditionalRedisDependency(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatalf("read README.md failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"Local example config default: `standard`",
		"Production template default: `basic`",
		"Basic edition does not require Redis by default",
		"Redis Stack is required when `vector_cache.enabled=true`",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("README.md must contain %q", needle)
		}
	}
}

func TestReleaseAcceptanceScript_ProvidesOptionalRuntimeSmokeGate(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-acceptance.sh"))
	if err != nil {
		t.Fatalf("read release-acceptance.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"--runtime-smoke-url",
		"--runtime-smoke-metrics-url",
		"--skip-runtime-smoke",
		"--runtime-smoke-allowed-origin",
		"--runtime-smoke-blocked-origin",
		"release-smoke.sh",
		"gate 5/5: runtime smoke",
		"--metrics-url",
		"--allowed-origin",
		"--blocked-origin",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-acceptance.sh must contain %q", needle)
		}
	}
}

func TestReleaseSmokeScript_CoversReleaseRuntimeChecks(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-smoke.sh"))
	if err != nil {
		t.Fatalf("read release-smoke.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"/health",
		"/ready",
		"/docs",
		"/docs/",
		"/swagger",
		"/swagger/",
		"/swagger/index.html",
		"docs center trailing slash",
		"swagger root redirect",
		"swagger trailing slash redirect",
		"Location: /swagger/index.html",
		"<!doctype html",
		"/trace",
		"/tmp/ai-gateway.log",
		"127.0.0.1:9090/metrics",
		"Metrics (localhost only)",
		"$BASE_URL/metrics",
		"metrics on gateway port",
		"--allowed-origin",
		"--blocked-origin",
		"cors allowed origin",
		"cors blocked origin",
		"Cache backend is memory",
		"Connected to Redis",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-smoke.sh must contain %q", needle)
		}
	}
}

func TestReleaseSmokeScript_HealthCurlShouldNotAbortOnConnectionError(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-smoke.sh"))
	if err != nil {
		t.Fatalf("read release-smoke.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"curl_status() {",
		"local url=\"$1\"",
		"curl -s -o /tmp/ai-gateway-smoke-body.txt -w \"%{http_code}\" \"$url\" || true",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-smoke.sh must contain %q", needle)
		}
	}
}

func TestReleaseSmokeScript_ConnectionFailureMessageShouldBeDeterministic(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-smoke.sh"))
	if err != nil {
		t.Fatalf("read release-smoke.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		": > /tmp/ai-gateway-smoke-body.txt",
		"if [ \"$code\" = \"000\" ]; then",
		"connection failed url=$url",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-smoke.sh must contain %q", needle)
		}
	}
}

func TestReleaseSmokeScript_ClosedEndpointCheckShouldFailOnConnectionError(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-smoke.sh"))
	if err != nil {
		t.Fatalf("read release-smoke.sh failed: %v", err)
	}
	text := string(content)

	start := strings.Index(text, "expect_not_http_200() {")
	if start == -1 {
		t.Fatal("release-smoke.sh must define expect_not_http_200")
	}
	end := strings.Index(text[start:], "\n}\n")
	if end == -1 {
		t.Fatal("release-smoke.sh expect_not_http_200 function must end with }")
	}
	functionBody := text[start : start+end]

	checks := []string{
		"code=\"$(curl_status \"$url\")\"",
		"if [ \"$code\" = \"000\" ]; then",
		"[release-smoke] FAIL: $name connection failed url=$url",
	}
	for _, needle := range checks {
		if !strings.Contains(functionBody, needle) {
			t.Fatalf("expect_not_http_200 must contain %q", needle)
		}
	}
}

func TestOpsDocsAndVerifyScript_DescribeLocalOnlyMetricsPolicy(t *testing.T) {
	root := projectRoot(t)

	verifyContent, err := os.ReadFile(filepath.Join(root, "scripts", "verify_all.sh"))
	if err != nil {
		t.Fatalf("read verify_all.sh failed: %v", err)
	}
	verifyText := string(verifyContent)
	verifyChecks := []string{
		"127.0.0.1:9090/metrics",
		"Metrics (localhost only)",
	}
	for _, needle := range verifyChecks {
		if !strings.Contains(verifyText, needle) {
			t.Fatalf("verify_all.sh must contain %q", needle)
		}
	}

	envContent, err := os.ReadFile(filepath.Join(root, "ENV-CONFIGURATION.md"))
	if err != nil {
		t.Fatalf("read ENV-CONFIGURATION.md failed: %v", err)
	}
	envText := string(envContent)
	envChecks := []string{
		"`METRICS_HOST`",
		"127.0.0.1",
		"仅监听本机",
		"`CORS_ALLOW_ORIGINS`",
		"CORS 白名单",
	}
	for _, needle := range envChecks {
		if !strings.Contains(envText, needle) {
			t.Fatalf("ENV-CONFIGURATION.md must contain %q", needle)
		}
	}
}
