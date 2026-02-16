import { devices } from '@playwright/test';

const appPort = 4173;
const appURL = `http://127.0.0.1:${appPort}`;

export default {
  testDir: './e2e',
  fullyParallel: false,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? [['github'], ['html', { open: 'never' }]] : [['list']],
  use: {
    baseURL: appURL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure'
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] }
    }
  ],
  webServer: {
    command: `pnpm dev --host 127.0.0.1 --port ${appPort}`,
    url: appURL,
    reuseExistingServer: !process.env.CI,
    timeout: 120000,
    env: {
      VITE_SUPABASE_URL: 'http://127.0.0.1:54321',
      VITE_SUPABASE_PUBLISHABLE_KEY: 'e2e-public-key',
      VITE_REFRESH_MS: '5000'
    }
  }
};
