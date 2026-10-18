# Testing

How the three suites are laid out: the Go tests beside the code, the
Playwright suite in `e2e`, and the Vitest suites for the Node checks in
`tests/scripts`. Running them, and the port the Playwright suite needs, is in
[docs/scripts.md](../docs/scripts.md).

## Go

- Go: a test file covers one subject, is named after it, and names each suite
  after what it exercises, so a handler's tests are found beside the handler.
  One `TestSubject` per function or type, with sentence-style `t.Run` names,
  `t.Helper` in helpers, and `-race`. `harness_test.go` is not a subject: it
  holds `TestMain` and the fixtures shared by every test that drives a real
  request.
- `design_test.go` is the one test that reads documentation rather than code.
  DESIGN.md's frontmatter lists the palette and `src/styles/theme.css` renders
  it, and nothing generates one from the other, so a colour changed in the
  sheet alone would leave the document quietly wrong. It resolves one level of
  `var(--color-x)`, which is what lets the sheet say the mark is brand-600
  while the document states the colour itself, and it treats `#fff` and
  `#ffffff` as the same. Both directions are checked, and both parsers refuse
  to pass when they match implausibly few colours.

## Playwright covers screens and page scripts separately

`e2e/tests` holds one spec per subject. A `*-screen.spec.ts` drives a screen
through captured traffic and asserts what a reader ends up seeing.
`render-body.spec.ts`, `page-helpers.spec.ts` and `har-export.spec.ts` drive the
page scripts directly through `loadPageScripts`, because traffic cannot
express everything those scripts handle: the server parses and re-serialises a
multipart body before storing it, so several part shapes a client can put on the
wire never reach the page intact, and neither a repeated header nor a body that
is not ASCII survives the round trip through the test client.
`capture-api.spec.ts` covers what only a client on the wire can observe, such as
the body limit the server enforces around the handler rather than inside it.

A spec's own fixtures sit at the top of that spec, as the locator and multipart
helpers do. Fixtures used by more than one spec live in
`e2e/tests/support/harness.ts`, along with the zod schemas that parse what the
suite reads at the `JSON.parse` and `response.json()` boundaries. The HAR
schemas are strict, because the export is what those tests are about; the
listing schema drops keys it does not know, so a field the API gains does not
fail an unrelated test. The helpers the page scripts publish on `window` are
declared once, in `types/page-scripts.d.ts`, and the suite reads that
declaration rather than keeping its own.

Screens are reached through `getByTestId`. `testIdAttribute` in
`e2e/playwright.config.ts` points it at the `data-test` attribute the templates
carry, so a test names a hook and never spells an attribute selector. An element
a test reaches for gets a `data-test`; anything a reader can identify by its
role or its text is reached that way instead.

## The Node checks

`tests/scripts` holds one Vitest suite per check under `scripts/`, and
`pnpm test:scripts` runs them. Each suite builds throwaway git repositories and
runs the script inside them, so it needs git and nothing else. The three checks
and their suites are vendored: they are kept identical to their canonical copy,
so a change to one is made there and copied in, never edited here.
