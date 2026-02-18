package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/thecoolrobot/task-agent/internal/ai"
	"github.com/thecoolrobot/task-agent/internal/asana"
	"github.com/thecoolrobot/task-agent/internal/config"
	"github.com/thecoolrobot/task-agent/internal/output"
	"github.com/thecoolrobot/task-agent/internal/tui"
)

var version = "1.0.0"

func main() {
	if err := newRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func loadConfig() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
		return config.Defaults()
	}
	return cfg
}

func newAsanaClient(cfg *config.Config) *asana.Client {
	client, err := asana.NewClient(cfg.AsanaCLIPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  asana-cli not found: %v\n", err)
		fmt.Fprintln(os.Stderr, "   Install: https://github.com/TheCoolRobot/asana-cli")
		return nil
	}
	return client
}

func doExecute(task *asana.Task, providerID, model, outDir string, cfg *config.Config) error {
	apiKey := config.GetAPIKey(cfg, providerID)
	if providerID != "ollama" && apiKey == "" {
		prov, _ := ai.GetProvider(providerID)
		return fmt.Errorf("no API key for %s — set %s or run: task-agent config", prov.Name, prov.EnvKey)
	}
	fmt.Printf("🤖 Provider : %s / %s\n", providerID, model)
	fmt.Printf("📋 Task     : %s\n", task.Name)
	fmt.Println("⚡ Running in YOLO mode...\n")
	client := ai.NewClient(providerID, model, apiKey)
	taskMD := asana.FormatTaskMarkdown(task)
	result, err := client.ExecuteTask(taskMD, func(msg string) { fmt.Println(" →", msg) })
	if err != nil {
		return fmt.Errorf("AI execution: %w", err)
	}
	outPath, err := output.Write(result, task, outDir)
	if err != nil {
		return fmt.Errorf("saving output: %w", err)
	}
	fmt.Printf("\n✅ Saved to: %s\n\n", outPath)
	fmt.Println(output.Preview(result))
	return nil
}

func printTaskTable(tasks []asana.Task) {
	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return
	}
	fmt.Printf("\n%-20s %-6s %-8s %s\n", "ID", "DONE", "PRIORITY", "NAME")
	fmt.Println(strings.Repeat("─", 70))
	for _, t := range tasks {
		id := t.GetID()
		if len(id) > 18 {
			id = id[:18]
		}
		done := "⏳"
		if t.Completed {
			done = "✅"
		}
		name := t.Name
		if len([]rune(name)) > 38 {
			name = string([]rune(name)[:37]) + "…"
		}
		fmt.Printf("%-20s %-6s %-8s %s\n", id, done, t.Priority, name)
	}
	fmt.Println()
}

func newRoot() *cobra.Command {
	root := &cobra.Command{
		Use:     "task-agent",
		Short:   "YOLO AI Task Executor for Asana",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			return tui.Run(cfg, newAsanaClient(cfg))
		},
	}
	root.AddCommand(newTUICmd(), newRunCmd(), newListCmd(), newSearchCmd(), newConfigCmd(), newProvidersCmd())
	return root
}

func newTUICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch the interactive Bubble Tea TUI (default)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			return tui.Run(cfg, newAsanaClient(cfg))
		},
	}
}

func newRunCmd() *cobra.Command {
	var providerID, model, outDir string
	cmd := &cobra.Command{
		Use:   "run <task-gid>",
		Short: "Execute a specific task by GID (no TUI)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			client := newAsanaClient(cfg)
			if client == nil {
				return fmt.Errorf("asana-cli required")
			}
			if providerID == "" {
				providerID = cfg.Provider
			}
			if model == "" {
				model = cfg.Model
			}
			if outDir == "" {
				outDir = cfg.OutputDir
			}
			fmt.Printf("🔍 Fetching task %s...\n", args[0])
			task, err := client.ViewTask(args[0])
			if err != nil {
				return err
			}
			return doExecute(task, providerID, model, outDir, cfg)
		},
	}
	cmd.Flags().StringVarP(&providerID, "provider", "p", "", "AI provider")
	cmd.Flags().StringVarP(&model, "model", "m", "", "Model name")
	cmd.Flags().StringVarP(&outDir, "output", "o", "", "Output directory")
	return cmd
}

func newListCmd() *cobra.Command {
	var project string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks in a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			client := newAsanaClient(cfg)
			if client == nil {
				return fmt.Errorf("asana-cli required")
			}
			if project == "" {
				project = cfg.ProjectGID
			}
			tasks, err := client.ListTasks(project)
			if err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(os.Stdout).Encode(tasks)
			}
			printTaskTable(tasks)
			return nil
		},
	}
	cmd.Flags().StringVarP(&project, "project", "P", "", "Project GID")
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON output")
	return cmd
}

func newSearchCmd() *cobra.Command {
	var workspace string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search tasks in a workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			client := newAsanaClient(cfg)
			if client == nil {
				return fmt.Errorf("asana-cli required")
			}
			if workspace == "" {
				workspace = cfg.WorkspaceGID
			}
			if workspace == "" {
				return fmt.Errorf("workspace GID required — run: task-agent config")
			}
			tasks, err := client.SearchTasks(workspace, args[0])
			if err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(os.Stdout).Encode(tasks)
			}
			printTaskTable(tasks)
			return nil
		},
	}
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace GID")
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON output")
	return cmd
}

func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Interactive configuration setup",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			prompt := func(label, current string, secret bool) string {
				disp := current
				if secret && current != "" {
					disp = strings.Repeat("*", 8)
				}
				if disp == "" {
					disp = "(not set)"
				}
				fmt.Printf("%s [%s]: ", label, disp)
				var inp string
				fmt.Scanln(&inp)
				inp = strings.TrimSpace(inp)
				if inp == "" {
					return current
				}
				return inp
			}
			fmt.Println("🔧 task-agent Configuration")
			fmt.Println(strings.Repeat("─", 40))
			cfg.WorkspaceGID = prompt("Workspace GID", cfg.WorkspaceGID, false)
			cfg.ProjectGID = prompt("Default Project GID", cfg.ProjectGID, false)
			cfg.OutputDir = prompt("Output directory", cfg.OutputDir, false)
			fmt.Println("\nAPI Keys:")
			for _, prov := range ai.Providers {
				if prov.EnvKey == "" {
					continue
				}
				key := prompt("  "+prov.Name, cfg.APIKeys[prov.ID], true)
				if key != "" {
					config.SetAPIKey(cfg, prov.ID, key)
				}
			}
			fmt.Println("\nAI Provider:")
			for i, p := range ai.Providers {
				marker := " "
				if p.ID == cfg.Provider {
					marker = "▶"
				}
				fmt.Printf("  %d. %s %s\n", i+1, marker, p.Name)
			}
			var choice int
			fmt.Printf("Choose [1-%d]: ", len(ai.Providers))
			fmt.Scan(&choice)
			if choice >= 1 && choice <= len(ai.Providers) {
				prov := ai.Providers[choice-1]
				cfg.Provider = prov.ID
				fmt.Printf("\nModels for %s:\n", prov.Name)
				for i, m := range prov.Models {
					marker := " "
					if m == cfg.Model {
						marker = "▶"
					}
					fmt.Printf("  %d. %s %s\n", i+1, marker, m)
				}
				var mChoice int
				fmt.Printf("Choose [1-%d]: ", len(prov.Models))
				fmt.Scan(&mChoice)
				if mChoice >= 1 && mChoice <= len(prov.Models) {
					cfg.Model = prov.Models[mChoice-1]
				}
			}
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}
			fmt.Printf("\n✅ Config saved to %s\n", config.ConfigPath())
			fmt.Printf("   Provider: %s / %s\n", cfg.Provider, cfg.Model)
			return nil
		},
	}
}

func newProvidersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "providers",
		Short: "List available AI providers and models",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			for _, prov := range ai.Providers {
				apiKey := config.GetAPIKey(cfg, prov.ID)
				status := "❌ no key"
				switch {
				case prov.ID == "ollama":
					status = "🔧 local (no key needed)"
				case apiKey != "":
					status = "✅ key found"
				}
				fmt.Printf("\n%s\n", strings.Repeat("─", 42))
				fmt.Printf("📦 %s (%s)  %s\n", prov.Name, prov.ID, status)
				if prov.EnvKey != "" {
					fmt.Printf("   Env: %s\n", prov.EnvKey)
				}
				for _, m := range prov.Models {
					marker := " "
					if m == cfg.Model && prov.ID == cfg.Provider {
						marker = "▶"
					}
					fmt.Printf("     %s %s\n", marker, m)
				}
			}
			fmt.Println()
			return nil
		},
	}
}
