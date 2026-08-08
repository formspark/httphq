---
target: landing page
total_score: 21
max_score: 40
na_heuristics: 
p0_count: 2
p1_count: 2
timestamp: 2026-08-08T10-17-31Z
slug: src-views-index-html
---
Method: dual-agent (A: design review · B: detector + browser evidence, run isolated and in parallel)

## Design Health Score

| # | Heuristic | Score | Key Issue |
|---|-----------|-------|-----------|
| 1 | Visibility of System Status | 2 | `Create endpoint` has no submit state: no disabled, no pending label. On a slow link the primary action is inert for seconds. |
| 2 | Match System / Real World | 3 | The lede promises "timing", but the product captures a relative `Time` timestamp, not duration or latency. "Endpoint" is system jargon for what the visitor wants: a URL to paste. |
| 3 | User Control and Freedom | 3 | One action, no traps, footer exits present: but no preview of what you get before an irreversible-feeling commit. |
| 4 | Consistency and Standards | 3 | `Common use cases` is an uppercase h2 at 0.875rem, breaking DESIGN.md's Two Uppercase Rule and its 0.75rem Label token. `hover:shadow-md` sits on non-interactive articles. |
| 5 | Error Prevention | 1 | No double-submit guard, no CSRF token on `POST /endpoint`, and no statement that the capture URL is public and unauthenticated before you point live webhook traffic at it. |
| 6 | Recognition Rather Than Recall | 1 | The entire product must be imagined. No screenshot, no sample capture, no method badge, no JSON body anywhere on the page. |
| 7 | Flexibility and Efficiency | 2 | Not n/a: PRODUCT.md names the returning regular a co-primary user who transits this page weekly. No autofocus, no Enter-to-create, no browser-local list of endpoints they made today. |
| 8 | Aesthetic and Minimalist Design | 3 | Genuinely restrained; all four anti-references avoided. But ~60% of mobile height is six cards carrying about three distinct ideas. |
| 9 | Error Recovery | 2 | Production rate limit is 125 req/min per IP; a rejected `POST /endpoint` has no designed surface on this page. |
| 10 | Help and Documentation | 1 | Not n/a: the first-timer needs four facts before clicking: free? account? how long does it live? is it public? None appear. |
| **Total** | | **21/40 (52.5%)** | **Acceptable** |

No heuristic scored n/a. Heuristics 7 and 10 both genuinely apply: 7 because a returning power user is a stated co-primary persona who passes through this exact page, and 10 because "no docs needed" covers operating the tool, not deciding to adopt it.

## Design Specificity Verdict

**LLM assessment: a neighbouring dev tool could ship this unchanged.** Swap the wordmark, the button label and the six card titles and this is Webhook.site, RequestBin, Beeceptor or Pipedream. The structure is the canonical open-source-dev-tool template, centred wordmark, centred display h1 with a hard `<br>`, 18px lede, one filled CTA with a trailing arrow, uppercase micro-label, 2×3 bordered card grid, three grey footer links, executed cleanly and without a single deviation.

The chassis is genuinely authored and the page obeys DESIGN.md exactly. But the chassis is the deliberately neutral part of the system, and this page uses only the neutral part:

- **The method badge, DESIGN.md's named signature component and the one place the system licenses full-spectrum colour, appears zero times.**
- The mono/sans evidence split survives as exactly one token: `<code>action</code>` in card two.
- `public/logo.svg`, the periwinkle diamond named in PRODUCT.md as the brand mark, is referenced by zero templates. The page renders 0 `<img>` elements.
- The six icons are stock Feather glyphs chosen by literal noun-matching: a clock for cron, a phone for mobile. None encodes anything about HTTP.

The North Star is "The Arrivals Board." The front of the arrivals board shows no arrivals.

**Deterministic scan: 0 findings, exit 0, across all four templates.** The detector did not choke on Go template syntax. But treat this as weak evidence, not a clean bill of health: these files are fragments with no `<head>` and no stylesheet, and every visual property lives in Tailwind utility strings resolved at runtime, so the static scanner had no CSS to analyse. The mode that would have covered rendered CSS, URL scanning, failed: `puppeteer is required for URL scanning`, and **it exited 0 anyway**, so a CI job invoking URL mode would silently pass on a broken toolchain.

**Overlay: injection succeeded and the detector ran in the page.** Console reported exactly one message: `[impeccable] No anti-patterns found.` Explicit re-runs of `impeccableDetectAsync()` and `impeccableScanAsync()` both returned `[]`. The overlay session has since been torn down, so no overlay is visible in the browser now.

## Overall Impression

The craft is real and the restraint is correct. What is missing is not polish, it is evidence and argument. This page describes a live stream of HTTP requests in 170 words of prose and demonstrates it nowhere, then asks for a click. Every one of PRODUCT.md's three "a competitor could not truthfully copy this" claims, no signup, open source and self-hostable, unpolluted captures: is absent from the rendered page. Word counts on the live DOM: `account` 0, `free` 0, `open source` 0, `self-host` 0, `4 hour` 0.

The single biggest opportunity is the cheapest one on the page: render one honest static capture, a POST badge, a path, three headers, a five-line JSON body, using components that already exist, and put two sentences of trust copy under the button.

## What's Working

**The restraint is real and correct for this product.** All four confirmed anti-references are genuinely avoided: no gradient mesh, no tilted dashboard mockup, no testimonial carousel, no green-on-black, no mascot. Measured contrast passes everywhere: h1 at 17.04:1, body copy at 7.58:1, CTA label at 6.46:1, worst case 4.55:1. The page earns "precise, quiet, fast" instead of claiming it.

**The hero is single-action and that shape is right.** One indigo fill, no competing CTA, no newsletter, no nav to get lost in, five tab stops on the whole page. The One Indigo Rule is honoured exactly, and the CTA's focus ring is the only authored focus style on the page: a 2px indigo-500 ring over a 2px white offset, measured and correct. For a user who is mid-debug and impatient, "click or don't" is the right decision space.

**Sentence-level copy voice.** "point any client at it", "see exactly what it puts on the wire: headers, body, every byte", and card six's *"is the payload from your side malformed, or mine?"* are written by someone who has had that argument at 6pm on a Friday. That last line is the only sentence on the page a competitor could not have written: and it is buried at the bottom of the sixth card, below the fold on every viewport tested.

## Priority Issues

### [P0] The product is invisible on the page that has to sell it
**Why it matters:** PRODUCT.md's Evidence section says it plainly: *"a live capture stream is the demonstration."* The only real evidence this project has, it declines to show. Heuristic 6 and the working-memory checklist item both fail as a direct consequence, and the visitor is asked to spend a click on faith at the moment they are least patient.
**Fix:** Put one static, honest capture between the CTA and the grid: a POST badge in Bench Emerald, `/to/<haiku-slug>`, three header rows (`content-type`, `user-agent`, `stripe-signature`), five lines of pretty-printed JSON in the borrowed hljs palette. Pure markup, zero JS, on existing panel/hairline/code-well components. This fabricates nothing: it renders the product's own output, not a testimonial or a metric.
**Suggested command:** `$impeccable bolder`

### [P0] Every positioning claim and every reassurance at the commit point is absent
**Why it matters:** The three claims are free to state, true, and unfalsifiable by a competitor. Meanwhile the CTA creates a resource that is public, unauthenticated, guessable-in-principle and dead in four hours: and states none of that before the click. Retention surfaces only afterward, on the endpoint page. The no-auth fact is never stated anywhere; card six sells shareability as a benefit and never names its corollary.
**Fix:** One line of body copy directly under `Create endpoint`: *"No account, no email, free forever. Requests are deleted after 4 hours, and anyone with the URL can read them."* Plus a footer-adjacent line: *"Open source, MIT licensed, self-hostable: a single Go binary."* Two sentences close the persuasion gap, the trust gap and the proximity failure at once.
**Suggested command:** `$impeccable clarify`

### [P1] The page ships 42.9% of its bytes to code it cannot use, and compiles 100% of its CSS in the browser
**Why it matters:** Measured on `/`: 158,273 B transferred, 82.2% third-party, of which **67,910 B (42.9%) is dead weight**, highlight.js (43,616 B) with 0 highlight targets on the page, Alpine (16,136 B) with 0 directives, plus `endpoint.js`, `har.js`, `render-body.js`. `document.styleSheets` shows **no first-party stylesheet at all**: every style is generated at runtime by a render-blocking 271 KB third-party script that Tailwind documents as development-only. If jsDelivr is slow, the page renders unstyled. This on a project whose design system explicitly rejects webfonts as a performance commitment. Also: no `preconnect`/`dns-prefetch` for either third-party origin, and `favicon.ico` alone is 15,706 B, 56% of first-party bytes.
**Fix:** Build Tailwind at compile time into a static stylesheet served from the Go binary. Move highlight.js, Alpine, `endpoint.js`, `har.js` and `render-body.js` out of `layouts/main.html` into `endpoint.html` where they are used.
**Suggested command:** `$impeccable optimize`

### [P1] Six cards advertise interactivity they do not have
**Why it matters:** Each `<article>` carries `hover:shadow-md transition` with `cursor: auto`, no href, no handler, no tabindex, no role. DESIGN.md's Shadow-Is-A-Response Rule says elevation above the resting whisper exists only as a response: and a response implies an action. Six surfaces lift under the cursor and do nothing, on a page where the visitor is hunting for what to click. Worse, the grid is a six-option decision point where every option resolves to no action, and cards one, three and five are three framings of the same activity.
**Fix:** Pick a side. Either make each card a real link that creates an endpoint and lands with a matching example pre-filled in the *Send a test request* panel, or delete `hover:shadow-md` and let them sit flat. Cut six to three either way.
**Suggested command:** `$impeccable distill`

### [P2] Neither arrival path PRODUCT.md names is served by the page's metadata
**Why it matters:** The `<title>` is literally `httphq`. No `og:*`, no `twitter:*`, no canonical. PRODUCT.md says the first-timer *"arrives from a search result or a colleague's link"*: a search result shows the bare word "httphq"; a Slack paste unfurls as a naked grey link with no image, title or description. `robots.txt` already allows only `/` and `/contact`, so this is the indexed surface.
**Fix:** `<title>httphq: inspect HTTP requests in real time</title>`, the four `og:` tags, `twitter:card=summary_large_image`, and a canonical. The `og:image` is the obvious first home for `public/logo.svg`, currently used by nothing a human sees at size.
**Suggested command:** `$impeccable clarify`

## Persona Red Flags

**Jordan (first-timer, mid-debug, from a search result)**: `Create endpoint` is the wrong noun, Jordan wants a URL to paste into the Stripe webhook field open in the next tab, not an "endpoint". Nothing on the page says free or no-signup, and Jordan has been burned before. The lede promises "timing", which the product does not capture, a credibility debit inside the first 90 seconds. Jordan's first sight of the actual product is an empty dashed box reading "Waiting for requests…", because the landing page did nothing to establish what will fill it.

**Riley (stress tester)**: Double-click on `Create endpoint` over a slow link fires two POSTs, no disabled state, nothing in `index.js`. The form has **no CSRF token and no confirmation step**, which Assessment B demonstrated accidentally: its automated interaction probe submitted the hero form and created a real endpoint record on the running server. Any cross-origin form post can mint endpoints. The 125 req/min rate limit has no designed error surface here. Riley reads card six, correctly infers there is no auth, and finds nothing to size the risk with. The header `<a href="/">` is 976px wide at 1440, clicking anywhere in the header band navigates.

**Casey (mobile, 390×844)**: All three footer links fail the WCAG 2.2 AA 24×24 minimum: measured 119.5×17, 138×20 and 71×17, with 13.5px and 16.5px vertical gaps, under the 24px spacing exception that would otherwise rescue them. They are the only escape hatches on the page. 1.87 screens of scroll, ~900px of it the six stacked cards. `Create endpoint` is a 195px pill inside a 358px column rather than the full-width control the system's own stacking implies. No repeated or sticky CTA, so the only action is off-screen for the last 1.2 screens.

**Sam (the integrations engineer who opens httphq three times a week, derived from PRODUCT.md's returning power user)**: Sam's entire relationship with `/` is land, click, leave, and Sam re-reads the same six cards every visit, extracting nothing. Method filtering, full-text search, HAR export and the send-a-test-request panel, the four things PRODUCT.md names as *why Sam returns*, appear on this page zero times; Sam cannot send a colleague a link to a page describing what the tool does for a regular. Sam wants the endpoint from twenty minutes ago and there is no path to it: a browser-local list of "endpoints you created here (they expire in 4 hours)" honours both the no-accounts and the ephemeral principle exactly, and its absence reads as a decision never made rather than one made and rejected.

## Minor Observations

- `<section aria-label="Common use cases">` duplicates the visible `<h2>`: screen readers announce the name twice. Use `aria-labelledby` or drop one.
- Heading order is clean (h1 → h2 → 6× h3, no skips), `lang="en"` present, all 5 interactive elements have accessible names, all 8 SVGs correctly hidden, `rel="noopener"` present. Zero console messages and 10/10 successful requests on load.
- Only the CTA has an authored focus style; the four links fall back to Chrome's UA default blue, which is engine-specific and unrelated to the indigo system. The CTA's ring also *fades in* via `transition` rather than appearing instantly.
- LCP element is the hero lede `<p>`, not the `<h1>`: the largest painted element is the supporting copy, not the headline.
- No `prefers-reduced-motion` rule exists in any loaded stylesheet while 7 elements carry `.transition`.
- Parallelism breaks on card four: five titles are verb-first (Test, Inspect, Debug, Verify, Share), and "Mobile & IoT traffic" names a channel instead of a job.
- Footer says *"A community project by Formspark"*; PRODUCT.md records Formspark as **sponsor**. Worth aligning.
- Measured performance is excellent where localhost can measure it: LCP 128ms, CLS 0, TTFB 2ms. The payload issue above is specifically the cold-cache, real-network case localhost cannot reproduce.

## Questions to Consider

1. The North Star is "The Arrivals Board." What is the argument for describing a live stream in six paragraphs instead of rendering one honest static capture beneath the button, from components that already exist and cost nothing to place?
2. Three claims sit in PRODUCT.md as the things no neighbouring tool could truthfully copy, and none of the three appear on this page. What is the six-card grid buying that those three sentences aren't?
3. The CTA creates a public, unauthenticated, guessable-in-principle URL that dies in four hours, and says none of that before the click. Which of those six facts would you be uncomfortable stating above the button: and what does that discomfort tell you?
4. Half this page's height exists to persuade a first-timer who "will not read documentation to get started", while the regular has read those cards fifty times. If neither audience reads the grid, who is it for?
