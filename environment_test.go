package main

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func environmentValue(entries []string, name string) []string {
	prefix := name + "="
	var values []string
	for _, entry := range entries {
		if strings.HasPrefix(entry, prefix) {
			values = append(values, strings.TrimPrefix(entry, prefix))
		}
	}
	return values
}

func TestBuildHookEnvironmentKeepsExtensionVariablesOutOfProjectHooks(t *testing.T) {
	parent := []string{
		"ZOT_EXTENSION_ROOT=/stale/root",
		"ZOT_EXTENSION_DATA=/inherited/data",
		"ZOT_EXTENSION_OPTION_MODE=inherited",
	}

	got := buildHookEnvironment(parent, "/project")

	for _, name := range []string{"ZOT_EXTENSION_ROOT", "ZOT_EXTENSION_DATA", "ZOT_EXTENSION_OPTION_MODE"} {
		if values := environmentValue(got, name); len(values) != 1 || values[0] != environmentValue(parent, name)[0] {
			t.Fatalf("%s entries = %#v, want inherited value", name, values)
		}
	}
}

func TestBuildHookEnvironmentSetsExtensionRootForOwnedHook(t *testing.T) {
	parent := []string{
		"ZOT_EXTENSION_ROOT=/stale/root",
		"ZOT_EXTENSION_ROOT=/duplicate/root",
		"ZOT_EXTENSION_DATA=/inherited/data",
		"ZOT_EXTENSION_OPTION_MODE=inherited",
	}
	source := HookSource{Owner: "owner", Root: "/extensions/owner", Extension: true}

	got := buildHookEnvironment(parent, "/project", source)

	if values := environmentValue(got, "ZOT_EXTENSION_ROOT"); len(values) != 1 || values[0] != source.Root {
		t.Fatalf("ZOT_EXTENSION_ROOT entries = %#v, want [%q]", values, source.Root)
	}
	for _, name := range []string{"ZOT_EXTENSION_DATA", "ZOT_EXTENSION_OPTION_MODE"} {
		if values := environmentValue(got, name); len(values) != 1 || values[0] != environmentValue(parent, name)[0] {
			t.Fatalf("%s entries = %#v, want inherited value", name, values)
		}
	}
}

func TestBuildHookEnvironmentDoesNotInventExtensionData(t *testing.T) {
	source := HookSource{Owner: "owner", Root: "/extensions/owner", Extension: true}

	got := buildHookEnvironment(nil, "/project", source)

	for _, name := range []string{"ZOT_EXTENSION_DATA", "ZOT_EXTENSION_OPTION_MODE"} {
		if values := environmentValue(got, name); len(values) != 0 {
			t.Fatalf("invented %s = %#v", name, values)
		}
	}
}

func TestBuildHookEnvironmentUsesEachExtensionOwnerRoot(t *testing.T) {
	first := buildHookEnvironment(nil, "/project", HookSource{Owner: "one", Root: "/extensions/one", Extension: true})
	second := buildHookEnvironment(nil, "/project", HookSource{Owner: "two", Root: "/extensions/two", Extension: true})

	if got := environmentValue(first, "ZOT_EXTENSION_ROOT"); len(got) != 1 || got[0] != "/extensions/one" {
		t.Fatalf("first owner root = %#v", got)
	}
	if got := environmentValue(second, "ZOT_EXTENSION_ROOT"); len(got) != 1 || got[0] != "/extensions/two" {
		t.Fatalf("second owner root = %#v", got)
	}
}

func TestRunHookPassesExtensionRootOnlyToOwnedHook(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell expansion test uses sh")
	}
	source := HookSource{Owner: "owner", Root: "/extensions/owner", Extension: true}
	hook := hook{Command: `printf '%s|%s' "$ZOT_EXTENSION_ROOT" "$ZOT_EXTENSION_DATA"`, SourceContext: source, Timeout: defaultTimeout}

	result := runHook(context.Background(), hook, nil, t.TempDir())
	if result.Code != 0 || result.Output != "/extensions/owner|" {
		t.Fatalf("runHook = code %d, output %q; want extension root and no invented data", result.Code, result.Output)
	}
}

func TestBuildHookEnvironmentPreservesInheritedValuesAndOwnsProjectDirectories(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "project")
	parent := []string{
		"CI=1",
		"MY_HOOK_MODE=",
		"ZOT_TEST_VALUE=provided",
		"CLAUDE_UNKNOWN_VALUE=preserve-me",
		"ZOT_PROJECT_DIR=/stale/zot",
		"ZOT_PROJECT_DIR=/duplicate/zot",
		"CLAUDE_PROJECT_DIR=/stale/claude",
	}

	got := buildHookEnvironment(parent, projectDir)

	for _, name := range []string{"CI", "MY_HOOK_MODE", "ZOT_TEST_VALUE", "CLAUDE_UNKNOWN_VALUE"} {
		if values := environmentValue(got, name); len(values) != 1 {
			t.Fatalf("%s entries = %#v, want one inherited entry", name, values)
		}
	}
	for _, name := range []string{"ZOT_PROJECT_DIR", "CLAUDE_PROJECT_DIR"} {
		if values := environmentValue(got, name); len(values) != 1 || values[0] != projectDir {
			t.Fatalf("%s entries = %#v, want [%q]", name, values, projectDir)
		}
	}
	if values := environmentValue(got, "UNSET_HOOK_VALUE"); len(values) != 0 {
		t.Fatalf("unsupported unset variable = %#v, want absent", values)
	}
}

func TestBuildHookEnvironmentScrubsInheritedEnvironmentFileAndDuplicates(t *testing.T) {
	parent := []string{"ZOT_ENV_FILE=stale", "ZOT_ENV_FILE=duplicate", "CI=1"}

	got := buildHookEnvironment(parent, "/project")

	if values := environmentValue(got, "ZOT_ENV_FILE"); len(values) != 0 {
		t.Fatalf("ZOT_ENV_FILE entries = %#v, want absent", values)
	}
	if values := environmentValue(got, "CI"); len(values) != 1 || values[0] != "1" {
		t.Fatalf("CI entries = %#v, want inherited value", values)
	}
}

func TestBuildHookEnvironmentAddsVerifiedSessionAndReplacesInheritedClaudeAlias(t *testing.T) {
	parent := []string{
		"ZOT_SESSION_ID=stale",
		"CLAUDE_CODE_SESSION_ID=stale-claude",
	}

	got := buildHookEnvironmentWithRuntime(parent, "/project", HookRuntime{SessionID: "session-123"})

	for _, name := range []string{"ZOT_SESSION_ID", "CLAUDE_CODE_SESSION_ID"} {
		if values := environmentValue(got, name); len(values) != 1 || values[0] != "session-123" {
			t.Fatalf("%s entries = %#v, want [session-123]", name, values)
		}
	}
	if values := environmentValue(got, "CLAUDE_SESSION_ID"); len(values) != 0 {
		t.Fatalf("incorrect Claude alias entries = %#v, want absent", values)
	}
}

func TestBuildHookEnvironmentLeavesSessionVariablesInheritedWhenUnavailable(t *testing.T) {
	parent := []string{
		"ZOT_SESSION_ID=inherited",
		"CLAUDE_CODE_SESSION_ID=inherited-claude",
		"ZOT_CHILD_SESSION=inherited-child",
	}

	got := buildHookEnvironmentWithRuntime(parent, "/project", HookRuntime{})

	for _, name := range []string{"ZOT_SESSION_ID", "CLAUDE_CODE_SESSION_ID", "ZOT_CHILD_SESSION"} {
		if values := environmentValue(got, name); len(values) != 1 || values[0] != environmentValue(parent, name)[0] {
			t.Fatalf("%s entries = %#v, want inherited value", name, values)
		}
	}
}

func TestBuildHookEnvironmentAddsChildSessionOnlyWhenVerified(t *testing.T) {
	parent := []string{"ZOT_CHILD_SESSION=stale"}

	got := buildHookEnvironmentWithRuntime(parent, "/project", HookRuntime{ChildSession: "agent-123"})

	if values := environmentValue(got, "ZOT_CHILD_SESSION"); len(values) != 1 || values[0] != "agent-123" {
		t.Fatalf("ZOT_CHILD_SESSION entries = %#v, want [agent-123]", values)
	}
}

func TestBuildHookEnvironmentDoesNotExportUnsupportedRuntimeValues(t *testing.T) {
	got := buildHookEnvironmentWithRuntime(nil, "/project", HookRuntime{SessionID: "session-123", ChildSession: "agent-123"})

	for _, name := range []string{"ZOT_EFFORT", "ZOT_REMOTE", "ZOT_BRIDGE", "ZOT_MESSAGE", "ZOT_PID", "ZOT_SHELL"} {
		if values := environmentValue(got, name); len(values) != 0 {
			t.Fatalf("invented %s = %#v", name, values)
		}
	}
}

func TestBuildHookEnvironmentPreservesDuplicateNonOwnedEntriesInOrder(t *testing.T) {
	parent := []string{
		"PATH=/first/bin",
		"CI=first",
		"CUSTOM=one",
		"PATH=/second/bin",
		"CI=second",
		"CUSTOM=two",
	}

	got := buildHookEnvironment(parent, "/project")
	want := append(append([]string{}, parent...),
		"ZOT_PROJECT_DIR=/project",
		"CLAUDE_PROJECT_DIR=/project",
	)
	if len(got) != len(want) {
		t.Fatalf("buildHookEnvironment length = %d, want %d; got %#v", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("buildHookEnvironment[%d] = %q, want %q; got %#v", index, got[index], want[index], got)
		}
	}
}

func TestRunHookUsesVerifiedRuntimeEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell expansion test uses sh")
	}
	hook := hook{Command: `printf '%s|%s|%s' "$ZOT_SESSION_ID" "$CLAUDE_CODE_SESSION_ID" "$ZOT_CHILD_SESSION"`, Timeout: defaultTimeout}

	result := runHookWithRuntime(context.Background(), hook, nil, t.TempDir(), "/project", HookRuntime{SessionID: "session-123", ChildSession: "agent-123"})
	if result.Code != 0 || result.Output != "session-123|session-123|agent-123" {
		t.Fatalf("runHookWithRuntime = code %d, output %q; want verified runtime values", result.Code, result.Output)
	}
}

func TestRunHookPreservesInheritedRuntimeWhenUnavailable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell expansion test uses sh")
	}
	t.Setenv("ZOT_SESSION_ID", "inherited-session")
	t.Setenv("CLAUDE_CODE_SESSION_ID", "inherited-claude")
	hook := hook{Command: `printf '%s|%s' "$ZOT_SESSION_ID" "$CLAUDE_CODE_SESSION_ID"`, Timeout: defaultTimeout}

	result := runHookWithRuntime(context.Background(), hook, nil, t.TempDir(), "/project", HookRuntime{})
	if result.Code != 0 || result.Output != "inherited-session|inherited-claude" {
		t.Fatalf("runHookWithRuntime = code %d, output %q; want inherited runtime values", result.Code, result.Output)
	}
}

func TestRunHookUsesInheritedEnvironmentAndProjectVariables(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell expansion test uses sh")
	}
	t.Setenv("ZOT_TEST_VALUE", "inherited")
	t.Setenv("ZOT_PROJECT_DIR", "/stale/zot")
	t.Setenv("CLAUDE_PROJECT_DIR", "/stale/claude")
	projectDir := t.TempDir()
	hook := hook{Command: "printf '%s|%s|%s|%s' \"$ZOT_TEST_VALUE\" \"$ZOT_PROJECT_DIR\" \"${CLAUDE_PROJECT_DIR}\" \"$UNSET_HOOK_VALUE\"", Timeout: defaultTimeout}

	result := runHook(context.Background(), hook, nil, projectDir)
	want := "inherited|" + projectDir + "|" + projectDir + "|"
	if result.Code != 0 || result.Output != want {
		t.Fatalf("runHook = code %d, output %q; want code 0, output %q", result.Code, result.Output, want)
	}

}
