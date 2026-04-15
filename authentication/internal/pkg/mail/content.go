package mail

import "strings"

// securityTokenContent returns subject, HTML body, and plain-text body for the
// security-token email in the requested locale.
// Supported locales: "sv", "sv-SE" (Swedish). All others default to English.
func SecurityTokenContent(token, locale string) (subject, htmlContent, textContent string) {
	if strings.HasPrefix(strings.ToLower(locale), "sv") {
		subject = "Din BilCool säkerhetskod"
		htmlContent = "Din säkerhetskod är: <div><b>" + token + "</b></div>Koden är giltig i 10 minuter."
		textContent = "Din säkerhetskod är: " + token + "\n\nKoden är giltig i 10 minuter."
		return
	}
	subject = "Your BilCool security code"
	htmlContent = "Your security code is: <div><b>" + token + "</b></div>This code is valid for 10 minutes."
	textContent = "Your security code is: " + token + "\n\nThis code is valid for 10 minutes."
	return
}
