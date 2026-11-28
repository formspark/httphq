package main

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
)

// securityContact is where a vulnerability in httphq is reported. It names the
// maintainers, so a self-hosted deployment points reporters at them too.
const securityContact = "mailto:bjorn@bjornkrols.com"

// securityTxtExpires is the date after which a reader treats the file as stale
// (RFC 9116). It has to be moved forward before it passes.
const securityTxtExpires = "2027-11-28T00:00:00.000Z"

// securityTxtTemplate is the RFC 9116 security.txt. Its canonical URL is
// substituted from the request, so a self-hosted deployment names its own host.
const securityTxtTemplate = `Contact: %[1]s
Expires: %[2]s
Preferred-Languages: en
Canonical: %[3]s/.well-known/security.txt
`

// renderSecurityTxt answers /.well-known/security.txt for the host it was
// asked on.
func renderSecurityTxt(c fiber.Ctx) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
	return c.SendString(fmt.Sprintf(securityTxtTemplate,
		securityContact, securityTxtExpires, pageBaseURL(c)))
}
