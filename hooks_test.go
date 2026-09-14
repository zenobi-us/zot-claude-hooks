package main

import (
	"testing"
	"time"
)

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
