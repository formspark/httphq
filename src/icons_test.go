package main

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every icon is a Lucide drawing kept in one partial per concept under
// views/partials/icons, and every view reaches it through that partial. An SVG
// pasted inline is how two drawings of one concept, or a glyph that means the
// wrong thing, get in without anyone choosing them. Brand marks are drawn the
// way their owners draw them, so they live under views/partials/brands and are
// held to none of the stroke rules.

var (
	iconsDir  = filepath.Join("views", "partials", "icons")
	brandsDir = filepath.Join("views", "partials", "brands")

	svgOpenTag       = regexp.MustCompile(`<svg\b[^>]*>`)
	iconPartialCall  = regexp.MustCompile(`\{\{template "partials/icons/([a-z0-9-]+)"`)
	selectCaretURI   = regexp.MustCompile(`\.app-select\s*\{[^}]*background-image:\s*url\("data:image/svg\+xml,([^"]+)"\)`)
	caretStrokeValue = regexp.MustCompile(`stroke='([^']+)'`)
)

// The attributes that make a drawing Lucide's rather than a lookalike: its
// grid, its stroke weight and its round ends. aria-hidden is ours: every icon
// sits beside a label, so reading it out would say the same thing twice.
var lucideAttributes = []string{
	`viewBox="0 0 24 24"`,
	`fill="none"`,
	`stroke="currentColor"`,
	`stroke-width="2"`,
	`stroke-linecap="round"`,
	`stroke-linejoin="round"`,
	`aria-hidden="true"`,
}

func inDir(path, dir string) bool {
	return strings.HasPrefix(path, dir+string(filepath.Separator))
}

// iconPartialTemplates hands fn every template under views/partials/icons.
func iconPartialTemplates(t *testing.T, fn func(path, source string)) {
	t.Helper()
	eachTemplate(t, func(path, source string) {
		if inDir(path, iconsDir) {
			fn(path, source)
		}
	})
}

func assertNoInlineSVG(t *testing.T) {
	eachTemplate(t, func(path, source string) {
		if inDir(path, iconsDir) || inDir(path, brandsDir) {
			return
		}
		assert.NotRegexp(t, svgOpenTag, source,
			"%s draws an inline svg; include a partial from %s instead", path, iconsDir)
	})
}

func assertLucideDrawing(t *testing.T, path, source string) {
	t.Helper()
	tags := svgOpenTag.FindAllString(source, -1)
	require.Len(t, tags, 1, "%s must hold exactly one svg", path)
	for _, attribute := range lucideAttributes {
		assert.Contains(t, tags[0], attribute, "%s is not drawn the Lucide way", path)
	}
	assert.Regexp(t, `class="icon[ "]`, tags[0], "%s must carry the .icon size hook", path)
}

func assertIconPartialsAreLucide(t *testing.T) {
	seen := 0
	iconPartialTemplates(t, func(path, source string) {
		seen++
		assertLucideDrawing(t, path, source)
	})
	// A walk that finds nothing would otherwise pass in silence.
	require.Greater(t, seen, 5, "found implausibly few icon partials")
}

func assertIconPartialsAreUsed(t *testing.T) {
	used := map[string]bool{}
	eachTemplate(t, func(_, source string) {
		for _, match := range iconPartialCall.FindAllStringSubmatch(source, -1) {
			used[match[1]] = true
		}
	})
	iconPartialTemplates(t, func(path, _ string) {
		name := strings.TrimSuffix(filepath.Base(path), ".html")
		assert.True(t, used[name], "partials/icons/%s is not included anywhere", name)
	})
}

func assertSelectCaretIsLucide(t *testing.T) {
	source, err := os.ReadFile("styles/components.css")
	require.NoError(t, err)
	match := selectCaretURI.FindSubmatch(source)
	require.NotNil(t, match, ".app-select has no data URI caret")
	svg, err := url.PathUnescape(string(match[1]))
	require.NoError(t, err)

	assert.Contains(t, svg, `d='m6 9 6 6 6-6'`, "the caret is not Lucide's ChevronDown")
	for _, attribute := range []string{`fill='none'`, `stroke-width='2'`, `stroke-linecap='round'`, `stroke-linejoin='round'`} {
		assert.Contains(t, svg, attribute, "the caret is not drawn the Lucide way")
	}
	stroke := caretStrokeValue.FindStringSubmatch(svg)
	require.NotNil(t, stroke, "the caret has no stroke colour")
	assert.Equal(t, themeColors(t)["neutral-500"], stroke[1],
		"the caret's stroke has drifted from --color-neutral-500")
}

func TestIcons(t *testing.T) {
	t.Run("no view draws an svg outside the icon and brand partials", assertNoInlineSVG)
	t.Run("each icon partial is one Lucide drawing with the icon class", assertIconPartialsAreLucide)
	t.Run("every icon partial is used by a view", assertIconPartialsAreUsed)
	t.Run("the select caret is Lucide's chevron stroked in neutral-500", assertSelectCaretIsLucide)
}
