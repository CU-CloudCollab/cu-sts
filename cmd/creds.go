package cmd

import (
	"fmt"

	"cu-sts/idp"
	"cu-sts/profile"

	"github.com/fatih/color"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/ini.v1"
)

var outFile string
var outProfile string
var outCfg *ini.File

// credsCmd represents the creds command
var credsCmd = &cobra.Command{
	Use:    "creds",
	Short:  "Writes new STS credentials to a file.",
	Long:   ``,
	Run:    credsCommand,
	PreRun: validateCredsArgs,
}

func init() {
	rootCmd.AddCommand(credsCmd)

	credsCmd.Flags().StringVar(&outFile, "out-file", "~/.aws/credentials", "file to write credentials to")
	credsCmd.Flags().StringVar(&outProfile, "out-profile", "saml", "name to write single credentials to")
}

func validateCredsArgs(cmd *cobra.Command, args []string) {
	var err error
	var p profile.Profile

	if len(profilesFlag) == 0 {
		p.Name = outProfile
		p.Account = account
		p.Role = role
		p.IDProvider = viper.GetString("id_provider")
		p.Duration = viper.GetInt("duration")
		profiles = append(profiles, p)
	} else {
		for _, k := range profilesFlag {
			if p, err = profile.NewFromConfig(k); err != nil {
				fatalError(err.Error())
			}
			profiles = append(profiles, p)
		}
	}

	if outFile != `~/.aws/credentials` {
		outCfg, err = ini.Load(outFile)
	} else {
		outFile, _ = homedir.Expand(outFile)
		outCfg, err = ini.Load(outFile)
	}
	if err != nil {
		fatalError(fmt.Sprintf("could not use out-file: %v", err))
	}
}

func credsCommand(cmd *cobra.Command, args []string) {
	var username = viper.GetString("username")
	var password = viper.GetString("password")
	var duoMethod = viper.GetString("duo_method")

	var SAMLResponse string
	if err := idp.GetSAMLResponse(username, password, duoMethod, &SAMLResponse); err != nil {
		fatalError(fmt.Sprintf("failed to fetch credentials via IdP: %v\n", err))
	}

	fmt.Printf("Writing credentials to %s.\n", outFile)

	for _, p := range profiles {
		creds, err := p.Credentials(SAMLResponse)
		if err != nil {
			fatalError(fmt.Sprintf("could net fetch STS credentials: %v.\n", err))
		}

		fmt.Printf("Received AWS STS credentials for %s, writing to file.\n", p.Name)

		sect := outCfg.Section(p.Name)
		// Clear any current keys to make sure we don't accidentaly carry over anything extra
		for _, k := range sect.KeyStrings() {
			sect.DeleteKey(k)
		}
		sect.Key("aws_access_key_id").SetValue(*creds.AccessKeyId)
		sect.Key("aws_secret_access_key").SetValue(*creds.SecretAccessKey)
		sect.Key("aws_session_token").SetValue(*creds.SessionToken)
		sect.Key("aws_security_token").SetValue(*creds.SessionToken)
		if err = outCfg.SaveTo(outFile); err != nil {
			color.Yellow(fmt.Sprintf("Problem saving profile: %v", err))
		}
	}
}
