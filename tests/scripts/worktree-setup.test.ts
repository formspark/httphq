import { spawnSync } from "node:child_process";
import {
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { afterEach, describe, expect, it } from "vitest";

const SCRIPT = resolve(import.meta.dirname, "../../scripts/worktree-setup");
const sandboxes: string[] = [];

afterEach(() => {
  for (const directory of sandboxes.splice(0)) {
    rmSync(directory, { recursive: true, force: true });
  }
});

const git = (cwd: string, args: string[]) => {
  const result = spawnSync("git", args, { cwd, encoding: "utf8" });
  expect(result.status, result.stderr).toBe(0);
};

const fixture = () => {
  const root = mkdtempSync(join(tmpdir(), "worktree-setup-"));
  sandboxes.push(root);
  const main = join(root, "main");
  const worktree = join(root, "worktree");
  const tools = join(root, "tools");
  mkdirSync(main);
  mkdirSync(tools);
  for (const name of ["go", "pnpm"]) {
    writeFileSync(join(tools, name), "#!/bin/sh\nexit 0\n", { mode: 0o755 });
  }
  git(main, ["init", "--quiet"]);
  writeFileSync(join(main, ".worktreeinclude"), ".env\n");
  git(main, ["add", ".worktreeinclude"]);
  git(main, [
    "-c",
    "user.name=Test",
    "-c",
    "user.email=test@example.test",
    "-c",
    "core.hooksPath=/dev/null",
    "commit",
    "--quiet",
    "-m",
    "Create fixture",
  ]);
  git(main, ["worktree", "add", "--quiet", "-b", "worktree", worktree]);
  const run = (cwd = worktree) =>
    spawnSync("bash", [SCRIPT], {
      cwd,
      encoding: "utf8",
      env: { ...process.env, PATH: `${tools}:${process.env.PATH}` },
    });
  return { main, worktree, run };
};

describe("worktree-setup", () => {
  it("preserves a copied variable without a final newline", () => {
    const { main, worktree, run } = fixture();
    writeFileSync(join(main, ".env"), "LOG_LEVEL=warn");

    const result = run();

    expect(result.status, result.stderr).toBe(0);
    expect(readFileSync(join(worktree, ".env"), "utf8")).toBe(
      "LOG_LEVEL=warn\nPORT=11710\n",
    );
    expect(readFileSync(join(main, ".env"), "utf8")).toBe("LOG_LEVEL=warn");
  });

  it("keeps its slot and local values when setup runs again", () => {
    const { worktree, run } = fixture();
    expect(run().status).toBe(0);
    writeFileSync(join(worktree, ".env"), "LOG_LEVEL=debug\nPORT=8080\n");

    expect(run().status).toBe(0);

    expect(readFileSync(join(worktree, ".dev-slot"), "utf8")).toBe("1\n");
    expect(readFileSync(join(worktree, ".env"), "utf8")).toBe(
      "LOG_LEVEL=debug\nPORT=11710\n",
    );
  });

  it("chooses a slot not held by another worktree", () => {
    const { main, worktree, run } = fixture();
    const other = join(main, "../other");
    git(main, ["worktree", "add", "--quiet", "-b", "other", other]);
    writeFileSync(join(other, ".dev-slot"), "1\n");

    expect(run().status).toBe(0);

    expect(readFileSync(join(worktree, ".env"), "utf8")).toBe("PORT=11720\n");
    expect(readFileSync(join(other, ".dev-slot"), "utf8")).toBe("1\n");
  });

  it("refuses the main checkout", () => {
    const { main, run } = fixture();
    const result = run(main);

    expect(result.status).toBe(1);
    expect(result.stderr).toContain("This is the main checkout");
  });
});
