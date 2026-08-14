import { test, expect } from '@playwright/test';

/**
 * Recovery from a dead session.
 *
 * Sessions live in the server's memory, so a restart (or an expiry) leaves an
 * open page holding a page id the server has never heard of. The EventSource
 * spec makes a non-200 response a permanent failure — the browser will never
 * retry it — so before recovery existed, such a page silently froze.
 *
 * The contract under test: when the server says this page's session is gone
 * (404 "unknown session"), the client probes, sees the server alive, reloads
 * the page once for a fresh session, and the fresh page is live again.
 *
 * The dead session is simulated at the network layer: every request that
 * names the old page id — stream, probe, or action — answers 404 exactly as
 * the real server would after a restart. The fresh session after reload uses
 * a different page id, so it passes through untouched. No fixture in the
 * server; the app under test is examples/todo, unmodified.
 */

async function ready(page) {
  await page.waitForFunction(() => {
    const el = document.getElementById('todos');
    return !!el?.shadowRoot?.querySelector('ul');
  });
}

test('a dead session recovers by reloading into a fresh one', async ({ page }) => {
  await page.goto('/');
  await ready(page);

  const firstPageID = await page.evaluate(
    () => document.querySelector('script[data-page]').dataset.page,
  );
  expect(firstPageID).toBeTruthy();

  // From now on the old session does not exist, whichever way it is asked for.
  await page.route('**/_alacris/live*', (route) => {
    const req = route.request();
    const dead =
      req.url().includes(`p=${firstPageID}`) ||
      (req.method() === 'POST' && (req.postData() ?? '').includes(firstPageID));
    if (dead) {
      return route.fulfill({ status: 404, contentType: 'text/plain', body: 'unknown session\n' });
    }
    return route.continue();
  });

  // Drive the module's own send(), the way a user's click would after a
  // restart. import() of the same URL returns the already-running module
  // instance, so this is the real recovery wiring, not a reimplementation.
  await page.evaluate(async () => {
    const script = document.querySelector('script[data-page][data-endpoint]');
    const mod = await import(script.src);
    await mod.send('nudge', '', null);
  });

  // Recovery probes, sees the server up and the session gone, and reloads.
  await page.waitForFunction(
    (old) => document.querySelector('script[data-page]')?.dataset.page !== old,
    firstPageID,
    { timeout: 15_000 },
  );
  await ready(page);

  const secondPageID = await page.evaluate(
    () => document.querySelector('script[data-page]').dataset.page,
  );
  expect(secondPageID).not.toBe(firstPageID);

  // The fresh page is genuinely live: a change made through the component
  // comes back as a server-driven update.
  const before = await page.evaluate(
    () => document.getElementById('todos').shadowRoot.querySelectorAll('li').length,
  );
  const input = page.locator('#todos input[aria-label="New todo"]');
  await input.fill('recovered and live');
  await input.press('Enter');
  await expect
    .poll(
      () =>
        page.evaluate(
          () => document.getElementById('todos').shadowRoot.querySelectorAll('li').length,
        ),
      { timeout: 5000 },
    )
    .toBe(before + 1);
});
