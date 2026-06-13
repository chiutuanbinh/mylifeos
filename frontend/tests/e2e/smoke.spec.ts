import { test, expect } from '@playwright/test';

// Smoke tests: pages load without crash, no console errors
test.describe('Smoke tests', () => {
  test('login page loads', async ({ page }) => {
    await page.goto('/login');
    await expect(page).toHaveTitle(/mylifeos|MyLifeOS|frontend/i);
    // Login page should show sign-in options
    await expect(page.getByRole('button', { name: /google|sign in/i }).first()).toBeVisible();
  });

  test('unauthenticated redirect to login', async ({ page }) => {
    await page.goto('/');
    // Should redirect to /login since no token
    await expect(page).toHaveURL(/\/login/);
  });

  test('auth callback page renders without crash', async ({ page }) => {
    const errors: string[] = [];
    page.on('pageerror', e => errors.push(e.message));
    await page.goto('/auth/callback');
    await page.waitForTimeout(500);
    const fatalErrors = errors.filter(e => !e.includes('supabase') && !e.includes('network'));
    expect(fatalErrors).toHaveLength(0);
  });

  test('unknown route does not crash', async ({ page }) => {
    const errors: string[] = [];
    page.on('pageerror', e => errors.push(e.message));
    await page.goto('/this-does-not-exist');
    await page.waitForTimeout(300);
    expect(errors).toHaveLength(0);
  });
});
