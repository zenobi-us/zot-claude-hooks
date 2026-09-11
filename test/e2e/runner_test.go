package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type e2eResult struct {
	actions     string
	eventLog    string
	exitCode    int
	protocolLog string
	stderr      string
	stdout      string
}

func extensionDirectory(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate the test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func copyTree(t *testing.T, source, destination string) {
	t.Helper()
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
	if err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
}

func readText(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func runCase(t *testing.T, name string) e2eResult {
	t.Helper()
	temporaryDirectory := t.TempDir()
	sourceProject := filepath.Join(extensionDirectory(t), "test", "e2e", "use-cases", name, "project")
	project := filepath.Join(temporaryDirectory, "project")
	copyTree(t, sourceProject, project)

	provider := startFakeProvider()
	t.Cleanup(provider.server.Close)
	actionLog := filepath.Join(temporaryDirectory, "actions.jsonl")
	eventLog := filepath.Join(temporaryDirectory, "zot-events.jsonl")
	protocolLog := filepath.Join(temporaryDirectory, "extension-protocol.jsonl")

	zotBinary := os.Getenv("ZOT_BIN")
	if zotBinary == "" {
		zotBinary = "zot"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, zotBinary,
		"--json", "--no-session", "--cwd", project,
		"--provider", "openai", "--model", "gpt-5", "--api-key", "fixture-key",
		"--base-url", provider.server.URL+"/v1", "--max-steps", "3",
		"--ext", extensionDirectory(t), "run the fixture tool",
	)
	command.Dir = project
	originalHome := os.Getenv("HOME")
	command.Env = append(os.Environ(),
		"HOME="+filepath.Join(temporaryDirectory, "home"),
		"GOMODCACHE="+filepath.Join(originalHome, "go", "pkg", "mod"),
		"GOCACHE="+filepath.Join(originalHome, ".cache", "go-build"),
		"XDG_CONFIG_HOME="+filepath.Join(temporaryDirectory, "config"),
		"XDG_STATE_HOME="+filepath.Join(temporaryDirectory, "state"),
		"ZOT_HOOK_TEST_LOG="+actionLog,
		"ZOT_HOOKS_PROTOCOL_TRACE="+protocolLog,
	)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() != nil {
			exitCode = -1
		} else {
			t.Fatalf("run zot: %v", err)
		}
	}
	if err := os.WriteFile(eventLog, stdout.Bytes(), 0o644); err != nil {
		t.Fatalf("write event log: %v", err)
	}
	return e2eResult{
		actions: readText(actionLog), eventLog: readText(eventLog), exitCode: exitCode,
		protocolLog: readText(protocolLog), stderr: stderr.String(), stdout: stdout.String(),
	}
}

func jsonLines(t *testing.T, text string) []map[string]any {
	t.Helper()
	var result []map[string]any
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("parse JSONL line %q: %v", line, err)
		}
		result = append(result, event)
	}
	return result
}

func TestEndToEnd(t *testing.T) {
	t.Run("allows a matching PreToolUse hook", func(t *testing.T) {
		result := runCase(t, "pre-tool-use-allow")
		if result.exitCode != 0 {
			t.Fatalf("zot exit code = %d, stderr = %s", result.exitCode, result.stderr)
		}
		var action map[string]any
		if err := json.Unmarshal([]byte(strings.Split(result.actions, "\n")[0]), &action); err != nil {
			t.Fatalf("parse action: %v", err)
		}
		payload, ok := action["payload"].(map[string]any)
		if !ok || payload["hook_event_name"] != "PreToolUse" || payload["tool_name"] != "bash" {
			t.Fatalf("unexpected action payload: %s", result.actions)
		}
		input, ok := payload["tool_input"].(map[string]any)
		if !ok || input["command"] != "printf tool-ran" {
			t.Fatalf("unexpected tool input: %#v", payload["tool_input"])
		}
		foundToolCall := false
		for _, event := range jsonLines(t, result.eventLog) {
			if event["type"] == "tool_call" {
				foundToolCall = true
			}
		}
		if !foundToolCall {
			t.Fatalf("event log contains no tool_call: %s", result.eventLog)
		}
		if !strings.Contains(result.stdout, "fixture complete") {
			t.Fatalf("stdout does not contain fixture complete: %s", result.stdout)
		}
	})

	t.Run("blocks on exit status 2", func(t *testing.T) {
		result := runCase(t, "pre-tool-use-block-exit")
		if result.exitCode != 0 {
			t.Fatalf("zot exit code = %d, stderr = %s", result.exitCode, result.stderr)
		}
		if !strings.Contains(result.actions, `"tool_name":"bash"`) ||
			!strings.Contains(result.stdout, "blocked by hook") ||
			!strings.Contains(result.stdout, `"is_error":true`) {
			t.Fatalf("unexpected blocked result: actions=%s stdout=%s", result.actions, result.stdout)
		}
		if strings.Contains(result.stdout, `"type":"tool_progress"`) {
			t.Fatalf("blocked call emitted tool progress: %s", result.stdout)
		}
	})

	t.Run("blocks on a JSON decision", func(t *testing.T) {
		result := runCase(t, "pre-tool-use-block-json")
		if result.exitCode != 0 {
			t.Fatalf("zot exit code = %d, stderr = %s", result.exitCode, result.stderr)
		}
		if !strings.Contains(result.actions, `"tool_name":"bash"`) ||
			!strings.Contains(result.stdout, "fixture blocked this tool") {
			t.Fatalf("unexpected JSON block result: actions=%s stdout=%s", result.actions, result.stdout)
		}
	})
}
