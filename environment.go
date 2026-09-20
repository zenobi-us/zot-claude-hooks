package main

import "strings"

var ownedHookEnvironmentKeys = map[string]struct{}{
	"ZOT_PROJECT_DIR":    {},
	"CLAUDE_PROJECT_DIR": {},
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
	source := HookSource{}
	if len(sources) > 0 {
		source = sources[0]
	}
	owned := ownedHookEnvironmentKeys
	if source.Extension && source.Root != "" {
		owned = map[string]struct{}{}
		for name := range ownedHookEnvironmentKeys {
			owned[name] = struct{}{}
		}
		owned["ZOT_EXTENSION_ROOT"] = struct{}{}
	}
	environment := make([]string, 0, len(parent)+len(owned)+1)
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
	return environment
}
