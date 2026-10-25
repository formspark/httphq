package main

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
)

// llmsTemplate is the llms.txt summary of httphq: a Markdown file that tells a
// language model what the site is and how to use it, in the shape llmstxt.org
// describes. Like the agent prompt, every figure and URL in it is substituted
// rather than typed, so moving a limit moves what this file states and a
// self-hosted deployment describes its own host.
const llmsTemplate = `# httphq

> httphq captures HTTP requests sent to a unique URL and shows them in real time:
> method, path, query string, headers, body and client IP. No account, no key and
> no setup. Free, open source and MIT licensed.

## How to use it

1. Open %[1]s/ and choose Create endpoint. The page that opens is the endpoint
   page, and its address ends in the endpoint ID.
2. Point any client at the capture URL, %[1]s/to/<endpoint-id>: a webhook
   provider, a form action, curl or your own code. Every method is accepted.
3. Watch each request appear on the endpoint page as it arrives, or read the
   captures as JSON from %[1]s/api/endpoints/<endpoint-id>/requests.

Example:

    curl -X POST -d '{"hello":"world"}' %[1]s/to/<endpoint-id>

An endpoint ID is lowercase words and digits joined by hyphens. It exists as
soon as something addresses it, so there is no create call in the API.

## Debugging with a coding agent

Every endpoint page offers a prompt to copy into a coding agent. It names the
capture URL and the JSON listing, and tells the agent to poll the listing with
the ` + "`since`" + ` cursor so it only reads captures it has not seen.

## Limits

- Captures are deleted after %[2]s.
- Request bodies are limited to %[3]s.
- Each client IP may send %[4]d requests per minute.
- Anyone with an endpoint URL can read its captures. Do not send secrets.

## Links

- [Home](%[1]s/): create an endpoint
- [Source code](%[5]s): the repository and its documentation
`

// renderLLMs answers /llms.txt for the host it was asked on.
func renderLLMs(c fiber.Ctx) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
	return c.SendString(fmt.Sprintf(llmsTemplate,
		pageBaseURL(c), retentionPhrase(retentionWindow), sizePhrase(bodyLimit),
		productionRequestsPerMinute, repositoryURL))
}

// sizePhrase renders a byte count for prose in the largest binary unit that
// divides it evenly.
func sizePhrase(bytes int) string {
	if bytes%(1<<20) == 0 {
		return fmt.Sprintf("%d MiB", bytes>>20)
	}
	return fmt.Sprintf("%d KiB", bytes>>10)
}
