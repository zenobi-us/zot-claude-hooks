package main

import "strings"

var ownedHookEnvironmentKeys = map[string]struct{}{
	"ZOT_PROJECT_DIR":    {},
	"CLAUDE_PROJECT_DIR": {},
}

// buildHookEnvironment preserves the parent environment and replaces values
// owned by the hook runner with values for the current project.
func buildHookEnvironment(parent []string, projectDir string) []string {
	environment := make([]string, 0, len(parent)+len(ownedHookEnvironmentKeys))
	positions := make(map[string]int, len(parent))
	for _, entry := range parent {
		name, _, hasValue := strings.Cut(entry, "=")
		if !hasValue {
			environment = append(environment, entry)
			continue
		}
		if _, owned := ownedHookEnvironmentKeys[name]; owned {
			continue
		}
		if position, exists := positions[name]; exists {
			environment[position] = entry
			continue
		}
		positions[name] = len(environment)
		environment = append(environment, entry)
	}
	environment = append(environment,
		"ZOT_PROJECT_DIR="+projectDir,
		"CLAUDE_PROJECT_DIR="+projectDir,
	)
	return environment
}
