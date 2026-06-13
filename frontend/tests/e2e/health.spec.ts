import { test, expect } from '@playwright/test';
import { gotoAuthed, mockAPI } from './helpers';

const HABITS = [
  { id: 'h-1', user_id: 'u1', name: 'Morning run', icon: '🏃', created_at: '2026-01-01T00:00:00Z' },
  { id: 'h-2', user_id: 'u1', name: 'Read book', icon: '📚', created_at: '2026-01-02T00:00:00Z' },
];
const LOGS = [
  { id: 'l-1', habit_id: 'h-1', user_id: 'u1', logged_date: '2026-06-13', done: true },
  { id: 'l-2', habit_id: 'h-2', user_id: 'u1', logged_date: '2026-06-13', done: false },
];

test.describe('Health/Habits page', () => {
  test.beforeEach(async ({ page }) => {
    await mockAPI(page, 'GET', '/habits', HABITS);
    await mockAPI(page, 'GET', '/habits/logs', LOGS);
  });

  test('loads habits list', async ({ page }) => {
    await gotoAuthed(page, '/health');
    await expect(page.getByText('Morning run')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Read book')).toBeVisible();
  });

  test('add habit form opens', async ({ page }) => {
    await gotoAuthed(page, '/health');
    const addBtn = page.getByRole('button', { name: /add|new|habit/i }).first();
    await expect(addBtn).toBeVisible({ timeout: 5000 });
    await addBtn.click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 3000 });
  });
});
