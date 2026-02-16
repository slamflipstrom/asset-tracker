import { expect, test, type Page } from '@playwright/test';

type AssetType = 'crypto' | 'stock';

interface AssetRow {
  id: number;
  symbol: string;
  name: string;
  type: AssetType;
}

interface LotRow {
  id: number;
  userId: string;
  assetId: number;
  quantity: number;
  unitCost: number;
  purchasedAt: string;
}

interface MockState {
  accessToken: string;
  userId: string;
  userEmail: string;
  assets: AssetRow[];
  lots: LotRow[];
  pricesByAssetId: Map<number, number>;
  nextLotID: number;
}

function seedMockState(): MockState {
  return {
    accessToken: 'smoke-access-token',
    userId: 'e2e-user-1',
    userEmail: 'smoke@example.com',
    assets: [
      { id: 1, symbol: 'BTC', name: 'Bitcoin', type: 'crypto' },
      { id: 2, symbol: 'ETH', name: 'Ethereum', type: 'crypto' },
      { id: 3, symbol: 'AAPL', name: 'Apple', type: 'stock' }
    ],
    lots: [
      {
        id: 1,
        userId: 'e2e-user-1',
        assetId: 1,
        quantity: 1,
        unitCost: 100,
        purchasedAt: '2026-02-16T00:00:00.000Z'
      }
    ],
    pricesByAssetId: new Map([[1, 150]]),
    nextLotID: 2
  };
}

function positionsResponse(state: MockState) {
  const byAsset = new Map<number, { qty: number; totalCost: number }>();

  for (const lot of state.lots) {
    const current = byAsset.get(lot.assetId) ?? { qty: 0, totalCost: 0 };
    current.qty += lot.quantity;
    current.totalCost += lot.quantity * lot.unitCost;
    byAsset.set(lot.assetId, current);
  }

  const positions = Array.from(byAsset.entries())
    .map(([assetId, aggregate]) => {
      const asset = state.assets.find((row) => row.id === assetId);
      const avgCost = aggregate.totalCost / aggregate.qty;
      const currentPrice = state.pricesByAssetId.get(assetId) ?? null;
      const unrealizedPL =
        currentPrice === null ? null : (currentPrice - avgCost) * aggregate.qty;

      return {
        asset_id: assetId,
        symbol: asset?.symbol ?? `#${assetId}`,
        name: asset?.name ?? 'Unknown asset',
        type: asset?.type ?? 'crypto',
        total_qty: Number(aggregate.qty.toFixed(10)),
        avg_cost: Number(avgCost.toFixed(10)),
        current_price: currentPrice,
        unrealized_pl: unrealizedPL === null ? null : Number(unrealizedPL.toFixed(10))
      };
    })
    .sort((a, b) => a.symbol.localeCompare(b.symbol));

  return positions;
}

function lotsResponse(state: MockState) {
  return state.lots.map((lot) => {
    const asset = state.assets.find((row) => row.id === lot.assetId);
    return {
      id: lot.id,
      asset_id: lot.assetId,
      symbol: asset?.symbol ?? `#${lot.assetId}`,
      name: asset?.name ?? 'Unknown asset',
      type: asset?.type ?? 'crypto',
      quantity: lot.quantity,
      unit_cost: lot.unitCost,
      purchased_at: lot.purchasedAt
    };
  });
}

async function setupMockRoutes(page: Page) {
  const state = seedMockState();

  await page.route('**/auth/v1/**', async (route) => {
    const request = route.request();
    const url = new URL(request.url());

    if (url.pathname.endsWith('/auth/v1/token') && request.method() === 'POST') {
      const grantType = url.searchParams.get('grant_type');
      if (grantType !== 'password' && grantType !== 'refresh_token') {
        await route.fulfill({
          status: 400,
          contentType: 'application/json',
          body: JSON.stringify({ error_description: 'unsupported grant type' })
        });
        return;
      }

      if (grantType === 'password') {
        const payload = request.postDataJSON() as { email?: string };
        if (payload.email) {
          state.userEmail = payload.email;
        }
      }

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          access_token: state.accessToken,
          token_type: 'bearer',
          expires_in: 3600,
          refresh_token: 'smoke-refresh-token',
          user: {
            id: state.userId,
            email: state.userEmail
          }
        })
      });
      return;
    }

    if (url.pathname.endsWith('/auth/v1/user') && request.method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: state.userId,
          email: state.userEmail
        })
      });
      return;
    }

    if (url.pathname.endsWith('/auth/v1/logout') && request.method() === 'POST') {
      await route.fulfill({ status: 204, body: '' });
      return;
    }

    await route.fulfill({
      status: 404,
      contentType: 'application/json',
      body: JSON.stringify({ message: 'not mocked' })
    });
  });

  await page.route('**/api/v1/**', async (route) => {
    const request = route.request();
    const method = request.method();
    const url = new URL(request.url());
    const path = url.pathname;

    if (method === 'GET' && path.endsWith('/api/v1/positions')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(positionsResponse(state))
      });
      return;
    }

    if (method === 'GET' && path.endsWith('/api/v1/lots')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(lotsResponse(state))
      });
      return;
    }

    if (method === 'GET' && path.endsWith('/api/v1/assets/search')) {
      const query = (url.searchParams.get('q') ?? '').trim().toLowerCase();
      const filteredAssets = state.assets.filter((asset) => {
        if (query === '') {
          return true;
        }
        return (
          asset.symbol.toLowerCase().includes(query) || asset.name.toLowerCase().includes(query)
        );
      });

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(filteredAssets)
      });
      return;
    }

    if (method === 'POST' && path.endsWith('/api/v1/lots')) {
      const payload = request.postDataJSON() as {
        asset_id: number;
        quantity: number;
        unit_cost: number;
        purchased_at: string;
      };
      const lotID = state.nextLotID++;
      state.lots.push({
        id: lotID,
        userId: state.userId,
        assetId: payload.asset_id,
        quantity: payload.quantity,
        unitCost: payload.unit_cost,
        purchasedAt: payload.purchased_at
      });

      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({ id: lotID })
      });
      return;
    }

    if (method === 'PATCH' && path.includes('/api/v1/lots/')) {
      const lotID = Number(path.split('/').at(-1));
      const payload = request.postDataJSON() as {
        quantity: number;
        unit_cost: number;
        purchased_at: string;
      };
      const lot = state.lots.find((row) => row.id === lotID);
      if (!lot) {
        await route.fulfill({
          status: 404,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'lot not found' })
        });
        return;
      }

      lot.quantity = payload.quantity;
      lot.unitCost = payload.unit_cost;
      lot.purchasedAt = payload.purchased_at;

      await route.fulfill({ status: 204, body: '' });
      return;
    }

    if (method === 'DELETE' && path.includes('/api/v1/lots/')) {
      const lotID = Number(path.split('/').at(-1));
      const before = state.lots.length;
      state.lots = state.lots.filter((row) => row.id !== lotID);
      if (state.lots.length === before) {
        await route.fulfill({
          status: 404,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'lot not found' })
        });
        return;
      }

      await route.fulfill({ status: 204, body: '' });
      return;
    }

    await route.fulfill({
      status: 404,
      contentType: 'application/json',
      body: JSON.stringify({ error: `unmocked endpoint ${method} ${path}` })
    });
  });

  return {
    setPrice(assetID: number, price: number) {
      state.pricesByAssetId.set(assetID, price);
    }
  };
}

function metricValue(page: Page, metricHeading: string) {
  return page
    .locator('article.metric-card')
    .filter({ has: page.getByRole('heading', { name: metricHeading, exact: true }) })
    .locator('.metric-value');
}

test('smoke: sign in, lot CRUD, and portfolio refresh rendering', async ({ page }) => {
  const mocks = await setupMockRoutes(page);
  page.on('dialog', (dialog) => dialog.accept());

  await page.goto('/');

  await page.getByLabel('Email').fill('smoke@example.com');
  await page.getByLabel('Password').fill('password123');
  await page.getByRole('button', { name: 'Sign In', exact: true }).click();

  await expect(page.getByRole('button', { name: 'Sign Out' })).toBeVisible();
  await expect(page.getByText('smoke@example.com')).toBeVisible();

  const lotsPanel = page
    .locator('section.panel')
    .filter({ has: page.getByRole('heading', { name: 'Lots', exact: true }) });

  await expect(lotsPanel.locator('tbody tr')).toHaveCount(1);
  await expect(metricValue(page, 'Portfolio Value')).toHaveText('$150.00');
  await expect(metricValue(page, 'Lots')).toHaveText('1');

  await page.getByLabel('Search Assets').fill('eth');
  await expect(page.locator('select option[value="2"]')).toHaveCount(1);
  await page.getByRole('combobox', { name: 'Asset' }).selectOption('2');
  await page.getByLabel('Quantity').fill('2');
  await page.getByLabel('Unit Cost (USD)').fill('2500');
  await page.getByRole('button', { name: 'Add Lot', exact: true }).click();

  await expect(lotsPanel.locator('tbody tr')).toHaveCount(2);
  await expect(lotsPanel.locator('tbody tr').filter({ hasText: 'ETH' })).toHaveCount(1);
  await expect(metricValue(page, 'Lots')).toHaveText('2');

  const ethRow = lotsPanel.locator('tbody tr').filter({ hasText: 'ETH' });
  await ethRow.getByRole('button', { name: 'Edit' }).click();
  await page.getByLabel('Quantity').fill('3');
  await page.getByRole('button', { name: 'Save Changes', exact: true }).click();
  await expect(lotsPanel.locator('tbody tr').filter({ hasText: 'ETH' })).toContainText('3');

  await ethRow.getByRole('button', { name: 'Delete' }).click();
  await expect(lotsPanel.locator('tbody tr')).toHaveCount(1);
  await expect(lotsPanel.locator('tbody tr').filter({ hasText: 'ETH' })).toHaveCount(0);
  await expect(metricValue(page, 'Lots')).toHaveText('1');

  mocks.setPrice(1, 180);
  await page.getByRole('button', { name: 'Refresh', exact: true }).click();
  await expect(metricValue(page, 'Portfolio Value')).toHaveText('$180.00');

  await page.getByRole('button', { name: 'Sign Out', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Sign In', exact: true })).toBeVisible();
});
