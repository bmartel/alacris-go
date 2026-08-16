// Copy the runtime the Go module vendors into public/lib, so every runnable
// example on this site executes the same bytes a Go server would serve.
//
// If the docs ever drift from the module, the demos break loudly instead of
// quietly documenting something that no longer exists.
import { cp, mkdir, rm, readdir, stat } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const sources = [
  { from: new URL('../../assets/', import.meta.url), only: (f) => f.endsWith('.js') },
  { from: new URL('../../live/assets/', import.meta.url), only: (f) => f.endsWith('.js') },
];

const out = fileURLToPath(new URL('../public/lib/', import.meta.url));

for (const source of sources) {
  try {
    await stat(fileURLToPath(source.from));
  } catch {
    console.error(`\n  docs: ${fileURLToPath(source.from)} is missing — is this running from the repo?\n`);
    process.exit(1);
  }
}

await rm(out, { recursive: true, force: true });
await mkdir(out, { recursive: true });

/** Recursively copy .js files, preserving the directory tree under assets/ui/. */
async function copyJs(from, to, only) {
  await mkdir(to, { recursive: true });
  let n = 0;
  for (const entry of await readdir(from, { withFileTypes: true })) {
    const src = join(from, entry.name);
    const dest = join(to, entry.name);
    if (entry.isDirectory()) {
      n += await copyJs(src, dest, only);
      continue;
    }
    if (!only(entry.name)) continue;
    await cp(src, dest);
    n++;
  }
  return n;
}

let total = 0;
for (const source of sources) {
  total += await copyJs(fileURLToPath(source.from), out, source.only);
}

console.log(`  docs: synced ${total} files into public/lib`);

// public/AGENTS.md is not synced from anywhere. It is the drop-in file a
// project using this library curls, and it is committed where Astro serves it
// from. The repository root holds a different file — the guide to working on
// this module — and copying one over the other is how that mistake would be
// made.
