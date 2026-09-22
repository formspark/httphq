#!/usr/bin/env node
// Fails when an em dash appears in a tracked text file. The rule, and how to
// rewrite a sentence that has one, is under "Writing" in AGENTS.md.
//
// A markdown table cell holding nothing but a dash marks a missing value and
// passes. Anything else that has to keep a dash, such as a code or a format
// taken from outside, goes in scripts/prose-allowlist.json with its reason:
//   [{ "path": "src/pages/Foundations.tsx", "pattern": "MT—\\d{3}", "reason": "…" }]
// `path` is a file, or a directory ending in "/". `pattern` is a regular
// expression; its matches on that path are ignored.
//
// The same file lives in every repository that deploys to the lab. Change it
// in all of them together. Every statement fits in 80 columns, so Prettier
// prints it the same at any print width and the copies stay identical.
import { execFileSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";

const DASH = "—";
const PLACEHOLDER_CELL = /\|\s*—\s*(?=\|)/g;
const ALLOWLIST = "scripts/prose-allowlist.json";

const allowlist = existsSync(ALLOWLIST) ? readAllowlist() : [];

function readAllowlist() {
  const entries = JSON.parse(readFileSync(ALLOWLIST, "utf8"));
  if (!Array.isArray(entries)) {
    throw new Error(`${ALLOWLIST} must hold an array`);
  }
  return entries.map(toEntry);
}

function toEntry(entry, index) {
  const { path, pattern, reason } = entry ?? {};
  const fields = [path, pattern, reason];
  if (fields.some((field) => typeof field !== "string")) {
    throw new Error(`${ALLOWLIST}[${index}] needs path, pattern and reason`);
  }
  return { path, pattern: new RegExp(pattern, "g") };
}

function appliesTo(entry, file) {
  if (entry.path.endsWith("/")) return file.startsWith(entry.path);
  return file === entry.path;
}

const tracked = execFileSync("git", ["ls-files", "-z"], { encoding: "utf8" });
const self = ["scripts/check-prose.mjs", ALLOWLIST];
const files = tracked.split("\0").filter((f) => f && !self.includes(f));

const findings = [];
for (const file of files) {
  if (!existsSync(file)) continue;
  const buffer = readFileSync(file);
  // A NUL byte in the first block means a binary file: an image, a font.
  if (buffer.subarray(0, 8000).includes(0)) continue;
  const text = buffer.toString("utf8");
  if (!text.includes(DASH)) continue;

  const entries = allowlist.filter((entry) => appliesTo(entry, file));
  text.split("\n").forEach((line, index) => {
    let rest = line.replace(PLACEHOLDER_CELL, "|");
    for (const { pattern } of entries) rest = rest.replace(pattern, "");
    if (rest.includes(DASH)) {
      findings.push(`${file}:${index + 1}: ${line.trim()}`);
    }
  });
}

if (findings.length > 0) {
  const n = findings.length;
  const count = n === 1 ? "1 em dash" : `${n} em dashes`;
  console.error(findings.join("\n"));
  console.error(`\n${count} in prose. Rewrite the clause (AGENTS.md,`);
  console.error(`"Writing"), or allowlist a dash that is data in`);
  console.error(ALLOWLIST);
  // exitCode rather than exit(): exit() can cut the report off while a pipe
  // is still draining it.
  process.exitCode = 1;
} else {
  console.log(`No em dashes in ${files.length} tracked files.`);
}
