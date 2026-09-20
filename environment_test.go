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
