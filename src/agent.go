package main

import (
	"fmt"
	"time"
)

// agentPromptTemplate is the text an endpoint page offers for pasting into a
// coding agent. It is a prompt rather than documentation: it is read by
// something that will act on it, so it states the loop, the interval and the
// stopping condition rather than describing the API.
//
// Every figure in it is substituted, never typed. A prompt that restated the
// rate limit or the retention window in prose would keep quoting the old number
// after the constant behind it moved, and the reader has no way to tell.
const agentPromptTemplate = `You are debugging HTTP requests with httphq.

Capture URL:   %[1]s
Read captures: %[2]s

Anything sent to the capture URL (a webhook provider, a form action, a script,
your own code) is stored and readable as JSON from the read URL.

To watch requests as they arrive:

1. Call GET %[2]s and read ` + "`requests`" + ` and ` + "`cursor`" + ` from the response.
2. On every call after the first, send the previous response's cursor back as
   ?since=<cursor>. You will only be given captures you have not seen.
3. If ` + "`hasMore`" + ` is true, call again immediately: a burst is still draining.
   Otherwise wait 2 seconds before the next call.
4. Keep polling until you have what you need, then stop.

Example:
  curl -s "%[2]s?since=2026-08-09T07:20:05Z"

Each capture carries method, path, query string, headers, body, client IP and
timestamp. With ?since they come back oldest first.

Do not poll faster than every 2 seconds. httphq allows %[3]d requests per minute
per IP and that budget is shared with everything else you and this browser send
it.

Captures are deleted after %[4]s and anyone with the URL can read them, so do not
send anything secret through this endpoint.`

// agentPrompt builds that text for one endpoint.
func agentPrompt(endpointURL, apiURL string, requestsPerMinute int, retention time.Duration) string {
	return fmt.Sprintf(agentPromptTemplate,
		endpointURL, apiURL, requestsPerMinute, retentionPhrase(retention))
}

// retentionPhrase renders the window for prose, in whichever unit divides it
// evenly. It exists because Go's own duration formatting would offer a reader
// "4h0m0s".
func retentionPhrase(d time.Duration) string {
	unit, count := "hour", int(d.Hours())
	if count < 1 || d%time.Hour != 0 {
		unit, count = "minute", int(d.Minutes())
	}
	if count == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", count, unit)
}
