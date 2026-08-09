---
name: httphq
description: "A live arrivals board for HTTP requests. Neutral chassis, one accent drawn from the mark, colour reserved for classifying traffic."
colors:
  white: "#ffffff"
  black: "#000000"
  mark: "#707ee7"
  brand-50: "oklch(96.5% 0.022 275.25)"
  brand-100: "oklch(93.2% 0.042 275.25)"
  brand-200: "oklch(87.5% 0.075 275.25)"
  brand-300: "oklch(79.5% 0.11 275.25)"
  brand-400: "oklch(70% 0.14 275.25)"
  brand-500: "oklch(57.5% 0.157 275.25)"
  brand-600: "oklch(52% 0.157 275.25)"
  brand-700: "oklch(46% 0.15 275.25)"
  brand-800: "oklch(39% 0.13 275.25)"
  brand-900: "oklch(33% 0.105 275.25)"
  neutral-50: "oklch(98.4% 0.003 274)"
  neutral-100: "oklch(96.6% 0.006 274)"
  neutral-200: "oklch(92.8% 0.011 274)"
  neutral-300: "oklch(86.6% 0.018 274)"
  neutral-400: "oklch(70.2% 0.032 274)"
  neutral-500: "oklch(55.2% 0.038 274)"
  neutral-600: "oklch(44.4% 0.036 274)"
  neutral-700: "oklch(37% 0.034 274)"
  neutral-800: "oklch(27.8% 0.032 274)"
  neutral-900: "oklch(20.6% 0.03 274)"
  neutral-950: "oklch(12.8% 0.026 274)"
  get-ink: "oklch(50% 0.155 255)"
  get-wash: "oklch(97% 0.025 255)"
  post-ink: "oklch(50% 0.115 157)"
  post-wash: "oklch(97% 0.018 157)"
  put-ink: "oklch(50% 0.135 62)"
  put-wash: "oklch(97% 0.022 62)"
  patch-ink: "oklch(50% 0.165 308)"
  patch-wash: "oklch(97% 0.026 308)"
  delete-ink: "oklch(50% 0.17 19)"
  delete-wash: "oklch(97% 0.027 19)"
  options-ink: "oklch(50% 0.13 213)"
  options-wash: "oklch(97% 0.021 213)"
  head-ink: "oklch(37% 0.034 274)"
  head-wash: "oklch(96.6% 0.006 274)"
  danger-50: "oklch(97% 0.02 19)"
  danger-100: "oklch(94% 0.04 19)"
  danger-200: "oklch(89% 0.07 19)"
  danger-500: "oklch(62% 0.19 19)"
  danger-600: "oklch(55% 0.185 19)"
  danger-700: "oklch(48% 0.165 19)"
  syntax-key: "#005cc5"
  syntax-string: "#032f62"
  syntax-name: "#22863a"
  syntax-keyword: "#d73a49"
  live: "oklch(62% 0.145 157)"
  pending: "oklch(68% 0.145 62)"
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
  btn:
    backgroundColor: "{colors.white}"
    textColor: "{colors.neutral-700}"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: "0.375rem 0.75rem"
  btn-primary:
    backgroundColor: "{colors.brand-600}"
    textColor: "{colors.white}"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: "0.5rem 1rem"
  btn-primary-hover:
    backgroundColor: "{colors.brand-500}"
  btn-lg:
    backgroundColor: "{colors.brand-600}"
    textColor: "{colors.white}"
    typography: "{typography.lede}"
    rounded: "{rounded.md}"
    padding: "0.75rem 1.5rem"
  btn-secondary:
    backgroundColor: "{colors.white}"
    textColor: "{colors.neutral-700}"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: "0.375rem 0.75rem"
  btn-secondary-hover:
    backgroundColor: "{colors.neutral-100}"
  btn-danger:
    backgroundColor: "{colors.danger-600}"
    textColor: "{colors.white}"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: "0.375rem 0.75rem"
  btn-danger-hover:
    backgroundColor: "{colors.danger-700}"
  btn-inline:
    textColor: "{colors.neutral-500}"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: "0.25rem 0.25rem"
  field:
    backgroundColor: "{colors.white}"
    textColor: "{colors.neutral-800}"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: "0.5rem 0.75rem"
  field-label:
    textColor: "{colors.neutral-500}"
    typography: "{typography.label}"
  panel:
    backgroundColor: "{colors.white}"
    rounded: "{rounded.lg}"
    padding: "1.25rem"
  code-block:
    backgroundColor: "{colors.neutral-50}"
    textColor: "{colors.neutral-800}"
    typography: "{typography.mono}"
    rounded: "{rounded.sm}"
    padding: "0.75rem"
  badge:
    typography: "{typography.badge}"
    rounded: "{rounded.sm}"
    padding: "0.125rem 0.5rem"
  empty-value:
    textColor: "{colors.neutral-500}"
    typography: "{typography.mono}"
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

- A neutral-50 field with white panels; hairline seams and no shadows anywhere make the edges
- One indigo action per view; every other control is a slate outline or bare text
- Full-spectrum colour is spent entirely on the seven HTTP-method badges
- Everything the client sent is monospace; everything httphq says is sans
- A single breakpoint (40rem) governs the entire responsive system
- Liveness is stated, never performed: a connection indicator on the page, a count in the tab title, a dot on the favicon. The page body never flashes, animates or auto-scrolls to claim attention

## Colors

Authored in OKLCH, not taken from a framework's defaults. Three families, built
so they read as one system: a neutral chassis, one accent drawn from the mark,
and a seven-hue spectrum spent only on classifying traffic.

### Primary

- **The Mark** (`#707ee7`): the periwinkle diamond in `public/logo.svg` and the favicon set. It is the origin of the whole accent family and appears unchanged wherever the logo appears.
- **Brand 600** (`oklch(52% 0.157 275.25)`): the accent that fills the one primary control per view. It is the mark's own hue and chroma, darkened until white text clears the contrast floor: the mark itself measures 3.6:1 against white and cannot hold a label. Brand 500 is the hover step and the focus ring; brand 700 sets the endpoint URL; brand 50 tints the icon tiles; brand 400 is the quieter ring on form fields.

**The Chroma Rule.** The accent carries the mark's chroma of 0.157, not more. A
higher-chroma violet reads as a generic framework purple and fights the neutral
chassis; the softness is the identity, and lightness is the only axis that moves
when contrast demands it.

### Tertiary

The method spectrum. Conventional meanings, because GET-blue and POST-green and
DELETE-red are near-universal in HTTP tooling, but built to one construction
rather than inherited: **every ink sits at 50% lightness and every wash at 97%**,
with chroma tuned per hue so the seven carry equal visual weight.

- **GET** ink `oklch(50% 0.155 255)` on wash `oklch(97% 0.025 255)`
- **POST** ink `oklch(50% 0.115 157)` on wash `oklch(97% 0.018 157)`
- **PUT** ink `oklch(50% 0.135 62)` on wash `oklch(97% 0.022 62)`
- **PATCH** ink `oklch(50% 0.165 308)` on wash `oklch(97% 0.026 308)`
- **DELETE** ink `oklch(50% 0.17 19)` on wash `oklch(97% 0.027 19)`
- **OPTIONS** ink `oklch(50% 0.13 213)` on wash `oklch(97% 0.021 213)`
- **HEAD** carries no hue at all: it is metadata, so it takes the neutral ink and wash.

PATCH sits at 308 and GET at 255 deliberately, pushed away from the accent's 275
so a badge is never mistaken for an action. Every pair measures between 4.97:1
and 9.48:1.

### Neutral

The chassis sits on the mark's hue at very low chroma, which is why the greys
read as belonging to the brand rather than as a generic blue-grey.

- **Neutral 50** (`oklch(98.4% 0.003 274)`): the page field, and the fill of every code well.
- **White** (`#ffffff`): every card, disclosure and input.
- **Neutral 200** (`oklch(92.8% 0.011 274)`): the 1px seam around panels and wells. The primary edge in the system.
- **Neutral 100**: internal rules between rows, one step quieter than a panel edge.
- **Neutral 300**: the stroke on inputs and secondary buttons, and the dashed edge of the waiting state.
- **Neutral 400**: input placeholders only.
- **Neutral 500**: field labels, meta rows, and the italic "None" for an absent value.
- **Neutral 600**: descriptive prose.
- **Neutral 700**: secondary button labels, the disclosure summary, and the wordmark.
- **Neutral 800**: default document text.
- **Neutral 900**: headings and card titles.

### Named Rules

**The Board Rule.** Full-spectrum colour belongs to the method badge and nothing
else. If a new element wants blue, green, amber or violet, the answer is no:
that vocabulary means "this is the class of traffic that arrived", and every
additional user dilutes the only colour-coding on the page.

**The One Accent Rule.** At most one brand-filled control exists per view.
Everything else that is actionable is a neutral outline or bare text that turns
brand on hover.

**The Wash-and-Ink Rule.** Coloured chips are a 97%-lightness wash carrying a
50%-lightness ink of the same hue. There are no saturated fills with white text
anywhere except the single primary button.

**The Danger Is Removal Rule.** Danger means something is about to be destroyed:
the delete controls and the DELETE badge, which share the hue deliberately. Its
one sanctioned exception is the unread dot on the favicon. It is never used for
form validation or generic error text.

**The Borrowed Palette Rule.** The four syntax-highlighting colours are a
vendored theme, not system tokens. Never restyle individual `hljs-*` classes to
bring them closer to this palette: either swap the whole theme or leave it
alone. A half-retinted syntax palette reads as a bug in the highlighter.

### Syntax Highlighting

The one part of the palette httphq does not author. Captured JSON, XML and
multipart bodies are highlighted by the pinned highlight.js GitHub light theme,
which ships plain sRGB hex outside the system above:

- **Key Blue** (`#005cc5`): object keys, numbers, booleans and the XML prolog.
- **String Navy** (`#032f62`): quoted string values.
- **Element Green** (`#22863a`): XML element names.
- **Keyword Red** (`#d73a49`): language keywords.
- Punctuation and tag brackets inherit body ink, which is what keeps a highlighted payload from turning into confetti.

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
- **Title** (500, 1rem): card headings. Medium weight, never semibold, because titles mark a region rather than competing with the display. The disclosure summary is deliberately not a Title: it is a control, so it takes the 0.875rem control size even though it is marked up as a heading.
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

A label belongs to its field, so the gap above a field is smaller than the gap
below it: 0.5rem from label to control, 0.75rem from control to the note that
follows. Equal gaps make a label read as floating between two things rather than
naming one of them. Do not reach for `space-y-*` where that distinction matters;
it distributes one gap evenly and cannot express the grouping.

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
element, so its neutral-100 divider spans the entire row rather than sitting
over one column.

Header lists scroll internally at 16rem, bodies at 24rem: long payloads never
push the next request off the board.

## Elevation & Depth

There are no shadows. Not "shadows used sparingly": the stylesheet contains
none, and `box-shadow` on a panel computes to `none`.

Depth is carried by two things only. The tonal step between the neutral-50 page
field and the white panel that sits on it, and a 1px neutral-200 seam around
that panel. A border and a shadow together is the ghost card: two mechanisms
declaring the same edge, each weakening the other. Picking one is what makes a
dense page of stacked panels stay calm.

The primary button is a flat brand fill with no border and no shadow. Nothing on
the page floats.

### Named Rules

**The One Edge Rule.** A surface declares its edge once, with a seam. If a new
component seems to need a shadow, the real problem is that it is not distinct
from what surrounds it, and the fix is tone or spacing.

**The No-Lift Rule.** Nothing changes elevation on hover, because nothing has
elevation to change. Hover is expressed in colour: a fill steps lighter, a
border steps darker, bare text turns brand.

**The Never-Hover-To-The-Field Rule.** A white surface must never hover to
neutral-50, because that is the page it sits on: the control dissolves into the
background at the exact moment it should respond. White surfaces hover to
neutral-100 and darken their seam to neutral-400. With no shadow to fall back
on, the colour step is the whole signal.

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
neutral-500 like every other icon. Keep it that way: an emoji brings its own
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

The system is implemented, not only described. `src/styles/components.css`
carries the classes these entries specify (`.btn` and its variants, `.field`,
`.field-label`, `.region-label`, `.panel`, `.kv-row`, `.icon`, `.badge`,
`.code-block`, `.empty-value`, `.btn-lg`), and templates compose them rather
than repeating utility strings. Every page uses them: a template that re-spells
a component as a utility string is the bug, not a shortcut. Utilities stay the default for one-off composition; anything whose
tokens must not drift between call sites belongs in that file. Before adding a
variant, check whether an existing class should absorb it. Several spellings of
one component is how two nominally identical controls come to sit a few pixels
apart.

There is one surface class, `.panel`: a white fill, a 1px neutral-200 seam and
no shadow. A second, quieter panel would be the One Edge Rule broken by another
name, so a surface that needs to read as nested takes its distinction from
spacing or tone rather than from a class of its own.

`.field-label` and `.region-label` carry one type token between them. The field
label owns the spacing above its input; the region label takes spacing from the
call site, because a heading inside a flex row must not carry a bottom margin.

### Buttons

- **Character:** calm instruments. Sized for accuracy rather than presence; nothing asks to be admired.
- **Shape:** control radius (0.375rem) on every variant.
- **Primary:** brand-600 fill, white label, medium weight, no border and no shadow.
- **Three sizes, each with a job:** `.btn` at 0.375rem/0.75rem carries in-panel secondary and destructive controls; `.btn-primary` at 0.5rem/1rem carries the primary action inside a panel; `.btn-lg` at 0.75rem/1.5rem with 1rem type is reserved for the landing page's single call to action, where the button is the reason the page exists.
- **Secondary:** white fill, neutral-300 stroke, control-ink label at 0.875rem medium, 0.5rem/0.75rem padding. Hovers to the neutral-50 field.
- **Destructive:** one variant only, a solid danger-600 fill with a white label, hovering to danger-700. A tinted destructive button shares its fill with the danger surfaces it sits on, so it stops reading as a button exactly where the stakes are highest. It marks the action that actually destroys, never the one that asks: a control that opens a confirmation is an ordinary secondary button, because colouring it red spends the alarm before anything is at stake and leaves nothing louder for the step that matters. The per-request Delete is not this button either: it is a bare text action, because a solid red fill repeated once per card would shout over the stream it sits in.
- **Text-only:** no fill, no border, neutral-500 at 0.875rem, resolving to brand-600 on hover, or to danger-600 when the action deletes. Used for _Copy_, _Copy request_, and per-request _Delete_. It carries its own padding so the hit area clears the 24px target minimum, and the same authored focus ring as every other control: inside a card that repeats N times, falling back to the browser default multiplies the inconsistency by N.
- **Hover / Focus:** fills shift one step lighter on primary, one step darker on destructive. Focus is never suppressed: `outline: none` is always paired with a 2px `focus-visible` ring in **brand-500** (the lighter step, not the fill colour), with a 2px white offset ring on filled buttons, so the ring reads against the brand fill it sits on. Destructive controls ring in danger-500 instead.
- **Disabled:** 50% opacity and `not-allowed` cursor; used on _Copy all_ when the stream is empty.

### Cards / Containers

- **Corner Style:** panel radius (0.5rem).
- **Background:** white on the neutral-50 field.
- **Shadow Strategy:** none. The seam is the edge; see Elevation & Depth.
- **Border:** 1px hairline, always.
- **Internal Padding:** 1rem below 40rem, 1.25rem above.
- **Composition:** a request article is a header rule-separated from its body, with the method badge and the requested path on the left and the copy/delete actions on the right. The path carries the query string inline, because to the person reading it they are one thing: the URL the client addressed; the body is a stack of labelled regions rather than a nested set of boxes. Its key/value rows follow the flex behaviour described in Layout: 10rem label column above the breakpoint, label stacked above value below it.

### Inputs / Fields

- **Label spacing:** 0.5rem from label to field, 1rem between field groups. Four pixels is a collision, not a relationship.

- **Style:** white fill, 1px neutral-300 stroke, control radius, 0.5rem/0.75rem padding, 0.875rem type. Textareas and header/body fields use the mono stack with `spellcheck="false"`; ordinary text fields use sans.
- **Focus:** `outline: none` paired with a 1px **brand-400** ring and a **brand-500** border. That is a tighter, quieter treatment than the 2px ring on buttons, because a focused field is already unambiguous. Note the three-way split: fields ring in brand-400, buttons ring in brand-500, and only fills use brand-600. Every control uses `:focus-visible`, so a mouse click never paints a ring on a button.
- **Labels:** one label component everywhere, uppercase at 0.75rem, 0.5rem above its field, with 1rem between field groups. The label belongs to its field, so it sits closer to it than the group does to the next one. There is no second label style for public forms: a contact field and a header field are the same control doing the same job, and two spellings of one control is drift, not intent.
- **Select:** native `appearance: none` with a slate chevron inlined as a data-URI background, 1.1em, positioned 0.5rem from the right with 2rem of padding reserved. Note this chevron is the single literal hex in the system (`#64748b`), because a data URI cannot read a CSS variable; it approximates neutral-500 and must be kept in step with it.

### Navigation

There is none, and that is deliberate: the header is the periwinkle mark beside
a centred wordmark linking home, at 1.875rem bold with tight tracking in control
ink. The link wraps only the mark and wordmark, never the full header band.
Every other route is reachable from the footer: feedback, the GitHub repository
with an inline brand glyph, and the Formspark credit: set at 0.875rem neutral-500,
centred, hovering to brand-600.

Every link carries its own padding and a 2.75rem minimum height. Bare inline
text has a hit area no larger than its text box, which lands under the 24px
target minimum and makes the page's only exits hard to hit on a phone. Links also carry
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

An absent body or query string renders as an italic _None_ in neutral-500 rather
than an empty well, so a card with nothing in it still reads as a complete
record. Escaping happens on every path: a captured body is attacker-controlled
text and is never trusted as markup.

### Browser Surfaces

The parts the product does not draw still carry its palette, set once in the
base layer: text selection is indigo wash behind indigo ink rather than the
browser's blue, and `accent-color` is brand-600 so native checkboxes and the
search field's clear affordance match the one accent. `color-scheme` is declared
light, so form controls and scrollbars render in the world the system actually
commits to rather than inverting under a dark OS preference.

Motion respects `prefers-reduced-motion`: transitions and the waiting ellipsis
collapse to a resting state. Every animation in this product is decorative, none carries meaning that is lost when it is removed, so reducing it is a clean
substitution, not a degradation.

### Render Window

The stream renders 25 cards and reveals another 25 per press of a Show more
control, rather than rendering everything the store holds. Each card is around a
hundred DOM nodes, so a full endpoint's worth is a five-figure node count and a
visible stall on every filter change. The store still holds every captured
request; only the DOM is bounded.

### Waiting State

White fill, dashed neutral-300 border, 3rem/1rem padding, centred neutral-500:
a 1rem line reading _Waiting for requests_ followed by an animated ellipsis, and
a 0.875rem line explaining that requests will appear in real time. The ellipsis
is a CSS `content` animation on four steps over 1.2s: the only looping motion
in the product.

### Liveness Indicator

When a request arrives while the tab is hidden, the count is prefixed to the
document title and the favicon is repainted on a canvas: a neutral-800 disc, a
white lowercase _h_, and a danger-600 dot at the upper right. Restoring
visibility clears both. This is where the board announces itself: the page
body never flashes, animates, or auto-scrolls to claim attention.

## Do's and Don'ts

### Do:

- **Do** put every new surface on the neutral-50 field with a white fill and a 1px neutral-200 seam. That pair is the system's default surface, and it carries no shadow.
- **Do** set anything the client sent in monospace at 0.75rem, and anything httphq says in sans.
- **Do** give a new region an uppercase 0.75rem/500/+0.025em neutral-500 heading, and separate its rows with neutral-100 rules instead of nesting another bordered box.
- **Do** pair `outline: none` with a visible `focus-visible` ring every single time: 2px brand-500 plus a white offset ring on buttons, 1px brand-400 plus a border shift on fields.
- **Do** take the radius from the category: 0.25rem for data, 0.375rem for controls, 0.5rem for panels.
- **Do** design to the 40rem breakpoint alone, stacking below it and going horizontal above it.
- **Do** keep icons at 24×24 viewBox, `stroke-width="2"`, `fill="none"`, `aria-hidden`, beside a text label.
- **Do** state connection and background activity plainly: an indicator on the page, a count in the tab title, a dot on the favicon. Never by animating the body.

### Don't:

- **Don't** spend a method hue on anything but a method badge. The seven wash-and-ink pairs are the traffic classification and nothing else.
- **Don't** add a second brand-filled button to one view. Everything beside the primary action is an outlined, bare, or destructive control. A disclosure panel counts as its own view: it is collapsed by default, and once opened it is the thing the reader is working in, so it carries its own primary action rather than deferring to the page's. Two panels on a page may therefore each hold one, which is why the Buttons entry scopes `.btn-primary` to a panel.
- **Don't** introduce a saturated fill with white text beyond the two that exist: the brand primary and the destructive button. Coloured chips are a 97% wash carrying a 50% ink of the same hue, never a fill.
- **Don't** use danger for validation errors or generic failure text: it means removal, plus the unread favicon dot.
- **Don't** add a shadow. There are none in this system, and a border under a shadow is two mechanisms declaring one edge.
- **Don't** add a webfont. The system stack is a performance commitment on a tool whose promise is a working URL in seconds.
- **Don't** add a second breakpoint. If a layout can't resolve with `sm` alone, simplify the layout.
- **Don't** uppercase anything that isn't a field label or a method badge.
- **Don't** borrow the dashed border for disabled, errored, or drop-target surfaces; it means "correctly empty, expecting content".
- **Don't** animate the page to announce arrivals, auto-scroll the stream, or add a second looping animation beside the waiting ellipsis.
- **Don't** reach for a stock framework colour. Tailwind's defaults are cleared in `theme.css`, so a stock utility silently renders nothing; every colour comes from the authored ramps.
