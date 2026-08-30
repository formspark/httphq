import { defineConfig, devices } from "@playwright/test";
import { BASE_URL } from "./tests/support/harness";

export default defineConfig({
  testDir: "./tests",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  // Bounded rather than left to the core count. Every page load pulls Alpine
  // and the syntax highlighter from public CDNs, with no cache shared between
  // contexts, so the ceiling that matters is how many simultaneous third-party
  // fetches stay reliable rather than how many browsers the machine can run.
  workers: process.env.CI ? 1 : 4,
  reporter: process.env.CI ? "github" : "list",
  use: {
    baseURL: BASE_URL,
    trace: "on-first-retry",
    permissions: ["clipboard-read", "clipboard-write"],
    // The templates mark their test hooks with data-test, so getByTestId reads
    // them directly. Spelling the attribute here rather than writing the
    // selector at each call site keeps a hook's name the only thing a test
    // states about it.
    testIdAttribute: "data-test",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  webServer: {
    command: "./bin/httphq",
    cwd: "..",
    url: `${BASE_URL}/api/health`,
    reuseExistingServer: !process.env.CI,
    timeout: 30_000,
  },
});
