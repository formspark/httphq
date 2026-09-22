// Every file extension in the tree that Prettier can format must be covered by
// a lint-staged pattern, or the pre-commit hook silently skips it. The gap is
// invisible: files of that type go in unformatted, and nothing fails until
// `format:check` does.
//
// Prettier decides what counts, rather than a list kept here. Asking it which
// parser it would use for a path is the same question the hook asks, so the two
// cannot drift.
//
// Every statement fits in 80 columns, so Prettier prints this file the same at
// any print width and the copies in the sibling repositories stay identical.

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
const BRACES = /^\*\.\{([^}]+)\}$/;
const SINGLE = /^\*\.([a-z0-9]+)$/;
const covered = new Set();
for (const pattern of patterns) {
  const match = BRACES.exec(pattern) ?? SINGLE.exec(pattern);
  if (!match) {
    console.error("Cannot read extensions from lint-staged pattern:");
    console.error(`  ${pattern}`);
    console.error("This check understands `*.{a,b,c}` and `*.ext`.");
    process.exit(1);
  }
  for (const extension of match[1].split(",")) {
    covered.add(extension.trim());
  }
}

const listed = execFileSync("git", ["ls-files"], { encoding: "utf8" });
const tracked = listed.split("\n").filter(Boolean);

const ignored = new Set(
  readFileSync(".prettierignore", "utf8")
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith("#")),
);

function isIgnored(file) {
  return [...ignored].some((path) => file === path || file.startsWith(path));
}

const missing = new Map();
for (const file of tracked) {
  const extension = file.includes(".") ? file.split(".").pop() : "";
  if (!extension || covered.has(extension)) continue;
  // A path Prettier is told to leave alone does not need a hook entry.
  if (isIgnored(file)) continue;

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
console.error("Add them to the lint-staged patterns in package.json, or add");
console.error("the paths to .prettierignore if the formatter should leave");
console.error("them alone.");
console.error("");
process.exit(1);
