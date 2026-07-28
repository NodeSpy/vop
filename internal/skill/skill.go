// Package skill holds the agent-facing instructions for using vop, embedded
// into the binary.
//
// The full guide is never written to disk: `vop skill` prints it from the
// embedded copy, so what an agent reads always matches the vop it is running.
// Only the stub — a short pointer that tells the agent to run `vop skill` — is
// installed into an agent's skills directory, and it is written to change
// rarely enough that a stale copy stays correct.
package skill

import (
	"embed"
	"strconv"
	"strings"
)

//go:embed stub.md guide.md
var files embed.FS

// StubVersion identifies the installed stub's content. Bump it when stub.md
// changes materially, so `vop skill status` can report an installed copy as
// out of date. It is not tied to the vop version: the whole point of the stub
// is that it survives releases untouched.
const StubVersion = 1

// InstalledName is the filename an agent's skills directory expects.
const InstalledName = "SKILL.md"

// Vars are the live values interpolated into the guide at print time. Anything
// in here is a fact about the running binary or the current directory, which is
// exactly what static documentation gets wrong.
type Vars struct {
	Version       string
	ProfileStatus string
	ExitCooldown  string
	ExitLocked    string
}

// Guide returns the full agent instructions with vars substituted.
func Guide(v Vars) string {
	return strings.NewReplacer(
		"{{VERSION}}", v.Version,
		"{{PROFILE_STATUS}}", v.ProfileStatus,
		"{{EXIT_COOLDOWN}}", v.ExitCooldown,
		"{{EXIT_LOCKED}}", v.ExitLocked,
	).Replace(read("guide.md"))
}

// Stub returns the file installed into an agent's skills directory.
func Stub() string {
	return strings.ReplaceAll(read("stub.md"), "{{STUB_VERSION}}", strconv.Itoa(StubVersion))
}

// StubVersionOf reports the stub version recorded in an installed copy, and
// whether one was found. Used to tell "installed, current" from "installed by
// an older vop" without diffing the whole file — the user is allowed to have
// edited prose, and that shouldn't read as out of date.
func StubVersionOf(content string) (int, bool) {
	const marker = "vop skill stub v"
	i := strings.Index(content, marker)
	if i < 0 {
		return 0, false
	}
	rest := content[i+len(marker):]
	end := 0
	for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
		end++
	}
	n, err := strconv.Atoi(rest[:end])
	if err != nil {
		return 0, false
	}
	return n, true
}

func read(name string) string {
	data, err := files.ReadFile(name)
	if err != nil {
		// Unreachable: the files are embedded at build time, so a failure here
		// means the binary itself is malformed.
		panic("skill: missing embedded file " + name + ": " + err.Error())
	}
	return string(data)
}
