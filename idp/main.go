// Package idp contains the chromedp / headless tasks to navigate the IdP Shib+DUO pages
// and return the base-64 encoded SAMLAssertion.
package idp

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/runner"
	"github.com/fatih/color"
	"github.com/howeyc/gopass"
)

// Chrome holds the chromedp Chrome instance and root context for tasks.
type Chrome struct {
	Ctxt   context.Context
	C      *chromedp.CDP
	Cancel context.CancelFunc
}

var chrome Chrome
var timeoutContext context.Context

// GetSAMLResponse takes a NetID and Password and gets the base-64 encoded
// SAMLResponse from the final signin-sts.aws.cucloud.net POST.
func GetSAMLResponse(username, password, duoMethod string, debug bool, response *string) error {
	var err error

	if password == "" {
		c := color.New(color.FgYellow)
		c.Printf("Password: ")
		passwordBytes, _ := gopass.GetPasswdMasked()
		password = string(passwordBytes)
	}

	if password == "" {
		return fmt.Errorf("ERROR: must enter a password.")
	}

	chrome.Ctxt, chrome.Cancel = context.WithTimeout(
		context.Background(),
		120*time.Second,
	)
	defer chrome.Cancel()

	if !debug {
		chrome.C, err = chromedp.New(chrome.Ctxt,
			chromedp.WithRunnerOptions(
				runner.Flag("disable-web-security", true),
				runner.Flag("headless", true),
				runner.Flag("no-first-run", true),
				runner.Flag("no-default-browser-check", true),
			),
		)
	} else {
		chrome.C, err = chromedp.New(chrome.Ctxt,
			chromedp.WithRunnerOptions(
				runner.Flag("disable-web-security", true),
				runner.Flag("headless", true),
				runner.Flag("no-first-run", true),
				runner.Flag("no-default-browser-check", true),
			),
			chromedp.WithLog(log.Printf),
		)
	}

	if err != nil {
		return fmt.Errorf("Unable to start a chrome instance: %s\n.", err)
	}

	// register SIGINT handler to make sure we cleanup chrome
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT)
	go func() {
		for _ = range signalChan {
			exitChromeQuietly()
			os.Exit(1)
		}
	}()

	// ensure the chrome instance gets quietly killed on exit
	// otherwise we can end up with an orphaned chrome-headless process
	defer exitChromeQuietly()

	fmt.Println("(chrome) Fetching IdP Shibboleth login page.")
	if err = navToLogin(`https://signin-sts.aws.cucloud.net`); err != nil {
		return err
	}

	fmt.Println("(chrome) Submitting username & password.")
	if err = submitCredentials(username, password); err != nil {
		return err
	}

	fmt.Println("(chrome) Submitting selected DUO method.")
	if err = submitAuthMethod(duoMethod); err != nil {
		return err
	}

	fmt.Println("(chrome) Waiting for DUO response and SAML assertion.")
	if err = getSAMLResponse(response); err != nil {
		return err
	}
	return nil
}

func exitChromeQuietly() {
	_ = chrome.C.Shutdown(chrome.Ctxt)
	_ = chrome.C.Wait()
	return
}

func navToLogin(url string) error {
	return chrome.C.Run(chrome.Ctxt, chromedp.Tasks{
		chromedp.Navigate(url),
	})
}
