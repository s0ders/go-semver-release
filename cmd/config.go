package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.yaml.in/yaml/v3"

	"github.com/s0ders/go-semver-release/v8/internal/appcontext"
	"github.com/s0ders/go-semver-release/v8/internal/branch"
	"github.com/s0ders/go-semver-release/v8/internal/monorepo"
	"github.com/s0ders/go-semver-release/v8/internal/rule"
)

// envPrefix prefixes the environment variables configuring the program: flag "access-token" is configured by
// GO_SEMVER_RELEASE_ACCESS_TOKEN.
const envPrefix = "GO_SEMVER_RELEASE_"

// defaultConfigFiles are looked for, in order, in the working directory when no configuration file is given.
var defaultConfigFiles = []string{defaultConfigFile + ".yaml", defaultConfigFile + ".yml", defaultConfigFile + ".json"}

// initializeConfig sets every flag that was not given on the command line from, in order of precedence, its
// environment variable then the configuration file.
func initializeConfig(cmd *cobra.Command, ctx *appcontext.AppContext) error {
	path, err := configFilePath(ctx.CfgFile)
	if err != nil {
		return err
	}

	values := map[string]any{}

	if path != "" {
		ctx.Logger.Debug().Str("path", path).Msg("using the following configuration file")

		if values, err = readConfigFile(path); err != nil {
			return err
		}
	}

	return bindFlags(cmd, values)
}

// configFilePath returns the configuration file to read: the given one, which must exist, or else the first default
// configuration file found in the working directory, if any.
func configFilePath(cfgFile string) (string, error) {
	if cfgFile == "" {
		cfgFile = os.Getenv(envName("config"))
	}

	if cfgFile != "" {
		return filepath.Abs(cfgFile)
	}

	for _, name := range defaultConfigFiles {
		info, err := os.Stat(name)
		if err == nil && !info.IsDir() {
			return filepath.Abs(name)
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("looking for configuration file: %w", err)
		}
	}

	return "", nil
}

// readConfigFile parses a YAML (or JSON) configuration file. Keys are case-insensitive.
func readConfigFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading configuration file: %w", err)
	}

	var values map[string]any
	if err = yaml.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("parsing configuration file %q: %w", path, err)
	}

	return lowercaseKeys(values), nil
}

// lowercaseKeys lowercases the keys of the given map and of the maps it contains.
func lowercaseKeys(values map[string]any) map[string]any {
	lowered := make(map[string]any, len(values))

	for key, value := range values {
		if nested, ok := value.(map[string]any); ok {
			value = lowercaseKeys(nested)
		}

		lowered[strings.ToLower(key)] = value
	}

	return lowered
}

// bindFlags sets each flag not given on the command line from its environment variable or, if unset, from the
// configuration file values.
func bindFlags(cmd *cobra.Command, values map[string]any) error {
	var err error

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if err != nil || f.Changed {
			return
		}

		value, ok, valueErr := configValue(f, values)
		if valueErr != nil {
			err = valueErr
			return
		}

		if !ok {
			return
		}

		if err = cmd.Flags().Set(f.Name, value); err != nil {
			err = fmt.Errorf("setting %q from configuration: %w", f.Name, err)
		}
	})

	return err
}

// configValue returns the value configuring the given flag, as the flag parses it, and whether there is one.
// Environment variables take precedence over the configuration file; empty environment variables are ignored.
func configValue(f *pflag.Flag, values map[string]any) (string, bool, error) {
	if value := os.Getenv(envName(f.Name)); value != "" {
		return value, true, nil
	}

	value, ok := values[f.Name]
	if !ok || value == nil {
		return "", false, nil
	}

	switch f.Value.(type) {
	// These flags parse JSON, while the configuration file holds their YAML structure.
	case *branch.Flag, *rule.Flag, *monorepo.Flag:
		b, err := json.Marshal(value)
		if err != nil {
			return "", false, fmt.Errorf("marshaling %q value: %w", f.Name, err)
		}

		return string(b), true, nil
	default:
		return fmt.Sprintf("%v", value), true, nil
	}
}

// envName returns the name of the environment variable configuring the given flag.
func envName(flagName string) string {
	return envPrefix + strings.ToUpper(strings.ReplaceAll(flagName, "-", "_"))
}
