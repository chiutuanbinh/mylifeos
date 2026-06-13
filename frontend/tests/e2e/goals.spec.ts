import { test, expect } from '@playwright/test';
import { gotoAuthed, mockAPI } from './helpers';

const GOALS = [
  {
    id: 'g-1',
    user_id: 'u1',
    name: 'Learn TypeScript',
    description: 'Get proficient',
    target_date: '2026-12-31',
    progress: 40,
    color: '#1677ff',
    created_at: '2026-01-01T00:00:00Z',
    key_results: [
      { id: 'kr-1', goal_id: 'g-1', user_id: 'u1', description: 'Finish course', done: false },
    ],
  },
  {
    id: 'g-2',
    user_id: 'u1',
    name: 'Run marathon',
    description: '',
    target_date: null,
    progress: 10,
    color: '#52c41a',
    created_at: '2026-01-02T00:00:00Z',
    key_results: [],
  },
];

test.describe('Goals page', () => {
  test.beforeEach(async ({ page }) => {
    await mockAPI(page, 'GET', '/goals', GOALS);
  });

  test('loads goals list', async ({ page }) => {
    await gotoAuthed(page, '/goals');
    await expect(page.getByText('Learn TypeScript')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Run marathon')).toBeVisible();
  });

  test('shows key results without crash', async ({ page }) => {
    // This was the original bug — key_results.map crash
    const errors: string[] = [];
    page.on('pageerror', e => errors.push(e.message));
    await gotoAuthed(page, '/goals');
    await expect(page.getByText('Learn TypeScript')).toBeVisible({ timeout: 5000 });
    expect(errors.filter(e => e.includes('map') || e.includes('undefined'))).toHaveLength(0);
  });

  test('goal with empty key_results does not crash', async ({ page }) => {
    const errors: string[] = [];
    page.on('pageerror', e => errors.push(e.message));
    await gotoAuthed(page, '/goals');
    await expect(page.getByText('Run marathon')).toBeVisible({ timeout: 5000 });
    expect(errors).toHaveLength(0);
  });

  test('add goal form opens', async ({ page }) => {
    await gotoAuthed(page, '/goals');
    const addBtn = page.getByRole('button', { name: /add|new|goal/i }).first();
    await expect(addBtn).toBeVisible({ timeout: 5000 });
    await addBtn.click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 3000 });
  });
});
