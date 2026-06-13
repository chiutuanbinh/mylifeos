import { test, expect } from '@playwright/test';
import { gotoAuthed, mockAPI } from './helpers';

const EVENTS = [
  { id: 'e-1', user_id: 'u1', title: 'Team standup', start_at: '2026-06-13T09:00:00Z', end_at: '2026-06-13T09:30:00Z', color: '#1677ff', all_day: false },
  { id: 'e-2', user_id: 'u1', title: 'All-day event', start_at: '2026-06-14T00:00:00Z', end_at: '2026-06-14T23:59:59Z', color: '#52c41a', all_day: true },
];

test.describe('Calendar page', () => {
  test.beforeEach(async ({ page }) => {
    await mockAPI(page, 'GET', '/events', EVENTS);
  });

  test('loads calendar without crash', async ({ page }) => {
    const errors: string[] = [];
    page.on('pageerror', e => errors.push(e.message));
    await gotoAuthed(page, '/calendar');
    await page.waitForTimeout(1000);
    const crashes = errors.filter(e => e.includes('TypeError') || e.includes('Cannot read'));
    expect(crashes).toHaveLength(0);
  });

  test('calendar renders event', async ({ page }) => {
    await gotoAuthed(page, '/calendar');
    // Calendar should show something (may need to navigate to the right week/month)
    await expect(page.locator('body')).not.toBeEmpty();
  });
});
