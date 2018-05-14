// NOTES
// https://github.com/yogesh-desai/WebCrawlerTokopedia/blob/47e9309eb964f34ccf38b410aaf44514e6d208d8/main.go
// https://github.com/zqqiang/go-capwap/blob/0f6e2f2533299013f675cf62dd54afa469f77dab/ac/tools/chromedp/testcase.go
//
// https://github.com/go-ini/ini
package idp

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/runner"
	"github.com/fatih/color"
	"github.com/howeyc/gopass"
)

type Chrome struct {
	Ctxt   context.Context
	C      *chromedp.CDP
	Cancel context.CancelFunc
}

var chrome Chrome
var timeoutContext context.Context

func GetSAMLResponse(username, password, duoMethod string, response *string) error {
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

	chrome.C, err = chromedp.New(chrome.Ctxt,
		chromedp.WithRunnerOptions(
			runner.Flag("disable-web-security", true),
			runner.Flag("headless", true),
			runner.Flag("no-first-run", true),
			runner.Flag("no-default-browser-check", true),
		),
		//chromedp.WithLog(log.Printf),
	)
	if err != nil {
		return fmt.Errorf("Unable to start a chrome instance: %s\n.", err)
	}

	// register SIGINT handler to make sure we cleanup chrome
	signal_chan := make(chan os.Signal, 1)
	signal.Notify(signal_chan, syscall.SIGINT)
	go func() {
		for _ = range signal_chan {
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
