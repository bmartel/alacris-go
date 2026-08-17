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

const LIST = '#board';

/** Wait for the custom element to upgrade and render its cards. */
async function ready(page) {
  await page.waitForFunction(() => {
    const el = document.getElementById('board');
    return !!el?.shadowRoot?.querySelector('ui-card');
  });
}

function todoLane(page) {
  return page.locator(LIST).locator('[data-column="todo"]');
}

/** The visible card titles, read from the board. */
function rows(page) {
  return page.evaluate(() =>
    [...document.getElementById('board').shadowRoot.querySelectorAll('ui-card')].map(
      (n) => n.getAttribute('data-text') || n.querySelector('.title')?.textContent?.trim() || '',
    ),
  );
}

/**
 * Stamp every current card so it can be recognised afterwards.
 *
 * An expando survives a node being moved and does not survive it being
 * rebuilt, which is precisely the distinction under test.
 */
function markRows(page) {
  return page.evaluate(() => {
    const sr = document.getElementById('board').shadowRoot;
    [...sr.querySelectorAll('ui-card')].forEach((li, i) => {
      li.dataset.e2eMark = `row-${i}`;
    });
    return sr.querySelectorAll('ui-card').length;
  });
}

function readMarks(page) {
  return page.evaluate(() =>
    [...document.getElementById('board').shadowRoot.querySelectorAll('ui-card')].map(
      (li) => li.dataset.e2eMark ?? null,
    ),
  );
}

/** Add a card through the component, the way a user would. */
async function addCard(page, text) {
  const before = (await rows(page)).length;
  const todo = todoLane(page);
  await todo.getByRole('button', { name: 'Add a card' }).click();
  const input = todo.getByRole('textbox', { name: 'Add a card' });
  await input.fill(text);
  await input.press('Enter');
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
  await addCard(page, 'identity check');

  const marked = await markRows(page);
  expect(marked).toBeGreaterThan(1);

  await page.evaluate(() => {
    const board = document.getElementById('board');
    const todo = board.items.filter((it) => it.column === 'todo');
    const last = todo[todo.length - 1];
    board.dispatchEvent(
      new CustomEvent('move', {
        bubbles: true,
        composed: true,
        detail: { id: last.id, column: 'todo', index: 0 },
      }),
    );
  });

  await expect
    .poll(async () => {
      const todo = (await page.evaluate(() => document.getElementById('board').items)).filter(
        (it) => it.column === 'todo',
      );
      return todo[0]?.text;
    }, { timeout: 5000 })
    .toBe('identity check');

  const marks = await readMarks(page);
  expect(marks).toHaveLength(marked);
  expect(
    marks,
    'the rows were rebuilt: look for each() inside a conditional template',
  ).not.toContain(null);
});

test('an update leaves focus and a half-typed draft alone', async ({ page }) => {
  const draft = 'half typed';

  const todo = todoLane(page);
  await todo.getByRole('button', { name: 'Add a card' }).click();
  const input = todo.getByRole('textbox', { name: 'Add a card' });
  await input.fill(draft);
  // Put the caret in the middle: an update that replaced the node would lose
  // the position even if it somehow preserved the value.
  await page.evaluate(() => {
    const field = document
      .getElementById('board')
      .shadowRoot.querySelector('[data-column="todo"] ui-text-field');
    const el = field.shadowRoot.querySelector('input');
    el.focus();
    el.setSelectionRange(4, 4);
  });

  // Reorder a card in another lane without moving focus into it.
  await page.evaluate(() => {
    const board = document.getElementById('board');
    const it = board.items.find((c) => c.column === 'doing') || board.items[0];
    board.dispatchEvent(
      new CustomEvent('move', {
        bubbles: true,
        composed: true,
        detail: { id: it.id, column: 'done', index: 0 },
      }),
    );
  });
  await page.waitForTimeout(500);

  const state = await page.evaluate(() => {
    const field = document
      .getElementById('board')
      .shadowRoot.querySelector('[data-column="todo"] ui-text-field');
    const el = field.shadowRoot.querySelector('input');
    return {
      focused: field.shadowRoot.activeElement === el,
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
  await addCard(second, text);

  // The first page was never touched.
  await expect.poll(async () => (await rows(page)).includes(text), { timeout: 5000 }).toBe(true);
  expect((await rows(page)).length).toBe(before.length + 1);

  await second.close();
});

test('server-rendered slot content is replaced, not appended', async ({ page }) => {
  // SetHTML replaces the children assigned to one slot. Appending instead is a
  // failure that duplicates the stamp — which is how the encoding bug that
  // caused it was found.
  const stamps = () => page.locator('#members [data-cards]');
  expect(await stamps().count()).toBe(1);
  const before = await stamps().getAttribute('data-cards');

  await addCard(page, 'slot check');

  await expect.poll(() => stamps().getAttribute('data-cards'), { timeout: 5000 }).not.toBe(before);
  expect(await stamps().count()).toBe(1);
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
  expect(html).toContain('<ala-board');
  expect(html).toContain('slot="empty"');
  // Props are in the markup, not in a hydration payload.
  expect(html).toMatch(/items="\[\{/);

  await context.close();
});

test('the session capability is a cookie, and never a URL', async ({ page, context }) => {
  // The whole point of the cookie: a page id in an access log or a referer
  // reaches nothing without it.
  const requests = [];
  page.on('request', (r) => requests.push(r.url()));
  await page.goto('/');
  await ready(page);

  const cookies = await context.cookies();
  const live = cookies.find((c) => c.name === 'alacris_live');

  expect(live, 'the live cookie was not set').toBeTruthy();
  expect(live.httpOnly, 'script must not be able to read the capability').toBe(true);
  expect(live.sameSite, 'SameSite is what stops a cross-site POST carrying it').toBe('Lax');
  expect(live.path).toBe('/_alacris/');

  // The token must not appear anywhere a log or a referer would pick it up.
  const token = live.value;
  expect(token.length).toBeGreaterThan(20);
  for (const url of requests) {
    expect(url, `the capability leaked into a URL: ${url}`).not.toContain(token);
  }
  expect(await page.content()).not.toContain(token);

  // Script cannot reach it either.
  const visible = await page.evaluate(() => document.cookie);
  expect(visible).not.toContain(token);

  // What is in the URL is the page id, which is useless on its own.
  const streamURL = requests.find((u) => u.includes('/_alacris/live?'));
  expect(streamURL, 'the stream never connected').toBeTruthy();
  expect(streamURL).toContain('p=');
  expect(streamURL).not.toContain(token);
});

test('a page id without the cookie is refused', async ({ page, browser }) => {
  await page.goto('/');
  await ready(page);

  const streamURL = await page.evaluate(() => {
    const s = document.querySelector('script[data-page]');
    return new URL(s.dataset.endpoint + '?p=' + s.dataset.page, location.href).href;
  });

  // A fresh context is a different browser: it has the page id, from a log
  // say, and no cookie.
  const clean = await browser.newContext();
  const response = await clean.request.get(streamURL);
  expect(response.status()).toBe(404);
  await clean.close();
});

test('a card can be reordered in its lane', async ({ page }) => {
  const before = await page.evaluate(() =>
    document
      .getElementById('board')
      .items.filter((it) => it.column === 'todo')
      .map((it) => it.text),
  );
  expect(before.length).toBeGreaterThan(1);

  const from = await page.evaluate(() => {
    const cards = [
      ...document
        .getElementById('board')
        .shadowRoot.querySelectorAll('[data-column="todo"] .card'),
    ];
    const r = cards[1].getBoundingClientRect();
    return { x: r.x + 40, y: r.y + 16 };
  });
  const to = await page.evaluate(() => {
    const r = document
      .getElementById('board')
      .shadowRoot.querySelector('[data-column="todo"] .card')
      .getBoundingClientRect();
    return { x: r.x + 40, y: r.y + 8 };
  });

  await page.mouse.move(from.x, from.y);
  await page.mouse.down();
  await page.mouse.move(to.x, to.y, { steps: 12 });
  await page.mouse.up();

  await expect
    .poll(
      async () => {
        const todo = (await page.evaluate(() => document.getElementById('board').items)).filter(
          (it) => it.column === 'todo',
        );
        return todo[0]?.text;
      },
      { timeout: 5000 },
    )
    .toBe(before[1]);
});

test('a card can be dragged into another lane', async ({ page }) => {
  const title = await page.evaluate(
    () =>
      document
        .getElementById('board')
        .shadowRoot.querySelector('[data-column="todo"] .card').dataset.text,
  );

  const from = await page.evaluate(() => {
    const r = document
      .getElementById('board')
      .shadowRoot.querySelector('[data-column="todo"] .card')
      .getBoundingClientRect();
    return { x: r.x + 40, y: r.y + 16 };
  });
  const to = await page.evaluate(() => {
    const r = document
      .getElementById('board')
      .shadowRoot.querySelector('[data-column="doing"]')
      .getBoundingClientRect();
    return { x: r.x + r.width / 2, y: r.y + 120 };
  });

  await page.mouse.move(from.x, from.y);
  await page.mouse.down();
  await page.mouse.move(to.x, to.y, { steps: 12 });
  await page.mouse.up();

  await expect
    .poll(
      async () => {
        const items = await page.evaluate(() => document.getElementById('board').items);
        return items.find((it) => it.text === title)?.column;
      },
      { timeout: 5000 },
    )
    .toBe('doing');
});

test('a list title can be renamed in place', async ({ page }) => {
  const n = await markRows(page);
  const doing = page.locator(LIST).locator('[data-column="doing"]');
  await doing.getByRole('button', { name: 'Doing' }).click();
  const input = doing.getByRole('textbox', { name: 'List title' });
  await input.fill('In progress');
  await input.press('Enter');
  await expect
    .poll(
      async () =>
        (await page.evaluate(() => document.getElementById('board').columns)).find(
          (c) => c.id === 'doing',
        )?.title,
      { timeout: 5000 },
    )
    .toBe('In progress');
  expect(await readMarks(page)).toEqual(Array.from({ length: n }, (_, i) => `row-${i}`));
});

test('a card opens an editor dialog', async ({ page }) => {
  const n = await markRows(page);
  const card = await page.evaluate(() => {
    const it = document.getElementById('board').items.find((c) => c.column === 'doing');
    return { id: it.id, labels: (it.labels || []).slice().sort().join(',') };
  });
  await page.locator(LIST).locator('[data-column="doing"] .card').first().click();
  await expect(page.getByRole('dialog', { name: 'Card' })).toBeVisible();
  await page.getByRole('option', { name: 'Blocked' }).click();
  await expect
    .poll(
      async () => {
        const it = (await page.evaluate(() => document.getElementById('board').items)).find(
          (c) => c.id === card.id,
        );
        return (it?.labels || []).slice().sort().join(',');
      },
      { timeout: 5000 },
    )
    .not.toBe(card.labels);
  await page.getByRole('textbox', { name: 'New label' }).fill('design');
  await page.getByRole('button', { name: 'Add label' }).click();
  await expect
    .poll(
      async () =>
        (await page.evaluate(() => document.getElementById('board').items)).find((c) => c.id === card.id)
          ?.labels || [],
      { timeout: 5000 },
    )
    .toContain('design');
  const addWho = await page.evaluate((id) => {
    const it = document.getElementById('board').items.find((c) => c.id === id);
    const have = new Set(it?.who || []);
    return ['You', 'Ada Lovelace', 'Ben Linus', 'Cara Moss'].find((n) => !have.has(n));
  }, card.id);
  const members = page.getByRole('combobox', { name: 'Members' });
  await members.click();
  await members.fill(addWho.split(/\s/)[0]);
  await page.getByRole('listbox', { name: 'Members' }).getByRole('option', { name: addWho }).click();
  await expect
    .poll(
      async () =>
        (await page.evaluate(() => document.getElementById('board').items)).find((c) => c.id === card.id)
          ?.who || [],
      { timeout: 5000 },
    )
    .toContain(addWho);
  await page.getByRole('combobox', { name: 'List' }).click();
  await page.getByRole('option', { name: 'Done' }).click();
  await expect(page.getByRole('dialog', { name: 'Card' })).toBeVisible();
  await expect
    .poll(
      async () =>
        (await page.evaluate(() => document.getElementById('board').items)).find((c) => c.id === card.id)
          ?.column,
      { timeout: 5000 },
    )
    .toBe('done');
  await page.getByRole('combobox', { name: 'List' }).click();
  await page.getByRole('option', { name: 'In progress' }).click();
  await expect
    .poll(
      async () =>
        (await page.evaluate(() => document.getElementById('board').items)).find((c) => c.id === card.id)
          ?.column,
      { timeout: 5000 },
    )
    .toBe('doing');
  await page.getByRole('textbox', { name: 'Title' }).fill('Cut the launch film');
  await page.getByRole('button', { name: 'Close' }).click();
  await expect
    .poll(
      async () =>
        (await page.evaluate(() => document.getElementById('board').items)).find((it) => it.id === card.id)
          ?.text,
      { timeout: 5000 },
    )
    .toBe('Cut the launch film');
  // Leaving a lane remounts that card (each lane has its own each()). The
  // others must keep their nodes.
  expect((await readMarks(page)).filter(Boolean)).toHaveLength(n - 1);
});

test('a list can be dragged to a new position', async ({ page }) => {
  const from = await page.evaluate(() => {
    const r = document
      .getElementById('board')
      .shadowRoot.querySelector('[data-column="doing"] .head')
      .getBoundingClientRect();
    return { x: r.x + 24, y: r.y + 10 };
  });
  const to = await page.evaluate(() => {
    const r = document
      .getElementById('board')
      .shadowRoot.querySelector('[data-column="todo"] .head')
      .getBoundingClientRect();
    return { x: r.x + 8, y: r.y + 10 };
  });

  await page.mouse.move(from.x, from.y);
  await page.mouse.down();
  await page.mouse.move(to.x, to.y, { steps: 12 });
  await page.mouse.up();

  await expect
    .poll(
      async () =>
        (await page.evaluate(() => document.getElementById('board').columns)).map((c) => c.id),
      { timeout: 5000 },
    )
    .toEqual(['doing', 'todo', 'done']);
});

test('a list can be deleted after confirming', async ({ page }) => {
  const doing = page.locator(LIST).locator('[data-column="doing"]');
  await doing.getByRole('button', { name: 'List actions' }).click();
  await page.getByRole('menuitem', { name: 'Delete list' }).click();
  await expect(page.getByRole('dialog', { name: 'Delete list?' })).toBeVisible();
  await page.getByRole('button', { name: 'Delete', exact: true }).click();
  await expect
    .poll(
      async () =>
        (await page.evaluate(() => document.getElementById('board').columns)).map((c) => c.id),
      { timeout: 5000 },
    )
    .not.toContain('doing');
  const items = await page.evaluate(() => document.getElementById('board').items);
  expect(items.every((it) => it.column !== 'doing')).toBe(true);
});

test('Alacris UI is on the page', async ({ page }) => {
  await expect
    .poll(() => page.evaluate(() => !!customElements.get('ui-button')))
    .toBe(true);
  await expect(page.locator('#board')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Share' })).toBeVisible();
});
