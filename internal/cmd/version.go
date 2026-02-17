package cmd

import (
	"fmt"

	"github.com/NodeSpy/vop/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println(version.Full())
		},
	}
}
