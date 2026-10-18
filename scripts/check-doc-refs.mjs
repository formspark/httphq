#!/usr/bin/env node
// Fails when the docs point at something that is not there: a relative link
// or its #anchor, a backticked repository path, a `pnpm <script>` the package
// does not define, an agents/*.md file AGENTS.md never links, or a backticked
// pnpm-workspace.yaml key such as `allowBuilds.some-package`.
//
// Run it from the repository root. It reads the docs agents and contributors
// rely on: AGENTS.md, agents/, README.md, DESIGN.md, PRODUCT.md,
// CONTRIBUTING.md, docs/, tests/README.md and the same files inside each
// apps/* and packages/* directory.
//
// A path counts as existing when git tracks it (or would, once added), or
// when git ignores it on purpose, such as `.env` or a build output. That keeps
// a fresh CI checkout and a working copy full of build output in agreement.
//
// A reference that is meant to dangle, such as a file a doc describes as
// removed, goes in scripts/doc-refs-allowlist.json with its reason:
//   [{ "file": "docs/history.md", "token": "src/old.ts", "reason": "…" }]
// `token` is the text the finding quotes. An entry that no longer matches a
// finding is itself an error, so the list cannot rot.
//
// Every statement fits in 80 columns, so Prettier prints it the same at any
// print width.
import { execFileSync } from "node:child_process";
import { existsSync, readdirSync, readFileSync } from "node:fs";
import { join, posix } from "node:path";

const ALLOWLIST = "scripts/doc-refs-allowlist.json";
const SKIP_DIRS = new Set([
  "node_modules",
  "dist",
  ".next",
  "coverage",
  ".git",
  ".claude",
  ".turbo",
  "storybook-static",
]);

// Extensions that mark a bare backticked word as a file name.
const FILE_EXTENSIONS = new Set(
  (
    "ts tsx mts cts js jsx mjs cjs json jsonc json5 md mdx yml yaml toml " +
    "go mod sum sql css scss html sh bash zsh py rb rs prisma graphql gql " +
    "tf tfvars hcl lock ini conf cfg env svg png jpg jpeg gif webp ico " +
    "avif woff woff2 csv xml txt hbs njk vue svelte astro proto dockerfile"
  ).split(" "),
);

// File names with no extension that are worth checking on their own.
const PLAIN_NAMES = new Set(
  "Dockerfile Makefile Procfile Justfile Caddyfile Brewfile Gemfile".split(" "),
);

// Extensions of files a site serves or a build writes, which a bare name in
// prose is more likely to mean as a URL than as a file in this tree.
const ASSET_EXTENSIONS = new Set(
  "txt xml html svg png jpg jpeg gif webp ico avif woff woff2 csv".split(" "),
);

// Extensions a module path may leave off, as an import specifier does.
const MODULE_SUFFIXES = [".ts", ".tsx", ".mts", ".js", ".jsx", ".mjs"];
const INDEX_SUFFIXES = MODULE_SUFFIXES.map((suffix) => `/index${suffix}`);

// Product names that read like file names.
const NOT_FILES = new Set(
  (
    "node.js next.js vue.js nuxt.js express.js chart.js three.js d3.js " +
    "react.js day.js moment.js alpine.js ember.js socket.io deno.js " +
    "htmx.js p5.js tone.js pixi.js vite.js solid.js"
  ).split(" "),
);

// pnpm commands of its own, which never name a package script.
const PNPM_BUILTINS = new Set(
  (
    "add approve-builds audit bin cache cat-file cat-index config create " +
    "dedupe deploy dlx doctor env exec fetch find-hash help i ignored-builds " +
    "import init install install-test it licenses link list ln ls outdated " +
    "pack patch patch-commit patch-remove peers prune publish rb rebuild " +
    "remove rm root sbom self-update server setup store un uninstall unlink " +
    "up update upgrade version why x"
  ).split(" "),
);

// pnpm flags that take the next word as their value.
const PNPM_VALUE_FLAGS = new Set([
  "--filter",
  "-F",
  "--filter-prod",
  "-C",
  "--dir",
  "--reporter",
  "--workspace-concurrency",
  "--test-pattern",
  "--changed-files-ignore-pattern",
  "--resume-from",
]);

// Command names that differ from the package that ships them.
const BIN_PACKAGES = {
  tsc: "typescript",
  tsserver: "typescript",
  playwright: "@playwright/test",
  sb: "storybook",
  "next-sitemap": "next-sitemap",
  "drizzle-kit": "drizzle-kit",
  "graphql-codegen": "@graphql-codegen/cli",
  gql: "@graphql-codegen/cli",
};

// pnpm-workspace.yaml settings whose dotted keys docs tend to quote.
const CONFIG_KEY = /^(audit|auditConfig|overrides|allowBuilds)\.\S+$/;

// Setting names a doc may point at before anything is set under them, as in
// "an advisory with no fix goes in `audit.ignore`". Entries under overrides
// and allowBuilds name packages, so those have to be there.
const SETTING_NAMES = new Set([
  "audit.ignore",
  "audit.level",
  "auditConfig.ignoreCves",
  "auditConfig.ignoreGhsas",
]);

function isConfigKey(span) {
  if (!CONFIG_KEY.test(span)) return false;
  // `audit.yml` is a file name, not a key.
  const rest = span.slice(span.indexOf(".") + 1);
  return !(FILE_EXTENSIONS.has(rest) && !rest.includes("."));
}

// Dotfiles a doc may name on their own. Any other `.word` is an extension
// or a class name, such as `.snap` or `.active`.
const DOTFILE_NAMES = [
  "env git github npmrc nvmrc node-version tool-versions editorconfig",
  "vscode husky changeset storybook vitepress devcontainer config docker",
].join(" ");
const DOTFILE = new RegExp(
  `^\\.(${DOTFILE_NAMES.replace(/ /g, "|")})|(rc|ignore)$`,
);

const INLINE_COMMAND =
  /^(\S+=\S*\s+)*(pnpm|npm|npx|node|tsx|go|make|bash|sh|git|docker)\s/;
const SHELL_FENCES = new Set(["", "sh", "bash", "shell", "console", "zsh"]);
const COMMAND_START =
  /^(\$\s+)?(pnpm|npm|npx|node|tsx|bash|sh|go|make|docker|\.\/)/;

// The tree --------------------------------------------------------------

function git(args, input) {
  return execFileSync("git", args, {
    encoding: "utf8",
    input,
    stdio: ["pipe", "pipe", "ignore"],
    maxBuffer: 1 << 28,
  });
}

function listFiles() {
  try {
    const args = ["ls-files", "-z", "--cached", "--others"];
    const out = git([...args, "--exclude-standard"]);
    return out.split("\0").filter((f) => f && existsSync(f));
  } catch {
    return walk(".").map((f) => f.replace(/^\.\//, ""));
  }
}

function walk(directory) {
  const found = [];
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    if (SKIP_DIRS.has(entry.name)) continue;
    const path = join(directory, entry.name);
    if (entry.isDirectory()) found.push(...walk(path));
    else found.push(path);
  }
  return found;
}

const files = new Set(listFiles());
const directories = new Set();
const byBasename = new Map();
for (const file of files) {
  const parts = file.split("/");
  for (let i = 1; i < parts.length; i++) {
    directories.add(parts.slice(0, i).join("/"));
  }
  const base = parts.at(-1);
  if (!byBasename.has(base)) byBasename.set(base, []);
  byBasename.get(base).push(file);
}

const topLevel = new Set([...files, ...directories].map(firstSegment));

function firstSegment(path) {
  return path.split("/")[0];
}

function exists(path) {
  const clean = path.replace(/\/+$/, "");
  if (clean === "" || clean === ".") return true;
  return files.has(clean) || directories.has(clean);
}

// Answers in one git call per batch, since a call per path is slow. Each
// path is asked about as a directory too, since `out/` only matches one.
const ignoredCache = new Map();
function ignoredByGit(given) {
  const paths = given.flatMap((p) => [p, `${p.replace(/\/+$/, "")}/`]);
  const unknown = paths.filter((p) => !ignoredCache.has(p));
  if (unknown.length > 0) {
    let out = "";
    try {
      out = git(["check-ignore", "--no-index", "--stdin"], unknown.join("\n"));
    } catch {
      // Exit status 1 means none of them is ignored.
    }
    const hits = new Set(out.split("\n").filter(Boolean));
    for (const p of unknown) ignoredCache.set(p, hits.has(p));
  }
  return paths.some((p) => ignoredCache.get(p));
}

// Workspace packages ------------------------------------------------------

function readJson(path) {
  try {
    return JSON.parse(readFileSync(path, "utf8"));
  } catch {
    return undefined;
  }
}

// A small YAML reader: block mappings and block sequences of scalars, which
// is all pnpm-workspace.yaml holds. Flow collections are kept as strings.
function parseYaml(text) {
  const root = {};
  const stack = [{ indent: -1, node: root }];
  let open = null;
  for (const raw of text.split("\n")) {
    const line = raw.replace(/(^|\s)#.*$/, "");
    if (!line.trim()) continue;
    const indent = line.length - line.trimStart().length;
    const content = line.trim();
    const item = /^-(?:\s+(.*))?$/.exec(content);
    if (item) {
      // A sequence item belongs to the last key that opened with no value.
      if (open && indent >= open.indent) {
        const value = open.parent[open.key];
        if (!Array.isArray(value)) open.parent[open.key] = [];
        open.parent[open.key].push(unquote(item[1] ?? ""));
      }
      continue;
    }
    const pair = /^("[^"]*"|'[^']*'|[^:]+?)\s*:(?:\s+(.*))?$/.exec(content);
    if (!pair) continue;
    while (stack.at(-1).indent >= indent) stack.pop();
    const parent = stack.at(-1).node;
    if (Array.isArray(parent)) continue;
    const key = unquote(pair[1]);
    if (pair[2]) {
      parent[key] = unquote(pair[2]);
      open = null;
    } else {
      parent[key] = {};
      stack.push({ indent, node: parent[key] });
      open = { parent, key, indent };
    }
  }
  return root;
}

function unquote(value) {
  const match = /^(["'])(.*)\1$/.exec(value.trim());
  return match ? match[2] : value.trim();
}

const workspaceYaml = existsSync("pnpm-workspace.yaml")
  ? parseYaml(readFileSync("pnpm-workspace.yaml", "utf8"))
  : undefined;

function workspaceGlobs() {
  const fromYaml = workspaceYaml?.packages;
  const rootJson = readJson("package.json");
  const fromJson = rootJson?.workspaces?.packages ?? rootJson?.workspaces;
  const globs = [
    ...(Array.isArray(fromYaml) ? fromYaml : []),
    ...(Array.isArray(fromJson) ? fromJson : []),
  ];
  // apps/* and packages/* hold packages by convention even without a
  // workspace file, as a repository of separate apps does.
  return [...globs, "apps/*", "packages/*"];
}

function expandGlob(glob) {
  if (glob.startsWith("!")) return [];
  const clean = glob.replace(/^\.\//, "").replace(/\/+$/, "");
  const star = clean.indexOf("*");
  const base = star === -1 ? clean : clean.slice(0, star).replace(/\/$/, "");
  const deep = clean.includes("**");
  const found = [];
  for (const file of files) {
    if (!file.endsWith("package.json")) continue;
    const dir = posix.dirname(file);
    if (star === -1) {
      if (dir === clean) found.push(dir);
    } else if (dir.startsWith(`${base}/`) || base === "") {
      const depth = dir.slice(base ? base.length + 1 : 0).split("/").length;
      if (deep || depth === 1) found.push(dir);
    }
  }
  return found;
}

function loadPackage(dir) {
  const json = readJson(join(dir, "package.json"));
  if (!json) return undefined;
  const deps = {
    ...json.dependencies,
    ...json.devDependencies,
    ...json.optionalDependencies,
    ...json.peerDependencies,
  };
  return {
    dir,
    name: json.name ?? "",
    scripts: new Set(Object.keys(json.scripts ?? {})),
    deps: new Set(Object.keys(deps)),
    bins: new Set(Object.keys(typeof json.bin === "object" ? json.bin : {})),
  };
}

const rootPackage = existsSync("package.json") ? loadPackage(".") : undefined;
const packageDirs = [...new Set(workspaceGlobs().flatMap(expandGlob))].filter(
  (dir) => dir !== ".",
);
const packages = packageDirs.map(loadPackage).filter(Boolean);

// Every apps/* and packages/* directory, with a package.json or not, every
// workspace package, and every project nested a level or two down with a
// manifest of its own: a doc may name a path relative to any of them.
const MANIFEST_NAME = /^(package\.json|pyproject\.toml|go\.mod|Cargo\.toml)$/;

function manifestDir(file) {
  const parts = file.split("/");
  const shallow = parts.length === 2 || parts.length === 3;
  if (!shallow || !MANIFEST_NAME.test(parts.at(-1))) return null;
  return parts.slice(0, -1).join("/");
}

const appDirs = [
  ...new Set([
    ...[...directories].filter((d) => /^(apps|packages)\/[^/]+$/.test(d)),
    ...packageDirs,
    ...[...files].map(manifestDir).filter(Boolean),
  ]),
].filter((d) => !d.split("/").some((part) => SKIP_DIRS.has(part)));

function packageFor(file) {
  let best;
  for (const pkg of packages) {
    if (file.startsWith(`${pkg.dir}/`)) {
      if (!best || pkg.dir.length > best.dir.length) best = pkg;
    }
  }
  return best;
}

// The docs ---------------------------------------------------------------

function isDoc(file) {
  if (!file.endsWith(".md")) return false;
  if (file.split("/").some((part) => SKIP_DIRS.has(part))) return false;
  const top = [
    "AGENTS.md",
    "README.md",
    "DESIGN.md",
    "PRODUCT.md",
    "CONTRIBUTING.md",
    "tests/README.md",
  ];
  if (top.includes(file)) return true;
  if (/^(agents|docs)\//.test(file)) return true;
  if (/^(apps|packages)\/[^/]+\//.test(file)) {
    const base = posix.basename(file);
    return ["README.md", "AGENTS.md", "DESIGN.md", "PRODUCT.md"].includes(base);
  }
  return false;
}

const docs = [...files].filter(isDoc).sort();

// Splits a document into lines, each marked as prose or code. Code lines
// carry whether they read as shell, so commands in them are still checked.
function classify(text) {
  const lines = text.split("\n");
  const out = [];
  let fence = null;
  let previousBlank = true;
  let inList = false;
  let indentedCode = false;
  for (const line of lines) {
    if (fence) {
      const close = new RegExp(`^\\s*${fence.marker[0]}{${fence.size},}\\s*$`);
      if (close.test(line)) {
        fence = null;
        out.push({ line, kind: "fence-edge" });
      } else {
        out.push({ line, kind: "code", shell: fence.shell });
      }
      previousBlank = false;
      continue;
    }
    const open = /^\s*(`{3,}|~{3,})\s*([\w+-]*)/.exec(line);
    if (open) {
      const lang = open[2].toLowerCase();
      fence = {
        marker: open[1],
        size: open[1].length,
        shell: SHELL_FENCES.has(lang) ? lang || "plain" : false,
      };
      out.push({ line, kind: "fence-edge" });
      indentedCode = false;
      continue;
    }
    const blank = line.trim() === "";
    if (/^\s{0,3}([-*+]|\d+[.)])\s/.test(line)) inList = true;
    else if (!blank && !/^\s/.test(line)) inList = false;
    const indented = /^( {4,}|\t)/.test(line);
    if (indented && !inList && (previousBlank || indentedCode)) {
      indentedCode = true;
      out.push({ line, kind: "code", shell: "plain" });
    } else {
      if (!blank) indentedCode = false;
      out.push({ line, kind: "prose" });
    }
    previousBlank = blank;
  }
  return out;
}

const INLINE_CODE = /(`+)(?!`)([\s\S]*?[^`])\1(?!`)/g;

function codeSpans(line) {
  return [...line.matchAll(INLINE_CODE)].map((m) => m[2].trim());
}

// Anchors -----------------------------------------------------------------

const anchorCache = new Map();

// GitHub's rule: lower-case, drop everything but letters, digits, spaces,
// hyphens and underscores, turn each space into a hyphen, and number repeats.
function slug(heading) {
  const text = heading
    .replace(/<[^>]+>/g, "")
    .replace(/!?\[([^\]]*)\]\([^)]*\)/g, "$1")
    .replace(/[`*~]/g, "")
    .replace(/(^|\s)_+|_+(\s|$)/g, "$1$2")
    .trim()
    .toLowerCase();
  return text.replace(/[^\p{L}\p{M}\p{N}\s_-]/gu, "").replace(/\s/g, "-");
}

// The directory of a docs site the file belongs to, if any: one holding a
// .vitepress, .vuepress or docusaurus config.
function siteRoot(file) {
  const parts = file.split("/");
  for (let i = parts.length - 1; i >= 0; i--) {
    const dir = parts.slice(0, i).join("/") || ".";
    const prefix = dir === "." ? "" : `${dir}/`;
    if (directories.has(`${prefix}.vitepress`)) return dir;
    if (directories.has(`${prefix}.vuepress`)) return dir;
    if ([...files].some((f) => f.startsWith(`${prefix}docusaurus.config.`))) {
      return dir;
    }
  }
  return null;
}

function anchorsOf(file) {
  if (anchorCache.has(file)) return anchorCache.get(file);
  const anchors = new Set();
  const counts = new Map();
  const add = (base) => {
    const n = counts.get(base) ?? 0;
    counts.set(base, n + 1);
    anchors.add(n === 0 ? base : `${base}-${n}`);
  };
  const lines = classify(readFileSync(file, "utf8"));
  lines.forEach(({ line, kind }, i) => {
    if (kind !== "prose") return;
    for (const m of line.matchAll(/<a\s[^>]*(?:id|name)="([^"]+)"/g)) {
      anchors.add(m[1]);
    }
    const atx = /^\s{0,3}#{1,6}\s+(.*?)\s*#*\s*$/.exec(line);
    const next = lines[i + 1];
    const setext =
      next?.kind === "prose" &&
      line.trim() &&
      !/^\s{0,3}([-*+]|\d+[.)])\s/.test(line) &&
      /^\s{0,3}(=+|-+)\s*$/.test(next.line) &&
      (i === 0 || lines[i - 1].line.trim() === "");
    const heading = atx ? atx[1] : setext ? line.trim() : null;
    if (heading === null) return;
    const custom = /\s*\{#([\w-]+)\}\s*$/.exec(heading);
    if (custom) anchors.add(custom[1]);
    const base = slug(custom ? heading.slice(0, custom.index) : heading);
    add(base);
    // A docs site generator collapses the runs of hyphens GitHub keeps.
    if (siteRoot(file)) add(base.replace(/-{2,}/g, "-").replace(/^-|-$/g, ""));
  });
  anchorCache.set(file, anchors);
  return anchors;
}

// Findings and the allowlist -----------------------------------------------

const findings = [];
function report(file, line, token, message) {
  findings.push({ file, line, token, message });
}

const counts = { links: 0, paths: 0, scripts: 0, keys: 0 };

// Links ----------------------------------------------------------------

// [text](target "title"): the text may hold one level of brackets, and the
// target one level of parentheses.
const LINK_TEXT = String.raw`!?\[((?:[^[\]]|\[[^\]]*\])*)\]`;
const LINK_TARGET = String.raw`(<[^>]*>|(?:[^\s()]|\([^\s()]*\))+)`;
const LINK_TITLE = String.raw`(?:\s+(?:"[^"]*"|'[^']*'|\([^)]*\)))?`;
const LINK = new RegExp(
  String.raw`${LINK_TEXT}\(\s*${LINK_TARGET}${LINK_TITLE}\s*\)`,
  "g",
);
const REFERENCE = /^\s{0,3}\[([^\]]+)\]:\s*(<[^>]*>|\S+)/;

function checkLink(doc, lineNo, rawTarget) {
  const target = rawTarget.replace(/^<|>$/g, "");
  if (/^[a-z][a-z0-9+.-]*:/i.test(target) || target.startsWith("//")) return;
  if (/[{}]|\$\{/.test(target)) return;
  counts.links++;
  const hash = target.indexOf("#");
  const pathPart = hash === -1 ? target : target.slice(0, hash);
  const anchor = hash === -1 ? null : target.slice(hash + 1);
  let path;
  try {
    path = decodeURIComponent(pathPart.split("?")[0]);
  } catch {
    path = pathPart;
  }

  let resolved = doc;
  if (path !== "") {
    resolved = resolveLink(doc, path);
    if (!resolved) {
      if (ignoredByGit(linkCandidates(doc, path))) return;
      report(doc, lineNo, target, `link target \`${path}\` does not exist`);
      return;
    }
  }
  if (anchor && resolved.endsWith(".md") && files.has(resolved)) {
    let decoded = anchor;
    try {
      decoded = decodeURIComponent(anchor);
    } catch {
      // Keep the anchor as written.
    }
    const anchors = anchorsOf(resolved);
    const wanted = [decoded, decoded.toLowerCase()];
    if (!wanted.some((a) => anchors.has(a))) {
      const where = resolved === doc ? "this file" : resolved;
      report(doc, lineNo, target, `no heading for #${anchor} in ${where}`);
    }
  }
}

function linkCandidates(doc, path) {
  const docDir = posix.dirname(doc);
  if (!path.startsWith("/")) return [posix.join(docDir, path)];
  // A leading slash means the repository root on GitHub, and the site root in
  // a docs site built from docs/ (or whichever directory the doc sits in).
  const roots = [""];
  const docsRoot = /^(.*?\bdocs)\//.exec(doc)?.[1];
  if (docsRoot) roots.push(docsRoot);
  return roots.map((root) => posix.join(root || ".", path.slice(1)));
}

function resolveLink(doc, path) {
  // A docs site serves a page without its .md; GitHub needs the file name.
  const site = path.startsWith("/");
  for (const candidate of linkCandidates(doc, path)) {
    const clean = posix.normalize(candidate).replace(/\/+$/, "");
    if (clean.startsWith("..")) continue;
    const tries = site
      ? [clean, `${clean}.md`, `${clean}/index.md`, `${clean}/README.md`]
      : [clean];
    const hit = tries.find(exists);
    if (hit !== undefined) return hit === "" ? "." : hit;
  }
  return null;
}

// Backticked paths ---------------------------------------------------------

function cleanToken(word) {
  let token = word.trim();
  token = token.replace(/^[("'[{<]+|[)"'\]},;.:!?>]+$/g, "");
  // `--config=vite.config.ts` names the file after the equals sign.
  const eq = /^-{1,2}[\w-]+=(.+)$/.exec(token);
  if (eq) token = eq[1];
  token = token.replace(/^\w+=(?=\.{0,2}\/)/, "");
  // A line or column reference: src/a.ts:12, src/a.ts:12:4, src/a.ts#L12.
  token = token.replace(/(:\d+)+$/, "").replace(/#L\d+(-L?\d+)?$/, "");
  return token;
}

function extensionOf(token) {
  const base = token.split("/").at(-1);
  const dot = base.lastIndexOf(".");
  if (dot <= 0) return "";
  return base.slice(dot + 1).toLowerCase();
}

function isSkippable(token) {
  if (token.length < 3) return true;
  if (/:\/\/|^www\.|^mailto:|^[a-z]+:[^/]/i.test(token)) return true;
  if (/[*?{}$<>|`\\]|\.\.\.|^#|^@|^~|^\/|^-/.test(token)) return true;
  if (/\s/.test(token)) return true;
  if (/[()=,'"]/.test(token)) return true;
  // A go module path: the first segment is a domain.
  const domain = /^[\w-]+(\.[\w-]+)+\//.test(token);
  if (domain && !topLevel.has(firstSegment(token))) return true;
  return false;
}

// Decides whether a word names a repository path, and which kind.
function pathKind(token, doc) {
  if (isSkippable(token)) return null;
  const relative = token.startsWith("./") || token.startsWith("../");
  const stripped = token.replace(/^\.\//, "");
  if (relative) return "relative";
  if (stripped.includes("/")) {
    const first = firstSegment(stripped);
    const roots = rootsFor(doc);
    const known = roots.some((root) => exists(posix.join(root, first)));
    if (known) return "rooted";
    if (FILE_EXTENSIONS.has(extensionOf(stripped))) return "suffix";
    return null;
  }
  if (/^\.[\w-]/.test(stripped)) return DOTFILE.test(stripped) ? "bare" : null;
  if (PLAIN_NAMES.has(stripped)) return "bare";
  const ext = extensionOf(stripped);
  if (!FILE_EXTENSIONS.has(ext)) return null;
  // Served files such as robots.txt or sitemap.xml are often generated.
  if (ASSET_EXTENSIONS.has(ext)) return null;
  if (NOT_FILES.has(stripped.toLowerCase())) return null;
  // A version or a number, such as 1.2.json, is not worth guessing at.
  if (/^[\d.]+$/.test(stripped.slice(0, -ext.length - 1))) return null;
  return "bare";
}

function rootsFor(doc) {
  return [...new Set([".", posix.dirname(doc), ...appDirs])];
}

// Paths deleted somewhere in the history of HEAD. A bare file name that is
// in neither the tree nor this list never was a repository file: a runtime
// output, a remote object, a file in another project. One that is here was
// renamed or removed, and the doc still names it.
let deletedPaths;
function wasDeleted(paths, anywhere) {
  if (!deletedPaths) {
    deletedPaths = new Set();
    try {
      const args = ["log", "--no-renames", "--diff-filter=D", "--name-only"];
      const out = git([...args, "--format=", "HEAD"]);
      for (const path of out.split("\n")) if (path) deletedPaths.add(path);
    } catch {
      // No history, as in a fresh repository: nothing was deleted.
    }
  }
  if (paths.some((path) => deletedPaths.has(path))) return true;
  if (!anywhere) return false;
  for (const path of deletedPaths) {
    if (path.endsWith(`/${anywhere}`)) return true;
  }
  return false;
}

function resolvesAt(base) {
  if (exists(base)) return true;
  const ext = extensionOf(base);
  if (FILE_EXTENSIONS.has(ext) && ext !== "") return false;
  const suffixes = [...MODULE_SUFFIXES, ...INDEX_SUFFIXES];
  return suffixes.some((suffix) => exists(base + suffix));
}

function checkPath(doc, lineNo, word) {
  const token = cleanToken(word);
  const kind = pathKind(token, doc);
  if (!kind) return;
  counts.paths++;
  const stripped = token.replace(/^\.\//, "").replace(/\/+$/, "");
  const docDir = posix.dirname(doc);
  let candidates;
  if (kind === "relative") {
    // Relative to the doc, or to the root a command runs from.
    candidates = [posix.normalize(posix.join(docDir, token))];
    if (token.startsWith("./")) candidates.push(stripped);
  } else {
    candidates = rootsFor(doc).map((root) =>
      posix.normalize(posix.join(root, stripped)),
    );
  }
  candidates = candidates.filter((c) => !c.startsWith(".."));
  // A relative path that climbs out of the repository is not ours to check.
  if (candidates.length === 0) return;
  if (candidates.some(resolvesAt)) return;
  const loose = kind === "bare" || kind === "suffix";
  if (kind !== "relative") {
    // A file name on its own, or a path written from inside some nested
    // project, means one somewhere in the tree.
    const base = posix.basename(stripped);
    const hits = byBasename.get(base) ?? [];
    if (hits.some((hit) => hit === stripped || hit.endsWith(`/${stripped}`))) {
      return;
    }
  }
  if (token.includes("[")) return;
  if (ignoredByGit(candidates)) return;
  // A bare name counts where it would sit: the root, the doc's directory or
  // a project root. A partial path counts wherever it once was.
  const partial = kind === "suffix" ? stripped : null;
  if (loose && !wasDeleted(candidates, partial)) return;
  report(doc, lineNo, token, `\`${token}\` does not exist`);
}

const SCRIPT_EXTENSIONS = new Set("sh bash mjs cjs js ts mts py rb".split(" "));

function isScriptPath(token) {
  const first = firstSegment(token.replace(/^\.\//, ""));
  if (first === "scripts" || first === "bin") return token.includes("/");
  return SCRIPT_EXTENSIONS.has(extensionOf(token)) && token.includes("/");
}

// pnpm scripts ------------------------------------------------------------

function checkCommands(doc, lineNo, text) {
  const segments = text.split(/&&|\|\||[;|]|\$\(|`/);
  for (const segment of segments) {
    const words = segment.trim().split(/\s+/).filter(Boolean);
    const at = words.findIndex((w) => w === "pnpm" || w === "npm");
    if (at === -1) continue;
    checkPnpm(doc, lineNo, words[at], words.slice(at + 1));
  }
}

function checkPnpm(doc, lineNo, tool, args) {
  const filters = [];
  let dir = null;
  let recursive = false;
  let ifPresent = false;
  let command = null;
  let i = 0;
  for (; i < args.length; i++) {
    const arg = args[i];
    if (arg.startsWith("-")) {
      const [flag, value] = arg.split(/=(.*)/s);
      const takes = PNPM_VALUE_FLAGS.has(flag) && value === undefined;
      const v = takes ? args[++i] : value;
      if (flag === "--filter" || flag === "-F" || flag === "--filter-prod") {
        if (v) filters.push(v);
      } else if (flag === "-C" || flag === "--dir") {
        dir = v;
      } else if (flag === "-r" || flag === "--recursive") {
        recursive = true;
      } else if (flag === "--if-present") {
        ifPresent = true;
      }
      continue;
    }
    command = arg;
    break;
  }
  if (!command) return;
  let script = command;
  if (command === "run" || command === "run-script") {
    script = args.slice(i + 1).find((a) => !a.startsWith("-"));
    if (args.slice(i + 1).includes("--if-present")) ifPresent = true;
    if (!script) return;
  } else if (tool === "npm" && !["test", "start"].includes(command)) {
    return;
  } else if (tool === "pnpm" && PNPM_BUILTINS.has(command)) {
    return;
  }
  script = script.replace(/[`"',.;:)]+$/, "").replace(/^["'(]+/, "");
  if (!script || /[<>${}*]|^\d/.test(script) || ifPresent) return;
  if (!rootPackage && packages.length === 0) return;

  let scope;
  let label = `${tool} ${command === script ? script : `run ${script}`}`;
  if (filters.length > 0) {
    scope = filters.flatMap(matchFilter);
    const named = filters.join(" ");
    label = `${tool} --filter ${named} ${script}`;
    if (scope.length === 0) {
      counts.scripts++;
      const msg = `no workspace package matches --filter ${named}`;
      report(doc, lineNo, label, msg);
      return;
    }
  } else if (dir) {
    const target = posix.normalize(dir.replace(/^\.\//, ""));
    const fromDoc = posix.normalize(posix.join(posix.dirname(doc), dir));
    scope = packages.filter((p) => p.dir === target || p.dir === fromDoc);
    if (target === "." && rootPackage) scope.push(rootPackage);
    if (scope.length === 0) return;
  } else if (recursive) {
    scope = packages.length > 0 ? packages : [rootPackage].filter(Boolean);
  } else {
    const own = packageFor(doc);
    // A doc in an app with no package.json of its own describes some other
    // tool's commands, which this tree cannot check.
    const app = /^(apps|packages)\/[^/]+/.exec(doc)?.[0];
    if (app && !own) return;
    scope = [rootPackage, own].filter(Boolean);
    // A command in a package's own doc runs in that package; a command in a
    // root doc of a repository with no root package.json runs in some app.
    if (!rootPackage && !own) scope = packages;
  }
  counts.scripts++;
  if (scope.some((pkg) => pkg.scripts.has(script))) return;
  if (tool === "pnpm" && command !== "run" && isBin(script, scope)) return;
  // Prose often sets the directory in words ("run it in apps/api"), so a
  // script any workspace package defines is taken as meant for that one.
  if (filters.length === 0 && !dir) {
    if (packages.some((pkg) => pkg.scripts.has(script))) return;
    if (tool === "pnpm" && isBin(script, packages)) return;
    scope = [rootPackage, ...packages].filter(Boolean);
  }
  const names = scope.map((p) => (p.dir === "." ? "package.json" : p.dir));
  const where = names.length > 3 ? "any package" : names.join(", ");
  report(doc, lineNo, label, `no "${script}" script in ${where}`);
}

function isBin(name, scope) {
  const all = [rootPackage, ...scope].filter(Boolean);
  for (const pkg of all) {
    if (existsSync(join(pkg.dir, "node_modules/.bin", name))) return true;
    if (pkg.bins.has(name)) return true;
    const want = BIN_PACKAGES[name] ?? name;
    if (pkg.deps.has(want)) return true;
    for (const dep of pkg.deps) {
      if (dep.endsWith(`/${name}`) || dep === `${name}-cli`) return true;
    }
  }
  return false;
}

function matchFilter(raw) {
  if (raw.startsWith("!")) return [];
  const filter = raw.replace(/^\.\.\.\^?|\^?\.\.\.$/g, "");
  const pathish = /^\{(.+)\}$/.exec(filter)?.[1] ?? filter;
  const dir = posix.normalize(pathish.replace(/^\.\//, ""));
  const pattern = new RegExp(
    `^${filter.replace(/[.+?^$()|[\]\\]/g, "\\$&").replace(/\*/g, ".*")}$`,
  );
  const all = [rootPackage, ...packages].filter(Boolean);
  return all.filter(
    (p) =>
      pattern.test(p.name) ||
      p.name.split("/").at(-1) === filter ||
      p.dir === dir ||
      (filter.startsWith("./") && p.dir.startsWith(`${dir}/`)),
  );
}

// pnpm-workspace.yaml keys ------------------------------------------------

function hasKey(node, path) {
  if (path === "") return true;
  if (!node || typeof node !== "object" || Array.isArray(node)) return false;
  if (Object.hasOwn(node, path)) return true;
  // A key may hold dots itself, as a package name can.
  let dot = path.indexOf(".");
  while (dot !== -1) {
    const head = path.slice(0, dot);
    if (Object.hasOwn(node, head) && hasKey(node[head], path.slice(dot + 1))) {
      return true;
    }
    dot = path.indexOf(".", dot + 1);
  }
  return false;
}

function checkConfigKey(doc, lineNo, span) {
  if (!workspaceYaml || !isConfigKey(span)) return;
  counts.keys++;
  const key = span.replace(/["']/g, "");
  if (SETTING_NAMES.has(key) || hasKey(workspaceYaml, key)) return;
  report(doc, lineNo, span, `\`${span}\` is not set in pnpm-workspace.yaml`);
}

// agents/*.md --------------------------------------------------------------

function checkAgentsIndex() {
  const agents = [...files].filter((f) => /^agents\/[^/]+\.md$/.test(f));
  if (agents.length === 0) return;
  if (!files.has("AGENTS.md")) {
    for (const file of agents) {
      report(file, 1, file, "AGENTS.md does not exist to link this file");
    }
    return;
  }
  const linked = new Set();
  const text = readFileSync("AGENTS.md", "utf8");
  for (const { line, kind } of classify(text)) {
    if (kind !== "prose") continue;
    const bare = line.replace(INLINE_CODE, "");
    const targets = [
      ...[...bare.matchAll(LINK)].map((m) => m[2]),
      ...(REFERENCE.exec(bare) ? [REFERENCE.exec(bare)[2]] : []),
    ];
    for (const target of targets) {
      const path = target.replace(/^<|>$/g, "").split("#")[0];
      linked.add(posix.normalize(path));
    }
  }
  for (const file of agents) {
    if (!linked.has(file)) {
      report("AGENTS.md", 1, file, `AGENTS.md does not link ${file}`);
    }
  }
}

// The run ----------------------------------------------------------------

function checkDoc(doc) {
  const lines = classify(readFileSync(doc, "utf8"));
  lines.forEach(({ line, kind, shell }, index) => {
    const lineNo = index + 1;
    if (kind === "fence-edge") return;
    if (kind === "code") {
      if (!shell) return;
      const command = line.trim().replace(/^\$\s+/, "");
      if (command.startsWith("#") || !command) return;
      if (shell === "plain" && !COMMAND_START.test(command)) return;
      const words = command.replace(/\s+#\s.*$/, "").split(/\s+/);
      checkCommands(doc, lineNo, command.replace(/\s+#\s.*$/, ""));
      // In a command, only a script it runs counts. Other paths in it may be
      // what the command creates.
      for (const word of words) {
        const token = cleanToken(word);
        if (!isScriptPath(token)) continue;
        const kindOf = pathKind(token, doc);
        if (kindOf === "rooted" || kindOf === "relative") {
          checkPath(doc, lineNo, word);
        }
      }
      return;
    }
    for (const span of codeSpans(line)) {
      checkCommands(doc, lineNo, span);
      checkConfigKey(doc, lineNo, span);
      if (isConfigKey(span)) continue;
      const single = !/\s/.test(span);
      const command = INLINE_COMMAND.test(span);
      const words = single || command ? span.split(/\s+/) : [];
      for (const word of words) checkPath(doc, lineNo, word);
    }
    const bare = line.replace(INLINE_CODE, "");
    for (const m of bare.matchAll(LINK)) checkLink(doc, lineNo, m[2]);
    const ref = REFERENCE.exec(bare);
    if (ref) checkLink(doc, lineNo, ref[2]);
  });
}

function readAllowlist() {
  if (!existsSync(ALLOWLIST)) return [];
  const entries = JSON.parse(readFileSync(ALLOWLIST, "utf8"));
  if (!Array.isArray(entries)) {
    throw new Error(`${ALLOWLIST} must hold an array`);
  }
  return entries.map((entry, index) => {
    const { file, token, reason } = entry ?? {};
    if ([file, token, reason].some((f) => typeof f !== "string")) {
      throw new Error(`${ALLOWLIST}[${index}] needs file, token and reason`);
    }
    return { file, token, index, used: false };
  });
}

const allowlist = readAllowlist();
for (const doc of docs) checkDoc(doc);
checkAgentsIndex();

const errors = [];
for (const finding of findings) {
  const entry = allowlist.find(
    (e) => e.file === finding.file && e.token === finding.token,
  );
  if (entry) {
    entry.used = true;
    continue;
  }
  errors.push(`${finding.file}:${finding.line}: ${finding.message}`);
}
const allowlistLines = existsSync(ALLOWLIST)
  ? readFileSync(ALLOWLIST, "utf8").split("\n")
  : [];
for (const entry of allowlist) {
  if (entry.used) continue;
  const quoted = JSON.stringify(entry.token);
  const at = allowlistLines.findIndex((line) => line.includes(quoted));
  const what = `${entry.file} no longer has ${entry.token}`;
  errors.push(`${ALLOWLIST}:${at + 1 || 1}: stale entry, ${what}`);
}

if (errors.length > 0) {
  const n = errors.length;
  console.error(errors.join("\n"));
  const count = n === 1 ? "1 broken reference" : `${n} broken references`;
  console.error(`\n${count} in the docs. Fix the doc, or allowlist`);
  console.error(`a reference that is meant to dangle in ${ALLOWLIST}.`);
  process.exitCode = 1;
} else {
  const { links, paths, scripts, keys } = counts;
  const summary = `${links} links, ${paths} paths, ${scripts} scripts`;
  console.log(
    `Doc references hold in ${docs.length} files: ${summary}, ${keys} keys.`,
  );
}
