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
// repository vendors it with this file under tests/scripts/.
const SIBLING = resolve(import.meta.dirname, "check-prose.mjs");
const SCRIPT = existsSync(SIBLING)
  ? SIBLING
  : resolve(import.meta.dirname, "../../scripts/check-prose.mjs");

// Built from its code point so this file passes the check it tests.
const DASH = String.fromCodePoint(0x2014);

const sandboxes: string[] = [];

afterEach(() => {
  for (const directory of sandboxes.splice(0)) {
    rmSync(directory, { recursive: true, force: true });
  }
});

/** Runs the check in a throwaway git repository holding `files`, tracked. */
const run = (files: Record<string, string | Uint8Array>) => {
  const directory = mkdtempSync(join(tmpdir(), "check-prose-"));
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
  return {
    status: result.status,
    stdout: result.stdout,
    stderr: result.stderr,
  };
};

const allowlist = (entries: unknown) => JSON.stringify(entries, null, 2);

describe("check-prose.mjs", () => {
  it("passes a tree with no em dash", () => {
    const result = run({ "README.md": "Plain prose, with commas.\n" });
    expect(result.status).toBe(0);
    expect(result.stdout).toContain("No em dashes in 1 tracked files.");
  });

  it("fails on an em dash in prose and names the file and line", () => {
    const result = run({ "README.md": `Title\n\nOne ${DASH} two.\n` });
    expect(result.status).toBe(1);
    expect(result.stderr).toContain(`README.md:3: One ${DASH} two.`);
    expect(result.stderr).toContain("1 em dash in prose.");
  });

  it("counts every finding across files", () => {
    const result = run({
      "a.md": `One ${DASH} two.\n`,
      "src/b.ts": `// Three ${DASH} four.\n`,
    });
    expect(result.status).toBe(1);
    expect(result.stderr).toContain("2 em dashes in prose.");
  });

  it("passes a table cell holding only a dash", () => {
    const table = `| Name | Value |\n| --- | --- |\n| a | ${DASH} |\n`;
    const result = run({ "DOCS.md": table });
    expect(result.status).toBe(0);
  });

  it("fails on a dash in a table cell that holds prose", () => {
    const row = `| a | ${DASH} | One ${DASH} two |\n`;
    const result = run({ "DOCS.md": row });
    expect(result.status).toBe(1);
    expect(result.stderr).toContain("DOCS.md:1:");
  });

  it("ignores a file that is not tracked", () => {
    const result = run({
      ".gitignore": "notes.md\n",
      "notes.md": `Scratch ${DASH} notes.\n`,
    });
    expect(result.status).toBe(0);
  });

  it("skips a binary file", () => {
    const bytes = new Uint8Array([0, 1, 2, ...Buffer.from(DASH)]);
    const result = run({ "cover.png": bytes });
    expect(result.status).toBe(0);
  });

  describe("allowlist", () => {
    it("ignores the pattern on the file it names", () => {
      const result = run({
        "scripts/prose-allowlist.json": allowlist([
          { path: "src/a.ts", pattern: `ID${DASH}`, reason: "A code." },
        ]),
        "src/a.ts": `const id = "ID${DASH}001";\n`,
      });
      expect(result.status).toBe(0);
    });

    it("ignores every match of the pattern on a line", () => {
      const result = run({
        "scripts/prose-allowlist.json": allowlist([
          { path: "src/a.ts", pattern: `ID${DASH}\\d{3}`, reason: "A code." },
        ]),
        "src/a.ts": `const ids = ["ID${DASH}001", "ID${DASH}002"];\n`,
      });
      expect(result.status).toBe(0);
    });

    it("covers every file under a directory entry ending in a slash", () => {
      const result = run({
        "scripts/prose-allowlist.json": allowlist([
          { path: "src/", pattern: `ID${DASH}`, reason: "A code." },
        ]),
        "src/deep/a.ts": `const id = "ID${DASH}001";\n`,
      });
      expect(result.status).toBe(0);
    });

    it("does not cover a file outside the path it names", () => {
      const result = run({
        "scripts/prose-allowlist.json": allowlist([
          { path: "src/", pattern: `ID${DASH}`, reason: "A code." },
        ]),
        "docs/part.md": `The code is ID${DASH}001.\n`,
      });
      expect(result.status).toBe(1);
      expect(result.stderr).toContain("docs/part.md:1:");
    });

    it("still fails a second dash on a line the pattern matched once", () => {
      const result = run({
        "scripts/prose-allowlist.json": allowlist([
          { path: "src/a.ts", pattern: `ID${DASH}`, reason: "A code." },
        ]),
        "src/a.ts": `// ID${DASH}001 ${DASH} the first release.\n`,
      });
      expect(result.status).toBe(1);
    });

    it("rejects an entry without a reason", () => {
      const result = run({
        "scripts/prose-allowlist.json": allowlist([
          { path: "src/a.ts", pattern: `ID${DASH}` },
        ]),
      });
      expect(result.status).not.toBe(0);
      expect(result.stderr).toContain(
        "scripts/prose-allowlist.json[0] needs path, pattern and reason",
      );
    });

    it("rejects a file that does not hold an array", () => {
      const result = run({
        "scripts/prose-allowlist.json": allowlist({ path: "src/" }),
      });
      expect(result.status).not.toBe(0);
      expect(result.stderr).toContain("must hold an array");
    });
  });
});
