package main

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// DESIGN.md is the reference for the palette and src/styles/theme.css is what
// the interface actually renders. Nothing generates one from the other, so the
// only thing keeping them together is this test: a colour changed in the sheet
// and not in the document leaves the document quietly wrong, which is worse
// than having no document at all.

var (
	designColorBlock = regexp.MustCompile(`(?m)^colors:\n((?:  \S+:.*\n)+)`)
	designColorLine  = regexp.MustCompile(`(?m)^  ([a-z0-9-]+):\s*"?([^"\n]+?)"?\s*$`)
	cssColorLine     = regexp.MustCompile(`(?m)^\s*--color-([a-z0-9-]+):\s*([^;]+);`)
	cssVarReference  = regexp.MustCompile(`^var\(--color-([a-z0-9-]+)\)$`)
)

// Tailwind's own keywords rather than palette entries: they carry no value to
// document, so DESIGN.md does not list them.
var cssOnlyKeywords = map[string]bool{"current": true, "transparent": true}

// #fff and #ffffff are the same colour. The sheet writes the short form and the
// document the long one, and neither is wrong.
func expandHex(value string) string {
	if len(value) != 4 || !strings.HasPrefix(value, "#") {
		return value
	}
	var b strings.Builder
	b.WriteByte('#')
	for _, c := range value[1:] {
		b.WriteRune(c)
		b.WriteRune(c)
	}
	return b.String()
}

func designColors(t *testing.T) map[string]string {
	t.Helper()
	source, err := os.ReadFile("../DESIGN.md")
	require.NoError(t, err)
	block := designColorBlock.FindSubmatch(source)
	require.NotNil(t, block, "DESIGN.md has no colors block in its frontmatter")

	colors := map[string]string{}
	for _, match := range designColorLine.FindAllStringSubmatch(string(block[1]), -1) {
		colors[match[1]] = expandHex(strings.TrimSpace(match[2]))
	}
	// A parser that matches nothing would otherwise pass this test in silence.
	require.Greater(t, len(colors), 40, "parsed implausibly few colours from DESIGN.md")
	return colors
}

func themeColors(t *testing.T) map[string]string {
	t.Helper()
	source, err := os.ReadFile("styles/theme.css")
	require.NoError(t, err)

	colors := map[string]string{}
	for _, match := range cssColorLine.FindAllStringSubmatch(string(source), -1) {
		colors[match[1]] = expandHex(strings.TrimSpace(match[2]))
	}
	require.Greater(t, len(colors), 40, "parsed implausibly few colours from theme.css")

	// One level of indirection is resolved, which is what lets the sheet say
	// the mark is brand-600 while the document states the colour itself.
	for name, value := range colors {
		if reference := cssVarReference.FindStringSubmatch(value); reference != nil {
			target, ok := colors[reference[1]]
			require.True(t, ok, "--color-%s points at --color-%s, which is not declared", name, reference[1])
			colors[name] = target
		}
	}
	return colors
}

func TestEveryDocumentedColourMatchesTheStylesheet(t *testing.T) {
	theme := themeColors(t)

	for name, documented := range designColors(t) {
		rendered, ok := theme[name]
		if assert.True(t, ok, "DESIGN.md documents %s, which theme.css does not declare", name) {
			assert.Equal(t, documented, rendered, "%s differs between DESIGN.md and theme.css", name)
		}
	}
}

func TestEveryStylesheetColourIsDocumented(t *testing.T) {
	design := designColors(t)

	for name := range themeColors(t) {
		if cssOnlyKeywords[name] {
			continue
		}
		assert.Contains(t, design, name, "theme.css declares %s, which DESIGN.md does not document", name)
	}
}
