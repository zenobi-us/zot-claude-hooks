package main

import "strings"

var ownedHookEnvironmentKeys = map[string]struct{}{
	"ZOT_PROJECT_DIR":    {},
	"CLAUDE_PROJECT_DIR": {},
}

// HookRuntime contains runtime identities that the zot SDK exposes with a
// defined meaning. Empty values mean that zot did not provide that value.
type HookRuntime struct {
	SessionID    string
	ChildSession string
}

// HookSource identifies the file that supplied a hook. Extension roots are
// scoped to hooks from that extension. Data and options stay unset because
// zot has no persistent-data or option contract for these sources.
type HookSource struct {
	Path      string
	Owner     string
	Root      string
	Extension bool
}

// buildHookEnvironment preserves the parent environment and replaces values
// owned by the hook runner with values for the current project and source.
func buildHookEnvironment(parent []string, projectDir string, sources ...HookSource) []string {
	return buildHookEnvironmentWithRuntime(parent, projectDir, HookRuntime{}, sources...)
}

func buildHookEnvironmentWithRuntime(parent []string, projectDir string, runtime HookRuntime, sources ...HookSource) []string {
	source := HookSource{}
	if len(sources) > 0 {
		source = sources[0]
	}
	owned := map[string]struct{}{}
	for name := range ownedHookEnvironmentKeys {
		owned[name] = struct{}{}
	}
	if source.Extension && source.Root != "" {
		owned["ZOT_EXTENSION_ROOT"] = struct{}{}
	}
	if runtime.SessionID != "" {
		owned["ZOT_SESSION_ID"] = struct{}{}
		owned["CLAUDE_CODE_SESSION_ID"] = struct{}{}
	}
	if runtime.ChildSession != "" {
		owned["ZOT_CHILD_SESSION"] = struct{}{}
	}
	environment := make([]string, 0, len(parent)+len(owned)+3)
	for _, entry := range parent {
		name, _, hasValue := strings.Cut(entry, "=")
		if !hasValue {
			environment = append(environment, entry)
			continue
		}
		if _, isOwned := owned[name]; isOwned {
			continue
		}
		environment = append(environment, entry)
	}
	environment = append(environment,
		"ZOT_PROJECT_DIR="+projectDir,
		"CLAUDE_PROJECT_DIR="+projectDir,
	)
	if source.Extension && source.Root != "" {
		environment = append(environment, "ZOT_EXTENSION_ROOT="+source.Root)
	}
	if runtime.SessionID != "" {
		environment = append(environment, "ZOT_SESSION_ID="+runtime.SessionID, "CLAUDE_CODE_SESSION_ID="+runtime.SessionID)
	}
	if runtime.ChildSession != "" {
		environment = append(environment, "ZOT_CHILD_SESSION="+runtime.ChildSession)
	}
	return environment
}
