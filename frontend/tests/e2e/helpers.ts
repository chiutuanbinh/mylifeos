import { Page, Route } from '@playwright/test';

const API = 'http://localhost:8080/api/v1';

// Seed auth token via sessionStorage BEFORE page scripts run.
// The auth store reads sessionStorage 'e2e_token' on init (dev mode only).
export async function seedToken(page: Page) {
  await page.addInitScript(() => {
    sessionStorage.setItem('e2e_token', 'e2e-dev-token');
  });
}

// Mock a backend route. Matches path prefix with optional query string.
export async function mockAPI(page: Page, method: string, path: string, body: unknown, status = 200) {
  await page.route(`${API}${path}**`, (route: Route) => {
    if (route.request().method() !== method) { route.continue(); return; }
    route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
  });
}

// Navigate to a protected page with auth pre-seeded.
export async function gotoAuthed(page: Page, path: string) {
  await seedToken(page);
  await page.goto(path);
  await page.waitForLoadState('networkidle');
}
