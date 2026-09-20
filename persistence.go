package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var safeEnvironmentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var blockedEnvironmentNames = map[string]struct{}{
	"BASH_ENV": {}, "ENV": {}, "LD_PRELOAD": {}, "LD_LIBRARY_PATH": {},
	"PATH": {}, "SHELL": {}, "ZOT_ENV_FILE": {},
}

// hookEnvironmentState is private to one app instance. Its file is a transport
// for hook output, not a shell script. Values are parsed before they reach a
// later hook process.
type hookEnvironmentState struct {
	mu sync.RWMutex

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
	s.mu.Lock()
	defer s.mu.Unlock()
	for key := range s.values {
		delete(s.values, key)
	}
	if s.file != "" {
		_ = os.WriteFile(s.file, nil, 0o600)
	}
}

func (s *hookEnvironmentState) close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file == "" {
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
	if s == nil {
		return errors.New("environment state is unavailable")
	}
	s.mu.RLock()
	file := s.file
	s.mu.RUnlock()
	if file == "" {
		return errors.New("environment state is unavailable")
	}
	if err := os.WriteFile(file, []byte(contents), 0o600); err != nil {
		return err
	}
	s.refresh()
	return nil
}

func (s *hookEnvironmentState) refresh() {
	if s == nil {
		return
	}
	s.mu.RLock()
	file := s.file
	s.mu.RUnlock()
	if file == "" {
		return
	}
	data, err := os.ReadFile(file)
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
	s.mu.Lock()
	if s.file == file {
		s.values = values
	}
	s.mu.Unlock()
}

func safeEnvironmentAssignment(line string) (string, string, bool) {
	name, value, ok := strings.Cut(line, "=")
	if !ok || !safeEnvironmentName.MatchString(name) || isBlockedEnvironmentName(name) || strings.ContainsAny(value, "\r\n$`;&|<>") {
		return "", "", false
	}
	return name, value, true
}

func isBlockedEnvironmentName(name string) bool {
	if _, blocked := blockedEnvironmentNames[name]; blocked {
		return true
	}
	return strings.HasPrefix(name, "LD_") || strings.HasPrefix(name, "DYLD_")
}

func (s *hookEnvironmentState) env() []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := make([]string, 0, len(s.values))
	for name, value := range s.values {
		entries = append(entries, fmt.Sprintf("%s=%s", name, value))
	}
	return entries
}

func (s *hookEnvironmentState) environmentFile() string {
	if s == nil {
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.file
}

func appendPersistedEnvironment(parent []string, state *hookEnvironmentState) []string {
	if state == nil {
		return parent
	}
	state.mu.RLock()
	owned := make(map[string]string, len(state.values))
	for name, value := range state.values {
		owned[name] = value
	}
	state.mu.RUnlock()
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
