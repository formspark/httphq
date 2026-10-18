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
const SIBLING = resolve(import.meta.dirname, "check-doc-refs.mjs");
const SCRIPT = existsSync(SIBLING)
  ? SIBLING
  : resolve(import.meta.dirname, "../../scripts/check-doc-refs.mjs");

const sandboxes: string[] = [];

afterEach(() => {
  for (const directory of sandboxes.splice(0)) {
    rmSync(directory, { recursive: true, force: true });
  }
});

const git = (cwd: string, ...args: string[]) => {
  const identity = ["-c", "user.name=test", "-c", "user.email=test@test"];
  const result = spawnSync("git", [...identity, ...args], {
    cwd,
    encoding: "utf8",
  });
  if (result.status !== 0) {
    throw new Error(`git ${args.join(" ")} failed: ${result.stderr}`);
  }
};

const write = (directory: string, files: Record<string, string>) => {
  for (const [path, content] of Object.entries(files)) {
    const target = join(directory, path);
    mkdirSync(dirname(target), { recursive: true });
    writeFileSync(target, content);
  }
};

/**
 * Runs the check in a throwaway git repository holding `files`, every one of
 * them tracked. `deleted` names files that were committed and then removed,
 * so they are in the history and not in the tree.
 */
const run = (files: Record<string, string>, deleted: string[] = []) => {
  const directory = mkdtempSync(join(tmpdir(), "check-doc-refs-"));
  sandboxes.push(directory);
  git(directory, "init", "--quiet");
  if (deleted.length > 0) {
    write(directory, Object.fromEntries(deleted.map((p) => [p, "gone\n"])));
    git(directory, "add", "--all");
    git(directory, "commit", "--quiet", "-m", "add");
    git(directory, "rm", "--quiet", ...deleted);
    git(directory, "commit", "--quiet", "-m", "remove");
  }
  write(directory, files);
  git(directory, "add", "--all");
  const result = spawnSync(process.execPath, [SCRIPT], {
    cwd: directory,
    encoding: "utf8",
  });
  return {
    status: result.status,
    output: `${result.stdout}${result.stderr}`,
  };
};

const pkg = (scripts: Record<string, string> = {}, extra = {}) =>
  JSON.stringify({ name: "fixture", scripts, ...extra });

describe("check-doc-refs.mjs", () => {
  it("passes a tree whose docs point at what is there", () => {
    const result = run({
      "package.json": pkg({ build: "tsc" }),
      "src/main.ts": "export {};\n",
      "README.md": "Run `pnpm build`. The entry is `src/main.ts`.\n",
    });
    expect(result.status).toBe(0);
    expect(result.output).toContain("Doc references hold in 1 files");
  });

  describe("links", () => {
    it("fails on a relative link to a missing file", () => {
      const result = run({ "README.md": "See [the guide](docs/guide.md).\n" });
      expect(result.status).toBe(1);
      expect(result.output).toContain(
        "README.md:1: link target `docs/guide.md` does not exist",
      );
    });

    it("resolves a link from the doc's own directory", () => {
      const result = run({
        "docs/a.md": "See [b](b.md) and [the root](../README.md).\n",
        "docs/b.md": "# B\n",
        "README.md": "# Root\n",
      });
      expect(result.status).toBe(0);
    });

    it("checks an anchor against the target's headings", () => {
      const result = run({
        "README.md":
          "[Setup](DESIGN.md#set-up) and [Colour](DESIGN.md#color).\n",
        "DESIGN.md": "# Design\n\n## Set up\n\n## Colour\n",
      });
      expect(result.status).toBe(1);
      expect(result.output).toContain("no heading for #color in DESIGN.md");
      expect(result.output).not.toContain("#set-up");
    });

    it("follows GitHub's slug rules for punctuation and repeats", () => {
      const doc = [
        "## `pnpm verify`: what it runs",
        "## Notes",
        "## Notes",
        "",
        "[a](#pnpm-verify-what-it-runs) [b](#notes) [c](#notes-1)",
        "",
      ].join("\n");
      const result = run({ "AGENTS.md": doc });
      expect(result.status).toBe(0);
    });

    it("fails on a same-file anchor with no heading", () => {
      const result = run({ "README.md": "# Title\n\n[Up](#top)\n" });
      expect(result.status).toBe(1);
      expect(result.output).toContain("no heading for #top in this file");
    });

    it("leaves URLs and mail links alone", () => {
      const result = run({
        "README.md": "[a](https://example.com/x.md) [b](mailto:a@b.c)\n",
      });
      expect(result.status).toBe(0);
    });

    it("ignores a link inside a code block", () => {
      const result = run({
        "README.md": "```md\n[gone](gone.md)\n```\n\n`[gone](gone.md)`\n",
      });
      expect(result.status).toBe(0);
    });
  });

  describe("backticked paths", () => {
    it("fails on a path under an existing top-level directory", () => {
      const result = run({
        "src/a.ts": "export {};\n",
        "README.md": "The code is in `src/b.ts`.\n",
      });
      expect(result.status).toBe(1);
      expect(result.output).toContain("README.md:1: `src/b.ts` does not exist");
    });

    it("accepts a module path without its extension or index file", () => {
      const result = run({
        "src/lib/db.ts": "export {};\n",
        "src/ui/index.tsx": "export {};\n",
        "README.md": "See `src/lib/db` and `src/ui`, and `src/lib/`.\n",
      });
      expect(result.status).toBe(0);
    });

    it("accepts a line reference on a path", () => {
      const result = run({
        "src/a.ts": "export {};\n",
        "README.md": "See `src/a.ts:12` and `src/a.ts#L3`.\n",
      });
      expect(result.status).toBe(0);
    });

    it("skips module specifiers, URLs, globs and placeholders", () => {
      const doc = [
        "`next/link` `hono/factory` `golang.org/x/net` `@scope/pkg/sub`",
        "`https://example.com/a.ts` `src/**/*.ts` `src/<name>.ts`",
        "`text/html` `origin/main` `/api/forms` `~/notes.md`",
        "",
      ].join("\n");
      const result = run({ "src/a.ts": "export {};\n", "README.md": doc });
      expect(result.status).toBe(0);
    });

    it("resolves a path from a workspace package's root", () => {
      const result = run({
        "pnpm-workspace.yaml": 'packages:\n  - "apps/*"\n',
        "package.json": pkg(),
        "apps/web/package.json": JSON.stringify({ name: "web" }),
        "apps/web/src/page.tsx": "export {};\n",
        "apps/web/README.md": "The page is `src/page.tsx`.\n",
        "README.md": "Also `src/page.tsx`, from the web app.\n",
      });
      expect(result.status).toBe(0);
    });

    it("accepts a path git ignores on purpose", () => {
      const result = run({
        ".gitignore": ".env\ndist/\n",
        "src/a.ts": "export {};\n",
        "README.md": "Copy `.env` and build to `src/out/` via `dist/x.js`.\n",
        "src/.gitignore": "out/\n",
      });
      expect(result.status).toBe(0);
    });

    it("fails on a bare file name that was deleted", () => {
      const result = run(
        { "next.config.ts": "export {};\n", "README.md": "`next.config.js`\n" },
        ["next.config.js"],
      );
      expect(result.status).toBe(1);
      expect(result.output).toContain("`next.config.js` does not exist");
    });

    it("accepts a bare file name that exists anywhere in the tree", () => {
      const result = run({
        "src/components/Button.tsx": "export {};\n",
        "README.md": "Edit `Button.tsx` or `components/Button.tsx`.\n",
      });
      expect(result.status).toBe(0);
    });

    it("leaves a bare name that never was a file in this tree", () => {
      const doc = "It writes `manifest.json`; `Node.js` reads `.ts` files.\n";
      const result = run({ "README.md": doc });
      expect(result.status).toBe(0);
    });

    it("skips paths in a code block unless a command runs them", () => {
      const doc = [
        "```ts",
        'import x from "./src/gone";',
        "```",
        "",
        "```sh",
        "mkdir ./restored && cp src/a.ts ./restored/",
        "node scripts/gone.mjs",
        "```",
        "",
      ].join("\n");
      const result = run({
        "src/a.ts": "export {};\n",
        "scripts/other.mjs": "\n",
        "README.md": doc,
      });
      expect(result.status).toBe(1);
      expect(result.output).toContain("README.md:7: `scripts/gone.mjs`");
      expect(result.output).not.toContain("restored");
      expect(result.output).not.toContain("src/gone");
    });
  });

  describe("pnpm scripts", () => {
    it("fails on a script the package does not define", () => {
      const result = run({
        "package.json": pkg({ build: "tsc" }),
        "README.md": "Run `pnpm build`, then `pnpm run deploy`.\n",
      });
      expect(result.status).toBe(1);
      expect(result.output).toContain('no "deploy" script in package.json');
      expect(result.output).not.toContain('"build"');
    });

    it("ignores pnpm's own commands", () => {
      const doc = "`pnpm install` `pnpm exec tsc` `pnpm dlx x` `pnpm why y`\n";
      const result = run({ "package.json": pkg(), "README.md": doc });
      expect(result.status).toBe(0);
    });

    it("accepts a command a dependency ships", () => {
      const result = run({
        "package.json": pkg({}, { devDependencies: { typescript: "5" } }),
        "README.md": "`pnpm tsc --noEmit`\n",
      });
      expect(result.status).toBe(0);
    });

    it("checks a --filter command against that package", () => {
      const doc = [
        "`pnpm --filter web dev` and `pnpm -F web test`",
        "`pnpm --filter nope dev`",
        "",
      ].join("\n");
      const result = run({
        "pnpm-workspace.yaml": "packages:\n  - apps/*\n",
        "package.json": pkg(),
        "apps/web/package.json": JSON.stringify({
          name: "web",
          scripts: { dev: "vite" },
        }),
        "README.md": doc,
      });
      expect(result.status).toBe(1);
      expect(result.output).toContain('README.md:1: no "test" script in');
      expect(result.output).toContain(
        "README.md:2: no workspace package matches --filter nope",
      );
    });

    it("checks commands in a shell code block", () => {
      const doc = "```sh\n$ pnpm build && pnpm lint\n```\n";
      const result = run({
        "package.json": pkg({ build: "tsc" }),
        "README.md": doc,
      });
      expect(result.status).toBe(1);
      expect(result.output).toContain('README.md:2: no "lint" script');
    });
  });

  describe("agents index", () => {
    it("fails on an agents file AGENTS.md does not link", () => {
      const result = run({
        "AGENTS.md": "- [API](agents/api.md)\n- [Old](./agents/db.md)\n",
        "agents/api.md": "# API\n",
        "agents/db.md": "# DB\n",
        "agents/email.md": "# Email\n",
      });
      expect(result.status).toBe(1);
      expect(result.output).toContain(
        "AGENTS.md:1: AGENTS.md does not link agents/email.md",
      );
      expect(result.output).not.toContain("agents/db.md");
    });
  });

  describe("pnpm-workspace.yaml keys", () => {
    const yaml = [
      "allowBuilds:",
      "  esbuild: false",
      '  "@scope/native": true',
      "overrides:",
      '  lodash.merge: ">=4.6.2" # Added for an advisory.',
      "",
    ].join("\n");

    it("passes keys that are set", () => {
      const doc =
        "`allowBuilds.esbuild` `allowBuilds.@scope/native` " +
        "`overrides.lodash.merge` `audit.ignore`\n";
      const result = run({ "pnpm-workspace.yaml": yaml, "AGENTS.md": doc });
      expect(result.status).toBe(0);
    });

    it("fails on a key that is not set", () => {
      const result = run({
        "pnpm-workspace.yaml": yaml,
        "AGENTS.md": "`allowBuilds.sharp`, see `audit.yml`.\n",
      });
      expect(result.status).toBe(1);
      expect(result.output).toContain(
        "`allowBuilds.sharp` is not set in pnpm-workspace.yaml",
      );
      expect(result.output).not.toContain("audit.yml");
    });
  });

  describe("allowlist", () => {
    const allowlist = (entries: unknown) => JSON.stringify(entries, null, 2);

    it("passes a finding the allowlist names", () => {
      const result = run({
        "src/a.ts": "export {};\n",
        "README.md": "There is no `src/old.ts` any more.\n",
        "scripts/doc-refs-allowlist.json": allowlist([
          { file: "README.md", token: "src/old.ts", reason: "Says it left." },
        ]),
      });
      expect(result.status).toBe(0);
    });

    it("fails on an entry that matches nothing", () => {
      const result = run({
        "README.md": "Plain prose.\n",
        "scripts/doc-refs-allowlist.json": allowlist([
          { file: "README.md", token: "src/old.ts", reason: "Says it left." },
        ]),
      });
      expect(result.status).toBe(1);
      expect(result.output).toContain(
        "scripts/doc-refs-allowlist.json:4: stale entry",
      );
    });

    it("refuses an entry without a reason", () => {
      const result = run({
        "README.md": "Plain prose.\n",
        "scripts/doc-refs-allowlist.json": allowlist([
          { file: "README.md", token: "x" },
        ]),
      });
      expect(result.status).not.toBe(0);
      expect(result.output).toContain("needs file, token and reason");
    });

    it("refuses an allowlist that is not an array", () => {
      const result = run({
        "README.md": "Plain prose.\n",
        "scripts/doc-refs-allowlist.json": "{}",
      });
      expect(result.status).not.toBe(0);
      expect(result.output).toContain("must hold an array");
    });
  });

  it("does not read docs outside the documented set", () => {
    const result = run({
      "notes/scratch.md": "`src/gone.ts` [x](gone.md)\n",
      "node_modules/pkg/README.md": "`src/gone.ts`\n",
      "src/a.ts": "export {};\n",
    });
    expect(result.status).toBe(0);
  });
});
