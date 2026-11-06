package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type commandFunc func() error

var errUsage = errors.New("usage: go run ./cmd/dev {setup|migrate|migration-status|test|run}")

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("Developer command failed", "error", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 {
		return errUsage
	}

	commands := map[string]commandFunc{
		"setup":            setup,
		"migrate":          migrate,
		"migration-status": migrationStatus,
		"test":             test,
		"run":              runServer,
	}
	command, found := commands[args[0]]
	if !found {
		return errUsage
	}
	return command()
}

func setup() error {
	if err := createEnvFile(); err != nil {
		return err
	}
	if err := execute("go", "mod", "download"); err != nil {
		return err
	}
	return execute("go", "mod", "verify")
}

func migrate() error {
	return executeWithEnv("go", "run", "./cmd/migrate", "up")
}

func migrationStatus() error {
	return executeWithEnv("go", "run", "./cmd/migrate", "status")
}

func test() error {
	if err := loadEnvFileIfPresent(".env"); err != nil {
		return err
	}
	return execute("go", "test", "-count=1", "./...")
}

func runServer() error {
	return executeWithEnv("go", "run", ".")
}

func executeWithEnv(name string, args ...string) error {
	if err := loadEnvFile(".env"); err != nil {
		return err
	}
	return execute(name, args...)
}

func createEnvFile() error {
	exists, err := fileExists(".env")
	if err != nil {
		return fmt.Errorf("check .env: %w", err)
	}
	if exists {
		fmt.Println("Kept existing .env")
		return nil
	}
	return copyEnvExample()
}

func copyEnvExample() error {
	source, err := os.Open(".env.example")
	if err != nil {
		return fmt.Errorf("open .env.example: %w", err)
	}
	defer source.Close()

	destination, err := os.OpenFile(".env", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create .env: %w", err)
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		return fmt.Errorf("write .env: %w", err)
	}

	fmt.Println("Created .env; add the database URLs supplied by the project owner")
	return nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func loadEnvFileIfPresent(path string) error {
	exists, err := fileExists(path)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	return loadEnvFile(path)
}

func loadEnvFile(path string) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("missing %s; run go run ./cmd/dev setup", path)
	}
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	values, err := parseEnv(file)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return applyEnv(values)
}

func applyEnv(values map[string]string) error {
	for key, value := range values {
		if _, alreadySet := os.LookupEnv(key); alreadySet {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s: %w", key, err)
		}
	}
	return nil
}

func parseEnv(reader io.Reader) (map[string]string, error) {
	values := make(map[string]string)
	scanner := bufio.NewScanner(reader)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		key, value, include, err := parseEnvLine(scanner.Text(), lineNumber)
		if err != nil {
			return nil, err
		}
		if !include {
			continue
		}
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func parseEnvLine(raw string, lineNumber int) (string, string, bool, error) {
	line := strings.TrimSpace(raw)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false, nil
	}

	line = strings.TrimPrefix(line, "export ")
	key, rawValue, found := strings.Cut(line, "=")
	if !found {
		return "", "", false, fmt.Errorf("line %d must use KEY=VALUE", lineNumber)
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return "", "", false, fmt.Errorf("line %d has an empty key", lineNumber)
	}

	value, err := parseEnvValue(strings.TrimSpace(rawValue), lineNumber)
	if err != nil {
		return "", "", false, err
	}
	return key, value, true, nil
}

func parseEnvValue(value string, lineNumber int) (string, error) {
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return value[1 : len(value)-1], nil
	}
	if !strings.HasPrefix(value, "\"") {
		return value, nil
	}

	unquoted, err := strconv.Unquote(value)
	if err != nil {
		return "", fmt.Errorf("line %d has an invalid quoted value", lineNumber)
	}
	return unquoted, nil
}

func execute(name string, args ...string) error {
	command := exec.Command(name, args...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Env = os.Environ()
	if err := command.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", name, err)
	}
	return nil
}
