package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var safeEnvironmentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// hookEnvironmentState is private to one app instance. Its file is a transport
// for hook output, not a shell script. Values are parsed before they reach a
// later hook process.
type hookEnvironmentState struct {
	projectDir string
	sessionID  string
	file       string
	values     map[string]string
}

func newHookEnvironmentState(projectDir, sessionID string) *hookEnvironmentState {
	root, err := os.MkdirTemp("", "zot-cluade-hooks-env-")
	if err != nil {
		return &hookEnvironmentState{projectDir: projectDir, sessionID: sessionID, values: map[string]string{}}
	}
	file := filepath.Join(root, "environment")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		return &hookEnvironmentState{projectDir: projectDir, sessionID: sessionID, values: map[string]string{}}
	}
	return &hookEnvironmentState{projectDir: projectDir, sessionID: sessionID, file: file, values: map[string]string{}}
}

func (s *hookEnvironmentState) reset() {
	if s == nil {
		return
	}
	for key := range s.values {
		delete(s.values, key)
	}
	if s.file != "" {
		_ = os.WriteFile(s.file, nil, 0o600)
	}
}

func (s *hookEnvironmentState) close() {
	if s == nil || s.file == "" {
		return
	}
	_ = os.Remove(s.file)
	_ = os.Remove(filepath.Dir(s.file))
	s.file = ""
	s.values = map[string]string{}
}

// persist replaces the parsed state with safe assignments from contents. It is
// used by tests; hooks write the file and call refresh instead.
func (s *hookEnvironmentState) persist(contents string) error {
	if s == nil || s.file == "" {
		return errors.New("environment state is unavailable")
	}
	if err := os.WriteFile(s.file, []byte(contents), 0o600); err != nil {
		return err
	}
	s.refresh()
	return nil
}

func (s *hookEnvironmentState) refresh() {
	if s == nil || s.file == "" {
		return
	}
	data, err := os.ReadFile(s.file)
	if err != nil {
		return
	}
	values := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		name, value, ok := safeEnvironmentAssignment(line)
		if ok {
			values[name] = value
		}
	}
	s.values = values
}

func safeEnvironmentAssignment(line string) (string, string, bool) {
	name, value, ok := strings.Cut(line, "=")
	if !ok || !safeEnvironmentName.MatchString(name) || strings.ContainsAny(value, "\r\n$`;&|<>") {
		return "", "", false
	}
	return name, value, true
}

func (s *hookEnvironmentState) env() []string {
	if s == nil {
		return nil
	}
	entries := make([]string, 0, len(s.values))
	for name, value := range s.values {
		entries = append(entries, fmt.Sprintf("%s=%s", name, value))
	}
	return entries
}

func appendPersistedEnvironment(parent []string, state *hookEnvironmentState) []string {
	if state == nil {
		return parent
	}
	owned := state.values
	result := make([]string, 0, len(parent)+len(owned))
	for _, entry := range parent {
		name, _, hasValue := strings.Cut(entry, "=")
		if hasValue {
			if _, replace := owned[name]; replace {
				continue
			}
		}
		result = append(result, entry)
	}
	return append(result, state.env()...)
}
