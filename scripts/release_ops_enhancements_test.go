package scripts_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseAcceptanceScript_ProvidesSpawnGatewayAndCorsFromEnvFlags(t *testing.T) {
	t.Parallel()
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-acceptance.sh"))
	if err != nil {
		t.Fatalf("read release-acceptance.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"--spawn-gateway",
		"--runtime-smoke-cors-from-env",
		"--runtime-smoke-swagger-json-url",
		"--runtime-smoke-cors-blocked-origin",
		"dev-restart.sh",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-acceptance.sh must contain %q", needle)
		}
	}
}

func TestReleaseAcceptanceScript_ValidatesFeatureBranchName(t *testing.T) {
	t.Parallel()
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-acceptance.sh"))
	if err != nil {
		t.Fatalf("read release-acceptance.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"source \"$SCRIPT_DIR/lib/git-branch.sh\"",
		"git_require_feature_branch",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-acceptance.sh must contain %q", needle)
		}
	}
}

func TestReleaseSmokeScript_UsesTotalChecksAndJsonHelper(t *testing.T) {
	t.Parallel()
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-smoke.sh"))
	if err != nil {
		t.Fatalf("read release-smoke.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"TOTAL_CHECKS=13",
		"log_check() {",
		"expect_json_200() {",
		"SWAGGER_DOC_JSON_URL=",
		"--swagger-json-url",
		"content-type",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-smoke.sh must contain %q", needle)
		}
	}
}

func TestReleaseSmokeScript_HasFailureClassificationAndRetry(t *testing.T) {
	t.Parallel()
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-smoke.sh"))
	if err != nil {
		t.Fatalf("read release-smoke.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"classify_curl_failure() {",
		"Operation not permitted",
		"connection_refused",
		"policy_blocked",
		"business_failure",
		"failure_detail() {",
		"detail=\"$(failure_detail \"$curl_err\" \"$failure_kind\")\"",
		"curl_status_with_retry() {",
		"retrying on connection failure",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-smoke.sh must contain %q", needle)
		}
	}
}

func TestDevRestartScript_ProvidesSkipWebBuildAndPostStartProbe(t *testing.T) {
	t.Parallel()
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "dev-restart.sh"))
	if err != nil {
		t.Fatalf("read dev-restart.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"--skip-web-build",
		"SKIP_WEB_BUILD",
		"POST_START_PROBE_DELAY_SECONDS=5",
		"二次探测",
		"建议验收命令",
		"http://localhost:$SERVER_PORT/health",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("dev-restart.sh must contain %q", needle)
		}
	}
}

func TestReleaseSmokeLocalScript_WiresRestartAndSmoke(t *testing.T) {
	t.Parallel()
	root := projectRoot(t)
	path := filepath.Join(root, "scripts", "release-smoke-local.sh")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read release-smoke-local.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"./scripts/dev-restart.sh",
		"./scripts/release-smoke.sh",
		"tail -n",
		"/tmp/ai-gateway.log",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-smoke-local.sh must contain %q", needle)
		}
	}
}

func TestMakefile_ProvidesTestSafeTarget(t *testing.T) {
	t.Parallel()
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"test-safe:",
		"Run tests safe for sandbox/limited-port environments",
		"./scripts -count=1",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("Makefile must contain %q", needle)
		}
	}
}

func TestStartFeatureBranchScript_EnforcesCodexFeaturePrefix(t *testing.T) {
	t.Parallel()
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "start-feature-branch.sh"))
	if err != nil {
		t.Fatalf("read start-feature-branch.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"source \"$SCRIPT_DIR/lib/git-branch.sh\"",
		"git_make_feature_branch_name",
		"codex/feature/",
		"git checkout -b",
		"origin/main",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("start-feature-branch.sh must contain %q", needle)
		}
	}
}

func TestReleaseSmokeScript_CheckNumberingIsContinuous(t *testing.T) {
	t.Parallel()
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-smoke.sh"))
	if err != nil {
		t.Fatalf("read release-smoke.sh failed: %v", err)
	}
	text := string(content)

	if !strings.Contains(text, "CHECK_TITLES=(") || !strings.Contains(text, "CHECK_HANDLERS=(") {
		t.Fatal("release-smoke.sh must define array-driven check metadata")
	}
	for i := 1; i <= 13; i++ {
		needle := fmt.Sprintf("check %d/13:", i)
		if !strings.Contains(text, needle) {
			t.Fatalf("release-smoke.sh must contain %q", needle)
		}
	}
}
