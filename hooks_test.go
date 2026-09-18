package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/patriceckhart/zot/packages/agent/ext"
)

func TestEventPayloadIncludesEffectiveToolResultDetails(t *testing.T) {
	a := &app{cwd: "/tmp/project"}
	executed := true
	ev := ext.Event{
		Name: "tool_result", SessionID: "session-1", Sequence: 7,
		ToolID: "call-1", ToolName: "bash", ToolArgs: json.RawMessage(`{"command":"printf ok"}`),
		Status: "completed", Executed: &executed,
	}
	payload := a.eventPayload("PostToolUse", ev)
	if payload["hook_event_name"] != "PostToolUse" || payload["tool_name"] != "bash" {
		t.Fatalf("got unexpected payload: %#v", payload)
	}
	if payload["tool_status"] != "completed" || payload["tool_executed"] != true {
		t.Fatalf("missing tool outcome fields: %#v", payload)
	}
	args, ok := payload["tool_input"].(json.RawMessage)
	if !ok || string(args) != `{"command":"printf ok"}` {
		t.Fatalf("got tool input %q, want effective arguments", args)
	}
}

func TestEventPayloadIncludesLifecycleFields(t *testing.T) {
	a := &app{cwd: "/tmp/project"}
	count, tokens := 12, 345
	payload := a.eventPayload("PreCompact", ext.Event{
		Name: "pre_compact", CompactionID: "compact-1", MessageCount: &count, TokenEstimate: &tokens,
	})
	if payload["compaction_id"] != "compact-1" || payload["message_count"] != 12 || payload["token_estimate"] != 345 {
		t.Fatalf("got unexpected lifecycle payload: %#v", payload)
	}
}

func TestParseHookDocumentBuildsTypedCommandHooks(t *testing.T) {
	data := []byte(`{
		"name": "unrelated setting",
		"hooks": {
			"PreToolUse": [{
				"matcher": "^bash$",
				"hooks": [{
					"type": "command",
					"command": "printf hook",
					"timeout": 1.5,
					"args": ["ignored by current runtime"]
				}]
			}]
		}
	}`)

	hooks := parseHookDocument(data, "settings.json", "")
	if len(hooks) != 1 {
		t.Fatalf("got %d hooks, want 1", len(hooks))
	}
	got := hooks[0]
	if got.Event != "PreToolUse" || got.Command != "printf hook" || got.Matcher != "^bash$" {
		t.Fatalf("got unexpected hook: %+v", got)
	}
	if got.Timeout != 1500*time.Millisecond {
		t.Fatalf("got timeout %s, want 1.5s", got.Timeout)
	}
	if !matches(got, "bash") || matches(got, "edit") {
		t.Fatal("compiled matcher does not match the expected tool names")
	}
}

func TestParseHookDocumentKeepsValidSiblings(t *testing.T) {
	data := []byte(`{
		"hooks": {
			"SessionStart": [{
				"hooks": [
					{"type": "http", "url": "https://example.test"},
					{"type": "command", "command": "printf valid"},
					{"type": "command", "command": ""}
				]
			}]
		}
	}`)

	hooks := parseHookDocument(data, "settings.json", "")
	if len(hooks) != 1 || hooks[0].Command != "printf valid" {
		t.Fatalf("got valid hooks %+v, want one command hook", hooks)
	}
}

func TestParseHookDocumentPreservesCurrentTimeoutCompatibility(t *testing.T) {
	data := []byte(`{
		"hooks": {
			"Stop": [{
				"hooks": [
					{"type": "command", "command": "printf default"},
					{"type": "command", "command": "printf minimum", "timeout": 0.01},
					{"type": "command", "command": "printf string", "timeout": "0.2"}
				]
			}]
		}
	}`)

	hooks := parseHookDocument(data, "settings.json", "")
	if len(hooks) != 3 {
		t.Fatalf("got %d hooks, want 3", len(hooks))
	}
	if hooks[0].Timeout != defaultTimeout {
		t.Fatalf("got default timeout %s, want %s", hooks[0].Timeout, defaultTimeout)
	}
	if hooks[1].Timeout != 100*time.Millisecond {
		t.Fatalf("got minimum timeout %s, want 100ms", hooks[1].Timeout)
	}
	if hooks[2].Timeout != 200*time.Millisecond {
		t.Fatalf("got string timeout %s, want 200ms", hooks[2].Timeout)
	}
}
