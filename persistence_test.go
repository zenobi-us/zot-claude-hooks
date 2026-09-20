package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestPersistedEnvironmentAcceptsSafeAssignments(t *testing.T) {
	state := newHookEnvironmentState(t.TempDir(), "session-a")
	if err := state.persist("SAFE_VALUE=hello world\nSECOND=/tmp/value\n"); err != nil {
		t.Fatal(err)
	}
	if got := state.values["SAFE_VALUE"]; got != "hello world" {
		t.Fatalf("SAFE_VALUE = %q, want hello world", got)
	}
	if got := state.values["SECOND"]; got != "/tmp/value" {
		t.Fatalf("SECOND = %q, want /tmp/value", got)
	}
}

func TestManifestIncludesPersistenceSource(t *testing.T) {
	data, err := os.ReadFile("extension.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Args []string `json:"args"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, arg := range manifest.Args {
		if arg == "persistence.go" {
			return
		}
	}
	t.Fatalf("manifest args = %#v, want persistence.go", manifest.Args)
}

func TestPersistedEnvironmentRejectsExecutableSyntaxMalformedAndDangerousNames(t *testing.T) {
	state := newHookEnvironmentState(t.TempDir(), "session-a")
	if err := state.persist(strings.Join([]string{
		"GOOD=kept",
		"BAD=$(touch " + filepath.Join(t.TempDir(), "created") + ")",
		"PIPE=one|two",
		"REDIRECT=one>two",
		"BROKEN",
		"1BAD=value",
		"LD_PRELOAD=/tmp/evil.so",
		"BASH_ENV=/tmp/evil.sh",
		"ENV=/tmp/evil.sh",
		"PATH=/tmp/bin",
		"SHELL=/tmp/evil-shell",
		"ZOT_ENV_FILE=/tmp/other-file",
		"", // empty lines are allowed.
	}, "\n")); err != nil {
		t.Fatal(err)
	}
	if got := state.values["GOOD"]; got != "kept" {
		t.Fatalf("GOOD = %q, want kept", got)
	}
	for _, name := range []string{"BAD", "PIPE", "REDIRECT", "BROKEN", "1BAD", "LD_PRELOAD", "BASH_ENV", "ENV", "PATH", "SHELL", "ZOT_ENV_FILE"} {
		if _, ok := state.values[name]; ok {
			t.Fatalf("rejected assignment %s was accepted", name)
		}
	}
}

func TestZotEnvironmentFileIsSessionStartOnlyAndNotDuplicated(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture uses sh")
	}
	project := t.TempDir()
	output := filepath.Join(t.TempDir(), "output")
	a := &app{cwd: project, hooks: []hook{
		{Event: "SessionStart", Command: "printf '%s' \"$ZOT_ENV_FILE\" > " + output, Timeout: defaultTimeout},
		{Event: "Notification", Command: "printf '%s' \"$ZOT_ENV_FILE\" > " + output, Timeout: defaultTimeout},
	}}
	t.Setenv("ZOT_ENV_FILE", "inherited-stale")
	a.event("SessionStart", "", map[string]any{"session_id": "session-a"})
	startValue, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(startValue) == "" || string(startValue) == "inherited-stale" {
		t.Fatalf("SessionStart ZOT_ENV_FILE = %q, want one current file path", startValue)
	}
	a.event("Notification", "", map[string]any{})
	laterValue, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(laterValue) != "" {
		t.Fatalf("later hook ZOT_ENV_FILE = %q, want empty", laterValue)
	}
}

func TestSessionStartProvidesZotEnvironmentFileAndLaterHooksSeeValues(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture uses sh")
	}
	project := t.TempDir()
	output := filepath.Join(t.TempDir(), "output")
	writer := "printf 'PERSISTED=visible\\n' > \"$ZOT_ENV_FILE\""
	reader := "printf '%s' \"$PERSISTED\" > " + output
	a := &app{cwd: project, hooks: []hook{
		{Event: "SessionStart", Command: writer, Timeout: defaultTimeout},
		{Event: "Notification", Command: reader, Timeout: defaultTimeout},
	}}

	a.event("SessionStart", "", map[string]any{"session_id": "session-a"})
	a.event("Notification", "", map[string]any{})
	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "visible" {
		t.Fatalf("later hook value = %q, want visible", got)
	}
}

func TestSessionEndAndRestartClearPersistedEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture uses sh")
	}
	project := t.TempDir()
	output := filepath.Join(t.TempDir(), "output")
	a := &app{cwd: project, hooks: []hook{
		{Event: "SessionStart", Command: "printf 'PERSISTED=old\\n' > \"$ZOT_ENV_FILE\"", Timeout: defaultTimeout},
		{Event: "Notification", Command: "printf '%s' \"$PERSISTED\" > " + output, Timeout: defaultTimeout},
	}}

	a.event("SessionStart", "", map[string]any{"session_id": "session-a"})
	a.event("SessionEnd", "", map[string]any{})
	a.event("Notification", "", map[string]any{})
	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "" {
		t.Fatalf("after session end value = %q, want empty", got)
	}

	a.event("SessionStart", "", map[string]any{"session_id": "session-b"})
	a.event("Notification", "", map[string]any{})
	got, err = os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old" {
		t.Fatalf("restarted session value = %q, want old", got)
	}
}

func TestPersistedEnvironmentOverridesInheritedValueOnlyForLaterHook(t *testing.T) {
	state := newHookEnvironmentState(t.TempDir(), "session-a")
	if err := state.persist("PERSISTED=new\n"); err != nil {
		t.Fatal(err)
	}
	got := appendPersistedEnvironment([]string{"PERSISTED=old", "OTHER=value"}, state)
	if values := environmentValue(got, "PERSISTED"); len(values) != 1 || values[0] != "new" {
		t.Fatalf("persisted environment = %#v, want one new value", values)
	}
}

func TestPersistedEnvironmentStateIsSafeForConcurrentCallbacks(t *testing.T) {
	state := newHookEnvironmentState(t.TempDir(), "session-a")
	var wait sync.WaitGroup
	for i := 0; i < 20; i++ {
		wait.Add(2)
		go func(i int) {
			defer wait.Done()
			_ = state.persist("VALUE=" + string(rune('a'+i)) + "\n")
		}(i)
		go func() {
			defer wait.Done()
			state.refresh()
		}()
	}
	wait.Wait()
	if got := state.env(); len(got) > 1 {
		t.Fatalf("concurrent state values = %#v, want at most one value", got)
	}
}

func TestPersistedEnvironmentDoesNotChangeZotProcessEnvironment(t *testing.T) {
	const name = "ZOT_PERSISTENCE_ISOLATION"
	_ = os.Unsetenv(name)
	state := newHookEnvironmentState(t.TempDir(), "session-a")
	if err := state.persist(name + "=child-only\n"); err != nil {
		t.Fatal(err)
	}
	if _, ok := os.LookupEnv(name); ok {
		t.Fatal("persisted value changed zot process environment")
	}

	result := runHookWithEnvironment(context.Background(), hook{Command: "printf '%s' \"$" + name + "\"", Timeout: defaultTimeout}, nil, t.TempDir(), "/project", HookRuntime{}, state, false)
	if result.Code != 0 || result.Output != "child-only" {
		t.Fatalf("later hook = code %d, output %q; want child-only", result.Code, result.Output)
	}
	if _, ok := os.LookupEnv(name); ok {
		t.Fatal("hook execution changed zot process environment")
	}
}
