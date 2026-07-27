package cmd

import (
	"fmt"
	"os"

	"github.com/NodeSpy/vop/internal/config"
	"github.com/NodeSpy/vop/internal/ui"
)

// profileEnvVars are the environment variables that pin a session to a
// profile, in precedence order.
//
// VOP_PROFILE is exported by `vop shell`, so it reflects the shell you are
// actually standing in. AGENT_DECK_VOP_PROFILE is set by agent-deck to lock
// an agent session to one account.
var profileEnvVars = []string{"VOP_PROFILE", "AGENT_DECK_VOP_PROFILE"}

// profileSource describes where a resolved profile name came from, for
// display and for deciding whether to announce it.
type profileSource string

const (
	sourceArg profileSource = "argument"
	sourceEnv profileSource = "env"
	sourceDir profileSource = "dir"
)

// resolvedProfile is the outcome of default-profile resolution.
type resolvedProfile struct {
	Name   string
	Source profileSource
	Origin string // env var name, or path to the .vop file
}

// Describe renders the origin for display, e.g. "VOP_PROFILE" or "./.vop".
func (r resolvedProfile) Describe() string {
	if r.Source == sourceDir {
		return displayPath(r.Origin)
	}
	return r.Origin
}

// defaultProfile determines which profile to use when the user didn't name
// one on the command line. Precedence:
//
//  1. VOP_PROFILE / AGENT_DECK_VOP_PROFILE
//  2. a .vop file in the current directory or an ancestor
//
// The environment deliberately wins. A .vop file is repository content, and
// letting repository content silently redirect credentials to a different
// AWS account would defeat the point of pinning a session to a profile. When
// a .vop file is passed over for that reason, the user is told.
//
// Returns an empty Name when nothing is configured; callers fall back to
// their own behaviour (an interactive picker, or an error).
func defaultProfile() resolvedProfile {
	for _, key := range profileEnvVars {
		if v := os.Getenv(key); v != "" {
			warnIfDirProfileIgnored(v, key)
			return resolvedProfile{Name: v, Source: sourceEnv, Origin: key}
		}
	}

	dp := findDirProfile()
	if dp == nil {
		return resolvedProfile{}
	}
	return resolvedProfile{Name: dp.Name, Source: sourceDir, Origin: dp.Path}
}

// findDirProfile looks for a .vop file, reporting malformed ones as warnings
// rather than errors — a broken marker file shouldn't make vop unusable when
// the user can still name a profile explicitly.
func findDirProfile() *config.DirProfile {
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}
	dp, err := config.FindDirProfile(cwd)
	if err != nil {
		ui.Warn("Ignoring %s: %s", config.DirProfileFilename, err)
		return nil
	}
	return dp
}

// warnIfDirProfileIgnored tells the user when a .vop file asked for a
// different profile than the one the environment pinned. Silence here would
// be the confusing case: commands would quietly use an account the directory
// didn't ask for.
func warnIfDirProfileIgnored(active, envKey string) {
	dp := findDirProfile()
	if dp == nil || dp.Name == active {
		return
	}
	ui.Warn("%s requests profile '%s' — ignored, this session is pinned to '%s' by %s",
		displayPath(dp.Path), dp.Name, active, envKey)
}

// announceProfile prints which profile a command settled on when it wasn't
// named explicitly, so an implicit default is never invisible.
func announceProfile(r resolvedProfile) {
	if r.Name == "" || r.Source == sourceArg {
		return
	}
	ui.Info("Profile: %s (from %s)", r.Name, r.Describe())
}

// profileOrDefault returns the profile named in args, or the resolved
// default when args is empty. The returned resolvedProfile has an empty
// Name if neither is available.
func profileOrDefault(args []string) resolvedProfile {
	if len(args) > 0 && args[0] != "" {
		return resolvedProfile{Name: args[0], Source: sourceArg}
	}
	return defaultProfile()
}

// requireDefaultProfile resolves a default profile or explains how to set
// one. Used by commands that have no interactive fallback.
func requireDefaultProfile(args []string, cmdName string) (resolvedProfile, error) {
	r := profileOrDefault(args)
	if r.Name == "" {
		return r, fmt.Errorf("no profile specified and no default found.\n"+
			"  Name one: vop %s <profile>\n"+
			"  Or create a %s file containing the profile name in this directory or the repo root",
			cmdName, config.DirProfileFilename)
	}
	announceProfile(r)
	return r, nil
}

// displayPath shortens a path under $HOME to ~/… for readable output.
func displayPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if len(path) > len(home)+1 && path[:len(home)] == home && path[len(home)] == os.PathSeparator {
		return "~" + string(os.PathSeparator) + path[len(home)+1:]
	}
	return path
}
