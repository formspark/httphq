// Renders scripts/social-card.html to public/social-card.png.
//
// The card exists as markup rather than as a binary someone once exported, so
// it takes its colours from the built stylesheet and cannot drift away from the
// palette the way a hand-made image does. Run `pnpm run social-card` after any
// change to the palette or to the card.

import { execFileSync } from "node:child_process";
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";

// Playwright is a dev dependency of the e2e package rather than of the root, so
// it is resolved from there instead of being installed a second time.
const require = createRequire(new URL("../e2e/package.json", import.meta.url));
const { chromium } = require("playwright");

const cardPath = fileURLToPath(new URL("./social-card.html", import.meta.url));
const outputPath = fileURLToPath(
  new URL("../public/social-card.png", import.meta.url),
);

// The card reads these from the built stylesheet. Tailwind emits a theme
// variable only where something uses it, so a token the product stops using
// disappears from app.css and would leave the card with an unset colour and no
// visible error. Checking them here turns that into a failed build.
const REQUIRED_TOKENS = [
  "--color-white",
  "--color-neutral-50",
  "--color-neutral-200",
  "--color-neutral-500",
  "--color-neutral-600",
  "--color-neutral-800",
  "--color-neutral-900",
  "--color-post-wash",
  "--color-post-ink",
];

const browser = await chromium.launch();
try {
  const page = await browser.newPage({
    viewport: { width: 1200, height: 630 },
    deviceScaleFactor: 1,
  });
  await page.goto(`file://${cardPath}`);

  const unset = await page.evaluate((tokens) => {
    const styles = getComputedStyle(document.documentElement);
    return tokens.filter(
      (token) => styles.getPropertyValue(token).trim() === "",
    );
  }, REQUIRED_TOKENS);
  if (unset.length > 0) {
    throw new Error(
      `stylesheet does not define ${unset.join(", ")}; run \`pnpm run css\` first, ` +
        `and check the product still uses these tokens`,
    );
  }

  await page.locator(".card").screenshot({ path: outputPath });
} finally {
  await browser.close();
}

// An unfurled card is composited onto a background this process cannot know, so
// it ships opaque rather than carrying an alpha channel for a client to resolve.
execFileSync("magick", [
  outputPath,
  "-background",
  "white",
  "-alpha",
  "remove",
  "-alpha",
  "off",
  outputPath,
]);

console.log(`wrote ${outputPath}`);
