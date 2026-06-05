package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/gkmz/sharp/internal/tools"
	"github.com/gkmz/sharp/internal/tui"
	"github.com/gkmz/sharp/pkg/tool"
	"github.com/spf13/cobra"
)

// Version is the semantic version printed by `sharp version`.
// Release builds can override it with -ldflags "-X github.com/gkmz/sharp/internal/cli.Version=X.Y.Z".
var Version = "0.0.0"

const logo = `   _____ __  _____    ____  ____
  / ___// / / /   |  / __ \/ __ \
  \__ \/ /_/ / /| | / /_/ / /_/ /
 ___/ / __  / ___ |/ _, _/ ____/
/____/_/ /_/_/  |_/_/ |_/_/
        local dev toolkit`

// Execute builds and runs the root CLI command.
func Execute() error {
	registry := tools.NewRegistry()
	root := NewRootCommand(registry)
	return root.Execute()
}

// NewRootCommand builds the root command around registry.
func NewRootCommand(registry *tool.Registry) *cobra.Command {
	root := &cobra.Command{
		Use:   "sharp",
		Short: "Interactive terminal toolkit for developers",
		Long:  logo + "\n\nsharp is a keyboard-first developer toolkit for JSON, encoding, crypto, time, text, network, conversion, JWT inspection, and generators.",
		// 运行期错误由 main 统一打印一次，避免脚本里出现 Cobra usage 和重复错误文本。
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return tui.Run(registry)
			}
			return cmd.Help()
		},
	}
	root.AddCommand(listCommand(registry))
	root.AddCommand(versionCommand())
	addToolCommands(root, registry)
	return root
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

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), logo)
			fmt.Fprintf(cmd.OutOrStdout(), "\nversion: %s\n", Version)
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
		Use:                name + " [input-or-file]",
		Short:              t.Description(),
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if toolHelpRequested(args) {
				return cmd.Help()
			}
			options, inputArgs, err := parseToolCommandArgs(args, t.Options())
			if err != nil {
				return err
			}
			input, err := resolveInput(cmd, inputArgs)
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

func toolHelpRequested(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func parseToolCommandArgs(args []string, optionDefs []tool.Option) (tool.Options, []string, error) {
	options := tool.Options{}
	known := map[string]tool.Option{}
	for _, opt := range optionDefs {
		options[opt.Name] = opt.Default
		known[opt.Name] = opt
	}

	inputArgs := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			// `--` 是 POSIX 风格的选项结束符；后续内容即使以短横线开头也必须保留为原始输入。
			inputArgs = append(inputArgs, args[i+1:]...)
			break
		}
		if strings.HasPrefix(arg, "--") && len(arg) > 2 {
			raw := strings.TrimPrefix(arg, "--")
			name, value, hasValue := strings.Cut(raw, "=")
			opt, ok := known[name]
			if name == "" || !ok {
				return nil, nil, fmt.Errorf("unknown option: --%s", name)
			}
			if !hasValue {
				if i+1 >= len(args) {
					return nil, nil, fmt.Errorf("missing value for option: --%s", name)
				}
				i++
				value = args[i]
			}
			if value == "" {
				value = opt.Default
			}
			options[name] = value
			continue
		}
		inputArgs = append(inputArgs, arg)
	}

	for _, opt := range optionDefs {
		if opt.Required && options[opt.Name] == "" {
			return nil, nil, fmt.Errorf("missing required option: --%s", opt.Name)
		}
	}
	return options, inputArgs, nil
}

func resolveInput(cmd *cobra.Command, args []string) (string, error) {
	if len(args) > 0 {
		candidate := strings.Join(args, " ")
		if len(args) == 1 {
			// 单个参数优先按文件路径解析；多个参数保持为用户输入文本，适合 `sharp hash sha256 hello world`。
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
		// 只去掉 shell 管道常见的单个尾随换行，保留用户输入主体中的空白和换行。
		return strings.TrimSuffix(string(b), "\n"), nil
	}
	return "", nil
}
