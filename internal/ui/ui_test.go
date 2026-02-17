package ui

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w

	fn()

	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestInfo(t *testing.T) {
	out := captureStdout(t, func() {
		Info("hello %s", "world")
	})
	if !strings.Contains(out, ">>>") {
		t.Error("expected '>>>' prefix")
	}
	if !strings.Contains(out, "hello world") {
		t.Errorf("expected 'hello world' in output, got %q", out)
	}
}

func TestWarn(t *testing.T) {
	out := captureStderr(t, func() {
		Warn("warning %d", 42)
	})
	if !strings.Contains(out, ">>>") {
		t.Error("expected '>>>' prefix")
	}
	if !strings.Contains(out, "warning 42") {
		t.Errorf("expected 'warning 42' in output, got %q", out)
	}
}

func TestError(t *testing.T) {
	out := captureStderr(t, func() {
		Error("error: %s", "bad")
	})
	if !strings.Contains(out, ">>>") {
		t.Error("expected '>>>' prefix")
	}
	if !strings.Contains(out, "error: bad") {
		t.Errorf("expected 'error: bad' in output, got %q", out)
	}
}

func TestSuccess(t *testing.T) {
	out := captureStdout(t, func() {
		Success("done %s", "ok")
	})
	if !strings.Contains(out, "ok") {
		t.Error("expected 'ok' prefix")
	}
	if !strings.Contains(out, "done ok") {
		t.Errorf("expected 'done ok' in output, got %q", out)
	}
}

func TestColorConstants(t *testing.T) {
	// Verify ANSI codes are non-empty
	colors := map[string]string{
		"Red":    Red,
		"Green":  Green,
		"Yellow": Yellow,
		"Cyan":   Cyan,
		"Bold":   Bold,
		"Dim":    Dim,
		"Reset":  Reset,
	}
	for name, val := range colors {
		if val == "" {
			t.Errorf("expected %s to be non-empty", name)
		}
		if !strings.HasPrefix(val, "\033[") {
			t.Errorf("expected %s to start with ESC[, got %q", name, val)
		}
	}
}

func TestPrompt_WithDefault(t *testing.T) {
	// Simulate empty input (user just presses enter)
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.Write([]byte("\n"))
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	// Capture stdout (prompt text)
	out := captureStdout(t, func() {
		result := Prompt("Name", "default-val")
		if result != "default-val" {
			t.Errorf("expected 'default-val', got %q", result)
		}
	})
	if !strings.Contains(out, "Name") {
		t.Error("expected prompt to contain label")
	}
	if !strings.Contains(out, "default-val") {
		t.Error("expected prompt to show default value")
	}
}

func TestPrompt_WithInput(t *testing.T) {
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.Write([]byte("custom-value\n"))
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	_ = captureStdout(t, func() {
		result := Prompt("Name", "default")
		if result != "custom-value" {
			t.Errorf("expected 'custom-value', got %q", result)
		}
	})
}

func TestPromptYN_DefaultYes(t *testing.T) {
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.Write([]byte("\n"))
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	_ = captureStdout(t, func() {
		result := PromptYN("Continue?", true)
		if !result {
			t.Error("expected true (default yes)")
		}
	})
}

func TestPromptYN_DefaultNo(t *testing.T) {
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.Write([]byte("\n"))
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	_ = captureStdout(t, func() {
		result := PromptYN("Continue?", false)
		if result {
			t.Error("expected false (default no)")
		}
	})
}

func TestPromptYN_ExplicitYes(t *testing.T) {
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.Write([]byte("y\n"))
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	_ = captureStdout(t, func() {
		result := PromptYN("Continue?", false)
		if !result {
			t.Error("expected true for 'y' input")
		}
	})
}

func TestPromptYN_ExplicitNo(t *testing.T) {
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.Write([]byte("n\n"))
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	_ = captureStdout(t, func() {
		result := PromptYN("Continue?", true)
		if result {
			t.Error("expected false for 'n' input")
		}
	})
}
