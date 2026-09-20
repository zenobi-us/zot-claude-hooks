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
