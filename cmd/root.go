package cmd

import (
	"fmt"
	"os"

	"cu-sts/profile"

	"github.com/fatih/color"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile           string
	account           string
	role              string
	username          string
	duration          int
	idProvider        string
	profilesFlag      []string
	singleProfileFlag string
	profiles          []profile.Profile
	duoMethod         string
	debug             bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:              "cu-sts",
	Short:            "Fetch AWS STS credentials via Cornell IdP + DUO.",
	Long:             ``,
	Version:          `0.0.2`,
	PersistentPreRun: validateRootArgs,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cu-sts.toml)")
	rootCmd.PersistentFlags().StringVar(&username, "username", "", "username for IdP login")
	rootCmd.PersistentFlags().StringVar(&duoMethod, "duo-method", "push", "DUO method to use (push or call)")
	rootCmd.PersistentFlags().StringVar(&account, "account", "", "account number of role")
	rootCmd.PersistentFlags().StringVar(&role, "role", "", "name of the role")
	rootCmd.PersistentFlags().IntVar(&duration, "duration", 3600, "requested duration of credentials, in seconds")
	rootCmd.PersistentFlags().StringVar(&idProvider, "id-provider", "cornell_idp", "name of the Identity Provider in IAM")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable Chrome debug output to STDERR")

	rootCmd.PersistentFlags().StringSliceVar(&profilesFlag, "profiles", nil, "profiles to get STS credentials for")
	rootCmd.PersistentFlags().StringVar(&singleProfileFlag, "profile", "", "single profile to get STS credentials for")

	viper.BindPFlag("username", rootCmd.PersistentFlags().Lookup("username"))
	viper.BindPFlag("duo_method", rootCmd.PersistentFlags().Lookup("duo-method"))
	viper.BindPFlag("duration", rootCmd.PersistentFlags().Lookup("duration"))
	viper.BindPFlag("id_provider", rootCmd.PersistentFlags().Lookup("id-provider"))
}

func validateRootArgs(cmd *cobra.Command, args []string) {
	// The combination of cobra + viper, config+flags, AND allowing --account/role
	// to be used in conjunction with profile.* sections makes the flag validation
	// kind of dumb and verbose because we can't mark flags as *required* or set
	// reasonable defaults. So we're stuck just checking all invalid combinations.

	// Hack to allow a single profile to be passed by --profile
	if singleProfileFlag != "" {
		profilesFlag = append(profilesFlag, singleProfileFlag)
	}

	if profilesFlag == nil && account == "" {
		fatalError("must use --profiles or --username/--role.")
	}

	if profilesFlag != nil && (account != "" || role != "") {
		fatalError("cannot use --profiles and --username/--role together.")
	}

	if account != "" && role == "" {
		fatalError("--account and --role must be used together.")
	}

	if viper.GetString("username") == "" {
		fatalError("username must be set via --username flag or config file.")
		os.Exit(1)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fatalError(err.Error())
		}

		// Search config in home directory with name ".cu-sts" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".cu-sts")
	}

	viper.SetEnvPrefix("CUSTS")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Loaded config file:", viper.ConfigFileUsed())
	} else {
		fmt.Println("Error reading config file:", err)
	}
}

func fatalError(message string) {
	color.Red("ERROR: %s", message)
	os.Exit(1)
}
