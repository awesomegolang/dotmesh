package commands

import (
	"fmt"
	"io"
	"os"

	//"github.com/dotmesh-io/dotmesh/cmd/dm/pkg/remotes"
	"github.com/spf13/cobra"
)

var runDataVolumes *[]string

func NewCmdRun(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <dot> <mountpoint>",
		Short: "Help for dm run.",
		Long:  `Help for dm run.`,

		Run: func(cmd *cobra.Command, args []string) {
			err := runJob(cmd, args, out)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		},
	}

	runDataVolumes = cmd.PersistentFlags().StringSliceP("data", "d", []string{},
		"Specify the data dots used for a job run command.")

	return cmd
}

func runJob(cmd *cobra.Command, args []string, out io.Writer) error {
	fmt.Fprintf(out, "params %+v -- %+v\n", args, runDataVolumes)
	return nil
}
