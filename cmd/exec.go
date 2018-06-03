package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"cu-sts/idp"
	"cu-sts/profile"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	subCmd  string
	subArgs []string
)

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use:    "exec",
	Short:  "Execute a command or spawn a shell with new AWS STS credentials.",
	Long:   ``,
	Run:    execCommand,
	PreRun: validateExecArgs,
}

func init() {
	rootCmd.AddCommand(execCmd)
}

func validateExecArgs(cmd *cobra.Command, args []string) {
	var err error
	profilesCount := len(profilesFlag)
	p := profile.New()

	if profilesCount > 1 {
		fatalError("exec command can only use a single --profile argument.")
	}
	if profilesCount == 0 {
		p.Account = account
		p.Role = role
		p.Name = fmt.Sprintf("%s/%s", p.Account, p.Role)
		p.IDProvider = viper.GetString("id_provider")
		p.Duration = viper.GetInt("duration")
	} else {
		if p, err = profile.NewFromConfig(profilesFlag[0]); err != nil {
			fatalError(err.Error())
		}
	}

	profiles = append(profiles, p)
}

func execCommand(cmd *cobra.Command, args []string) {
	var username = viper.GetString("username")
	var password = viper.GetString("password")
	var duoMethod = viper.GetString("duo_method")

	p := profiles[0]

	var SAMLResponse string
	if err := idp.GetSAMLResponse(username, password, duoMethod, debug, &SAMLResponse); err != nil {
		fatalError(fmt.Sprintf("failed to fetch credentials via IdP %v\n", err))
		os.Exit(1)
	}

	creds, err := p.Credentials(SAMLResponse)
	if err != nil {
		fatalError(fmt.Sprintf("could net fetch STS credentials %v.\n", err))
	}

	fmt.Printf("Received AWS STS credentials for %s, spawning sub-command.\n", p.Name)

	env := environ(os.Environ())
	env.Unset("AWS_ACCESS_KEY_ID")
	env.Unset("AWS_SECRET_ACCESS_KEY")
	env.Unset("AWS_CREDENTIAL_FILE")
	env.Unset("AWS_DEFAULT_PROFILE")
	env.Unset("AWS_PROFILE")

	env.Set("AWS_ACCESS_KEY_ID", *creds.AccessKeyId)
	env.Set("AWS_SECRET_ACCESS_KEY", *creds.SecretAccessKey)
	env.Set("AWS_SESSION_TOKEN", *creds.SessionToken)
	env.Set("AWS_SECURITY_TOKEN", *creds.SessionToken)
	env.Set("CUSTS_PROFILE", p.Name)

	subCmd = os.Getenv("SHELL")
	subArgs = nil
	if len(args) > 0 {
		subCmd = args[:1][0]
		subArgs = args[1:]
	}

	sh := exec.Command(subCmd, subArgs...)
	sh.Env = env
	sh.Stdin = os.Stdin
	sh.Stdout = os.Stdout
	sh.Stderr = os.Stderr

	if err := sh.Start(); err != nil {
		fatalError(err.Error())
	}

	// Apologies to 99designs
	// https://github.com/99designs/aws-vault/blob/master/cli/exec.go
	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- sh.Wait()
		close(waitCh)
	}()

	for {
		select {
		case sig := <-signals:
			if err := sh.Process.Signal(sig); err != nil {
				fatalError(err.Error())
			}
		case err := <-waitCh:
			var waitStatus syscall.WaitStatus
			if exitError, ok := err.(*exec.ExitError); ok {
				waitStatus = exitError.Sys().(syscall.WaitStatus)
				os.Exit(waitStatus.ExitStatus())
			}
			if err != nil {
				fatalError(err.Error())
			}
			return
		}
	}
}

// environ is a slice of strings representing the environment, in the form "key=value".
type environ []string

// Unset an environment variable by key
func (e *environ) Unset(key string) {
	for i := range *e {
		if strings.HasPrefix((*e)[i], key+"=") {
			(*e)[i] = (*e)[len(*e)-1]
			*e = (*e)[:len(*e)-1]
			break
		}
	}
}

// Set adds an environment variable, replacing any existing ones of the same key
func (e *environ) Set(key, val string) {
	e.Unset(key)
	*e = append(*e, key+"="+val)
}
