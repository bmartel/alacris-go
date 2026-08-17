#!/usr/bin/env node
/**
 * Record a looping clip of the example board for the docs hero.
 *
 * Story: add a card, mark it blocked, drag it into Doing, add and
 * reorder a Review list, then drop that card in Review.
 *
 * Motion is eased and the pointer is drawn on the page so the clip
 * reads as a product take, not a test run. Encode uses Playwright's
 * ffmpeg (the Homebrew build is broken on this machine) at a higher
 * bitrate than Playwright's default realtime 1 Mbps screencast.
 *
 *   node docs/scripts/record-hero.mjs
 *
 * Starts the example app on 127.0.0.1:8099 unless ALACRIS_HERO_PORT is set.
 * Writes docs/public/board-hero.webm and docs/public/board-hero.jpg.
 */
import { chromium } from '../../e2e/node_modules/playwright/index.mjs';
import { spawn } from 'node:child_process';
import { access, copyFile, mkdir, readdir, rm } from 'node:fs/promises';
import { homedir } from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, '../..');
const OUT_DIR = path.join(ROOT, 'docs/public');
const TMP = path.join(ROOT, 'docs/.hero-record');
const PORT = process.env.ALACRIS_HERO_PORT ?? '8099';
const BASE = `http://127.0.0.1:${PORT}`;
const WIDTH = 1920;
const HEIGHT = 1080;
const FPS = 30;

const pause = (page, ms) => page.waitForTimeout(ms);

const easeInOut = (t) =>
  t < 0.5 ? 4 * t * t * t : 1 - (-2 * t + 2) ** 3 / 2;

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

async function playwrightFfmpeg() {
  const bin =
    process.platform === 'darwin'
      ? 'ffmpeg-mac'
      : process.platform === 'win32'
        ? 'ffmpeg-win64.exe'
        : 'ffmpeg-linux';
  const roots = [
    process.env.PLAYWRIGHT_BROWSERS_PATH,
    path.join(homedir(), 'Library/Caches/ms-playwright'),
    path.join(homedir(), '.cache/ms-playwright'),
  ].filter(Boolean);
  for (const root of roots) {
    let entries = [];
    try {
      entries = await readdir(root);
    } catch {
      continue;
    }
    for (const dir of entries.filter((n) => n.startsWith('ffmpeg-')).sort().reverse()) {
      const p = path.join(root, dir, bin);
      try {
        await access(p);
        return p;
      } catch {
        /* try next */
      }
    }
  }
  throw new Error('Playwright ffmpeg not found; run the e2e install once');
}

function createRecorder(page, outFile, ffmpegBin) {
  let current = null;
  let running = false;
  let session;
  let ff;
  let pump;

  const start = async () => {
    session = await page.context().newCDPSession(page);
    ff = spawn(
      ffmpegBin,
      [
        '-y',
        '-f',
        'image2pipe',
        '-vcodec',
        'mjpeg',
        '-r',
        String(FPS),
        '-i',
        'pipe:0',
        '-an',
        '-c:v',
        'vp8',
        '-qmin',
        '0',
        '-qmax',
        '32',
        '-crf',
        '8',
        '-b:v',
        '2M',
        '-deadline',
        'good',
        '-speed',
        '2',
        '-threads',
        '1',
        outFile,
      ],
      { stdio: ['pipe', 'inherit', 'inherit'] },
    );
    session.on('Page.screencastFrame', async ({ data, sessionId }) => {
      current = Buffer.from(data, 'base64');
      try {
        await session.send('Page.screencastFrameAck', { sessionId });
      } catch {
        /* stopped */
      }
    });
    await session.send('Page.startScreencast', {
      format: 'jpeg',
      quality: 88,
      maxWidth: WIDTH,
      maxHeight: HEIGHT,
      everyNthFrame: 1,
    });
    const t0 = Date.now();
    while (!current) {
      if (Date.now() - t0 > 8_000) throw new Error('screencast produced no frames');
      await new Promise((r) => setTimeout(r, 20));
    }
    running = true;
    pump = (async () => {
      const frame = 1000 / FPS;
      while (running) {
        const started = Date.now();
        if (current && ff.stdin.writable) {
          const ok = ff.stdin.write(current);
          if (!ok) await new Promise((r) => ff.stdin.once('drain', r));
        }
        const wait = frame - (Date.now() - started);
        if (wait > 0) await new Promise((r) => setTimeout(r, wait));
      }
    })();
  };

  const stop = async () => {
    running = false;
    await pump;
    try {
      await session.send('Page.stopScreencast');
    } catch {
      /* already gone */
    }
    await new Promise((resolve, reject) => {
      ff.stdin.end();
      ff.on('close', (code) =>
        code === 0 ? resolve() : reject(new Error(`ffmpeg exited ${code}`)),
      );
    });
  };

  return { start, stop };
}

async function ready(page) {
  await page.waitForFunction(() => {
    const el = document.getElementById('board');
    return !!el?.shadowRoot?.querySelector('ui-card');
  });
  await page.evaluate(() => document.fonts.ready);
  await pause(page, 400);
}

async function installCursor(page) {
  await page.addStyleTag({
    content: `
      html, body, * { cursor: none !important; }
      #hero-cursor {
        position: fixed;
        top: 0;
        left: 0;
        z-index: 2147483647;
        width: 28px;
        height: 28px;
        pointer-events: none;
        opacity: 0;
        filter: drop-shadow(0 2px 4px rgb(0 0 0 / 0.45));
        transition: opacity 420ms ease, transform 80ms ease;
        will-change: transform, opacity;
      }
      #hero-cursor.down { transform: translate(-3px, -1px) scale(0.86); }
      #hero-cursor svg { display: block; width: 28px; height: 28px; }
    `,
  });
  await page.evaluate(() => {
    const el = document.createElement('div');
    el.id = 'hero-cursor';
    el.innerHTML = `<svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path d="M4.2 3.4 19 12.1l-7.1 1.4 3.6 7.2-2.5 1.2-3.6-7.3-5.2 4.1z"
        fill="#fff" stroke="#111" stroke-width="1.1" stroke-linejoin="round"/>
    </svg>`;
    document.documentElement.appendChild(el);
    window.addEventListener(
      'mousemove',
      (e) => {
        el.style.translate = `${e.clientX}px ${e.clientY}px`;
      },
      true,
    );
    window.addEventListener('mousedown', () => el.classList.add('down'), true);
    window.addEventListener('mouseup', () => el.classList.remove('down'), true);
  });
}

function createPointer(page) {
  let pos = { x: WIDTH * 0.58, y: HEIGHT * 0.42 };

  const move = async (x, y, { ms = 700, arc = 0 } = {}) => {
    const from = pos;
    const t0 = Date.now();
    for (;;) {
      const t = Math.min(1, (Date.now() - t0) / ms);
      const e = easeInOut(t);
      const px = from.x + (x - from.x) * e;
      const py = from.y + (y - from.y) * e - Math.sin(e * Math.PI) * arc;
      await page.mouse.move(px, py);
      pos = { x: px, y: py };
      if (t >= 1) break;
      await new Promise((r) => setTimeout(r, 8));
    }
    pos = { x, y };
  };

  const centerOf = async (locator) => {
    const box = await locator.boundingBox();
    if (!box) throw new Error('locator has no box');
    return { x: box.x + box.width / 2, y: box.y + Math.min(22, box.height / 2) };
  };

  const click = async (locator, { ms = 640 } = {}) => {
    const at = await centerOf(locator);
    await move(at.x, at.y, { ms });
    await pause(page, 160);
    await page.mouse.down();
    await pause(page, 70);
    await page.mouse.up();
  };

  const type = async (locator, text, { delay = 72 } = {}) => {
    await click(locator);
    await pause(page, 220);
    await locator.pressSequentially(text, { delay });
  };

  const drag = async (from, to, { ms = 1200, arc = 56 } = {}) => {
    await move(from.x, from.y, { ms: 620 });
    await pause(page, 200);
    await page.mouse.down();
    await pause(page, 90);
    await move(from.x + 12, from.y - 10, { ms: 240 });
    await pause(page, 320);
    await move(to.x, to.y, { ms, arc });
    await pause(page, 360);
    await page.mouse.up();
    await pause(page, 820);
  };

  return { move, click, type, drag, set: (p) => (pos = p), pos: () => pos };
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
      return { x: r.x + Math.min(48, r.width / 2), y: r.y + 18 };
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
      const x = edge === 'start' ? r.x + 16 : r.x + r.width / 2;
      return { x, y: r.y + 12 };
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
    return { x: r.x + r.width / 2, y: r.y + Math.min(200, r.height * 0.42) };
  }, column);
}

/** A drag sets suppressOpen, so the next card click is swallowed. */
async function openCard(page, pointer, card) {
  const dialog = page.getByRole('dialog', { name: 'Card' });
  await pointer.click(card);
  try {
    await dialog.waitFor({ state: 'visible', timeout: 1600 });
  } catch {
    await pointer.click(card, { ms: 420 });
    await dialog.waitFor({ state: 'visible', timeout: 8000 });
  }
}

let server;
try {
  server = startServer();
  await waitForURL(BASE);
  const ffmpegBin = await playwrightFfmpeg();

  const browser = await chromium.launch({
    headless: true,
    args: [
      '--autoplay-policy=no-user-gesture-required',
      '--force-color-profile=srgb',
      '--disable-background-timer-throttling',
      '--disable-renderer-backgrounding',
      '--disable-backgrounding-occluded-windows',
    ],
  });
  await rm(TMP, { recursive: true, force: true });
  await mkdir(TMP, { recursive: true });

  const context = await browser.newContext({
    viewport: { width: WIDTH, height: HEIGHT },
    deviceScaleFactor: 2,
    colorScheme: 'dark',
  });
  const page = await context.newPage();
  await page.goto(BASE, { waitUntil: 'networkidle' });
  await ready(page);
  await installCursor(page);

  const pointer = createPointer(page);
  await page.mouse.move(pointer.pos().x, pointer.pos().y);
  const rawWebm = path.join(TMP, 'hero.webm');
  const rec = createRecorder(page, rawWebm, ffmpegBin);
  await rec.start();

  await page.evaluate(() => {
    const c = document.getElementById('hero-cursor');
    if (c) c.style.opacity = '1';
  });
  await pause(page, 1400);

  const board = page.locator('#board');
  const todo = board.locator('[data-column="todo"]');

  await pointer.click(todo.getByRole('button', { name: 'Add a card' }));
  await pause(page, 360);
  await pointer.type(todo.getByRole('textbox', { name: 'Add a card' }), 'Cut the trailer');
  await pause(page, 280);
  await page.keyboard.press('Enter');
  await page.waitForFunction(() =>
    document.getElementById('board').items.some((it) => it.text === 'Cut the trailer'),
  );
  await pause(page, 900);

  const added = board.locator('[data-column="todo"] .card[data-text="Cut the trailer"]');
  await openCard(page, pointer, added);
  await pause(page, 520);
  const dialog = page.getByRole('dialog', { name: 'Card' });
  await pointer.click(page.getByRole('option', { name: 'Blocked' }));
  await page.waitForFunction(() =>
    (document.getElementById('board').items.find((it) => it.text === 'Cut the trailer')?.labels || []).includes(
      'blocked',
    ),
  );
  await pause(page, 480);
  await pointer.type(page.getByRole('textbox', { name: 'Description' }), 'Need the Berlin cut.');
  await pause(page, 640);
  await pointer.click(page.getByRole('button', { name: 'Close' }));
  await dialog.waitFor({ state: 'hidden', timeout: 8000 });
  await pause(page, 720);

  await pointer.drag(await cardPoint(page, 'todo', 'Cut the trailer'), await laneBody(page, 'doing'), {
    ms: 1300,
    arc: 64,
  });
  await page.waitForFunction(() => {
    const it = document.getElementById('board').items.find((c) => c.text === 'Cut the trailer');
    return it?.column === 'doing';
  });
  await pause(page, 800);

  await pointer.click(page.getByRole('button', { name: 'Add another list' }));
  await pause(page, 360);
  await pointer.type(page.getByRole('textbox', { name: 'List title' }), 'Review');
  await pause(page, 240);
  await pointer.click(page.getByRole('button', { name: 'Add list' }));
  await page.waitForFunction(() =>
    document.getElementById('board').columns.some((c) => c.title === 'Review'),
  );
  await pause(page, 1000);

  const reviewId = await page.evaluate(
    () => document.getElementById('board').columns.find((c) => c.title === 'Review').id,
  );

  await pointer.drag(await laneHead(page, reviewId), await laneHead(page, 'todo', 'start'), {
    ms: 1400,
    arc: 28,
  });
  await page.waitForFunction((id) => {
    const ids = document.getElementById('board').columns.map((c) => c.id);
    return ids[0] === id || ids[1] === id;
  }, reviewId);
  await pause(page, 900);

  await pointer.drag(
    await cardPoint(page, 'doing', 'Cut the trailer'),
    await laneBody(page, reviewId),
    { ms: 1400, arc: 72 },
  );
  await page.waitForFunction((id) => {
    const it = document.getElementById('board').items.find((c) => c.text === 'Cut the trailer');
    return it?.column === id;
  }, reviewId);

  await pointer.move(WIDTH * 0.72, HEIGHT * 0.28, { ms: 900, arc: 18 });
  await pause(page, 1600);
  await page.evaluate(() => {
    const c = document.getElementById('hero-cursor');
    if (c) c.style.opacity = '0';
  });
  await pause(page, 700);

  await rec.stop();

  await page.evaluate(() => document.getElementById('hero-cursor')?.remove());
  await page.screenshot({
    path: path.join(OUT_DIR, 'board-hero.jpg'),
    type: 'jpeg',
    quality: 86,
  });

  await context.close();
  await browser.close();

  await mkdir(OUT_DIR, { recursive: true });
  await copyFile(rawWebm, path.join(OUT_DIR, 'board-hero.webm'));
  await rm(TMP, { recursive: true, force: true });
  console.log('wrote docs/public/board-hero.webm and docs/public/board-hero.jpg');
} finally {
  stopServer(server);
}
