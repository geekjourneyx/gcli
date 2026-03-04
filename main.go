package main

import (
	"os"

	"github.com/geekjourneyx/gcli/cmd/gcli"
	"github.com/geekjourneyx/gcli/pkg/errorsx"
	"github.com/geekjourneyx/gcli/pkg/output"
)

func main() {
	streams := gcli.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
	rootCmd, state := gcli.NewRootCommand(streams)

	if err := rootCmd.Execute(); err != nil {
		appErr := errorsx.From(err)
		format := output.FormatJSON
		if state != nil {
			if parsed, parseErr := output.ParseFormat(state.Output); parseErr == nil {
				format = parsed
			}
		}
		_ = output.RenderError(appErr, output.Options{Format: format, Writer: streams.ErrOut})
		os.Exit(errorsx.ExitCode(appErr))
	}
}
