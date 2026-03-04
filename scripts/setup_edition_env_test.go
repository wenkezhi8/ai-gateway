package scripts_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func projectRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
}

func writeExecutable(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write %s failed: %v", path, err)
	}
}

func runScript(t *testing.T, scriptPath string, env []string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("bash", append([]string{scriptPath}, args...)...)
	cmd.Env = env
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	return output.String(), err
}

func TestRedisContainerNames_AreUnifiedAcrossSetupScripts(t *testing.T) {
	root := projectRoot(t)

	containerNamesPath := filepath.Join(root, "scripts", "lib", "container-names.sh")
	containerNames, err := os.ReadFile(containerNamesPath)
	if err != nil {
		t.Fatalf("read %s failed: %v", containerNamesPath, err)
	}
	if !strings.Contains(string(containerNames), `REDIS_CONTAINER="ai-gateway-redis-stack"`) {
		t.Fatalf("REDIS_CONTAINER constant missing or incorrect")
	}
	if !strings.Contains(string(containerNames), `REDIS_CONTAINER_PROD="ai-gateway-redis-stack-prod"`) {
		t.Fatalf("REDIS_CONTAINER_PROD constant missing or incorrect")
	}

	devRestartPath := filepath.Join(root, "scripts", "dev-restart.sh")
	devRestart, err := os.ReadFile(devRestartPath)
	if err != nil {
		t.Fatalf("read %s failed: %v", devRestartPath, err)
	}
	if !strings.Contains(string(devRestart), "container-names.sh") {
		t.Fatalf("dev-restart.sh must source shared container names")
	}
	if strings.Contains(string(devRestart), "--name redis-stack") {
		t.Fatalf("dev-restart.sh should not hardcode legacy redis-stack container name")
	}

	setupPath := filepath.Join(root, "scripts", "setup-edition-env.sh")
	setupScript, err := os.ReadFile(setupPath)
	if err != nil {
		t.Fatalf("read %s failed: %v", setupPath, err)
	}
	if !strings.Contains(string(setupScript), "container-names.sh") {
		t.Fatalf("setup-edition-env.sh must source shared container names")
	}
	if strings.Contains(string(setupScript), "REDIS_CONTAINER=\"ai-gateway-redis-stack\"") {
		t.Fatalf("setup-edition-env.sh should read REDIS_CONTAINER from shared names file")
	}

	upgradePath := filepath.Join(root, "scripts", "upgrade.sh")
	upgradeScript, err := os.ReadFile(upgradePath)
	if err != nil {
		t.Fatalf("read %s failed: %v", upgradePath, err)
	}
	if !strings.Contains(string(upgradeScript), "container-names.sh") {
		t.Fatalf("upgrade.sh must source shared container names")
	}
	if !strings.Contains(string(upgradeScript), `docker exec "$REDIS_CONTAINER" redis-cli ping`) {
		t.Fatalf("upgrade.sh should verify redis with unified REDIS_CONTAINER variable")
	}

	composePath := filepath.Join(root, "docker-compose.yml")
	composeContent, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatalf("read %s failed: %v", composePath, err)
	}
	if !strings.Contains(string(composeContent), "container_name: ai-gateway-redis-stack") {
		t.Fatalf("docker-compose.yml must use unified redis container name")
	}

	prodComposePath := filepath.Join(root, "deploy", "docker-compose.prod.yml")
	prodComposeContent, err := os.ReadFile(prodComposePath)
	if err != nil {
		t.Fatalf("read %s failed: %v", prodComposePath, err)
	}
	if !strings.Contains(string(prodComposeContent), "container_name: ai-gateway-redis-stack-prod") {
		t.Fatalf("deploy/docker-compose.prod.yml must use unified prod redis container name")
	}

	legacyComposePath := filepath.Join(root, "deploy", "docker", "docker-compose.yml")
	legacyComposeContent, err := os.ReadFile(legacyComposePath)
	if err != nil {
		t.Fatalf("read %s failed: %v", legacyComposePath, err)
	}
	if !strings.Contains(string(legacyComposeContent), "container_name: ai-gateway-redis-stack") {
		t.Fatalf("deploy/docker/docker-compose.yml must use unified redis container name")
	}
}

func TestSetupEditionEnv_NativeRedis_NoDockerAutoFallback(t *testing.T) {
	root := projectRoot(t)
	setupPath := filepath.Join(root, "scripts", "setup-edition-env.sh")

	tempDir := t.TempDir()
	tempBin := filepath.Join(tempDir, "bin")
	if err := os.MkdirAll(tempBin, 0o755); err != nil {
		t.Fatalf("mkdir temp bin failed: %v", err)
	}

	dockerLogPath := filepath.Join(tempDir, "docker.log")
	if err := os.WriteFile(dockerLogPath, nil, 0o644); err != nil {
		t.Fatalf("create docker log failed: %v", err)
	}

	writeExecutable(t, filepath.Join(tempBin, "docker"), `#!/bin/bash
set -eu
: "${DOCKER_LOG_FILE:?}"
printf '%s\n' "$*" >>"$DOCKER_LOG_FILE"
exit 0
`)

	writeExecutable(t, filepath.Join(tempBin, "redis-cli"), `#!/bin/bash
exit 1
`)

	configPath := filepath.Join(tempDir, "config.json")
	env := append(os.Environ(),
		"PATH="+tempBin+":"+os.Getenv("PATH"),
		"DOCKER_LOG_FILE="+dockerLogPath,
	)

	output, err := runScript(
		t,
		setupPath,
		env,
		"--edition", "basic",
		"--runtime", "native",
		"--apply-config", "false",
		"--pull-embedding-model", "false",
		"--config-path", configPath,
	)
	if err == nil {
		t.Fatalf("expected native setup to fail when redis native is unavailable")
	}
	if !strings.Contains(output, "docker fallback disabled") {
		t.Fatalf("expected native failure guidance to mention docker fallback disabled, output=%s", output)
	}
	if !strings.Contains(output, "docker run -d --name ai-gateway-redis-stack") {
		t.Fatalf("expected native failure guidance to include manual docker command, output=%s", output)
	}

	dockerCalls, readErr := os.ReadFile(dockerLogPath)
	if readErr != nil {
		t.Fatalf("read docker log failed: %v", readErr)
	}
	if strings.TrimSpace(string(dockerCalls)) != "" {
		t.Fatalf("docker should not be called in native no-fallback path, calls=%s", string(dockerCalls))
	}
}

func TestSetupEditionEnv_DockerRedis_ConflictGateFailsFast(t *testing.T) {
	root := projectRoot(t)
	setupPath := filepath.Join(root, "scripts", "setup-edition-env.sh")

	tempDir := t.TempDir()
	tempBin := filepath.Join(tempDir, "bin")
	if err := os.MkdirAll(tempBin, 0o755); err != nil {
		t.Fatalf("mkdir temp bin failed: %v", err)
	}

	dockerLogPath := filepath.Join(tempDir, "docker.log")
	if err := os.WriteFile(dockerLogPath, nil, 0o644); err != nil {
		t.Fatalf("create docker log failed: %v", err)
	}

	writeExecutable(t, filepath.Join(tempBin, "docker"), `#!/bin/bash
set -eu
: "${DOCKER_LOG_FILE:?}"
printf '%s\n' "$*" >>"$DOCKER_LOG_FILE"
if [[ "$1" == "info" ]]; then
  exit 0
fi
if [[ "$1" == "ps" && "$2" == "--format" ]]; then
  printf '%b' "${DOCKER_PS_NAMES:-}"
  exit 0
fi
if [[ "$1" == "ps" && "$2" == "-a" && "$3" == "--format" ]]; then
  printf '%b' "${DOCKER_PSA_NAMES:-}"
  exit 0
fi
exit 0
`)

	writeExecutable(t, filepath.Join(tempBin, "redis-cli"), `#!/bin/bash
exit 0
`)

	configPath := filepath.Join(tempDir, "config.json")
	env := append(os.Environ(),
		"PATH="+tempBin+":"+os.Getenv("PATH"),
		"DOCKER_LOG_FILE="+dockerLogPath,
		"DOCKER_PS_NAMES=ai-gateway-redis-stack\nredis-stack\n",
		"DOCKER_PSA_NAMES=ai-gateway-redis-stack\nredis-stack\n",
	)

	output, err := runScript(
		t,
		setupPath,
		env,
		"--edition", "basic",
		"--runtime", "docker",
		"--apply-config", "false",
		"--pull-embedding-model", "false",
		"--config-path", configPath,
	)
	if err == nil {
		t.Fatalf("expected docker setup to fail when legacy and unified redis containers coexist")
	}
	if !strings.Contains(output, "conflicting redis containers") {
		t.Fatalf("expected conflict guidance, output=%s", output)
	}

	dockerCalls, readErr := os.ReadFile(dockerLogPath)
	if readErr != nil {
		t.Fatalf("read docker log failed: %v", readErr)
	}
	if strings.Contains(string(dockerCalls), "rm -f") {
		t.Fatalf("conflict gate must not auto-delete containers, calls=%s", string(dockerCalls))
	}
}

func TestSetupEditionEnv_NativeOllama_NoDockerAutoFallback(t *testing.T) {
	root := projectRoot(t)
	setupPath := filepath.Join(root, "scripts", "setup-edition-env.sh")

	tempDir := t.TempDir()
	tempBin := filepath.Join(tempDir, "bin")
	if err := os.MkdirAll(tempBin, 0o755); err != nil {
		t.Fatalf("mkdir temp bin failed: %v", err)
	}

	dockerLogPath := filepath.Join(tempDir, "docker.log")
	if err := os.WriteFile(dockerLogPath, nil, 0o644); err != nil {
		t.Fatalf("create docker log failed: %v", err)
	}

	writeExecutable(t, filepath.Join(tempBin, "docker"), `#!/bin/bash
set -eu
: "${DOCKER_LOG_FILE:?}"
printf '%s\n' "$*" >>"$DOCKER_LOG_FILE"
exit 0
`)

	writeExecutable(t, filepath.Join(tempBin, "redis-cli"), `#!/bin/bash
exit 0
`)

	configPath := filepath.Join(tempDir, "config.json")
	env := append(os.Environ(),
		"PATH="+tempBin+":/usr/bin:/bin",
		"DOCKER_LOG_FILE="+dockerLogPath,
	)

	output, err := runScript(
		t,
		setupPath,
		env,
		"--edition", "standard",
		"--runtime", "native",
		"--apply-config", "false",
		"--pull-embedding-model", "false",
		"--config-path", configPath,
	)
	if err == nil {
		t.Fatalf("expected native setup to fail when ollama binary is unavailable")
	}
	if !strings.Contains(output, "ollama binary not found") {
		t.Fatalf("expected native ollama failure message, output=%s", output)
	}
	if !strings.Contains(output, "docker fallback disabled") {
		t.Fatalf("expected native failure guidance to mention docker fallback disabled, output=%s", output)
	}
	if !strings.Contains(output, "docker run -d --name ai-gateway-ollama") {
		t.Fatalf("expected native failure guidance to include manual ollama docker command, output=%s", output)
	}

	dockerCalls, readErr := os.ReadFile(dockerLogPath)
	if readErr != nil {
		t.Fatalf("read docker log failed: %v", readErr)
	}
	if strings.TrimSpace(string(dockerCalls)) != "" {
		t.Fatalf("docker should not be called in native no-fallback path, calls=%s", string(dockerCalls))
	}
}

func TestSetupEditionEnv_NativeQdrant_NoDockerAutoFallback(t *testing.T) {
	root := projectRoot(t)
	setupPath := filepath.Join(root, "scripts", "setup-edition-env.sh")

	tempDir := t.TempDir()
	tempBin := filepath.Join(tempDir, "bin")
	if err := os.MkdirAll(tempBin, 0o755); err != nil {
		t.Fatalf("mkdir temp bin failed: %v", err)
	}

	dockerLogPath := filepath.Join(tempDir, "docker.log")
	if err := os.WriteFile(dockerLogPath, nil, 0o644); err != nil {
		t.Fatalf("create docker log failed: %v", err)
	}

	writeExecutable(t, filepath.Join(tempBin, "docker"), `#!/bin/bash
set -eu
: "${DOCKER_LOG_FILE:?}"
printf '%s\n' "$*" >>"$DOCKER_LOG_FILE"
exit 0
`)

	writeExecutable(t, filepath.Join(tempBin, "redis-cli"), `#!/bin/bash
exit 0
`)

	writeExecutable(t, filepath.Join(tempBin, "ollama"), `#!/bin/bash
exit 0
`)

	writeExecutable(t, filepath.Join(tempBin, "curl"), `#!/bin/bash
set -eu
if [[ "$*" == *"11434/api/tags"* ]]; then
  exit 0
fi
if [[ "$*" == *"6333/collections"* ]]; then
  exit 1
fi
exit 1
`)

	configPath := filepath.Join(tempDir, "config.json")
	env := append(os.Environ(),
		"PATH="+tempBin+":/usr/bin:/bin",
		"DOCKER_LOG_FILE="+dockerLogPath,
	)

	output, err := runScript(
		t,
		setupPath,
		env,
		"--edition", "enterprise",
		"--runtime", "native",
		"--apply-config", "false",
		"--pull-embedding-model", "false",
		"--config-path", configPath,
	)
	if err == nil {
		t.Fatalf("expected native setup to fail when qdrant is unavailable")
	}
	if !strings.Contains(output, "qdrant native service unavailable") {
		t.Fatalf("expected native qdrant failure message, output=%s", output)
	}
	if !strings.Contains(output, "docker fallback disabled") {
		t.Fatalf("expected native failure guidance to mention docker fallback disabled, output=%s", output)
	}
	if !strings.Contains(output, "docker run -d --name ai-gateway-qdrant") {
		t.Fatalf("expected native failure guidance to include manual qdrant docker command, output=%s", output)
	}

	dockerCalls, readErr := os.ReadFile(dockerLogPath)
	if readErr != nil {
		t.Fatalf("read docker log failed: %v", readErr)
	}
	if strings.TrimSpace(string(dockerCalls)) != "" {
		t.Fatalf("docker should not be called in native no-fallback path, calls=%s", string(dockerCalls))
	}
}

func TestSetupEditionEnv_ApplyConfig_PersistsRuntimeAndDependencyVersions(t *testing.T) {
	root := projectRoot(t)
	setupPath := filepath.Join(root, "scripts", "setup-edition-env.sh")

	tempDir := t.TempDir()
	tempBin := filepath.Join(tempDir, "bin")
	if err := os.MkdirAll(tempBin, 0o755); err != nil {
		t.Fatalf("mkdir temp bin failed: %v", err)
	}

	writeExecutable(t, filepath.Join(tempBin, "redis-cli"), `#!/bin/bash
exit 0
`)
	writeExecutable(t, filepath.Join(tempBin, "ollama"), `#!/bin/bash
exit 0
`)
	writeExecutable(t, filepath.Join(tempBin, "curl"), `#!/bin/bash
set -eu
if [[ "$*" == *"11434/api/tags"* ]]; then
  exit 0
fi
if [[ "$*" == *"6333/collections"* ]]; then
  exit 0
fi
exit 1
`)

	configPath := filepath.Join(tempDir, "config.json")
	env := append(os.Environ(),
		"PATH="+tempBin+":/usr/bin:/bin",
	)

	output, err := runScript(
		t,
		setupPath,
		env,
		"--edition", "enterprise",
		"--runtime", "native",
		"--apply-config", "true",
		"--pull-embedding-model", "false",
		"--config-path", configPath,
	)
	if err != nil {
		t.Fatalf("expected setup success, err=%v output=%s", err, output)
	}

	raw, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatalf("read config failed: %v", readErr)
	}
	var parsed map[string]any
	if unmarshalErr := json.Unmarshal(raw, &parsed); unmarshalErr != nil {
		t.Fatalf("unmarshal config failed: %v", unmarshalErr)
	}

	editionRaw, ok := parsed["edition"].(map[string]any)
	if !ok {
		t.Fatalf("edition object missing, config=%s", string(raw))
	}
	if runtime := strings.TrimSpace(asString(editionRaw["runtime"])); runtime != "native" {
		t.Fatalf("edition.runtime = %q, want %q", runtime, "native")
	}
	depVersions, ok := editionRaw["dependency_versions"].(map[string]any)
	if !ok {
		t.Fatalf("edition.dependency_versions missing, config=%s", string(raw))
	}
	if strings.TrimSpace(asString(depVersions["redis"])) == "" {
		t.Fatalf("edition.dependency_versions.redis should not be empty, config=%s", string(raw))
	}
	if strings.TrimSpace(asString(depVersions["ollama"])) == "" {
		t.Fatalf("edition.dependency_versions.ollama should not be empty, config=%s", string(raw))
	}
	if strings.TrimSpace(asString(depVersions["qdrant"])) == "" {
		t.Fatalf("edition.dependency_versions.qdrant should not be empty, config=%s", string(raw))
	}
}

func TestStartGatewayScript_NativeRuntime_ShouldFailBeforeDocker(t *testing.T) {
	root := projectRoot(t)
	scriptPath := filepath.Join(root, "scripts", "start-gateway.sh")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	configJSON := `{
  "server": {"port": "8566", "mode": "debug"},
  "redis": {"host": "localhost", "port": 6379, "password": "", "db": 0},
  "database": {"path": "./data/ai-gateway.db"},
  "providers": [],
  "limiter": {"enabled": true, "rate": 100, "burst": 200, "per_user": true},
  "intent_engine": {"enabled": false, "base_url": "http://127.0.0.1:18566", "timeout_ms": 1500, "language": "zh-CN", "expected_dimension": 1024},
  "vector_cache": {"enabled": true, "index_name": "idx_ai_cache_v2", "key_prefix": "ai:v2:cache:", "dimension": 1024, "query_timeout_ms": 1200, "thresholds": {"calc": 0.97}, "ttl_seconds": {"calc": 10}},
  "edition": {"type": "standard", "runtime": "native", "dependency_versions": {"redis":"7.2.0-v18","ollama":"latest","qdrant":"latest"}}
}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	tempBin := filepath.Join(tempDir, "bin")
	if err := os.MkdirAll(tempBin, 0o755); err != nil {
		t.Fatalf("mkdir temp bin failed: %v", err)
	}
	dockerLogPath := filepath.Join(tempDir, "docker.log")
	if err := os.WriteFile(dockerLogPath, nil, 0o644); err != nil {
		t.Fatalf("create docker log failed: %v", err)
	}
	writeExecutable(t, filepath.Join(tempBin, "docker"), `#!/bin/bash
set -eu
: "${DOCKER_LOG_FILE:?}"
printf '%s\n' "$*" >>"$DOCKER_LOG_FILE"
exit 0
`)

	env := append(os.Environ(),
		"PATH="+tempBin+":/usr/bin:/bin",
		"DOCKER_LOG_FILE="+dockerLogPath,
		"CONFIG_PATH="+configPath,
	)
	output, err := runScript(t, scriptPath, env)
	if err == nil {
		t.Fatalf("expected start-gateway.sh to fail for native runtime, output=%s", output)
	}
	if !strings.Contains(output, "runtime=native") {
		t.Fatalf("expected native runtime rejection message, output=%s", output)
	}
	if !strings.Contains(output, "dev-restart.sh") {
		t.Fatalf("expected guidance to use dev-restart.sh, output=%s", output)
	}

	dockerCalls, readErr := os.ReadFile(dockerLogPath)
	if readErr != nil {
		t.Fatalf("read docker log failed: %v", readErr)
	}
	if strings.TrimSpace(string(dockerCalls)) != "" {
		t.Fatalf("docker should not be invoked when native runtime is rejected, calls=%s", string(dockerCalls))
	}
}

func TestDockerScriptUp_NativeRuntime_ShouldFailBeforeDocker(t *testing.T) {
	root := projectRoot(t)
	scriptPath := filepath.Join(root, "scripts", "docker.sh")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	configJSON := `{
  "server": {"port": "8566", "mode": "debug"},
  "redis": {"host": "localhost", "port": 6379, "password": "", "db": 0},
  "database": {"path": "./data/ai-gateway.db"},
  "providers": [],
  "limiter": {"enabled": true, "rate": 100, "burst": 200, "per_user": true},
  "intent_engine": {"enabled": false, "base_url": "http://127.0.0.1:18566", "timeout_ms": 1500, "language": "zh-CN", "expected_dimension": 1024},
  "vector_cache": {"enabled": true, "index_name": "idx_ai_cache_v2", "key_prefix": "ai:v2:cache:", "dimension": 1024, "query_timeout_ms": 1200, "thresholds": {"calc": 0.97}, "ttl_seconds": {"calc": 10}},
  "edition": {"type": "enterprise", "runtime": "native", "dependency_versions": {"redis":"7.2.0-v18","ollama":"latest","qdrant":"latest"}}
}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	tempBin := filepath.Join(tempDir, "bin")
	if err := os.MkdirAll(tempBin, 0o755); err != nil {
		t.Fatalf("mkdir temp bin failed: %v", err)
	}
	dockerLogPath := filepath.Join(tempDir, "docker.log")
	if err := os.WriteFile(dockerLogPath, nil, 0o644); err != nil {
		t.Fatalf("create docker log failed: %v", err)
	}
	writeExecutable(t, filepath.Join(tempBin, "docker-compose"), `#!/bin/bash
set -eu
: "${DOCKER_LOG_FILE:?}"
printf '%s\n' "$*" >>"$DOCKER_LOG_FILE"
exit 0
`)

	env := append(os.Environ(),
		"PATH="+tempBin+":/usr/bin:/bin",
		"DOCKER_LOG_FILE="+dockerLogPath,
		"CONFIG_PATH="+configPath,
	)
	output, err := runScript(t, scriptPath, env, "up")
	if err == nil {
		t.Fatalf("expected docker.sh up to fail for native runtime, output=%s", output)
	}
	if !strings.Contains(output, "runtime=native") {
		t.Fatalf("expected native runtime rejection message, output=%s", output)
	}
	if !strings.Contains(output, "dev-restart.sh") {
		t.Fatalf("expected guidance to use dev-restart.sh, output=%s", output)
	}

	dockerCalls, readErr := os.ReadFile(dockerLogPath)
	if readErr != nil {
		t.Fatalf("read docker log failed: %v", readErr)
	}
	if strings.TrimSpace(string(dockerCalls)) != "" {
		t.Fatalf("docker compose should not run when native runtime is rejected, calls=%s", string(dockerCalls))
	}
}

func asString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	default:
		return ""
	}
}
