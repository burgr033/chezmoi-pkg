package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"

	"github.com/charmbracelet/fang"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type App struct {
	Hostname string
}

func main() {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	app := &App{
		Hostname: hostname,
	}

	root := NewRootCmd(app)

	if err := fang.Execute(context.Background(), root); err != nil {
		os.Exit(1)
	}
}

func NewRootCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chezmoi-pkg",
		Short: "Manage packages declaratively with chezmoi",
	}

	cmd.AddCommand(NewAddCmd(app))
	cmd.AddCommand(NewRemoveCmd(app))
	cmd.AddCommand(NewListCmd(app))

	return cmd
}

func NewAddCmd(app *App) *cobra.Command {
	var apply bool

	cmd := &cobra.Command{
		Use:   "add [packages...]",
		Short: "Add packages",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			list, config, filename := getList(app)

			for _, p := range args {
				if !slices.Contains(list, p) {
					list = append(list, p)
					fmt.Println("Added:", p)
				} else {
					fmt.Println("Already exists:", p)
				}
			}

			updatePackages(config, list, app.Hostname)
			saveConfig(config, filename)

			if apply {
				return applyChezmoi()
			}

			return nil
		},
	}
	cmd.Flags().BoolVar(&apply, "apply", false, "run chezmoi apply after modification")
	return cmd
}

func NewRemoveCmd(app *App) *cobra.Command {
	var apply bool

	cmd := &cobra.Command{
		Use:   "remove [package]",
		Short: "Remove a package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pkg := args[0]

			list, config, filename := getList(app)

			list = remove(list, pkg)

			fmt.Println("Removed:", pkg)

			updatePackages(config, list, app.Hostname)
			saveConfig(config, filename)

			if apply {
				return applyChezmoi()
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&apply, "apply", false, "run chezmoi apply after modification")

	return cmd
}

func NewListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			list, _, _ := getList(app)

			if len(list) == 0 {
				fmt.Println("No packages found for host:", app.Hostname)
				return nil
			}

			slices.Sort(list)

			for _, p := range list {
				fmt.Println(p)
			}

			return nil
		},
	}
}

func applyChezmoi() error {
	cmd := exec.Command("chezmoi", "apply")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func getList(app *App) ([]string, map[string]any, string) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("missing config directory: %v\n", err)
	}

	viper.AddConfigPath(filepath.Join(configDir, "chezmoi-pkg"))
	viper.SetConfigName("pkg")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config: %v", err)
	}

	filename := viper.GetString("file")

	config := loadMachinePackages(filename)

	packages := ensurePath(config, "packages", "linux", "arch", app.Hostname)

	list := getPackageList(packages)

	return list, config, filename
}

func updatePackages(config map[string]any, list []string, hostname string) {
	packages := ensurePath(config, "packages", "linux", "arch", hostname)

	var arr []any
	for _, p := range list {
		arr = append(arr, p)
	}

	packages["packages"] = arr
}

func loadMachinePackages(filename string) map[string]any {
	var config map[string]any

	data, err := os.ReadFile(filename)
	if err != nil {
		return make(map[string]any)
	}

	if err := toml.Unmarshal(data, &config); err != nil {
		panic(err)
	}

	return config
}

func saveConfig(config map[string]any, filename string) {
	output, err := toml.Marshal(config)
	if err != nil {
		panic(err)
	}

	if err := os.WriteFile(filename, output, 0o644); err != nil {
		panic(err)
	}
}

func ensurePath(m map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		v, ok := m[key]
		if !ok {
			child := make(map[string]any)
			m[key] = child
			m = child
			continue
		}

		child, ok := v.(map[string]any)
		if !ok {
			child = make(map[string]any)
			m[key] = child
		}
		m = child
	}
	return m
}

func getPackageList(section map[string]any) []string {
	var list []string

	if existing, ok := section["packages"].([]any); ok {
		for _, v := range existing {
			if s, ok := v.(string); ok {
				list = append(list, s)
			}
		}
	}

	return list
}

func remove(slice []string, item string) []string {
	var result []string
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}
