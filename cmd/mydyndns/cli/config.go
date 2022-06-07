package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/TylerHendrickson/mydyndns/internal"
)

func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Utilities to assist with configuring the mydyndns agent",
		Long: strings.TrimSpace(`
mydyndns reads configuration directives in from the following sources (in order of precedence): CLI flags, environment variables,
and a configuration file. Configuration files may be specified explicitly by setting the global --config-file flag to the
name of a file with a supported extension. When this flag is not set, mydyndns attempts to find a suitable configuration
file by looking in the current working directory for a file named "mydyndns.ext", where "ext" is one of any supported
config file extensions.`),
	}
}

func newConfigWriteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("write [filename.]{%s} ...", strings.Join(viper.SupportedExts, "|")),
		Short: "Writes one or more config files based on the effective configuration.",
		Long: `The write subcommand is useful for generating config file templates in a variety of supported formats.
Directives may be set via CLI flags, environment variables, and/or another detected config file, and the effective
configuration file(s) will be generated accordingly. If no configuration directives have been set, the directive
values set in the generated file(s) will be empty/default, which may be invalid for actual use (although still
useful for generating config file templates).`,
		Example: `
  - Generate a default-named config file in TOML format from effective configuration:
    mydyndns config write toml ⮕ ./mydyndns.toml
  - Generate several default-named config files in TOML, JSON, and YAML formats
    mydyndns config write toml json yaml ⮕ ./mydyndns.toml ./mydyndns.json ./mydyndns.yaml
  - Generate a custom-named config file in TOML format from effective configuration:
    mydyndns config write example.toml ⮕ ./example.toml
  - Generate a config file with default values, ignoring any effective configuration:
    mydyndns config write example.toml --defaults ⮕ ./example.toml
  - Generate config files in a different directory:
    mydyndns config write json ex.yml -d /examples ⮕ /examples/mydyndns.json /examples/ex.yml
    mydyndns config write $HOME/.config/mydyndns.toml ⮕ $HOME/.config/mydyndns.toml
    mydyndns config write toml -d $HOME/.config ⮕ $HOME/.config/mydyndns.toml
  - Convert an existing TOML-formatted config file to JSON format:
    mydyndns config write json --config-file /examples/conf.toml ⮕ ./mydyndns.json
  - Only write the effective configuration if valid:
    mydyndns config write toml --validate ⮕ ./mydyndns.toml (or ERROR!)
  - Only write the effective configuration if no existing file will be overwritten:
    mydyndns config write toml --safe ⮕ ./mydyndns.toml (or ERROR!)
  - This will fail because the format is not supported:
    mydyndns config write bespokeformat ⮕ (ERROR!)`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
				return err
			}
			return validateConfigFileNames(args)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			seenArgs := internal.NewStringCollection(args...)
			completions := make([]string, 0)
			for _, ext := range viper.SupportedExts {
				if !seenArgs.Contains(ext) {
					completions = append(completions, ext)
				}
			}
			if strings.Contains(toComplete, ".") {
				chunks := strings.Split(toComplete, ".")
				prefix := strings.Join(chunks[0:len(chunks)-1], ".")
				suffix := chunks[len(chunks)-1]
				for _, ext := range viper.SupportedExts {
					if !seenArgs.Contains(ext) && strings.HasPrefix(ext, suffix) {
						completions = append(completions, fmt.Sprintf("%s.%s", prefix, ext))
					}
				}
			}

			directive := cobra.ShellCompDirectiveDefault
			if viper.GetBool("safe") {
				directive = cobra.ShellCompDirectiveNoFileComp
			}

			return completions, directive
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if viper.GetBool("validate") {
				return firstValidationError(cmd,
					validateAPIKey, validateBaseURL, validatePollInterval)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				defaultBasePath = viper.GetString("directory")
				safeWrite       = viper.GetBool("safe")
				quiet           = viper.GetBool("quiet")
				defaultsOnly    = viper.GetBool("defaults")
			)

			// Ensure base path is absolute
			defaultBasePath, err := filepath.Abs(defaultBasePath)
			if err != nil {
				return err
			}

			// Get all viper settings, which we will use to create an isolated viper
			// with only the settings that make sense for a config file.
			configMap := viper.AllSettings()
			// These don't make sense for a config file:
			delete(configMap, configFileSettingKey)
			delete(configMap, configPathSettingKey)
			delete(configMap, "help")
			// Ignore directives that are only used for this ("config write") command
			cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
				delete(configMap, f.Name)
			})
			// Make an isolated Viper with only the remaining settings
			v := viper.New()
			v.MergeConfigMap(configMap)

			if defaultsOnly {
				// Replace remaining settings with the default value set on its corresponding flag
				cmd.Flags().VisitAll(func(f *pflag.Flag) {
					if v.IsSet(f.Name) {
						v.Set(f.Name, f.DefValue)
					}
				})
			}

			writeFunc := v.WriteConfigAs
			if safeWrite {
				writeFunc = v.SafeWriteConfigAs
			}

			for _, f := range args {
				basePath := defaultBasePath
				if filepath.IsAbs(f) {
					basePath, f = filepath.Split(f)
				}
				if filepath.Ext(f) == "" {
					f = fmt.Sprintf("%s.%s", defaultConfigFilename, f)
				}
				configPath := filepath.Join(basePath, f)
				if err := writeFunc(configPath); err != nil {
					return err
				}
				if !quiet {
					cmd.Println(configPath)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringP("directory", "d", ".",
		"Directory path where output files specified with relative paths will be written")
	cmd.MarkFlagDirname("directory")
	cmd.Flags().Bool("safe", false,
		"Fails when an existing file would be overwritten")
	cmd.Flags().Bool("validate", false,
		"Ensures that the effective configuration is valid before writing any file(s).")
	cmd.Flags().BoolP("quiet", "q", false,
		"If unset, filenames are printed as they are written.")
	cmd.Flags().Bool("defaults", false,
		"Ignore effective configuration and generate file(s) with defaults for directive values.")

	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Displays the effective configuration for the mydyndns agent.",
		Long: `The show subcommand is useful for checking the effective agent configuration, especially when multiple
configuration sources (environment variables, config file, and/or CLI flags) are in-use.

Note that the output from this command should not be used to create config files, as its output is meant to be human-
readable and is not intended to be compatible with any supported configuration file format. To generate usable config
files in a variety of supported formats, see the "agent config write" subcommand.`,
		Run: func(cmd *cobra.Command, args []string) {
			for k, v := range viper.AllSettings() {
				if k == "help" {
					continue
				}
				if k == configFileSettingKey {
					v = viper.ConfigFileUsed()
				}
				cmd.Printf("%s = %v\n", k, v)
			}
		},
	}
}

func newConfigTypesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "types",
		Short: "Utilities for supported configuration file types",
	}
}

func newConfigTypesListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Print a list of supported configuration file types (as extensions)",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if viper.GetBool("bare") {
				for _, ext := range viper.SupportedExts {
					cmd.Println(ext)
				}
			} else {
				cmd.Printf("Supported config file extensions: %s\n", strings.Join(viper.SupportedExts, ", "))
			}
		},
	}

	cmd.Flags().Bool("bare", false, "Outputs one extension per line")

	return cmd
}

func newConfigTypesCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   fmt.Sprintf("check [filename.]{%s}", strings.Join(viper.SupportedExts, "|")),
		Short: "Check if the supplied configuration file type is supported",
		Long: strings.TrimSpace(fmt.Sprintf(`
The check subcommand helps determine whether the bare config type (e.g. "toml") or config filename
(based on the extension, e.g. "config.toml") is a supported format. If the type type is not recognized,
the command will exit with an error.

Essentially, this command checks whether the single argument matches or ends with a match
preceded by a dot (as a file extension) any of the following values: %s`, strings.Join(viper.SupportedExts, ", "))),
		Example: `  mydyndns run config types check toml ⮕ (SUCCESS)
  mydyndns run config types check config.toml ⮕ (SUCCESS)
  mydyndns run config types check bespokeformat ⮕ (ERROR)`,
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return viper.SupportedExts, cobra.ShellCompDirectiveDefault
		},
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkExt := args[0]
			if e := filepath.Ext(checkExt); len(e) > 0 {
				checkExt = e[1:]
			}
			for _, supportedExt := range viper.SupportedExts {
				if checkExt == supportedExt {
					return nil
				}
			}
			return viper.UnsupportedConfigError(checkExt)
		},
	}
}

func newConfigValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Checks the effective agent configuration for issues",
		Long: `The validate subcommand isolates the configuration checks executed when the mydyndns agent starts. Use this to
check whether the agent would fail to start due to invalid configuration, without actually running the agent.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return firstValidationError(cmd,
				validateAPIKey, validateBaseURL, validatePollInterval)
		},
	}
}
