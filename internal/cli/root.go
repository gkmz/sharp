package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/hank/sharp/internal/tools"
	"github.com/hank/sharp/internal/tui"
	"github.com/hank/sharp/pkg/tool"
	"github.com/spf13/cobra"
)

func Execute() error {
	registry := tools.NewRegistry()
	root := &cobra.Command{
		Use:   "sharp",
		Short: "Interactive terminal toolkit for developers",
		Long:  "sharp is a keyboard-first developer toolkit for JSON, encoding, crypto, time, text, network, and generators.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return tui.Run(registry)
			}
			return cmd.Help()
		},
	}
	root.AddCommand(listCommand(registry))
	addToolCommands(root, registry)
	return root.Execute()
}

func listCommand(registry *tool.Registry) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available tools",
		Run: func(cmd *cobra.Command, args []string) {
			for _, t := range registry.All() {
				fmt.Fprintf(cmd.OutOrStdout(), "%-18s %-12s %s\n", t.ID(), t.Category(), t.Description())
			}
		},
	}
}

func addToolCommands(root *cobra.Command, registry *tool.Registry) {
	groups := map[string][]tool.Tool{}
	for _, t := range registry.All() {
		if !strings.Contains(t.ID(), ".") {
			root.AddCommand(toolCommand(t.ID(), t))
			continue
		}
		parts := strings.SplitN(t.ID(), ".", 2)
		groups[parts[0]] = append(groups[parts[0]], t)
	}
	var groupNames []string
	for group := range groups {
		groupNames = append(groupNames, group)
	}
	sort.Strings(groupNames)
	for _, group := range groupNames {
		groupCmd := &cobra.Command{
			Use:   group,
			Short: fmt.Sprintf("%s tools", group),
		}
		for _, currentTool := range groups[group] {
			t := currentTool
			name := strings.TrimPrefix(t.ID(), group+".")
			groupCmd.AddCommand(toolCommand(name, t))
		}
		root.AddCommand(groupCmd)
	}
}

func toolCommand(name string, t tool.Tool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   name + " [input-or-file]",
		Short: t.Description(),
		RunE: func(cmd *cobra.Command, args []string) error {
			options := tool.Options{}
			for _, opt := range t.Options() {
				value, err := cmd.Flags().GetString(opt.Name)
				if err != nil {
					return err
				}
				if value == "" {
					value = opt.Default
				}
				if opt.Required && value == "" {
					return fmt.Errorf("missing required option: --%s", opt.Name)
				}
				options[opt.Name] = value
			}
			input, err := resolveInput(cmd, args)
			if err != nil {
				return err
			}
			out, err := t.Run(context.Background(), tool.Input{Text: input}, options)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), out.Text)
			return nil
		},
	}
	for _, opt := range t.Options() {
		cmd.Flags().String(opt.Name, opt.Default, opt.Description)
	}
	return cmd
}

func resolveInput(cmd *cobra.Command, args []string) (string, error) {
	if len(args) > 0 {
		candidate := strings.Join(args, " ")
		if len(args) == 1 {
			if info, err := os.Stat(args[0]); err == nil && !info.IsDir() {
				b, err := os.ReadFile(args[0])
				if err != nil {
					return "", err
				}
				return string(b), nil
			}
		}
		return candidate, nil
	}
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}
	if stat.Mode()&os.ModeCharDevice == 0 {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return strings.TrimSuffix(string(b), "\n"), nil
	}
	return "", nil
}
