package idp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/fatih/color"
)

func submitAuthMethod(authMethod string) error {
	t, cancel := context.WithTimeout(chrome.Ctxt, 30*time.Second)
	timeoutContext = t
	defer cancel()

	_, err := waitForFrame()
	if err != nil {
		return err
	}

	if err := clickAuthMethod(authMethod); err != nil {
		return err
	}

	return nil
}

func waitForFrame() (bool, error) {
	// busy wait until the frame loads or > ~40seconds
	// for multi-device users the frame load might be "partial" w/ a checkbox available
	// before the buttons, so we check for both to be safe.
	for i := 0; i < 20; i++ {
		// "Remember me.." checkbox
		if ok := isPresent(`//input[@name='dampen_choice']`); ok {
			break
		}
		time.Sleep(1 * time.Second)
	}

	// busy wait for a button
	for i := 0; i < 20; i++ {
		if ok := isPresent(`//button[contains(., 'Push') or contains(., 'Call')]`); ok {
			return ok, nil
		}
		time.Sleep(1 * time.Second)
	}
	return false, errors.New("Timeout waiting for DUO frame.")
}

func clickAuthMethod(method string) error {
	var buf []byte
	m := make(map[string]string)
	m["push"] = "//button[contains(., 'Push')]"
	m["call"] = "//button[contains(., 'Call')]"
	js := fmt.Sprintf(`
		doc = document.querySelector('iframe[id="duo_iframe"]').contentWindow.document
		document.evaluate("%s", doc).iterateNext().click()
	`, m[method])

	//check if an auto-push/call is actually configured
	if isPresent(`//small[@class='used-automatically']`) {
		color.Yellow("(chrome) Auto-selected DUO method used, ignoring configured method '%s'.\n", method)
		return nil
	}

	return chrome.C.Run(timeoutContext,
		chromedp.Evaluate(js, &buf, chromedp.EvalIgnoreExceptions),
	)
}

func isPresent(xpath string) bool {
	var res interface{}
	js := fmt.Sprintf(`
	f = function(sel) {
	  doc = document.querySelector('iframe[id="duo_iframe"]').contentWindow.document
	  return document.evaluate(sel, doc).iterateNext() !== null
	}
	f("%s")
	`, xpath)
	if err := chrome.C.Run(timeoutContext, chromedp.Evaluate(js, &res)); err != nil {
		return false
	}
	return res.(bool)
}
