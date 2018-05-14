package idp

import (
	"github.com/chromedp/chromedp"
)

// ISSUE: https://github.com/chromedp/chromedp/issues/75
// Timeouts waiting for nodes to be ready can cause multi-second lockups and
// prints cdp output / errors to STDERR
func getSAMLResponse(res *string) error {
	var ok bool
	var signinSel = "#saml_response"

	return chrome.C.Run(chrome.Ctxt, chromedp.Tasks{
		chromedp.WaitReady(signinSel, chromedp.ByID),
		chromedp.AttributeValue(signinSel, "value", res, &ok),
	})
}
