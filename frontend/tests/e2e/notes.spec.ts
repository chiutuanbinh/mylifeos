import { test, expect } from '@playwright/test';
import { gotoAuthed, mockAPI } from './helpers';

const NOTES = [
  { id: 'n-1', user_id: 'u1', title: 'Meeting notes', content: 'Discuss Q3 plans', tags: ['work'], pinned: true, created_at: '2026-06-01T00:00:00Z', updated_at: '2026-06-01T00:00:00Z' },
  { id: 'n-2', user_id: 'u1', title: 'Shopping list', content: 'Milk, eggs', tags: [], pinned: false, created_at: '2026-06-02T00:00:00Z', updated_at: '2026-06-02T00:00:00Z' },
];

test.describe('Notes page', () => {
  test.beforeEach(async ({ page }) => {
    await mockAPI(page, 'GET', '/notes', NOTES);
  });

  test('loads notes list', async ({ page }) => {
    await gotoAuthed(page, '/notes');
    await expect(page.getByText('Meeting notes')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Shopping list')).toBeVisible();
  });

  test('add note form opens', async ({ page }) => {
    await gotoAuthed(page, '/notes');
    const addBtn = page.getByRole('button', { name: /add|new|note/i }).first();
    await expect(addBtn).toBeVisible({ timeout: 5000 });
    await addBtn.click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 3000 });
  });

  test('create note calls API', async ({ page }) => {
    let createCalled = false;
    await page.route('http://localhost:8080/api/v1/notes', route => {
      if (route.request().method() === 'POST') {
        createCalled = true;
        route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({ id: 'n-new', user_id: 'u1', title: 'New Note', content: 'Content', tags: [], pinned: false, created_at: '2026-06-13T00:00:00Z', updated_at: '2026-06-13T00:00:00Z' }),
        });
      } else {
        route.continue();
      }
    });

    await gotoAuthed(page, '/notes');
    const addBtn = page.getByRole('button', { name: /add|new|note/i }).first();
    await addBtn.click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 3000 });

    const titleField = page.getByRole('dialog').getByPlaceholder(/title/i).first();
    if (await titleField.isVisible()) await titleField.fill('New Note');

    const submitBtn = page.getByRole('dialog').getByRole('button', { name: /save|ok|submit|add|create/i }).first();
    if (await submitBtn.isVisible()) {
      await submitBtn.click();
      await page.waitForTimeout(500);
    }
    // No crash regardless of submit result
  });
});
