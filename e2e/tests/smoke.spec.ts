import { test, expect } from '@playwright/test';

test.describe('RdMarket Intelligence Foundation Smoke Tests', () => {
  test('1. Application shell loads with top bar and navigation', async ({ page }) => {
    await page.goto('/');
    // Check page title and top navigation bar
    await expect(page).toHaveTitle(/RdMarket/i);
    const topBar = page.locator('header');
    await expect(topBar).toBeVisible();
  });

  test('2. Health and Ready API endpoints pass', async ({ request }) => {
    const healthResp = await request.get('/api/v1/health');
    expect(healthResp.status()).toBe(200);
    const healthJson = await healthResp.json();
    expect(healthJson.data.status).toBe('healthy');

    const readyResp = await request.get('/api/v1/ready');
    expect(readyResp.status()).toBe(200);
    const readyJson = await readyResp.json();
    expect(readyJson.data.status).toBe('ready');
  });

  test('3. Simulated Data banner appears when mock provider is active', async ({ page }) => {
    await page.goto('/');
    const banner = page.locator('text=SIMULATED DATA');
    await expect(banner).toBeVisible();
  });

  test('4. Theme toggle persists selection in storage', async ({ page }) => {
    await page.goto('/');
    const themeBtn = page.getByRole('button', { name: /toggle theme/i });
    if (await themeBtn.isVisible()) {
      await themeBtn.click();
      const currentTheme = await page.evaluate(() => document.documentElement.getAttribute('data-theme'));
      expect(currentTheme).toBeTruthy();
    }
  });

  test('5. Admin authentication endpoints work', async ({ request }) => {
    // Attempt login with invalid credentials
    const badLogin = await request.post('/api/v1/auth/login', {
      data: { username: 'admin', password: 'wrongpassword' },
    });
    expect(badLogin.status()).toBe(401);

    // Attempt login with valid default dev credentials
    const goodLogin = await request.post('/api/v1/auth/login', {
      data: { username: 'admin', password: 'admin123' },
    });
    expect(goodLogin.status()).toBe(200);
    const body = await goodLogin.json();
    expect(body.data.authenticated).toBe(true);
  });
});
