package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/viper"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	action := os.Args[1]

	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

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
	packageFile := loadMachinePackages(filename)

	packages := ensurePath(packageFile, "packages", "linux", "arch", hostname)
	list := getPackageList(packages)

	switch action {

	case "add":
		if len(os.Args) != 3 {
			usage()
		}
		pkg := os.Args[2]

		if !slices.Contains(list, pkg) {
			list = append(list, pkg)
			fmt.Println("Added:", pkg)
		} else {
			fmt.Println("Already exists:", pkg)
		}

	case "remove":
		if len(os.Args) != 3 {
			usage()
		}
		pkg := os.Args[2]

		list = remove(list, pkg)
		fmt.Println("Removed:", pkg)

	case "list":
		if len(list) == 0 {
			fmt.Println("No packages found for host:", hostname)
			return
		}

		slices.Sort(list)
		for _, p := range list {
			fmt.Println(p)
		}
		return

	default:
		usage()
	}

	// Clean + sort before saving
	slices.Sort(list)
	list = slices.Compact(list)
	packages["packages"] = list

	saveConfig(packageFile, filename)

	// Run chezmoi apply ONLY after add/remove
	cmd := exec.Command("chezmoi", "apply")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}

func usage() {
	fmt.Println("Usage:")
	fmt.Println("  add <package>")
	fmt.Println("  remove <package>")
	fmt.Println("  list")
	os.Exit(1)
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
