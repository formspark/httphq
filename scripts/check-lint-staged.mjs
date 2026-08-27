// Every file extension in the tree that Prettier can format must be covered by
// a lint-staged pattern, or the pre-commit hook silently skips it.
//
// This is not hypothetical. The same gap appeared independently in two
// repositories: one ignored .mjs, .mts, .cjs and .cts while CI linted them, so
// the eslint config, the vitest configs and the build scripts were never
// formatted on the way in; another ignored .mdx while tracking one. Both were
// found by hand, months apart. Nothing stopped the third.
//
// Prettier decides what counts, rather than a list kept here. Asking it which
// parser it would use for a path is the same question the hook asks, so the two
// cannot drift.

import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";

import prettier from "prettier";

const packageJson = JSON.parse(readFileSync("package.json", "utf8"));
const patterns = Object.keys(packageJson["lint-staged"] ?? {});

if (patterns.length === 0) {
  console.error("No lint-staged configuration in package.json.");
  process.exit(1);
}

// The patterns are all of the form `*.{a,b,c}`. Reading the extensions out of
// them beats matching globs, and a pattern in another shape is worth failing on
// rather than quietly ignoring.
const covered = new Set();
for (const pattern of patterns) {
  const match =
    /^\*\.\{([^}]+)\}$/.exec(pattern) ?? /^\*\.([a-z0-9]+)$/.exec(pattern);
  if (!match) {
    console.error(
      `Cannot read extensions from lint-staged pattern: ${pattern}`,
    );
    console.error("This check understands `*.{a,b,c}` and `*.ext`.");
    process.exit(1);
  }
  for (const extension of match[1].split(",")) {
    covered.add(extension.trim());
  }
}

const tracked = execFileSync("git", ["ls-files"], { encoding: "utf8" })
  .split("\n")
  .filter(Boolean);

const ignored = new Set(
  readFileSync(".prettierignore", "utf8")
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith("#")),
);

const missing = new Map();
for (const file of tracked) {
  const extension = file.includes(".") ? file.split(".").pop() : "";
  if (!extension || covered.has(extension)) continue;
  // A path Prettier is told to leave alone does not need a hook entry.
  if ([...ignored].some((entry) => file === entry || file.startsWith(entry)))
    continue;

  const info = await prettier.getFileInfo(file);
  if (!info.inferredParser) continue;

  if (!missing.has(extension)) missing.set(extension, file);
}

if (missing.size === 0) {
  process.exit(0);
}

console.error("");
console.error("Prettier formats these extensions, and lint-staged does not:");
console.error("");
for (const [extension, example] of missing) {
  console.error(`  .${extension}  e.g. ${example}`);
}
console.error("");
console.error(
  "Add them to the lint-staged patterns in package.json, or add the",
);
console.error(
  "paths to .prettierignore if the formatter should leave them alone.",
);
console.error("");
process.exit(1);
