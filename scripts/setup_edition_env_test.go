package scripts_test

import (
	"bytes"
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
