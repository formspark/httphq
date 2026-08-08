---
name: httphq
description: A live arrivals board for HTTP requests: neutral chassis, one indigo control, color reserved for classifying traffic.
colors:
  beacon-periwinkle: "#707ee7"
  signal-indigo: "oklch(51.1% 0.262 276.966)"
  signal-indigo-lit: "oklch(58.5% 0.233 277.117)"
  signal-indigo-deep: "oklch(45.7% 0.24 277.023)"
  signal-indigo-wash: "oklch(96.2% 0.018 272.314)"
  focus-indigo: "oklch(67.3% 0.182 276.935)"
  panel-white: "#ffffff"
  board-field: "oklch(98.4% 0.003 247.858)"
  hairline-faint: "oklch(96.8% 0.007 247.896)"
  hairline: "oklch(92.9% 0.013 255.508)"
  edge-control: "oklch(86.9% 0.022 252.894)"
  ink-quiet: "oklch(70.4% 0.04 256.788)"
  ink-label: "oklch(55.4% 0.046 257.417)"
  ink-prose: "oklch(44.6% 0.043 257.281)"
  ink-control: "oklch(37.2% 0.044 257.287)"
  ink-body: "oklch(27.9% 0.041 260.031)"
  ink-strong: "oklch(20.8% 0.042 265.755)"
  method-get-ink: "oklch(48.8% 0.243 264.376)"
  method-get-wash: "oklch(97% 0.014 254.604)"
  method-post-ink: "oklch(50.8% 0.118 165.612)"
  method-post-wash: "oklch(97.9% 0.021 166.113)"
  method-put-ink: "oklch(55.5% 0.163 48.998)"
  method-put-wash: "oklch(98.7% 0.022 95.277)"
  method-patch-ink: "oklch(49.1% 0.27 292.581)"
  method-patch-wash: "oklch(96.9% 0.016 293.756)"
  method-delete-ink: "oklch(51.4% 0.222 16.935)"
  method-delete-wash: "oklch(96.9% 0.015 12.422)"
  method-options-ink: "oklch(50% 0.134 242.749)"
  method-options-wash: "oklch(97.7% 0.013 236.62)"
  alert-rose: "oklch(58.6% 0.253 17.585)"
  alert-rose-deep: "oklch(51.4% 0.222 16.935)"
  alert-rose-edge: "oklch(89.2% 0.058 10.001)"
  alert-rose-wash: "oklch(96.9% 0.015 12.422)"
  alert-rose-wash-hover: "oklch(94.1% 0.03 12.58)"
  syntax-key: "#005cc5"
  syntax-string: "#032f62"
  syntax-name: "#22863a"
  syntax-keyword: "#d73a49"
typography:
  display:
    fontFamily: "ui-sans-serif, system-ui, sans-serif"
    fontSize: "3rem"
    fontWeight: 600
    lineHeight: 1
    letterSpacing: "-0.025em"
  headline:
    fontFamily: "ui-sans-serif, system-ui, sans-serif"
    fontSize: "1.875rem"
    fontWeight: 600
    lineHeight: 1.2
    letterSpacing: "normal"
  title:
    fontFamily: "ui-sans-serif, system-ui, sans-serif"
    fontSize: "1rem"
    fontWeight: 500
    lineHeight: 1.5
  lede:
    fontFamily: "ui-sans-serif, system-ui, sans-serif"
    fontSize: "1.125rem"
    fontWeight: 400
    lineHeight: 1.556
  body:
    fontFamily: "ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.429
  label:
    fontFamily: "ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.75rem"
    fontWeight: 500
    lineHeight: 1.333
    letterSpacing: "0.025em"
  badge:
    fontFamily: "ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.75rem"
    fontWeight: 600
    lineHeight: 1.333
    letterSpacing: "0.025em"
  mono:
    fontFamily: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace"
    fontSize: "0.75rem"
    fontWeight: 400
    lineHeight: 1.333
rounded:
  sm: "0.25rem"
  md: "0.375rem"
  lg: "0.5rem"
spacing:
  "2": "0.5rem"
  "3": "0.75rem"
  "4": "1rem"
  "5": "1.25rem"
  "6": "1.5rem"
  "8": "2rem"
  "12": "3rem"
components:
  button-primary:
    backgroundColor: "{colors.signal-indigo}"
    textColor: "{colors.panel-white}"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: "0.75rem 1.5rem"
  button-primary-hover:
    backgroundColor: "{colors.signal-indigo-lit}"
  button-primary-compact:
    backgroundColor: "{colors.signal-indigo}"
    textColor: "{colors.panel-white}"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: "0.5rem 1rem"
  button-secondary:
    backgroundColor: "{colors.panel-white}"
    textColor: "{colors.ink-control}"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: "0.5rem 0.75rem"
  button-secondary-hover:
    backgroundColor: "{colors.board-field}"
  button-destructive:
    backgroundColor: "{colors.alert-rose-wash}"
    textColor: "{colors.alert-rose-deep}"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: "0.375rem 0.75rem"
  button-destructive-hover:
    backgroundColor: "{colors.alert-rose-wash-hover}"
  input-field:
    backgroundColor: "{colors.panel-white}"
    textColor: "{colors.ink-body}"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: "0.5rem 0.75rem"
  panel:
    backgroundColor: "{colors.panel-white}"
    rounded: "{rounded.lg}"
    padding: "1.25rem"
  code-well:
    backgroundColor: "{colors.board-field}"
    textColor: "{colors.ink-body}"
    typography: "{typography.mono}"
    rounded: "{rounded.sm}"
    padding: "0.75rem"
  method-badge:
    typography: "{typography.badge}"
    rounded: "{rounded.sm}"
    padding: "0.125rem 0.5rem"
  icon-tile:
    backgroundColor: "{colors.signal-indigo-wash}"
    textColor: "{colors.signal-indigo}"
    rounded: "{rounded.md}"
    width: "2.25rem"
    height: "2.25rem"
  empty-state:
    backgroundColor: "{colors.panel-white}"
    textColor: "{colors.ink-label}"
    rounded: "{rounded.lg}"
    padding: "3rem 1rem"
---

# Design System: httphq

## Overview

**Creative North Star: "The Arrivals Board"**

httphq is a board you watch, not a console you operate. Requests land at the
top of the stream, get colour-coded by class, and expire. The whole visual
system is built so a developer working in another window can glance over and
read the board, status first, detail on approach, then go back to what they
were doing. Everything that isn't the arriving traffic is chassis: a slate
field, white panels, hairline seams, and exactly one indigo control per view.

The register is precise, quiet and fast. Restraint here is not minimalism as a
style choice; it is what makes the board readable. The interface never claims
more than it can prove, never decorates a value it captured, and never puts a
second thing in colour next to a method badge. Type is system-native and
loads instantly: there is no webfont anywhere, and that is a feature of a tool
whose entire promise is being usable within seconds of arrival.

Four looks are rejected outright, and all four are confirmed prohibitions
rather than taste: marketing-SaaS gloss (gradient meshes, floating mockups,
testimonial carousels), the dense enterprise console (dark chrome, packed
toolbars, panels nested in panels), terminal cosplay (green-on-black, ASCII
framing, faux-CRT effects), and playful dev-tool mascotry (cartoon characters,
blob illustrations, jokey empty states). Monospace appears throughout, but as
evidence handling: never as costume.

**Key Characteristics:**

- A slate-50 field with white panels; hairline seams, not shadows, make the edges
- One indigo action per view; every other control is a slate outline or bare text
- Full-spectrum colour is spent entirely on the seven HTTP-method badges
- Everything the client sent is monospace; everything httphq says is sans
- A single breakpoint (40rem) governs the entire responsive system
- Liveness is signalled in the browser chrome, tab title count, favicon dot, not by animating the page

## Colors

A near-neutral chassis of eleven slate steps, one indigo accent that carries
every action, and a seven-hue spectrum spent exclusively on classifying
traffic.

### Primary

- **Beacon Periwinkle** (`#707ee7`): the brand primary, carried by the diamond
  mark in `public/logo.svg` and the favicon set. It is lighter and considerably
  softer than the interface accent below. **Recorded intent:** this is the true
  brand primary and the interface accent should converge toward it in a future
  pass. Until that happens the two coexist and neither moves: do not
  half-migrate individual components.
- **Signal Indigo** (`oklch(51.1% 0.262 276.966)`): the current interface
  accent. It fills the one primary button per view, sets the endpoint URL in
  its deeper step, tints the use-case icon tiles in its lightest wash, and is
  the hover colour every quiet control resolves to. Nothing else is indigo.

### Tertiary

The method spectrum. Seven hues, each existing only as a wash-and-ink pair on a
badge, and defined in `public/endpoint.js` rather than in markup:

- **GET: Instrument Blue** (ink `oklch(48.8% 0.243 264.376)` on wash `oklch(97% 0.014 254.604)`): reads, the default arrival.
- **POST: Bench Emerald** (ink `oklch(50.8% 0.118 165.612)` on wash `oklch(97.9% 0.021 166.113)`): creates, and the most common arrival on a webhook endpoint.
- **PUT: Signal Amber** (ink `oklch(55.5% 0.163 48.998)` on wash `oklch(98.7% 0.022 95.277)`): replaces.
- **PATCH: Filament Violet** (ink `oklch(49.1% 0.27 292.581)` on wash `oklch(96.9% 0.016 293.756)`): modifies.
- **DELETE: Warning Rose** (ink `oklch(51.4% 0.222 16.935)` on wash `oklch(96.9% 0.015 12.422)`): removes; shares its hue with the destructive controls, deliberately.
- **HEAD: Neutral Slate** (ink `oklch(37.2% 0.044 257.287)` on wash `oklch(96.8% 0.007 247.896)`): metadata only, so it gets no hue at all.
- **OPTIONS: Preflight Sky** (ink `oklch(50% 0.134 242.749)` on wash `oklch(97.7% 0.013 236.62)`): negotiation, usually a browser's preflight.

Any method outside this set falls back to the HEAD pair.

### Neutral

- **Board Field** (`oklch(98.4% 0.003 247.858)`): the page ground the panels sit on, and, reused deliberately, the fill of every code well and the endpoint URL chip, so raw data reads as recessed into the board.
- **Panel White** (`#ffffff`): every card, disclosure, input and empty state.
- **Hairline** (`oklch(92.9% 0.013 255.508)`): the 1px seam around panels, code wells and the sticky filter rail. The primary edge in the system.
- **Hairline Faint** (`oklch(96.8% 0.007 247.896)`): the internal rules between header rows and detail rows, one step quieter than a panel edge so nested structure never out-shouts the container.
- **Control Edge** (`oklch(86.9% 0.022 252.894)`): the visible stroke on inputs, selects, and secondary buttons; also the dashed edge of the waiting state.
- **Quiet Ink** (`oklch(70.4% 0.04 256.788)`): placeholders and the italic "None" for an absent query string or body.
- **Label Ink** (`oklch(55.4% 0.046 257.417)`): field labels, meta rows, timestamps, and the resting colour of every text-only control.
- **Prose Ink** (`oklch(44.6% 0.043 257.281)`): descriptive sentences on the home and contact pages.
- **Control Ink** (`oklch(37.2% 0.044 257.287)`): secondary button labels, the disclosure summary, and the wordmark.
- **Body Ink** (`oklch(27.9% 0.041 260.031)`): default document text.
- **Strong Ink** (`oklch(20.8% 0.042 265.755)`): headings and card titles.

### Syntax Highlighting

The one part of the palette httphq does not author. Captured JSON, XML and
multipart bodies are highlighted by the pinned highlight.js GitHub light theme
(`@highlightjs/cdn-assets@11.10.0/styles/github.min.css`), which ships plain
sRGB hex outside the OKLCH system above. Verified against rendered captures:

- **Key Blue** (`#005cc5`): object keys, numbers, booleans and the XML prolog: `hljs-attr`, `hljs-number`, `hljs-literal`, `hljs-meta`.
- **String Navy** (`#032f62`): every quoted string value: `hljs-string`.
- **Element Green** (`#22863a`): XML element names: `hljs-name`.
- **Keyword Red** (`#d73a49`): language keywords: `hljs-keyword`.
- Punctuation, braces and tag brackets carry no colour of their own; `hljs-punctuation` and `hljs-tag` inherit Body Ink, which is what keeps a highlighted payload from turning into confetti.

### Named Rules

**The Borrowed Palette Rule.** The four syntax colours are a vendored theme, not
system tokens. Never restyle individual `hljs-*` classes to bring them closer to
the slate-and-indigo world: either swap the whole theme or leave it alone. A
half-retinted syntax palette reads as a bug in the highlighter.

**The Board Rule.** Full-spectrum colour belongs to the method badge and
nothing else. If a new element wants blue, emerald, amber, violet or sky, the
answer is no: that vocabulary means "this is the class of traffic that
arrived", and every additional user dilutes the only colour-coding on the page.

**The One Indigo Rule.** At most one indigo-filled control exists per view. On
the home page it is _Create endpoint_; on the endpoint page it is _Send_;
on contact it is _Send message_. Everything else that is actionable is a slate
outline or bare label ink that turns indigo on hover.

**The Wash-and-Ink Rule.** Coloured chips are always a ~97%-lightness wash
carrying a ~50%-lightness ink of the same hue. There are no saturated fills
with white text anywhere except the single primary button.

**The Rose Is Removal Rule.** Rose means something is about to be destroyed: the _Delete all_ button, the per-request delete hover, the DELETE badge. Its one
sanctioned exception is the unread dot painted onto the favicon, where it means
"traffic landed while you were away". Rose is never used for form validation or
generic error text.

## Typography

**Display / Body Font:** system UI sans (`ui-sans-serif, system-ui, sans-serif`)
**Label/Mono Font:** system monospace (`ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace`)

**Character:** There is no webfont, and that is a decision, not an omission: a
tool promising a working URL in seconds cannot spend its first paint on a font
request. The same commitment governs the stylesheet: it is built ahead of time
from `src/styles/app.css` into a first-party `/app.css`, so no page waits on a
third party to have a layout. The pairing is the operating system's own voice
against its own terminal voice, which is exactly the contrast the product needs:
httphq speaks in sans, the captured traffic speaks in mono.

### Hierarchy

- **Display** (600, 3rem desktop / 1.875rem below 40rem, line-height 1, -0.025em): the home hero headline. Appears once, on one page.
- **Headline** (600, 1.875rem / 1.5rem below 40rem, line-height 1.2): page titles outside the hero, e.g. _Get in touch_.
- **Title** (500, 1rem): card headings and the disclosure summary. Medium weight, never semibold, because titles mark a region rather than competing with the display.
- **Lede** (400, 1.125rem, line-height ~1.56): the one paragraph under the hero, capped at 36rem.
- **Body** (400, 0.875rem, line-height ~1.43): the working size of the entire application chrome. Most of the product is set at 14px, not 16px.
- **Label** (500, 0.75rem, +0.025em, uppercase): field labels: TIME, CLIENT IP, PATH, HEADERS, QUERY STRING, BODY, YOUR UNIQUE URL.
- **Badge** (600, 0.75rem, +0.025em, uppercase): method badges only.
- **Mono** (400, 0.75rem): every captured value: headers, paths, IPs, query strings, bodies, the endpoint URL, and the request UUID.

### Named Rules

**The Evidence Rule.** Anything the user's client actually sent is set in
monospace; anything httphq says about it is set in sans. A captured value never
appears in sans, and interface copy never borrows mono for flavour.

**The Two Uppercase Rule.** Uppercase exists in exactly two places: field
labels and method badges. Both are 0.75rem with +0.025em tracking. Nothing else
in the system is uppercased: not buttons, not navigation, not headings.

## Layout

A single centred column: `max-w-5xl` (64rem) with 1rem gutters, widening to
1.5rem at the 40rem breakpoint. The page is a flex column with the footer
pushed to the bottom, so short pages still plant the footer at the viewport
edge rather than floating it mid-screen.

Measure is capped per content type rather than globally: the hero block at
42rem, its lede paragraph at 36rem, the use-case grid at 56rem, and the contact
form at 36rem. Nothing runs the full 64rem except the request stream, which
needs the width for header tables and body payloads.

**One breakpoint.** The entire system responds at `sm` (40rem) and nowhere
else: there is no `md`, `lg`, or `xl` anywhere in the templates. Below it the
layout is a single stacked column with 1rem gutters and the smaller step of
each type pair; above it the use-case grid becomes two columns, the URL row and
filter bar become horizontal, and secondary button labels appear next to their
icons. Design new surfaces to the same discipline: if a layout needs a second
breakpoint, it is probably too complex for this product.

Vertical rhythm runs on the 0.25rem base scale, using 0.5 / 0.75 / 1 / 1.25 /
1.5 / 2 / 3rem steps. Panels carry 1rem of internal padding below the
breakpoint and 1.25rem above it; the contact form is the one exception at 1.5 →
2rem, because it is a destination rather than a working surface.

The filter rail on the endpoint page is sticky at the top of the viewport,
bleeding into the gutters with a negative inline margin so its rules run edge
to edge, and it uses a 95%-opaque field colour over an 8px backdrop blur so the
stream reads as passing underneath it. It is the only sticky element and the
only backdrop filter in the system.

Detail and header rows are flex rows, not a two-column grid. Above the
breakpoint the label holds a fixed 10rem column with a 0.75rem gap and the value
takes the remainder; below it the label stacks above its value and both run the
full row width, so a long path or header value gets the whole screen instead of
competing with a label column that a phone cannot afford. Each row is its own
element, so its hairline-faint divider spans the entire row rather than sitting
over one column.

Header lists scroll internally at 16rem, bodies at 24rem: long payloads never
push the next request off the board.

## Elevation & Depth

Hairline-first. The 1px hairline seam _is_ the edge of a surface; the shadow
underneath it is a whisper whose only job is to stop a white panel from looking
pasted onto the slate field. Depth in this system is carried by the tonal step
between board field and panel white, reinforced by a seam: not by lift.

### Shadow Vocabulary

- **Resting whisper** (`box-shadow: 0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1)`): every panel, card, disclosure and primary button at rest. Barely perceptible by design.
- **Hover lift** (`box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)`): defined but **currently unused**. It was on the home page's use-case cards, which are not interactive: a surface that lifts under the cursor and then does nothing reads as a broken link. Available for a genuinely interactive card if one ever exists; not to be reintroduced on static content.

### Named Rules

**The Hairline-First Rule.** If a surface needs to be distinguished, give it a
hairline seam before you give it a shadow. A panel with no border and a heavier
shadow is wrong in this system even when it looks fine in isolation.

**The Shadow-Is-A-Response Rule.** Nothing above the resting whisper exists at
rest. `shadow-md` appears only as a hover response; there is no `shadow-lg` or
above anywhere, and adding one would break the flatness the board depends on.

## Shapes

Three radii, and each one encodes what a thing is:

- **0.25rem**: data. Code wells, the endpoint URL chip, method badges. The tightest corner, for things that hold captured bytes.
- **0.375rem**: controls. Every button, input, select, textarea and the use-case icon tiles.
- **0.5rem**: panels. Cards, disclosures, the empty state, request articles.

Borders are 1px and always present on a surface that has an edge; there are no
borderless cards. Request articles clip their contents so the header's bottom
rule meets the panel's rounded corner cleanly.

Iconography is a single family: 24×24 viewBox, `fill="none"`,
`stroke="currentColor"`, `stroke-width="2"`, rendered at 1rem inside controls
and 1.25rem inside icon tiles. Icons inherit their colour from the control and
are always `aria-hidden`, because every icon in this system sits beside a text
label rather than replacing one.

The family has no exceptions: there is no emoji anywhere in the product. The
retention notice carries a stroked alert triangle from the same set, inheriting
label ink like every other icon. Keep it that way: an emoji brings its own
colour and its own per-platform rendering, and one is enough to break the
uniformity that makes this icon set read as a system.

### Named Rules

**The Three Radii Rule.** 0.25 for data, 0.375 for controls, 0.5 for panels. A
new element takes the radius of the category it belongs to, not the radius that
looks best next to its neighbour.

**The Dashed-Means-Waiting Rule.** A dashed border means a container that is
correctly empty and expecting content. It is used once, the waiting state, and
must never be borrowed for a disabled, errored, or drop-target surface.

## Components

### Buttons

- **Character:** calm instruments. Sized for accuracy rather than presence; nothing asks to be admired.
- **Shape:** control radius (0.375rem) on every variant.
- **Primary:** signal indigo fill, white label, medium weight, resting whisper shadow. Two sizes only: 0.75rem/1.5rem padding at 1rem type for the page's main action, and 0.5rem/1rem at 0.875rem inside panels.
- **Secondary:** white fill, control-edge stroke, control-ink label at 0.875rem medium, 0.5rem/0.75rem padding. Hovers to the board field.
- **Destructive:** rose wash fill, rose edge stroke, deep rose label, 0.375rem/0.75rem padding. Hovers one wash step darker.
- **Text-only:** no fill, no border, label ink at 0.75–0.875rem, resolving to signal indigo on hover, or to alert rose when the action deletes. Used for _Copy_, _Copy request_, and per-request _Delete_.
- **Hover / Focus:** fills shift one step lighter on primary, one step darker on destructive. Focus is never suppressed: `outline: none` is always paired with a 2px `focus-visible` ring in **signal indigo lit** (the lighter step, not the fill colour), with a 2px white offset ring on filled buttons, so the ring reads against the indigo it sits on. Destructive controls ring in alert rose instead.
- **Disabled:** 50% opacity and `not-allowed` cursor; used on _Copy all_ when the stream is empty.

### Cards / Containers

- **Corner Style:** panel radius (0.5rem).
- **Background:** panel white on the board field.
- **Shadow Strategy:** resting whisper only. No surface in the product lifts on hover: elevation change is reserved for something that responds to the cursor, and every card here is static content.
- **Border:** 1px hairline, always.
- **Internal Padding:** 1rem below 40rem, 1.25rem above.
- **Composition:** a request article is a header rule-separated from its body, with the method badge and UUID on the left and the copy/delete actions on the right; the body is a stack of labelled regions rather than a nested set of boxes. Its key/value rows follow the flex behaviour described in Layout: 10rem label column above the breakpoint, label stacked above value below it.

### Inputs / Fields

- **Style:** white fill, 1px control-edge stroke, control radius, 0.5rem/0.75rem padding, 0.875rem type. Textareas and header/body fields use the mono stack with `spellcheck="false"`; ordinary text fields use sans.
- **Focus:** `outline: none` paired with a 1px **focus indigo** ring and a **signal indigo lit** border. That is a tighter, quieter treatment than the 2px ring on buttons, because a focused field is already unambiguous. Note the three-way split: fields ring in indigo-400, buttons ring in indigo-500, and only fills use indigo-600.
- **Labels:** the uppercase label style, 0.25–0.375rem above the field. The contact form is the one place labels are sentence-case medium body text, because it is a public form rather than an instrument panel.
- **Select:** native `appearance: none` with a slate chevron inlined as a data-URI background, 1.1em, positioned 0.5rem from the right with 2rem of padding reserved. Note this chevron is the single literal hex in the system (`#64748b`) and is the slate-500 equivalent.

### Navigation

There is none, and that is deliberate: the header is the periwinkle mark beside
a centred wordmark linking home, at 1.875rem bold with tight tracking in control
ink. The link wraps only the mark and wordmark, never the full header band.
Every other route is reachable from the footer: feedback, the GitHub repository
with an inline brand glyph, and the Formspark credit: set at 0.875rem label ink,
centred, hovering to signal indigo.

Every link carries its own padding and a 2.75rem minimum height. As bare inline
text their hit area collapses to the text box, which lands under the 24px target
minimum and makes the page's only exits hard to hit on a phone. Links also carry
the same authored `focus-visible` ring as buttons rather than falling back to the
browser's default outline, which is engine-specific and belongs to no design
system.

### Method Badge

The signature component. A 0.25rem chip, 0.5rem/0.125rem padding, 0.75rem
semibold uppercase with +0.025em tracking, carrying one of the seven wash-and-ink
pairs. It is the first thing on a request header and the only saturated colour
in the stream. Its class map lives in `public/endpoint.js`, not in markup, so a
new method is a one-line addition in one place.

### Body Rendering

A captured body is displayed through one of four paths, chosen by its
`Content-Type`. All four land in the same code well: board-field fill, hairline
border, data radius, 0.75rem mono, scrolling internally at 24rem: so the
container never signals which path ran; only the content does.

- **JSON**: reparsed, pretty-printed at two-space indent, then highlighted.
- **multipart/form-data**: parsed into a part list and serialised as a JSON array of `{name, value}` for fields and `{name, filename, contentType, size}` for files, then highlighted as JSON. A file's bytes are never shown, only its declared metadata.
- **XML**: highlighted in place, unformatted; the payload keeps whatever whitespace it arrived with.
- **Anything else**: HTML-escaped raw text, no highlighting, no reformatting.

An absent body or query string renders as an italic _None_ in quiet ink rather
than an empty well, so a card with nothing in it still reads as a complete
record. Escaping happens on every path: a captured body is attacker-controlled
text and is never trusted as markup.

### Browser Surfaces

The parts the product does not draw still carry its palette, set once in the
base layer: text selection is indigo wash behind indigo ink rather than the
browser's blue, and `accent-color` is signal indigo so native checkboxes and the
search field's clear affordance match the one accent. `color-scheme` is declared
light, so form controls and scrollbars render in the world the system actually
commits to rather than inverting under a dark OS preference.

Motion respects `prefers-reduced-motion`: transitions and the waiting ellipsis
collapse to a resting state. Every animation in this product is decorative, none carries meaning that is lost when it is removed, so reducing it is a clean
substitution, not a degradation.

### Waiting State

Panel-white, dashed control-edge border, 3rem/1rem padding, centred label ink:
a 1rem line reading _Waiting for requests_ followed by an animated ellipsis, and
a 0.875rem line explaining that requests will appear in real time. The ellipsis
is a CSS `content` animation on four steps over 1.2s: the only looping motion
in the product.

### Liveness Indicator

When a request arrives while the tab is hidden, the count is prefixed to the
document title and the favicon is repainted on a canvas: a slate-800 disc, a
white lowercase _h_, and a rose-600 dot at the upper right. Restoring
visibility clears both. This is where the board announces itself: the page
body never flashes, animates, or auto-scrolls to claim attention.

## Do's and Don'ts

### Do:

- **Do** put every new surface on the board field with a panel-white fill, a 1px hairline seam, and the resting whisper shadow. That trio is the system's default surface.
- **Do** set anything the client sent in monospace at 0.75rem, and anything httphq says in sans.
- **Do** give a new region an uppercase 0.75rem/500/+0.025em label ink heading, and separate its rows with hairline-faint rules instead of nesting another bordered box.
- **Do** pair `outline: none` with a visible `focus-visible` ring every single time: 2px signal-indigo-lit plus a white offset ring on buttons, 1px focus-indigo plus a border shift on fields.
- **Do** take the radius from the category: 0.25rem for data, 0.375rem for controls, 0.5rem for panels.
- **Do** design to the 40rem breakpoint alone, stacking below it and going horizontal above it.
- **Do** keep icons at 24×24 viewBox, `stroke-width="2"`, `fill="none"`, `aria-hidden`, beside a text label.
- **Do** signal background activity in the tab title and favicon rather than in the page.

### Don't:

- **Don't** spend blue, emerald, amber, violet or sky on anything but a method badge. That spectrum is the traffic classification and nothing else.
- **Don't** add a second indigo-filled button to a view. One primary action per screen, every other control outlined or bare.
- **Don't** introduce a saturated fill with white text; coloured chips are a ~97% wash with a ~50% ink of the same hue.
- **Don't** use rose for validation errors or generic failure text: rose means removal, plus the unread favicon dot.
- **Don't** reach past the hover lift for elevation. There is no `shadow-lg` in this system, and a borderless card with a bigger shadow is wrong here even when it looks fine alone.
- **Don't** add a webfont. The system stack is a performance commitment on a tool whose promise is a working URL in seconds.
- **Don't** add a second breakpoint. If a layout can't resolve with `sm` alone, simplify the layout.
- **Don't** uppercase anything that isn't a field label or a method badge.
- **Don't** borrow the dashed border for disabled, errored, or drop-target surfaces; it means "correctly empty, expecting content".
- **Don't** animate the page to announce arrivals, auto-scroll the stream, or add a second looping animation beside the waiting ellipsis.
- **Don't** half-migrate the accent toward the mark's periwinkle. Until that convergence is done deliberately across the mark, favicons and UI together, `signal-indigo` remains the interface accent everywhere.
