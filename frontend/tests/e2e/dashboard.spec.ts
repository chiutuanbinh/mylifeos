import { test, expect } from '@playwright/test';
import { gotoAuthed, mockAPI } from './helpers';

const SUMMARY = {
  habits_total: 5,
  habits_done_today: 3,
  goals_avg_progress: 65,
  budget_total: 3400,
  budget_spent: 2100,
  net_worth_trend: [110000, 115000, 118500, 121000, 125000, 127450],
  recent_transactions: [],
};

test.describe('Dashboard', () => {
  test('loads and shows summary data', async ({ page }) => {
    await mockAPI(page, 'GET', '/dashboard/summary', SUMMARY);
    await gotoAuthed(page, '/');

    await expect(page.getByText(/habit|dashboard/i).first()).toBeVisible({ timeout: 5000 });
    // No crash errors
    const errors: string[] = [];
    page.on('pageerror', e => errors.push(e.message));
    expect(errors).toHaveLength(0);
  });

  test('handles API error gracefully', async ({ page }) => {
    await mockAPI(page, 'GET', '/dashboard/summary', { error: 'internal' }, 500);
    const errors: string[] = [];
    page.on('pageerror', e => errors.push(e.message));
    await gotoAuthed(page, '/');
    await page.waitForTimeout(1000);
    // Page should not crash (JS error) even when API fails
    const crashes = errors.filter(e => e.includes('TypeError') || e.includes('Cannot read'));
    expect(crashes).toHaveLength(0);
  });
});
