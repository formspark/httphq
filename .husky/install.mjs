// Installs the git hooks, from `pnpm install` via the prepare script.
//
// Only where there is a repository to hook into. The image has no .git, and
// its production install has no husky either, being a dev dependency, so a
// prepare script that imported it unconditionally would fail the image build.

import { existsSync } from "node:fs";

if (existsSync(".git")) {
  const { default: husky } = await import("husky");
  const message = husky();
  if (message) console.log(message);
}
