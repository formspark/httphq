/**
 * What nothing imports: unused files, exports and dependencies.
 *
 * This repository is a Go application with a small JavaScript periphery, so this
 * check covers the periphery and nothing else. `src/` is Go, and the question
 * "does anything reach this?" is answered there by golangci-lint's `unused` in
 * the `go:lint` step and by `go:deadcode`. Do not read a quiet knip run as
 * coverage of the application.
 *
 * What is left for knip is the browser scripts under `public/`, the two build
 * scripts, and the Playwright suite in the `e2e` workspace.
 *
 * Every `entry` line is a claim that something outside the import graph loads
 * that file. Here that is mostly a `<script>` tag in a Go template, which no
 * module graph can see.
 */
import { defineConfig } from "knip/config";

const config = defineConfig({
  workspaces: {
    ".": {
      entry: [
        // Loaded by `<script>` tags in the Go templates under src/views, so
        // nothing imports them. `endpoint.js` is the one tsconfig.json leaves
        // out of its `files` list on purpose; it is still served.
        "public/*.js",
        // Run from package.json, never imported.
        "scripts/*.mjs",
      ],
      // Tailwind's output, written by `pnpm css` and committed. `css:check`
      // fails when it is stale, which is what keeps it honest.
      ignore: ["public/app.css"],
      ignoreDependencies: [
        // scripts/social-card.mjs resolves it through a `createRequire` rooted
        // at ../e2e/package.json, on purpose, so it is not installed twice. The
        // call is a real use that no static resolver can follow.
        "playwright",
      ],
      ignoreBinaries: [
        // The Go toolchain and the shell, which the pinned `go run` lines and
        // the coverage one-liners in package.json call. None comes from npm.
        "go",
        "gofmt",
        "awk",
        "tail",
        // `check:typos` runs typos through uv, a documented prerequisite.
        "uvx",
        // ImageMagick, used by the social-card script.
        "magick",
      ],
    },
  },

  // A symbol exported so a sibling file can reach it is a file-layout choice
  // rather than dead code.
  ignoreExportsUsedInFile: true,

  // An entry above that stops being needed fails the check rather than sitting
  // there.
  treatConfigHintsAsErrors: true,
});

export default config;
