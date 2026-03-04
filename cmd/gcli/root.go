package gcli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/geekjourneyx/gcli/pkg/config"
	"github.com/geekjourneyx/gcli/pkg/output"
)

// Version is overridden at build time via ldflags.
var Version = "dev"

type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

type State struct {
	Output       string
	OutputFormat output.Format
	JSON         bool
	Verbose      bool
	Timeout      time.Duration
}

func NewRootCommand(streams IOStreams) (*cobra.Command, *State) {
	state := &State{
		Output:       string(output.FormatJSON),
		OutputFormat: output.FormatJSON,
		Timeout:      30 * time.Second,
	}

	root := &cobra.Command{
		Use:           "gcli",
		Short:         "Gmail read-only CLI built on google-api-go-client",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := config.LoadStartupEnvFile(); err != nil {
				return err
			}
			if state.JSON {
				state.Output = string(output.FormatJSON)
			}
			parsed, err := output.ParseFormat(state.Output)
			if err != nil {
				return err
			}
			state.OutputFormat = parsed
			return nil
		},
	}

	root.Version = Version
	root.SetVersionTemplate("{{printf \"%s\\n\" .Version}}")

	root.PersistentFlags().StringVar(&state.Output, "output", string(output.FormatJSON), "Output format: json|table")
	root.PersistentFlags().BoolVar(&state.JSON, "json", false, "Alias for --output json")
	root.PersistentFlags().BoolVar(&state.Verbose, "verbose", false, "Enable verbose logging")
	root.PersistentFlags().DurationVar(&state.Timeout, "timeout", 30*time.Second, "Command timeout")

	root.AddCommand(newAuthCommand(state, streams))
	root.AddCommand(newMailCommand(state, streams))
	root.AddCommand(newVersionCommand(state, streams))

	return root, state
}

func newVersionCommand(state *State, streams IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print gcli version",
		RunE: func(cmd *cobra.Command, args []string) error {
			data := map[string]string{"version": Version}
			return output.RenderSuccess(data, output.Options{Format: state.OutputFormat, Writer: streams.Out})
		},
	}
}

func commandContext(cmd *cobra.Command, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}

func mustPositiveLimit(limit int64) error {
	if limit <= 0 {
		return fmt.Errorf("limit must be > 0")
	}
	return nil
}
