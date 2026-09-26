package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/patriceckhart/zot/packages/agent/ext"
)

func TestExtensionHookSourceCarriesOwningRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ZOT_HOME", root)
	extensionRoot := filepath.Join(root, "extensions", "owner-a")
	hooksDir := filepath.Join(extensionRoot, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "hooks.json"), []byte(`{"hooks":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	sources := extensionSources()
	if len(sources) != 1 {
		t.Fatalf("extensionSources() = %#v, want one source", sources)
	}
	if sources[0].Owner != "owner-a" || sources[0].Root != extensionRoot || !sources[0].Extension {
		t.Fatalf("extension source = %#v, want owner root and extension marker", sources[0])
	}
}

func TestEventPayloadUsesEventDirectoryWhenAvailable(t *testing.T) {
	a := &app{cwd: "/tmp/project"}
	for _, event := range []string{"SessionStart", "Stop", "Notification"} {
		payload := a.eventPayload(event, ext.Event{CWD: "/tmp/event"})
		if payload["cwd"] != "/tmp/event" {
			t.Fatalf("%s payload cwd = %q, want event directory", event, payload["cwd"])
		}
	}
}

func TestEventPayloadFallsBackToProjectDirectory(t *testing.T) {
	a := &app{cwd: "/tmp/project"}
	for _, event := range []string{"PreToolUse", "SessionStart", "Stop", "Notification"} {
		payload := a.eventPayload(event, ext.Event{})
		if payload["cwd"] != "/tmp/project" {
			t.Fatalf("%s payload cwd = %q, want project directory", event, payload["cwd"])
		}
	}
}

func TestEventRunsHookInEventDirectoryWithProjectEnvironment(t *testing.T) {
	projectDir := t.TempDir()
	eventDir := t.TempDir()
	output := filepath.Join(t.TempDir(), "hook-output")
	command := fmt.Sprintf("{ printf '%%s|%%s|%%s\\n' \"$PWD\" \"$ZOT_PROJECT_DIR\" \"$CLAUDE_PROJECT_DIR\"; cat; } > %q", output)
	a := &app{cwd: projectDir, hooks: []hook{{Event: "SessionStart", Command: command, Timeout: defaultTimeout}}}

	a.event("SessionStart", "", map[string]any{"hook_event_name": "SessionStart", "cwd": eventDir})
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.SplitN(string(data), "\n", 2)
	want := eventDir + "|" + projectDir + "|" + projectDir
	if len(lines) != 2 || lines[0] != want || !strings.Contains(lines[1], `"cwd":"`+eventDir+`"`) {
		t.Fatalf("hook output = %q, want process directory, project variables, and event cwd", string(data))
	}
}

func TestSessionEndClearsCachedRuntimeBeforeLaterHooks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell expansion test uses sh")
	}
	t.Setenv("ZOT_SESSION_ID", "")
	t.Setenv("CLAUDE_CODE_SESSION_ID", "")
	output := filepath.Join(t.TempDir(), "hook-output")
	a := &app{
		cwd:   t.TempDir(),
		hooks: []hook{{Event: "PreToolUse", MatcherRE: regexp.MustCompile(`.*`), Command: fmt.Sprintf("printf '%%s|%%s' \"$ZOT_SESSION_ID\" \"$CLAUDE_CODE_SESSION_ID\" > %q", output), Timeout: defaultTimeout}},
	}

	a.event("SessionStart", "", map[string]any{"session_id": "session-a"})
	a.event("SessionEnd", "", map[string]any{})
	allowed, reason := a.preTool("Bash", json.RawMessage(`{"command":"pwd"}`))
	if !allowed || reason != "" {
		t.Fatalf("preTool = %t, %q; want allowed", allowed, reason)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "|" {
		t.Fatalf("cached runtime = %q, want cleared session values", string(data))
	}
}

func TestPreToolUsesProjectDirectoryForProcessAndPayload(t *testing.T) {
	projectDir := t.TempDir()
	output := filepath.Join(t.TempDir(), "hook-output")
	command := fmt.Sprintf("{ printf '%%s|%%s|%%s\\n' \"$PWD\" \"$ZOT_PROJECT_DIR\" \"$CLAUDE_PROJECT_DIR\"; cat; } > %q", output)
	a := &app{cwd: projectDir, hooks: []hook{{Event: "PreToolUse", MatcherRE: regexp.MustCompile(`.*`), Command: command, Timeout: defaultTimeout}}}

	allowed, reason := a.preTool("Bash", json.RawMessage(`{"command":"pwd"}`))
	if !allowed || reason != "" {
		t.Fatalf("preTool = %t, %q; want allowed", allowed, reason)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.SplitN(string(data), "\n", 2)
	want := projectDir + "|" + projectDir + "|" + projectDir
	if len(lines) != 2 || lines[0] != want || !strings.Contains(lines[1], `"cwd":"`+projectDir+`"`) {
		t.Fatalf("hook output = %q, want project directory for process, variables, and payload cwd", string(data))
	}
}

func TestEventPayloadIncludesEffectiveToolResultDetails(t *testing.T) {
	a := &app{cwd: "/tmp/project"}
	executed := true
	ev := ext.Event{
		Name: "tool_result", SessionID: "session-1", Sequence: 7,
		ToolID: "call-1", ToolName: "bash", ToolArgs: json.RawMessage(`{"command":"printf ok"}`),
		Status: "completed", Executed: &executed,
	}
	payload := a.eventPayload("PostToolUse", ev)
	if payload["hook_event_name"] != "PostToolUse" || payload["tool_name"] != "bash" {
		t.Fatalf("got unexpected payload: %#v", payload)
	}
	if payload["tool_status"] != "completed" || payload["tool_executed"] != true {
		t.Fatalf("missing tool outcome fields: %#v", payload)
	}
	args, ok := payload["tool_input"].(json.RawMessage)
	if !ok || string(args) != `{"command":"printf ok"}` {
		t.Fatalf("got tool input %q, want effective arguments", args)
	}
}

func TestEventPayloadIncludesLifecycleFields(t *testing.T) {
	a := &app{cwd: "/tmp/project"}
	count, tokens := 12, 345
	payload := a.eventPayload("PreCompact", ext.Event{
		Name: "pre_compact", CompactionID: "compact-1", MessageCount: &count, TokenEstimate: &tokens,
	})
	if payload["compaction_id"] != "compact-1" || payload["message_count"] != 12 || payload["token_estimate"] != 345 {
		t.Fatalf("got unexpected lifecycle payload: %#v", payload)
	}
}

func TestParseHookDocumentBuildsTypedCommandHooks(t *testing.T) {
	data := []byte(`{
		"name": "unrelated setting",
		"hooks": {
			"PreToolUse": [{
				"matcher": "^bash$",
				"hooks": [{
					"type": "command",
					"command": "printf hook",
					"timeout": 1.5,
					"args": ["ignored by current runtime"]
				}]
			}]
		}
	}`)

	hooks := parseHookDocument(data, "settings.json", "")
	if len(hooks) != 1 {
		t.Fatalf("got %d hooks, want 1", len(hooks))
	}
	got := hooks[0]
	if got.Event != "PreToolUse" || got.Command != "printf hook" || got.Matcher != "^bash$" {
		t.Fatalf("got unexpected hook: %+v", got)
	}
	if got.Timeout != 1500*time.Millisecond {
		t.Fatalf("got timeout %s, want 1.5s", got.Timeout)
	}
	if !matches(got, "bash") || matches(got, "edit") {
		t.Fatal("compiled matcher does not match the expected tool names")
	}
}

func TestParseHookDocumentKeepsValidSiblings(t *testing.T) {
	data := []byte(`{
		"hooks": {
			"SessionStart": [{
				"hooks": [
					{"type": "http", "url": "https://example.test"},
					{"type": "command", "command": "printf valid"},
					{"type": "command", "command": ""}
				]
			}]
		}
	}`)

	hooks := parseHookDocument(data, "settings.json", "")
	if len(hooks) != 1 || hooks[0].Command != "printf valid" {
		t.Fatalf("got valid hooks %+v, want one command hook", hooks)
	}
}

func TestSharedHookDirectoriesLoadSortedJSONFilesWithoutRecursion(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	zot := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ZOT_HOME", zot)

	write := func(path, command string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		data := fmt.Sprintf(`{"hooks":{"SessionStart":[{"hooks":[{"type":"command","command":%q}]}]}}`, command)
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(home, ".agents/hooks", "b.json"), "home-agents-b")
	write(filepath.Join(home, ".agents/hooks", "a.json"), "home-agents-a")
	write(filepath.Join(project, ".claude/hooks", "nested", "ignored.json"), "nested")
	write(filepath.Join(project, ".claude/hooks", "c.json"), "project-claude")
	write(filepath.Join(zot, "hooks", "z.json"), "zot")

	hooks := loadHooks(project)
	var commands []string
	for _, hook := range hooks {
		commands = append(commands, hook.Command)
	}
	want := []string{"home-agents-a", "home-agents-b", "zot", "project-claude"}
	if !slices.Equal(commands, want) {
		t.Fatalf("loaded commands = %#v, want %#v", commands, want)
	}
}

func TestSharedHookDiscoverySupportsSymlinksAndCanonicalDeduplication(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	zot := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ZOT_HOME", zot)

	realDir := filepath.Join(t.TempDir(), "hooks")
	path := filepath.Join(realDir, "shared.json")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"shared"}]}]}}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realDir, filepath.Join(home, ".claude")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(project, ".agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realDir, filepath.Join(project, ".agents", "hooks")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(path, filepath.Join(zot, "hooks.json")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZOT_HOOKS_PATH", filepath.Join(zot, "hooks.json"))

	hooks := loadHooks(project)
	if len(hooks) != 1 || hooks[0].Command != "shared" {
		t.Fatalf("hooks = %#v, want one canonical hook", hooks)
	}
}

func TestHookLocationsIncludeSharedDirectories(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	zot := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ZOT_HOME", zot)
	a := &app{cwd: project}
	locations := a.formatLocations()
	for _, path := range []string{
		filepath.Join(home, ".agents", "hooks"),
		filepath.Join(home, ".claude", "hooks"),
		filepath.Join(zot, "hooks"),
		filepath.Join(project, ".agents", "hooks"),
		filepath.Join(project, ".claude", "hooks"),
		filepath.Join(project, ".zot", "hooks"),
	} {
		if !strings.Contains(locations, path+" (direct .json files)") {
			t.Fatalf("locations = %q, missing directory source %q", locations, path)
		}
	}
}

func TestParseHookDocumentPreservesCurrentTimeoutCompatibility(t *testing.T) {
	data := []byte(`{
		"hooks": {
			"Stop": [{
				"hooks": [
					{"type": "command", "command": "printf default"},
					{"type": "command", "command": "printf minimum", "timeout": 0.01},
					{"type": "command", "command": "printf string", "timeout": "0.2"}
				]
			}]
		}
	}`)

	hooks := parseHookDocument(data, "settings.json", "")
	if len(hooks) != 3 {
		t.Fatalf("got %d hooks, want 3", len(hooks))
	}
	if hooks[0].Timeout != defaultTimeout {
		t.Fatalf("got default timeout %s, want %s", hooks[0].Timeout, defaultTimeout)
	}
	if hooks[1].Timeout != 100*time.Millisecond {
		t.Fatalf("got minimum timeout %s, want 100ms", hooks[1].Timeout)
	}
	if hooks[2].Timeout != 200*time.Millisecond {
		t.Fatalf("got string timeout %s, want 200ms", hooks[2].Timeout)
	}
}
