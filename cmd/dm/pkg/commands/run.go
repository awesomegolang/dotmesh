package commands

import (
	"fmt"
	"io"
	"os"

	//"github.com/dotmesh-io/dotmesh/cmd/dm/pkg/remotes"
	"github.com/spf13/cobra"
)

var runInputDots *[]string
var runOutputDots *[]string
var runModelDot string
var runWorkDir string

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

	runInputDots = cmd.Flags().StringSliceP("input-dot", "i", []string{},
		"Specify the input data dots to be mounted in the workdir.")

	runOutputDots = cmd.Flags().StringSliceP("output-dot", "o", []string{},
		"Specify the output data dots to be mounted in the workdir.")

	cmd.Flags().StringVarP(
		&runModelDot, "model-dot", "m", "",
		"Specify the model dot that has the code for this run.",
	)

	cmd.Flags().StringVarP(
		&runWorkDir, "workdir", "w", "/work",
		"Specify the path used to mount the input, output and model dots.",
	)

	return cmd
}

func runJob(cmd *cobra.Command, args []string, out io.Writer) error {
	fmt.Fprintf(out, "args %+v\n", args)
	fmt.Fprintf(out, "runInputDots %+v\n", runInputDots)
	fmt.Fprintf(out, "runOutputDots %+v\n", runOutputDots)
	fmt.Fprintf(out, "runModelDot %+v\n", runModelDot)
	fmt.Fprintf(out, "runDataDir %+v\n", runWorkDir)
	return nil
}
