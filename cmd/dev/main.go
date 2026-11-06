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

	"github.com/BetterGuests/BetterGuests-Server/internal/logging"
)

type commandFunc func() error

var errUsage = errors.New("usage: go run ./cmd/dev {setup|migrate|migration-status|test|run}")

const (
	testComposeFile          = "compose.test.yaml"
	testDatabaseURL          = "postgres://betterguests_app:betterguests_app_test@127.0.0.1:55432/betterguests_test?sslmode=disable"
	testMigrationDatabaseURL = "postgres://betterguests_migrator:betterguests_migrator_test@127.0.0.1:55432/betterguests_test?sslmode=disable"
)

func main() {
	logging.Configure()

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

func test() (err error) {
	compose := []string{"compose", "-f", testComposeFile}

	// Always start from an empty, disposable database so tests cannot depend
	// on data left by a previous run.
	if err := execute("docker", append(compose, "down", "--volumes", "--remove-orphans")...); err != nil {
		return err
	}
	defer func() {
		cleanupErr := execute("docker", append(compose, "down", "--volumes", "--remove-orphans")...)
		if err == nil && cleanupErr != nil {
			err = cleanupErr
		}
	}()

	if err := execute("docker", append(compose, "up", "--detach", "--wait")...); err != nil {
		return err
	}
	if err := execute(
		"docker",
		append(compose,
			"exec", "-T", "postgres",
			"psql", "--username", "postgres", "--dbname", "betterguests_test",
			"--set", "ON_ERROR_STOP=1",
			"--file", "/betterguests/db/test/001_local_roles.sql",
		)...,
	); err != nil {
		return err
	}

	// These fixed loopback URLs deliberately override the shell and .env.
	// TEST_* enables the database security suite; DATABASE_URL keeps any
	// server-level tests on the same disposable database.
	testEnv := map[string]string{
		"DATABASE_URL":                testDatabaseURL,
		"MIGRATION_DATABASE_URL":      testMigrationDatabaseURL,
		"TEST_DATABASE_URL":           testDatabaseURL,
		"TEST_MIGRATION_DATABASE_URL": testMigrationDatabaseURL,
		"DB_EXPECTED_ROLE":            "betterguests_app",
	}
	if err := executeWithOverrides(testEnv, "go", "run", "./cmd/migrate", "up"); err != nil {
		return err
	}
	return executeWithOverrides(testEnv, "go", "test", "-count=1", "./...")
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
	return executeWithOverrides(nil, name, args...)
}

func executeWithOverrides(overrides map[string]string, name string, args ...string) error {
	command := exec.Command(name, args...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Env = environmentWithOverrides(os.Environ(), overrides)
	if err := command.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", name, err)
	}
	return nil
}

func environmentWithOverrides(base []string, overrides map[string]string) []string {
	environment := make([]string, 0, len(base)+len(overrides))
	for _, entry := range base {
		key, _, found := strings.Cut(entry, "=")
		if found {
			if _, overridden := overrides[key]; overridden {
				continue
			}
		}
		environment = append(environment, entry)
	}
	for key, value := range overrides {
		environment = append(environment, key+"="+value)
	}
	return environment
}
