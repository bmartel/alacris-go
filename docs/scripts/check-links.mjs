// Check that every internal link in the built site points at a page that
// exists.
//
// A docs site's most common defect is a link that used to work. Catching it at
// build time costs a second and saves a reader's trust.
import { readdir, readFile } from 'node:fs/promises';
import { join, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const dist = fileURLToPath(new URL('../dist/', import.meta.url));
const BASE = '/alacris-go';

async function walk(dir) {
  const out = [];
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) out.push(...(await walk(path)));
    else if (entry.name.endsWith('.html')) out.push(path);
  }
  return out;
}

const pages = await walk(dist);

// Every path the built site can serve, normalised without a trailing slash.
const served = new Set();
for (const page of pages) {
  const url = '/' + relative(dist, page).replaceAll('\\', '/');
  served.add(BASE + url.replace(/\/index\.html$/, '').replace(/\.html$/, ''));
}
// Non-HTML assets count too.
async function walkAll(dir) {
  const out = [];
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) out.push(...(await walkAll(path)));
    else out.push(path);
  }
  return out;
}
for (const asset of await walkAll(dist)) {
  served.add(BASE + '/' + relative(dist, asset).replaceAll('\\', '/'));
}
served.add(BASE);
served.add(BASE + '/');

const hrefPattern = /\shref="([^"]+)"/g;
let broken = 0;
let checked = 0;

for (const page of pages) {
  const html = await readFile(page, 'utf8');
  const from = '/' + relative(dist, page).replaceAll('\\', '/');

  for (const [, href] of html.matchAll(hrefPattern)) {
    // External, in-page and non-navigational links are somebody else's problem.
    if (!href.startsWith('/') || href.startsWith('//')) continue;
    checked++;

    const target = href.split('#')[0].split('?')[0].replace(/\/$/, '') || BASE;
    if (served.has(target) || served.has(target + '/')) continue;

    console.error(`  broken: ${from} -> ${href}`);
    broken++;
  }
}

if (broken > 0) {
  console.error(`\n  docs: ${broken} broken internal link(s) across ${pages.length} pages\n`);
  process.exit(1);
}
console.log(`  docs: ${checked} internal links across ${pages.length} pages, all resolve`);
