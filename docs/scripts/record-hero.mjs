#!/usr/bin/env node
/**
 * Record a looping clip of the example board for the docs hero.
 *
 * Story: add a card, mark it blocked, drag it into Doing, add and
 * reorder a Review list, filter to blocked, then drop that card in Review.
 *
 *   node docs/scripts/record-hero.mjs
 *
 * Starts the example app on 127.0.0.1:8099 unless ALACRIS_HERO_PORT is set.
 * Writes docs/public/board-hero.webm and docs/public/board-hero.jpg.
 */
import { chromium } from '../../e2e/node_modules/playwright/index.mjs';
import { spawn } from 'node:child_process';
import { mkdir, copyFile, rm, readdir } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, '../..');
const OUT_DIR = path.join(ROOT, 'docs/public');
const TMP = path.join(ROOT, 'docs/.hero-record');
const PORT = process.env.ALACRIS_HERO_PORT ?? '8099';
const BASE = `http://127.0.0.1:${PORT}`;
const BOARD = '#board';

const pause = (page, ms) => page.waitForTimeout(ms);

async function waitForURL(url, ms = 180_000) {
  const start = Date.now();
  while (Date.now() - start < ms) {
    try {
      const r = await fetch(url);
      if (r.ok) return;
    } catch {
      /* not up yet */
    }
    await new Promise((r) => setTimeout(r, 250));
  }
  throw new Error(`server at ${url} did not start`);
}

function startServer() {
  const child = spawn('go', ['run', './examples/todo', '-addr', `127.0.0.1:${PORT}`], {
    cwd: ROOT,
    stdio: ['ignore', 'pipe', 'pipe'],
    detached: true,
  });
  child.stdout.on('data', (d) => process.stdout.write(d));
  child.stderr.on('data', (d) => process.stderr.write(d));
  child.unref();
  return child;
}

function stopServer(child) {
  if (!child?.pid) return;
  try {
    process.kill(-child.pid, 'SIGTERM');
  } catch {
    try {
      process.kill(child.pid, 'SIGTERM');
    } catch {
      /* already gone */
    }
  }
}

async function ready(page) {
  await page.waitForFunction(() => {
    const el = document.getElementById('board');
    return !!el?.shadowRoot?.querySelector('ui-card');
  });
  await pause(page, 500);
}

async function drag(page, from, to, steps = 28) {
  await page.mouse.move(from.x, from.y);
  await page.mouse.down();
  await page.mouse.move(to.x, to.y, { steps });
  await page.mouse.up();
  await pause(page, 700);
}

async function cardPoint(page, column, text) {
  return page.evaluate(
    ({ column, text }) => {
      const cards = [
        ...document
          .getElementById('board')
          .shadowRoot.querySelectorAll(`[data-column="${column}"] .card`),
      ];
      const el =
        cards.find(
          (c) =>
            !c.classList.contains('filtered') &&
            (!text || (c.dataset.text || '') === text),
        ) || cards.find((c) => !c.classList.contains('filtered'));
      const r = el.getBoundingClientRect();
      return { x: r.x + Math.min(40, r.width / 2), y: r.y + 16 };
    },
    { column, text },
  );
}

async function laneHead(page, column, edge = 'mid') {
  return page.evaluate(
    ({ column, edge }) => {
      const r = document
        .getElementById('board')
        .shadowRoot.querySelector(`[data-column="${column}"] .head`)
        .getBoundingClientRect();
      const x = edge === 'start' ? r.x + 8 : r.x + 24;
      return { x, y: r.y + 10 };
    },
    { column, edge },
  );
}

async function laneBody(page, column) {
  return page.evaluate((column) => {
    const r = document
      .getElementById('board')
      .shadowRoot.querySelector(`[data-column="${column}"]`)
      .getBoundingClientRect();
    return { x: r.x + r.width / 2, y: r.y + 150 };
  }, column);
}

/** A drag sets suppressOpen, so the next card click is swallowed. */
async function openCard(page, card) {
  const dialog = page.getByRole('dialog', { name: 'Card' });
  await card.click();
  try {
    await dialog.waitFor({ state: 'visible', timeout: 1200 });
  } catch {
    await card.click();
    await dialog.waitFor({ state: 'visible', timeout: 8000 });
  }
}

let server;
try {
  server = startServer();
  await waitForURL(BASE);

  const browser = await chromium.launch({
    headless: true,
    args: ['--autoplay-policy=no-user-gesture-required'],
  });
  await rm(TMP, { recursive: true, force: true });
  await mkdir(TMP, { recursive: true });

  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 1,
    colorScheme: 'dark',
    recordVideo: { dir: TMP, size: { width: 1440, height: 900 } },
  });
  const page = await context.newPage();
  await page.goto(BASE, { waitUntil: 'networkidle' });
  await ready(page);
  await pause(page, 800);

  const board = page.locator(BOARD);
  const todo = board.locator('[data-column="todo"]');

  // Add a card.
  await todo.getByRole('button', { name: 'Add a card' }).click();
  await pause(page, 280);
  const addField = todo.getByRole('textbox', { name: 'Add a card' });
  await addField.pressSequentially('Cut the trailer', { delay: 55 });
  await pause(page, 200);
  await addField.press('Enter');
  await page.waitForFunction(() =>
    document.getElementById('board').items.some((it) => it.text === 'Cut the trailer'),
  );
  await pause(page, 700);

  // Edit it (before any drag, so the click is not swallowed).
  const added = board.locator('[data-column="todo"] .card[data-text="Cut the trailer"]');
  await openCard(page, added);
  await pause(page, 500);
  const dialog = page.getByRole('dialog', { name: 'Card' });
  await page.getByRole('option', { name: 'Blocked' }).click();
  await page.waitForFunction(() =>
    (document.getElementById('board').items.find((it) => it.text === 'Cut the trailer')?.labels || []).includes(
      'blocked',
    ),
  );
  await pause(page, 400);
  await page.getByRole('textbox', { name: 'Description' }).click();
  await page
    .getByRole('textbox', { name: 'Description' })
    .pressSequentially('Need the Berlin cut.', { delay: 40 });
  await pause(page, 500);
  await page.getByRole('button', { name: 'Close' }).click();
  await dialog.waitFor({ state: 'hidden', timeout: 8000 });
  await pause(page, 600);

  // Drag that card into Doing.
  const from = await cardPoint(page, 'todo', 'Cut the trailer');
  const doing = await laneBody(page, 'doing');
  await drag(page, from, doing);
  await page.waitForFunction(() => {
    const it = document.getElementById('board').items.find((c) => c.text === 'Cut the trailer');
    return it?.column === 'doing';
  });
  await pause(page, 700);

  // Add a list.
  await page.getByRole('button', { name: 'Add another list' }).click();
  await pause(page, 280);
  const listField = page.getByRole('textbox', { name: 'List title' });
  await listField.pressSequentially('Review', { delay: 55 });
  await pause(page, 180);
  await page.getByRole('button', { name: 'Add list' }).click();
  await page.waitForFunction(() =>
    document.getElementById('board').columns.some((c) => c.title === 'Review'),
  );
  await pause(page, 800);

  const reviewId = await page.evaluate(
    () => document.getElementById('board').columns.find((c) => c.title === 'Review').id,
  );

  // Reorder: slide the new list left, next to To do.
  const reviewHead = await laneHead(page, reviewId);
  const todoHead = await laneHead(page, 'todo', 'start');
  await drag(page, reviewHead, todoHead, 36);
  await page.waitForFunction((id) => {
    const ids = document.getElementById('board').columns.map((c) => c.id);
    return ids[0] === id || ids[1] === id;
  }, reviewId);
  await pause(page, 800);

  // Filter to blocked work, then drop that card in Review.
  const search = page.getByRole('textbox', { name: 'Filter cards' });
  await search.click();
  await search.pressSequentially('blocked', { delay: 60 });
  await pause(page, 900);
  const blockedCard = await cardPoint(page, 'doing', 'Cut the trailer');
  const reviewLane = await laneBody(page, reviewId);
  await drag(page, blockedCard, reviewLane);
  await page.waitForFunction((id) => {
    const it = document.getElementById('board').items.find((c) => c.text === 'Cut the trailer');
    return it?.column === id;
  }, reviewId);
  await pause(page, 800);
  await page.getByRole('button', { name: 'Clear' }).click();
  await pause(page, 1400);

  await page.screenshot({ path: path.join(OUT_DIR, 'board-hero.jpg'), type: 'jpeg', quality: 72 });

  await context.close();
  await browser.close();

  const files = (await readdir(TMP)).filter((f) => f.endsWith('.webm'));
  if (!files.length) throw new Error('playwright wrote no webm');
  await mkdir(OUT_DIR, { recursive: true });
  await copyFile(path.join(TMP, files[0]), path.join(OUT_DIR, 'board-hero.webm'));
  await rm(TMP, { recursive: true, force: true });
  console.log('wrote docs/public/board-hero.webm and docs/public/board-hero.jpg');
} finally {
  stopServer(server);
}
