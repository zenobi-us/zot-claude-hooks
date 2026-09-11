package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/patriceckhart/zot/packages/agent/ext"
)

const (
	name           = "zot-cluade-hooks"
	defaultTimeout = 10 * time.Second
)

var version = "0.0.0-dev"

var hookEvents = []string{"PreToolUse", "SessionStart", "Stop", "Notification", "UserPromptSubmit", "PostToolUse", "PermissionRequest", "SessionEnd", "PreCompact", "PostCompact", "SubagentStart", "SubagentStop"}

type hook struct {
	Event, Command, Matcher, Source, Owner string
	Timeout                                time.Duration
}

type hookResult struct {
	Code     int
	Output   string
	Response map[string]any
}

type app struct {
	mu    sync.RWMutex
	cwd   string
	hooks []hook
	ext   *ext.Extension

	panelMode        string
	panelEvent       string
	panelEventChoice int
	panelCommand     string
}

func logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[%s] %s\n", name, fmt.Sprintf(format, args...))
}

func object(v any) map[string]any {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	return m
}

func stringsValue(v any, fallback string) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fallback
}
func array(v any) []any {
	if a, ok := v.([]any); ok {
		return a
	}
	return nil
}

func zotHome() string {
	if value := os.Getenv("ZOT_HOME"); value != "" {
		return absolute(value)
	}
	state := os.Getenv("XDG_STATE_HOME")
	if state == "" {
		if home, err := os.UserHomeDir(); err == nil {
			state = filepath.Join(home, ".local", "state")
		}
	}
	return absolute(filepath.Join(state, "zot"))
}

func absolute(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	cwd, _ := os.Getwd()
	return filepath.Clean(filepath.Join(cwd, path))
}

func configPaths(cwd string) []string {
	home, _ := os.UserHomeDir()
	paths := []string{
		filepath.Join(home, ".claude", "settings.json"),
		filepath.Join(zotHome(), "zot-cluade-hooks.json"),
		filepath.Join(cwd, ".claude", "settings.json"),
		filepath.Join(cwd, ".claude", "settings.local.json"),
		filepath.Join(cwd, ".zot", "zot-cluade-hooks.json"),
		filepath.Join(cwd, ".zot", "zot-cluade-hooks.local.json"),
	}
	if value := os.Getenv("ZOT_HOOKS_PATH"); value != "" {
		paths = append(paths, filepath.Join(cwd, value))
	}
	seen := map[string]bool{}
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		path = filepath.Clean(path)
		if !seen[path] {
			seen[path] = true
			result = append(result, path)
		}
	}
	return result
}

func extensionSources() []struct{ path, owner string } {
	entries, err := os.ReadDir(filepath.Join(zotHome(), "extensions"))
	if err != nil {
		return nil
	}
	var result []struct{ path, owner string }
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == name {
			continue
		}
		files, err := os.ReadDir(filepath.Join(zotHome(), "extensions", entry.Name(), "hooks"))
		if err != nil {
			continue
		}
		sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
				result = append(result, struct{ path, owner string }{filepath.Join(zotHome(), "extensions", entry.Name(), "hooks", file.Name()), entry.Name()})
			}
		}
	}
	return result
}

func loadHooks(cwd string) []hook {
	var sources []struct{ path, owner string }
	for _, path := range configPaths(cwd) {
		sources = append(sources, struct{ path, owner string }{path, ""})
	}
	sources = append(sources, extensionSources()...)
	var result []hook
	for _, source := range sources {
		data, err := os.ReadFile(source.path)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				logf("cannot read %s: %v", source.path, err)
			}
			continue
		}
		var document map[string]any
		if err := json.Unmarshal(data, &document); err != nil {
			logf("cannot read %s: %v", source.path, err)
			continue
		}
		definitions := object(document["hooks"])
		if definitions == nil {
			logf("ignoring %s: hooks must be an object", source.path)
			continue
		}
		for event, groupsValue := range definitions {
			for _, groupValue := range array(groupsValue) {
				group := object(groupValue)
				if group == nil {
					continue
				}
				matcher := stringsValue(group["matcher"], ".*")
				if matcher == "" {
					matcher = ".*"
				}
				for _, itemValue := range array(group["hooks"]) {
					item := object(itemValue)
					if item == nil || item["type"] != "command" {
						continue
					}
					command := stringsValue(item["command"], "")
					if command == "" {
						logf("ignoring command without text in %s", source.path)
						continue
					}
					timeout := defaultTimeout
					switch value := item["timeout"].(type) {
					case float64:
						timeout = time.Duration(value * float64(time.Second))
					case string:
						if n, err := strconv.ParseFloat(value, 64); err == nil {
							timeout = time.Duration(n * float64(time.Second))
						}
					}
					if timeout < 100*time.Millisecond {
						timeout = 100 * time.Millisecond
					}
					result = append(result, hook{Event: event, Command: command, Matcher: matcher, Timeout: timeout, Source: source.path, Owner: source.owner})
				}
			}
		}
	}
	return result
}

func matches(h hook, toolName string) bool {
	re, err := regexp.Compile(h.Matcher)
	if err != nil {
		logf("invalid matcher in %s: %v", h.Source, err)
		return false
	}
	return re.MatchString(toolName)
}

func runHook(ctx context.Context, h hook, payload map[string]any, cwd string) hookResult {
	ctx, cancel := context.WithTimeout(ctx, h.Timeout)
	defer cancel()
	shell, args := "sh", []string{"-c", h.Command}
	if runtime.GOOS == "windows" {
		shell, args = "cmd.exe", []string{"/d", "/s", "/c", h.Command}
	}
	command := exec.CommandContext(ctx, shell, args...)
	command.Dir = cwd
	payloadBytes, _ := json.Marshal(payload)
	command.Stdin = bytes.NewReader(payloadBytes)
	var stdout, stderr strings.Builder
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if stderr.Len() > 0 {
		logf("%s: %s", h.Source, strings.TrimSpace(stderr.String()))
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		logf("timeout after %s: %s", h.Timeout, h.Source)
		return hookResult{Code: 0}
	}
	code := 0
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			code = exit.ExitCode()
		} else {
			code = 1
		}
	}
	result := hookResult{Code: code, Output: strings.TrimSpace(stdout.String())}
	if result.Output != "" {
		_ = json.Unmarshal([]byte(result.Output), &result.Response)
	}
	return result
}

func (a *app) snapshot() []hook {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return append([]hook(nil), a.hooks...)
}
func (a *app) reload() {
	a.mu.Lock()
	a.hooks = loadHooks(a.cwd)
	count := len(a.hooks)
	a.mu.Unlock()
	logf("loaded %d hook(s)", count)
}
func (a *app) forEvent(event, toolName string) []hook {
	var out []hook
	for _, h := range a.snapshot() {
		if h.Event == event && (toolName == "" || matches(h, toolName)) {
			out = append(out, h)
		}
	}
	return out
}

func (a *app) preTool(toolName string, input json.RawMessage) (bool, string) {
	payload := map[string]any{"hook_event_name": "PreToolUse", "cwd": a.cwd, "tool_name": toolName, "tool_input": json.RawMessage(input)}
	for _, h := range a.forEvent("PreToolUse", toolName) {
		result := runHook(context.Background(), h, payload, a.cwd)
		reason := "blocked by hook"
		if result.Response != nil {
			reason = stringsValue(result.Response["reason"], reason)
		}
		if result.Code == 2 || (result.Response != nil && result.Response["decision"] == "block") {
			return false, reason
		}
	}
	return true, ""
}

func (a *app) event(event string, payload map[string]any) {
	for _, h := range a.forEvent(event, "") {
		_ = runHook(context.Background(), h, payload, a.cwd)
	}
}

const hooksPanelID = "hooks-main"

func (a *app) command(args string) ext.Response {
	args = strings.TrimSpace(args)
	switch {
	case args == "":
		return ext.OpenPanel(hooksPanelID, "Hooks", a.panelLines(), a.panelFooter())
	case strings.EqualFold(args, "locations"):
		return ext.Display(a.formatLocations())
	case strings.EqualFold(args, "add") || strings.EqualFold(args, "help"):
		return ext.OpenPanel(hooksPanelID, "Add hook", a.beginAddPanel(), a.panelFooter())
	default:
		parts := strings.Fields(args)
		if len(parts) < 3 || !strings.EqualFold(parts[0], "add") {
			return ext.Display("Usage: /hooks add <hook-event> <command> (try /hooks for the interactive panel)")
		}
		event, command := parts[1], strings.TrimSpace(strings.TrimPrefix(args, parts[0]+" "))
		command = strings.TrimSpace(strings.TrimPrefix(command, event))
		path, err := a.addLocalHook(event, command)
		if err != nil {
			return ext.Display("Could not add hook: " + err.Error())
		}
		a.reload()
		return ext.OpenPanel(hooksPanelID, "Hooks", a.panelLinesWithStatus("Added "+event+" hook to "+path), a.panelFooter())
	}
}

func (a *app) panelLines() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.panelLinesLocked("")
}

func (a *app) panelLinesWithStatus(status string) []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.panelLinesLocked(status)
}

func (a *app) panelLinesLocked(status string) []string {
	if a.panelMode == "event" || a.panelMode == "command" {
		field := "Hook event"
		value := a.panelEvent
		if a.panelMode == "command" {
			field = "Command"
			value = a.panelCommand
		}
		lines := []string{
			"  Add a project-local hook",
			"",
			"▸ " + field + ": " + value + "▌",
		}
		if a.panelMode == "event" {
			options := a.panelEventOptionsLocked()
			if len(options) > 0 {
				lines = append(lines, "", "  Matching events:")
				for i, option := range options {
					marker := "  "
					if i == a.panelEventChoice {
						marker = "▸ "
					}
					lines = append(lines, marker+option)
				}
			} else {
				lines = append(lines, "", "  No matching hook event")
			}
			lines = append(lines, "", "  ↑/↓ choose, Enter continue")
		} else {
			lines = append(lines, "", "  Press Enter to save")
		}
		return lines
	}

	hooks := a.hooks
	lines := []string{"  Press a to add a project-local hook", "  Press r to reload, Esc to close", ""}
	if status != "" {
		lines = append(lines, "  ✓ "+status, "")
	}
	if len(hooks) == 0 {
		return append(lines, "  No active hooks found.")
	}
	for _, h := range hooks {
		owner := ""
		if h.Owner != "" {
			owner = " (" + h.Owner + ")"
		}
		lines = append(lines, fmt.Sprintf("  %s [%s]%s", h.Event, h.Matcher, owner), "    "+h.Command)
	}
	return lines
}

func (a *app) panelEventOptionsLocked() []string {
	query := strings.ToLower(strings.TrimSpace(a.panelEvent))
	options := make([]string, 0, len(hookEvents))
	for _, event := range hookEvents {
		if query == "" || strings.HasPrefix(strings.ToLower(event), query) {
			options = append(options, event)
		}
	}
	if a.panelEventChoice >= len(options) {
		a.panelEventChoice = 0
	}
	return options
}

func (a *app) panelFooter() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.panelMode != "" {
		return "type text - enter continue/save - backspace delete - esc cancel"
	}
	return "a add - r reload - esc close"
}

func (a *app) beginAddPanel() []string {
	a.mu.Lock()
	a.panelMode = "event"
	a.panelEvent = ""
	a.panelEventChoice = 0
	a.panelCommand = ""
	lines := a.panelLinesLocked("")
	a.mu.Unlock()
	return lines
}

func (a *app) handlePanelKey(key, text string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if key == "esc" {
		a.panelMode, a.panelEvent, a.panelCommand = "", "", ""
		a.panelEventChoice = 0
		a.ext.ClosePanel(hooksPanelID)
		return
	}
	if a.panelMode == "" {
		if key == "rune" {
			switch strings.ToLower(text) {
			case "a":
				a.panelMode, a.panelEvent, a.panelCommand = "event", "", ""
				a.panelEventChoice = 0
			case "r":
				a.hooks = loadHooks(a.cwd)
			}
		}
		a.ext.RenderPanel(hooksPanelID, "Hooks", a.panelLinesLocked(""), a.panelFooterLocked())
		return
	}

	value := &a.panelEvent
	if a.panelMode == "command" {
		value = &a.panelCommand
	}
	switch key {
	case "up", "down":
		if a.panelMode == "event" {
			options := a.panelEventOptionsLocked()
			if len(options) > 0 {
				if key == "up" {
					a.panelEventChoice = (a.panelEventChoice + len(options) - 1) % len(options)
				} else {
					a.panelEventChoice = (a.panelEventChoice + 1) % len(options)
				}
			}
		}
	case "backspace":
		runes := []rune(*value)
		if len(runes) > 0 {
			*value = string(runes[:len(runes)-1])
		}
		a.panelEventChoice = 0
	case "rune":
		*value += text
		a.panelEventChoice = 0
	case "enter":
		if a.panelMode == "event" {
			options := a.panelEventOptionsLocked()
			if len(options) > 0 {
				a.panelEvent = options[a.panelEventChoice]
				a.panelMode = "command"
			}
		} else if strings.TrimSpace(a.panelCommand) != "" {
			if _, err := a.addLocalHook(a.panelEvent, strings.TrimSpace(a.panelCommand)); err != nil {
				a.ext.Notify("error", "hooks: "+err.Error())
			} else {
				a.hooks = loadHooks(a.cwd)
				a.panelMode, a.panelEvent, a.panelCommand = "", "", ""
				a.panelEventChoice = 0
			}
		}
	}
	a.ext.RenderPanel(hooksPanelID, "Hooks", a.panelLinesLocked(""), a.panelFooterLocked())
}

func (a *app) panelFooterLocked() string {
	if a.panelMode != "" {
		return "type text - enter continue/save - backspace delete - esc cancel"
	}
	return "a add - r reload - esc close"
}

func (a *app) formatHooks() string {
	hooks := a.snapshot()
	if len(hooks) == 0 {
		return "No active hooks found."
	}
	grouped := map[string][]hook{}
	var order []string
	for _, h := range hooks {
		if _, ok := grouped[h.Source]; !ok {
			order = append(order, h.Source)
		}
		grouped[h.Source] = append(grouped[h.Source], h)
	}
	var b strings.Builder
	b.WriteString("Active hooks (merged from all discovered files):")
	for _, source := range order {
		b.WriteString("\n\n" + source)
		for _, h := range grouped[source] {
			owner := ""
			if h.Owner != "" {
				owner = " (" + h.Owner + ")"
			}
			fmt.Fprintf(&b, "\n  - %s [%s]%s %s", h.Event, h.Matcher, owner, h.Command)
		}
	}
	return b.String()
}
func (a *app) formatLocations() string {
	var b strings.Builder
	b.WriteString("Valid hook locations (checked in this order):")
	for i, path := range configPaths(a.cwd) {
		fmt.Fprintf(&b, "\n%d. %s", i+1, path)
	}
	sources := extensionSources()
	if len(sources) > 0 {
		b.WriteString("\n\nExtension hook files:")
		for _, s := range sources {
			b.WriteString("\n" + s.owner + ": " + s.path)
		}
	}
	b.WriteString("\n\nThe local hook command writes to: " + localConfigPath(a.cwd))
	return b.String()
}
func localConfigPath(cwd string) string { return filepath.Join(cwd, ".zot", "zot-cluade-hooks.json") }

func (a *app) addLocalHook(event, command string) (string, error) {
	path := localConfigPath(a.cwd)
	document := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &document); err != nil {
			return path, fmt.Errorf("cannot read %s: %w", path, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return path, err
	}
	hooks := object(document["hooks"])
	if hooks == nil {
		hooks = map[string]any{}
	}
	groups := array(hooks[event])
	group := map[string]any{"matcher": ".*", "hooks": []any{}}
	if len(groups) > 0 {
		if existing := object(groups[0]); existing != nil {
			group = existing
		}
	}
	items := array(group["hooks"])
	group["hooks"] = append(items, map[string]any{"type": "command", "command": command})
	if len(groups) > 0 {
		groups[0] = group
	} else {
		groups = []any{group}
	}
	hooks[event] = groups
	document["hooks"] = hooks
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return path, err
	}
	data, _ := json.MarshalIndent(document, "", "  ")
	return path, os.WriteFile(path, append(data, '\n'), 0o644)
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "list" {
		cwd, _ := os.Getwd()
		for _, h := range loadHooks(cwd) {
			fmt.Printf("%s\t%s\t%s\t%s\n", h.Event, h.Matcher, h.Source, h.Command)
		}
		return
	}
	a := &app{cwd: "", hooks: nil}
	a.ext = ext.New(name, version)
	a.ext.OnHello(func(info ext.HostInfo) {
		a.cwd = info.CWD
		a.reload()
		a.ext.Command("hooks", "show active hooks, valid locations, or add a local hook", a.command)
		a.ext.OnPanelKey(hooksPanelID, a.handlePanelKey, func() {
			a.mu.Lock()
			a.panelMode, a.panelEvent, a.panelCommand = "", "", ""
			a.panelEventChoice = 0
			a.mu.Unlock()
		})
		a.ext.On("session_start", func(ev ext.Event) {
			a.event("SessionStart", map[string]any{"hook_event_name": "SessionStart", "cwd": a.cwd})
		})
		a.ext.On("turn_end", func(ev ext.Event) {
			a.event("Stop", map[string]any{"hook_event_name": "Stop", "cwd": a.cwd, "stop_reason": ev.Stop, "error": ev.Error})
		})
		a.ext.On("tool_call", func(ev ext.Event) {
			a.event("Notification", map[string]any{"hook_event_name": "Notification", "cwd": a.cwd, "tool_name": ev.ToolName, "tool_input": json.RawMessage(ev.ToolArgs)})
		})
		a.ext.On("assistant_message", func(ev ext.Event) {
			a.event("Notification", map[string]any{"hook_event_name": "Notification", "cwd": a.cwd, "message": ev.Text})
		})
		a.ext.InterceptToolCall(func(toolName string, args json.RawMessage) (bool, string) {
			return a.preTool(toolName, args)
		})
	})
	// Registering the subscription through On calls makes the SDK emit the exact
	// lifecycle names supported by the current zot host.
	if err := a.ext.Run(); err != nil && !errors.Is(err, io.EOF) {
		logf("fatal: %v", err)
		os.Exit(1)
	}
}
