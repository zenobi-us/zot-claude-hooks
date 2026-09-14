package main

import (
	"encoding/json"
	"math"
	"regexp"
	"strconv"
	"time"
)

// HookDocument is the typed envelope for the hook portion of a Claude-style
// settings file. Event names remain open because Claude Code can add events.
type HookDocument struct {
	Hooks map[string]json.RawMessage `json:"hooks"`
}

type HookGroup struct {
	Matcher string           `json:"matcher,omitempty"`
	Hooks   []HookDefinition `json:"hooks"`
}

// HookDefinition is a tagged hook handler. The extension currently executes
// command handlers. Other handler types stay visible to the parser so they can
// be reported and supported later without changing the document shape.
type HookDefinition struct {
	Type    string
	Command *CommandHook
}

type CommandHook struct {
	Type        string
	Command     string
	Args        []string
	If          string
	Timeout     *TimeoutSeconds
	Async       bool
	AsyncRewake bool
	Shell       string
}

func (h *CommandHook) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	_ = json.Unmarshal(raw["type"], &h.Type)
	_ = json.Unmarshal(raw["command"], &h.Command)
	_ = json.Unmarshal(raw["args"], &h.Args)
	_ = json.Unmarshal(raw["if"], &h.If)
	_ = json.Unmarshal(raw["timeout"], &h.Timeout)
	_ = json.Unmarshal(raw["async"], &h.Async)
	_ = json.Unmarshal(raw["asyncRewake"], &h.AsyncRewake)
	_ = json.Unmarshal(raw["shell"], &h.Shell)
	return nil
}

// TimeoutSeconds accepts numbers and numeric strings because the old parser
// accepted both forms. Invalid values fall back to the normal timeout.
type TimeoutSeconds struct {
	Value float64
	Valid bool
}

func (t *TimeoutSeconds) UnmarshalJSON(data []byte) error {
	var number float64
	if err := json.Unmarshal(data, &number); err == nil {
		t.Value, t.Valid = number, true
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		value, err := strconv.ParseFloat(text, 64)
		if err == nil {
			t.Value, t.Valid = value, true
			return nil
		}
	}
	t.Valid = false
	return nil
}

func parseHookDocument(data []byte, source, owner string) []hook {
	var document HookDocument
	if err := json.Unmarshal(data, &document); err != nil {
		logf("cannot read %s: %v", source, err)
		return nil
	}

	definitions := document.Hooks
	if definitions == nil {
		logf("ignoring %s: hooks must be an object", source)
		return nil
	}

	var result []hook
	for event, groupsRaw := range definitions {
		var groups []json.RawMessage
		if err := json.Unmarshal(groupsRaw, &groups); err != nil {
			continue
		}
		for _, groupRaw := range groups {
			group, ok := decodeGroup(groupRaw)
			if !ok {
				continue
			}
			compiled, err := regexp.Compile(group.Matcher)
			if err != nil {
				logf("invalid matcher in %s: %v", source, err)
				continue
			}
			for _, definition := range group.Hooks {
				if definition.Command == nil || definition.Command.Command == "" {
					if definition.Type == "command" {
						logf("ignoring command without text in %s", source)
					}
					continue
				}
				timeout := normalizeTimeout(definition.Command.Timeout)
				result = append(result, hook{
					Event: event, Command: definition.Command.Command,
					Matcher: group.Matcher, MatcherRE: compiled, Timeout: timeout,
					Source: source, Owner: owner,
				})
			}
		}
	}
	return result
}

func decodeGroup(data []byte) (HookGroup, bool) {
	var group HookGroup
	if err := json.Unmarshal(data, &group); err != nil {
		return HookGroup{}, false
	}
	if group.Matcher == "" {
		group.Matcher = ".*"
	}
	return group, true
}

func (g *HookGroup) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	g.Matcher = ".*"
	if matcher, ok := raw["matcher"]; ok {
		if err := json.Unmarshal(matcher, &g.Matcher); err != nil {
			g.Matcher = ".*"
		}
	}
	var handlers []json.RawMessage
	if err := json.Unmarshal(raw["hooks"], &handlers); err != nil {
		return err
	}
	for _, handler := range handlers {
		var definition HookDefinition
		if err := json.Unmarshal(handler, &definition); err == nil {
			g.Hooks = append(g.Hooks, definition)
		}
	}
	return nil
}

func decodeHandler(data []byte) (HookDefinition, bool) {
	var definition HookDefinition
	if err := json.Unmarshal(data, &definition); err != nil {
		return HookDefinition{}, false
	}
	return definition, true
}

func (h *HookDefinition) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if err := json.Unmarshal(raw["type"], &h.Type); err != nil {
		return err
	}
	if h.Type == "command" {
		var command CommandHook
		if err := json.Unmarshal(data, &command); err != nil {
			return err
		}
		h.Command = &command
	}
	return nil
}

func normalizeTimeout(seconds *TimeoutSeconds) time.Duration {
	if seconds == nil || !seconds.Valid || math.IsNaN(seconds.Value) || math.IsInf(seconds.Value, 0) {
		return defaultTimeout
	}
	timeout := time.Duration(seconds.Value * float64(time.Second))
	if timeout < 100*time.Millisecond {
		return 100 * time.Millisecond
	}
	return timeout
}
