package scripts_test

import (
	"os"
	"os/exec"
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
		"/swagger/doc.json",
		"docs center trailing slash",
		"swagger root redirect",
		"swagger trailing slash redirect",
		"swagger index page",
		"swagger doc json",
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

func TestReleaseSmokeScript_UsesConsistentCheckProgressLabels(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-smoke.sh"))
	if err != nil {
		t.Fatalf("read release-smoke.sh failed: %v", err)
	}
	text := string(content)

	expectedChecks := []string{
		"check 1/13: health",
		"check 2/13: ready",
		"check 3/13: docs center",
		"check 4/13: docs center trailing slash",
		"check 5/13: swagger root redirect",
		"check 6/13: swagger index page",
		"check 7/13: swagger doc json",
		"check 8/13: trace page asset",
		"check 9/13: debug endpoints closed",
		"check 10/13: metrics on gateway port closed",
		"check 11/13: metrics localhost only",
		"check 12/13: cache backend hint",
		"check 13/13: cors whitelist (optional)",
	}
	for _, checkLabel := range expectedChecks {
		if !strings.Contains(text, checkLabel) {
			t.Fatalf("release-smoke.sh must contain %q", checkLabel)
		}
	}

	if strings.Contains(text, "check 1/11:") || strings.Contains(text, "check 2/11:") ||
		strings.Contains(text, "check 3/11:") || strings.Contains(text, "check 4/11:") ||
		strings.Contains(text, "check 5/11:") {
		t.Fatal("release-smoke.sh should not mix legacy x/11 progress labels with x/13 labels")
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
		"CURL_ARGS=(",
		"-o \"$SMOKE_BODY_FILE\"",
		"-w \"%{http_code}\"",
		"\"$url\" || true",
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
		": > \"$SMOKE_BODY_FILE\"",
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

func TestReleaseSmokeScript_UsesSharedSPAShellAssertionAndTempFiles(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-smoke.sh"))
	if err != nil {
		t.Fatalf("read release-smoke.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"SMOKE_BODY_FILE=",
		"SMOKE_HEADER_FILE=",
		"cleanup_smoke_files() {",
		"trap cleanup_smoke_files EXIT",
		"assert_spa_shell() {",
		"assert_spa_shell \"docs center\"",
		"assert_spa_shell \"docs center trailing slash\"",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-smoke.sh must contain %q", needle)
		}
	}
	if strings.Contains(text, ": > /tmp/ai-gateway-smoke-body.txt") {
		t.Fatal("release-smoke.sh should no longer hardcode /tmp body temp file directly")
	}
}

func TestReleaseSmokeScript_EnforcesLocalOnlyMetricsURLInput(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-smoke.sh"))
	if err != nil {
		t.Fatalf("read release-smoke.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"validate_local_metrics_url() {",
		"metrics url must target localhost/127.0.0.1/::1",
		"validate_local_metrics_url \"$METRICS_URL\"",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-smoke.sh must contain %q", needle)
		}
	}
}

func TestReleaseSmokeScript_RequiresCorsOriginsProvidedInPairs(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-smoke.sh"))
	if err != nil {
		t.Fatalf("read release-smoke.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"if [ -n \"$ALLOWED_ORIGIN\" ] && [ -z \"$BLOCKED_ORIGIN\" ]; then",
		"if [ -n \"$BLOCKED_ORIGIN\" ] && [ -z \"$ALLOWED_ORIGIN\" ]; then",
		"allowed-origin and blocked-origin must be provided together",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-smoke.sh must contain %q", needle)
		}
	}
}

func TestReleaseSmokeScript_CorsWhitelistChecksPreflightSemantics(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-smoke.sh"))
	if err != nil {
		t.Fatalf("read release-smoke.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"cors allowed preflight",
		"cors blocked preflight",
		"-X OPTIONS",
		"-H \"Access-Control-Request-Method: POST\"",
		"cors allowed preflight check failed",
		"cors blocked preflight should be 403",
		"cors allowed origin vary check failed",
		"cors allowed preflight vary check failed",
		"cors blocked origin vary check failed",
		"cors blocked preflight vary check failed",
		"Vary:",
		"Origin",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-smoke.sh must contain %q", needle)
		}
	}
}

func TestReleaseSmokeScript_DocsCenterShouldNotExposeRedirectHeaders(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-smoke.sh"))
	if err != nil {
		t.Fatalf("read release-smoke.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"docs center should not include redirect location header",
		"docs center trailing slash should not include redirect location header",
		"docsLocationLine=",
		"docsSlashLocationLine=",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-smoke.sh must contain %q", needle)
		}
	}
}

func TestReleaseSmokeScript_SwaggerIndexShouldExposeSwaggerUIMarker(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-smoke.sh"))
	if err != nil {
		t.Fatalf("read release-smoke.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"swagger index should expose swagger ui marker",
		"assert_swagger_ui_shell",
		"SwaggerUIBundle",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-smoke.sh must contain %q", needle)
		}
	}
}

func TestReleaseSmokeScript_AllowsOptionalLimitedNetworkSkip(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-smoke.sh"))
	if err != nil {
		t.Fatalf("read release-smoke.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"--allow-limited-network-skip",
		"ALLOW_LIMITED_NETWORK_SKIP=false",
		"contains_limited_network_marker() {",
		"Operation not permitted",
		"SKIP: limited network environment detected",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-smoke.sh must contain %q", needle)
		}
	}
}

func TestReleaseAcceptanceScript_PrefightsRuntimeSmokeConnectivity(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-acceptance.sh"))
	if err != nil {
		t.Fatalf("read release-acceptance.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"preflight_runtime_smoke_connectivity() {",
		"SKIP: runtime smoke connectivity preflight detected limited network environment",
		"curl command is unavailable",
		"Could not resolve host",
		"Network is unreachable",
		"Connection timed out",
		"Operation not permitted",
		"Failed to connect to",
		"gate 5/5: runtime smoke skipped by connectivity preflight",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-acceptance.sh must contain %q", needle)
		}
	}
}

func TestReleaseAcceptanceScript_ExposesLimitedNetworkSkipToggleForRuntimeSmoke(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-acceptance.sh"))
	if err != nil {
		t.Fatalf("read release-acceptance.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"--runtime-smoke-allow-limited-network-skip",
		"--allow-limited-network-skip",
		"RUNTIME_SMOKE_ALLOW_LIMITED_NETWORK_SKIP=false",
		"RUNTIME_SMOKE_ARGS+=(--allow-limited-network-skip)",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-acceptance.sh must contain %q", needle)
		}
	}
}

func TestReleaseAcceptanceScript_ProvidesRuntimeSmokeVaryOriginGateToggle(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-acceptance.sh"))
	if err != nil {
		t.Fatalf("read release-acceptance.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"--runtime-smoke-require-vary-origin",
		"--runtime-smoke-no-require-vary-origin",
		"RUNTIME_SMOKE_REQUIRE_VARY_ORIGIN=true",
		"RUNTIME_SMOKE_ARGS+=(--require-vary-origin)",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-acceptance.sh must contain %q", needle)
		}
	}
}

func TestReleaseAcceptanceScript_ProvidesSpawnGatewayPortAndConfigPassThrough(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-acceptance.sh"))
	if err != nil {
		t.Fatalf("read release-acceptance.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"--spawn-gateway-port",
		"--spawn-gateway-config",
		"SPAWN_GATEWAY_PORT=",
		"SPAWN_GATEWAY_CONFIG=",
		"--port",
		"--config",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-acceptance.sh must contain %q", needle)
		}
	}
}

func TestReleaseAcceptanceScript_RequiresCorsRuntimeSmokeArgsWhenWhitelistEnabled(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-acceptance.sh"))
	if err != nil {
		t.Fatalf("read release-acceptance.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"CORS_ALLOW_ORIGINS",
		"runtime smoke CORS whitelist is enabled",
		"runtime-smoke-allowed-origin and --runtime-smoke-blocked-origin are required together",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-acceptance.sh must contain %q", needle)
		}
	}
}

func TestReleaseAcceptanceScript_WildcardWhitelistDoesNotRequireCorsRuntimeSmokeArgs(t *testing.T) {
	root := projectRoot(t)
	cmd := exec.Command(
		"bash",
		filepath.Join(root, "scripts", "release-acceptance.sh"),
		"--dry-run",
		"--skip-backend",
		"--skip-frontend",
		"--skip-delivery-status",
	)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CORS_ALLOW_ORIGINS=https://console.example.com,*")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("release-acceptance should allow wildcard whitelist without runtime smoke origin pair, err=%v, out=%s", err, out)
	}
	if !strings.Contains(string(out), "[release-acceptance] completed") {
		t.Fatalf("unexpected release-acceptance output: %s", out)
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
		"空字符串或 `*` 表示允许全部",
		"仅包含空白或逗号的无效白名单会拒绝跨域请求",
		"CORS_ALLOW_ORIGINS=https://console.example.com,https://ops.example.com",
		"CORS_ALLOW_ORIGINS= ,   ,",
	}
	for _, needle := range envChecks {
		if !strings.Contains(envText, needle) {
			t.Fatalf("ENV-CONFIGURATION.md must contain %q", needle)
		}
	}
}

func TestReleaseSmokeScript_UsesArrayDrivenCheckMetadataAndHelpers(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-smoke.sh"))
	if err != nil {
		t.Fatalf("read release-smoke.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"CHECK_TITLES=(",
		"CHECK_OPTIONAL_FLAGS=(",
		"check_title() {",
		"is_optional_check() {",
		"log_check \"$check_index\"",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-smoke.sh must contain %q", needle)
		}
	}
}

func TestReleaseSmokeScript_SupportsSwaggerDocJsonSampleLimit(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-smoke.sh"))
	if err != nil {
		t.Fatalf("read release-smoke.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"MAX_JSON_SAMPLE_BYTES=",
		"--swagger-json-max-bytes",
		"head -c \"$MAX_JSON_SAMPLE_BYTES\"",
		"swagger doc json sample exceeds limit",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-smoke.sh must contain %q", needle)
		}
	}
}

func TestReleaseSmokeScript_SupportsRequireVaryOriginToggleAndCorsHelpers(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-smoke.sh"))
	if err != nil {
		t.Fatalf("read release-smoke.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"--require-vary-origin",
		"--no-require-vary-origin",
		"REQUIRE_VARY_ORIGIN=true",
		"extract_header_value() {",
		"assert_vary_origin_if_required() {",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-smoke.sh must contain %q", needle)
		}
	}
}

func TestReleaseSmokeLocalScript_SupportsSkipRestartFlag(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "scripts", "release-smoke-local.sh"))
	if err != nil {
		t.Fatalf("read release-smoke-local.sh failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"--skip-restart",
		"SKIP_RESTART=false",
		"if [ \"$SKIP_RESTART\" = false ]; then",
		"[release-smoke-local] skip restart",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("release-smoke-local.sh must contain %q", needle)
		}
	}
}

func TestReleaseScripts_UseSharedFeatureBranchRuleLibrary(t *testing.T) {
	root := projectRoot(t)
	libContent, err := os.ReadFile(filepath.Join(root, "scripts", "lib", "git-branch.sh"))
	if err != nil {
		t.Fatalf("read scripts/lib/git-branch.sh failed: %v", err)
	}
	libText := string(libContent)
	if !strings.Contains(libText, "codex/feature/") || !strings.Contains(libText, "git_require_feature_branch()") {
		t.Fatalf("git-branch.sh must define codex/feature/ rule and git_require_feature_branch helper")
	}

	acceptanceContent, err := os.ReadFile(filepath.Join(root, "scripts", "release-acceptance.sh"))
	if err != nil {
		t.Fatalf("read release-acceptance.sh failed: %v", err)
	}
	if !strings.Contains(string(acceptanceContent), "source \"$SCRIPT_DIR/lib/git-branch.sh\"") {
		t.Fatal("release-acceptance.sh must source scripts/lib/git-branch.sh")
	}

	startBranchContent, err := os.ReadFile(filepath.Join(root, "scripts", "start-feature-branch.sh"))
	if err != nil {
		t.Fatalf("read start-feature-branch.sh failed: %v", err)
	}
	if !strings.Contains(string(startBranchContent), "source \"$SCRIPT_DIR/lib/git-branch.sh\"") {
		t.Fatal("start-feature-branch.sh must source scripts/lib/git-branch.sh")
	}
}

func TestReleaseChecklistDocument_CoversDocsSwaggerCorsMetricsRiskPoints(t *testing.T) {
	root := projectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "docs", "release", "RELEASE_CHECKLIST.md"))
	if err != nil {
		t.Fatalf("read docs/release/RELEASE_CHECKLIST.md failed: %v", err)
	}
	text := string(content)

	checks := []string{
		"/docs",
		"/swagger",
		"CORS_ALLOW_ORIGINS",
		"metrics",
		"release-smoke.sh",
		"release-acceptance.sh",
	}
	for _, needle := range checks {
		if !strings.Contains(text, needle) {
			t.Fatalf("RELEASE_CHECKLIST.md must contain %q", needle)
		}
	}
}
