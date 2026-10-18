import { spawnSync } from "node:child_process";
import {
  existsSync,
  mkdirSync,
  mkdtempSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { afterEach, describe, expect, it } from "vitest";

// The script sits beside this file where it is kept, and in scripts/ once a
// repository vendors it with this file under tests/scripts/. It imports
// prettier, which resolves from the script's own location.
const SIBLING = resolve(import.meta.dirname, "check-lint-staged.mjs");
const SCRIPT = existsSync(SIBLING)
  ? SIBLING
  : resolve(import.meta.dirname, "../../scripts/check-lint-staged.mjs");

const sandboxes: string[] = [];

afterEach(() => {
  for (const directory of sandboxes.splice(0)) {
    rmSync(directory, { recursive: true, force: true });
  }
});

/** Runs the check in a throwaway git repository holding `files`, tracked. */
const run = (files: Record<string, string>) => {
  const directory = mkdtempSync(join(tmpdir(), "check-lint-staged-"));
  sandboxes.push(directory);
  for (const [path, content] of Object.entries(files)) {
    const target = join(directory, path);
    mkdirSync(dirname(target), { recursive: true });
    writeFileSync(target, content);
  }
  spawnSync("git", ["init", "--quiet"], { cwd: directory });
  spawnSync("git", ["add", "--all"], { cwd: directory });
  const result = spawnSync(process.execPath, [SCRIPT], {
    cwd: directory,
    encoding: "utf8",
  });
  return { status: result.status, stderr: result.stderr };
};

const packageJson = (lintStaged?: Record<string, string>) =>
  JSON.stringify({ name: "fixture", "lint-staged": lintStaged }) + "\n";

describe("check-lint-staged.mjs", () => {
  it("passes when every formattable extension has a pattern", () => {
    const result = run({
      "package.json": packageJson({
        "package.json": "sort-package-json",
        "*.{ts,json,md}": "prettier --write",
      }),
      ".prettierignore": "",
      "src/a.ts": "export {};\n",
      "README.md": "# Fixture\n",
    });
    expect(result.status).toBe(0);
  });

  it("names an extension Prettier formats and no pattern covers", () => {
    const result = run({
      "package.json": packageJson({ "*.{ts,json}": "prettier --write" }),
      ".prettierignore": "",
      "src/a.ts": "export {};\n",
      "docs/guide.md": "# Guide\n",
      "styles/site.css": "a {}\n",
    });
    expect(result.status).toBe(1);
    expect(result.stderr).toContain(".md  e.g. docs/guide.md");
    expect(result.stderr).toContain(".css  e.g. styles/site.css");
  });

  it("names each uncovered extension once, with the first file found", () => {
    const result = run({
      "package.json": packageJson({ "*.json": "prettier --write" }),
      ".prettierignore": "",
      "a.md": "# A\n",
      "b.md": "# B\n",
    });
    expect(result.status).toBe(1);
    expect(result.stderr).toContain(".md  e.g. a.md");
    expect(result.stderr).not.toContain("b.md");
  });

  it("reads extensions listed with spaces after the commas", () => {
    const result = run({
      "package.json": packageJson({ "*.{json, md}": "prettier --write" }),
      ".prettierignore": "",
      "README.md": "# Fixture\n",
    });
    expect(result.status).toBe(0);
  });

  it("reads a single-extension pattern", () => {
    const result = run({
      "package.json": packageJson({
        "*.json": "prettier --write",
        "*.css": "prettier --write",
      }),
      ".prettierignore": "",
      "styles/site.css": "a {}\n",
    });
    expect(result.status).toBe(0);
  });

  it("claims no extension for a pattern that names a single file", () => {
    const result = run({
      "package.json": packageJson({ "package.json": "sort-package-json" }),
      ".prettierignore": "",
    });
    expect(result.status).toBe(1);
    expect(result.stderr).toContain(".json  e.g. package.json");
  });

  it("leaves alone a file type Prettier has no parser for", () => {
    const result = run({
      "package.json": packageJson({ "*.json": "prettier --write" }),
      ".prettierignore": "",
      "_typos.toml": "[default]\n",
      "notes.txt": "Plain text.\n",
      Dockerfile: "FROM scratch\n",
    });
    expect(result.status).toBe(0);
  });

  it("skips a path listed in .prettierignore", () => {
    const result = run({
      "package.json": packageJson({ "*.json": "prettier --write" }),
      ".prettierignore": "# Generated.\ngenerated\n/styles/\n",
      "generated/routes.ts": "export {};\n",
      "styles/site.css": "a {}\n",
    });
    expect(result.status).toBe(0);
  });

  it("does not skip a sibling that only shares an ignored prefix", () => {
    const result = run({
      "package.json": packageJson({ "*.json": "prettier --write" }),
      ".prettierignore": "coverage\n",
      "coverage-notes.md": "# Notes\n",
    });
    expect(result.status).toBe(1);
    expect(result.stderr).toContain(".md  e.g. coverage-notes.md");
  });

  it("fails without a lint-staged configuration", () => {
    const result = run({
      "package.json": packageJson(),
      ".prettierignore": "",
    });
    expect(result.status).toBe(1);
    expect(result.stderr).toContain("No lint-staged configuration");
  });

  it("fails on a pattern it cannot read extensions from", () => {
    const result = run({
      "package.json": packageJson({ "src/**/*.ts": "eslint --fix" }),
      ".prettierignore": "",
    });
    expect(result.status).toBe(1);
    expect(result.stderr).toContain("Cannot read extensions");
    expect(result.stderr).toContain("src/**/*.ts");
  });
});
