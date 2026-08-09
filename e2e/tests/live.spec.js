import { test, expect } from '@playwright/test';

/**
 * The claims the Go tests cannot reach.
 *
 * `go test` covers the wire format, the session lifecycle and the encoding. It
 * cannot cover what actually justifies the live layer: that a server-driven
 * update writes one property and therefore leaves the DOM the user is touching
 * alone. That is real-DOM behaviour — custom element upgrade, alacris' `each`
 * reordering by key, focus and selection — and a simulated DOM would happily
 * pass while the real one regressed.
 *
 * These run against `examples/todo`, unmodified. If they ever need a fixture
 * instead, the thing being tested has stopped being the thing people copy.
 */

const LIST = '#todos';
const NEW_TODO = 'input[aria-label="New todo"]';

/** Wait for the custom element to upgrade and render its rows. */
async function ready(page) {
  await page.waitForFunction(() => {
    const el = document.getElementById('todos');
    return !!el?.shadowRoot?.querySelector('ul');
  });
}

/** The visible row text, read through the shadow root. */
function rows(page) {
  return page.evaluate(() =>
    [...document.getElementById('todos').shadowRoot.querySelectorAll('li .text')].map(
      (n) => n.textContent,
    ),
  );
}

/**
 * Stamp every current row so it can be recognised afterwards.
 *
 * An expando survives a node being moved and does not survive it being
 * rebuilt, which is precisely the distinction under test.
 */
function markRows(page) {
  return page.evaluate(() => {
    const sr = document.getElementById('todos').shadowRoot;
    [...sr.querySelectorAll('li')].forEach((li, i) => {
      li.dataset.e2eMark = `row-${i}`;
    });
    return sr.querySelectorAll('li').length;
  });
}

function readMarks(page) {
  return page.evaluate(() =>
    [...document.getElementById('todos').shadowRoot.querySelectorAll('li')].map(
      (li) => li.dataset.e2eMark ?? null,
    ),
  );
}

/** Add a todo through the component, the way a user would. */
async function addTodo(page, text) {
  const before = (await rows(page)).length;
  await page.locator(`${LIST} ${NEW_TODO}`).fill(text);
  await page.locator(`${LIST} ${NEW_TODO}`).press('Enter');
  await expect
    .poll(async () => (await rows(page)).length, { timeout: 5000 })
    .toBe(before + 1);
}

test.beforeEach(async ({ page }) => {
  await page.goto('/');
  await ready(page);
});

test('a server-driven update moves rows instead of rebuilding them', async ({ page }) => {
  // This is the one that catches an `each` placed inside a conditional
  // template, which rebuilds every row on every change and quietly undoes the
  // whole point of the update path.
  await addTodo(page, 'identity check');

  const marked = await markRows(page);
  expect(marked).toBeGreaterThan(1);

  // Toggle a row through the full round trip: click -> event -> POST ->
  // handler -> items prop -> SSE frame -> DOM.
  await page.locator(`${LIST} li input[type=checkbox]`).first().check();

  await expect
    .poll(async () => (await page.evaluate(() => document.getElementById('todos').items))[0].done, {
      timeout: 5000,
    })
    .toBe(true);

  const marks = await readMarks(page);
  expect(marks).toHaveLength(marked);
  expect(
    marks,
    'the rows were rebuilt: look for each() inside a conditional template',
  ).not.toContain(null);
});

test('an update leaves focus and a half-typed draft alone', async ({ page }) => {
  const draft = 'half typed';

  const input = page.locator(`${LIST} ${NEW_TODO}`);
  await input.fill(draft);
  // Put the caret in the middle: an update that replaced the node would lose
  // the position even if it somehow preserved the value.
  await page.evaluate(() => {
    const el = document.getElementById('todos').shadowRoot.querySelector('form input');
    el.focus();
    el.setSelectionRange(4, 4);
  });

  // Change the list underneath the user.
  await page.locator(`${LIST} li input[type=checkbox]`).first().check();
  await page.waitForTimeout(500);

  const state = await page.evaluate(() => {
    const sr = document.getElementById('todos').shadowRoot;
    const el = sr.querySelector('form input');
    return {
      focused: sr.activeElement === el,
      value: el.value,
      caret: el.selectionStart,
    };
  });

  expect(state.focused).toBe(true);
  expect(state.value).toBe(draft);
  expect(state.caret).toBe(4);
});

test('every open page stays in step without polling', async ({ context, page }) => {
  const second = await context.newPage();
  await second.goto('/');
  await ready(second);

  const before = await rows(page);
  const text = `from the second tab ${Date.now()}`;
  await addTodo(second, text);

  // The first page was never touched.
  await expect.poll(async () => (await rows(page)).at(-1), { timeout: 5000 }).toBe(text);
  expect((await rows(page)).length).toBe(before.length + 1);

  await second.close();
});

test('server-rendered slot content is replaced, not appended', async ({ page }) => {
  // SetHTML replaces the children assigned to one slot. Appending instead is a
  // failure that reads as "3 left4 left" — which is how the encoding bug that
  // caused it was found.
  // textContent, not innerText: the chip sits inside an uppercased heading, and
  // innerText would return what CSS renders rather than what the server sent.
  const remaining = () => page.locator('#remaining').textContent();

  const before = await remaining();
  expect(before).toMatch(/^\d+ left$/);

  await addTodo(page, 'slot check');

  await expect.poll(remaining, { timeout: 5000 }).not.toBe(before);
  expect(await remaining()).toMatch(/^\d+ left$/);
});

test('the page works before the module arrives', async ({ browser }) => {
  // Slot content is light DOM, so it is in the HTML the server sent. With
  // JavaScript off, the component's internals never appear — but what the
  // server rendered still does, and that is the part that matters for a
  // crawler or a reader who never gets the module.
  const context = await browser.newContext({ javaScriptEnabled: false });
  const page = await context.newPage();
  const response = await page.goto('/');

  expect(response.status()).toBe(200);

  const html = await page.content();
  expect(html).toContain('<ala-todo-list');
  expect(html).toContain('slot="empty"');
  // Props are in the markup, not in a hydration payload.
  expect(html).toMatch(/items="\[\{/);

  await context.close();
});
