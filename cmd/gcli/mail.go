package gcli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/geekjourneyx/gcli/pkg/errorsx"
	"github.com/geekjourneyx/gcli/pkg/gmail"
	"github.com/geekjourneyx/gcli/pkg/output"
)

func newMailCommand(state *State, streams IOStreams) *cobra.Command {
	mailCmd := &cobra.Command{
		Use:   "mail",
		Short: "Read Gmail messages",
	}
	mailCmd.AddCommand(newMailListCommand(state, streams))
	mailCmd.AddCommand(newMailSearchCommand(state, streams))
	mailCmd.AddCommand(newMailGetCommand(state, streams))
	return mailCmd
}

func newMailListCommand(state *State, streams IOStreams) *cobra.Command {
	var (
		label     string
		limit     int64
		pageToken string
		hydrate   bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List messages from a label",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mustPositiveLimit(limit); err != nil {
				return errorsx.Wrap(errorsx.CodeInputInvalid, "invalid limit", false, err)
			}
			ctx, cancel := commandContext(cmd, state.Timeout)
			defer cancel()

			client, err := newGmailClient(ctx)
			if err != nil {
				return err
			}

			res, err := client.ListMessages(ctx, gmail.ListOptions{
				Label:     label,
				PageToken: pageToken,
				Limit:     limit,
				Hydrate:   hydrate,
			})
			if err != nil {
				return err
			}
			return output.RenderSuccess(res, output.Options{Format: state.OutputFormat, Writer: streams.Out})
		},
	}

	cmd.Flags().StringVar(&label, "label", "INBOX", "Gmail label to filter")
	cmd.Flags().Int64Var(&limit, "limit", 20, "Maximum results to return")
	cmd.Flags().StringVar(&pageToken, "page-token", "", "Pagination token")
	cmd.Flags().BoolVar(&hydrate, "hydrate", false, "Fetch each message metadata for from/subject/date (extra API calls)")

	return cmd
}

func newMailSearchCommand(state *State, streams IOStreams) *cobra.Command {
	var (
		query     string
		limit     int64
		pageToken string
		hydrate   bool
	)

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search messages with Gmail q syntax",
		RunE: func(cmd *cobra.Command, args []string) error {
			queryFlag := strings.TrimSpace(query)
			queryArg := strings.TrimSpace(strings.Join(args, " "))
			if queryFlag != "" && queryArg != "" {
				return errorsx.New(errorsx.CodeInputInvalid, "use either --q or positional query, not both", false)
			}
			if queryFlag == "" {
				queryFlag = queryArg
			}
			if queryFlag == "" {
				return errorsx.New(errorsx.CodeInputInvalid, "--q or positional query is required", false)
			}
			if err := mustPositiveLimit(limit); err != nil {
				return errorsx.Wrap(errorsx.CodeInputInvalid, "invalid limit", false, err)
			}
			ctx, cancel := commandContext(cmd, state.Timeout)
			defer cancel()

			client, err := newGmailClient(ctx)
			if err != nil {
				return err
			}

			res, err := client.ListMessages(ctx, gmail.ListOptions{
				Query:     queryFlag,
				PageToken: pageToken,
				Limit:     limit,
				Hydrate:   hydrate,
			})
			if err != nil {
				return err
			}
			return output.RenderSuccess(res, output.Options{Format: state.OutputFormat, Writer: streams.Out})
		},
	}

	cmd.Flags().StringVar(&query, "q", "", "Gmail search query")
	cmd.Flags().Int64Var(&limit, "limit", 20, "Maximum results to return")
	cmd.Flags().Int64Var(&limit, "max", 20, "Alias for --limit")
	cmd.Flags().StringVar(&pageToken, "page-token", "", "Pagination token")
	cmd.Flags().StringVar(&pageToken, "page", "", "Alias for --page-token")
	cmd.Flags().BoolVar(&hydrate, "hydrate", false, "Fetch each message metadata for from/subject/date (extra API calls)")

	return cmd
}

func newMailGetCommand(state *State, streams IOStreams) *cobra.Command {
	var (
		id     string
		format string
	)

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get one message by Gmail message ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(id) == "" {
				return errorsx.New(errorsx.CodeInputInvalid, "--id is required", false)
			}
			ctx, cancel := commandContext(cmd, state.Timeout)
			defer cancel()

			client, err := newGmailClient(ctx)
			if err != nil {
				return err
			}

			res, err := client.GetMessage(ctx, id, format)
			if err != nil {
				return err
			}
			return output.RenderSuccess(res, output.Options{Format: state.OutputFormat, Writer: streams.Out})
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "Gmail message ID")
	cmd.Flags().StringVar(&format, "format", "metadata", "Message format: metadata|full|minimal|raw")
	if err := cmd.MarkFlagRequired("id"); err != nil {
		panic(fmt.Sprintf("failed to mark --id required: %v", err))
	}

	return cmd
}
