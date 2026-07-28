package cmd

import (
	"fmt"

	"github.com/NodeSpy/vop/internal/ui"
	"github.com/spf13/cobra"
)

// newProfileCmd reports the profile vop would use here, in a form that can be
// carried elsewhere. Directory-based resolution is invisible to anything that
// isn't standing in that directory — an agent that runs one command in a repo
// and the next one in /tmp loses the profile silently. Printing the name on
// stdout, alone, lets a caller pin it once:
//
//	export VOP_PROFILE=$(vop profile)
func newProfileCmd() *cobra.Command {
	var export bool
	var quiet bool

	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Print the profile vop would use here",
		Long: "Print the profile vop resolves in the current directory, from " +
			"VOP_PROFILE / AGENT_DECK_VOP_PROFILE or a .vop file.\n\n" +
			"The name goes to stdout on its own line; where it came from goes to " +
			"stderr. Exits non-zero when no profile can be resolved.",
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if quiet {
				ui.Quiet = true
			}

			r := defaultProfile()
			if r.Name == "" {
				return fmt.Errorf("no profile resolved here.\n" +
					"  Pin one: export VOP_PROFILE=<profile>\n" +
					"  Or create a .vop file containing the profile name in this directory or any parent\n" +
					"  See configured profiles: vop ls")
			}

			// Resolution is name-only, so a stale .vop or a typo'd env var
			// resolves fine and then fails later inside another command. Say
			// so here instead, without derailing the output.
			if c, err := loadConfig(); err == nil {
				if _, ok := c.Profiles[r.Name]; !ok {
					ui.Warn("Profile '%s' (from %s) is not configured — run vop ls", r.Name, r.Describe())
				}
			}

			ui.Info("Profile: %s (from %s)", r.Name, r.Describe())

			if export {
				fmt.Printf("export VOP_PROFILE=%s\n", r.Name)
			} else {
				fmt.Println(r.Name)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&export, "export", false, "print as a shell export statement for eval")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress the source line on stderr")

	return cmd
}
