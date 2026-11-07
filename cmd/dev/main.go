package main

import (
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
	testComposeEnvFile       = "compose.test.env"
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
	// Point Compose at a dedicated env file so its dotenv parser never has to
	// read the server's .env, which may contain multi-line credentials.
	compose := []string{"compose", "--env-file", testComposeEnvFile, "-f", testComposeFile}

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
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// Normalize Windows line endings so multi-line values work on every
	// platform without changing the value's own line breaks.
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	values := make(map[string]string)
	for index := 0; index < len(lines); index++ {
		line := strings.TrimSpace(lines[index])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.TrimPrefix(line, "export ")
		key, rawValue, found := strings.Cut(line, "=")
		if !found {
			return nil, fmt.Errorf("line %d must use KEY=VALUE", index+1)
		}

		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("line %d has an empty key", index+1)
		}

		value, end, err := parseEnvValue(strings.TrimSpace(rawValue), lines, index)
		if err != nil {
			return nil, err
		}
		values[key] = value
		index = end
	}
	return values, nil
}

// parseEnvValue reads a value that may be quoted across several lines. It
// returns the decoded value and the index of the last line it consumed.
func parseEnvValue(value string, lines []string, start int) (string, int, error) {
	switch {
	case strings.HasPrefix(value, "'"):
		literal, end, err := readQuoted(value, lines, start, '\'')
		if err != nil {
			return "", 0, err
		}
		return literal[1 : len(literal)-1], end, nil
	case strings.HasPrefix(value, "\""):
		literal, end, err := readQuoted(value, lines, start, '"')
		if err != nil {
			return "", 0, err
		}
		// strconv.Unquote rejects literal newlines, so escape them first and
		// let Unquote turn them back into the newlines the value carried.
		unquoted, err := strconv.Unquote(strings.ReplaceAll(literal, "\n", "\\n"))
		if err != nil {
			return "", 0, fmt.Errorf("line %d has an invalid quoted value", start+1)
		}
		return unquoted, end, nil
	default:
		return value, start, nil
	}
}

// readQuoted returns the quoted literal, including its surrounding quotes, and
// the index of the line where the closing quote was found.
func readQuoted(value string, lines []string, start int, quote byte) (string, int, error) {
	literal := value
	line := start
	for {
		if index := closingQuote(literal, quote); index >= 0 {
			trailing := strings.TrimSpace(literal[index+1:])
			if trailing != "" && !strings.HasPrefix(trailing, "#") {
				return "", 0, fmt.Errorf("line %d has unexpected text after the closing quote", line+1)
			}
			return literal[:index+1], line, nil
		}

		line++
		if line >= len(lines) {
			return "", 0, fmt.Errorf("line %d has an unterminated quoted value", start+1)
		}
		literal += "\n" + lines[line]
	}
}

// closingQuote finds the first closing quote after the opening quote. Only
// double-quoted values treat a backslash as an escape, matching shell and Go
// string-literal conventions.
func closingQuote(value string, quote byte) int {
	escaped := false
	for index := 1; index < len(value); index++ {
		if escaped {
			escaped = false
			continue
		}
		if quote == '"' && value[index] == '\\' {
			escaped = true
			continue
		}
		if value[index] == quote {
			return index
		}
	}
	return -1
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
