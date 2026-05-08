import { test, expect } from '@playwright/test';

test.describe('Authentication Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('should redirect to login if not authenticated', async ({ page }) => {
    await expect(page).toHaveURL(/.*login/);
  });

  test('should request OTP code', async ({ page }) => {
    await page.fill('input[type="tel"]', '+79990000000');
    await page.click('button[type="submit"]');
    
    // Wait for OTP input to appear
    await expect(page.locator('input#otp')).toBeVisible();
  });

  test('should show error for invalid phone number', async ({ page }) => {
    await page.fill('input[type="tel"]', 'invalid');
    await page.click('button[type="submit"]');
    
    // Form should not submit or show validation error
    const otpInput = page.locator('input#otp');
    await expect(otpInput).not.toBeVisible();
  });

  test('should display app title on login page', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Messenger');
  });

  test('should have accessible form elements', async ({ page }) => {
    const phoneInput = page.locator('input#phone');
    await expect(phoneInput).toBeVisible();
    await expect(phoneInput).toHaveAttribute('type', 'tel');
    
    const submitButton = page.locator('button[type="submit"]');
    await expect(submitButton).toBeVisible();
    await expect(submitButton).toHaveText('Получить код');
  });
});
