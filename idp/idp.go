package idp

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

func submitCredentials(username, password string) error {
	t, cancel := context.WithTimeout(chrome.Ctxt, 15*time.Second)
	timeoutContext = t
	defer cancel()

	var err error
	var failed bool

	if err = submitCredentialsRunner(username, password); err != nil {
		return err
	}

	if failed, err = failedLogin(); err != nil {
		return err
	}

	if failed {
		return errors.New("Login failed, invalid credentials.")
	}
	return nil
}

func submitCredentialsRunner(username, password string) error {
	var usernameSel = "#netid"
	var passwordSel = "#password"

	return chrome.C.Run(timeoutContext, chromedp.Tasks{
		chromedp.WaitVisible(usernameSel),
		chromedp.SendKeys(usernameSel, username),
		chromedp.SendKeys(passwordSel, password),
		chromedp.Submit(passwordSel),
		chromedp.Sleep(1 * time.Second),
	})
}

func failedLogin() (bool, error) {
	var s string
	var reasonSel = "#reason"

	if err := chrome.C.Run(chrome.Ctxt, chromedp.Title(&s)); err != nil {
		return true, err
	}

	if strings.Contains(s, `Cornell Two-Step Login`) {
		return false, nil
	}

	if err := chrome.C.Run(chrome.Ctxt,
		chromedp.InnerHTML(reasonSel, &s, chromedp.ByQuery)); err != nil {
		return true, err
	}

	return strings.Contains(s, "Unable"), nil
}
