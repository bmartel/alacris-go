import { defineConfig, devices } from '@playwright/test';

const PORT = Number(process.env.ALACRIS_E2E_PORT ?? 8210);

export default defineConfig({
  testDir: './tests',

  // These tests are about DOM identity across a server round trip. A retry
  // would hide a real flake in the live layer, which is exactly the thing worth
  // knowing about.
  retries: 0,
  forbidOnly: !!process.env.CI,
  reporter: process.env.CI ? [['github'], ['list']] : [['list']],

  // The example app holds one shared list, and the multi-tab test asserts on
  // it. Parallel workers would race over the same state.
  workers: 1,
  fullyParallel: false,

  use: {
    baseURL: `http://127.0.0.1:${PORT}`,
    trace: process.env.CI ? 'retain-on-failure' : 'off',
  },

  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],

  // `go run` builds on first use, which can take a while on a cold module
  // cache. The server is the real example app, not a fixture: these tests are
  // only worth anything if they exercise the code someone would copy.
  webServer: {
    command: `go run ../examples/todo -addr 127.0.0.1:${PORT}`,
    url: `http://127.0.0.1:${PORT}/`,
    reuseExistingServer: !process.env.CI,
    timeout: 180_000,
    stdout: 'pipe',
    stderr: 'pipe',
  },
});
