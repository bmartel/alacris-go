// Copy the runtime the Go module vendors into public/lib, so every runnable
// example on this site executes the same bytes a Go server would serve.
//
// If the docs ever drift from the module, the demos break loudly instead of
// quietly documenting something that no longer exists.
import { cp, mkdir, rm, readdir, stat } from 'node:fs/promises';
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

for (const source of sources) {
  const dir = fileURLToPath(source.from);
  for (const name of await readdir(dir)) {
    if (!source.only(name)) continue;
    await cp(dir + name, out + name);
  }
}

const files = (await readdir(out)).sort();
console.log(`  docs: synced ${files.length} files into public/lib — ${files.join(', ')}`);

// AGENTS.md is served from the site so agents can curl it, but the repository
// root is where it lives — a project using this library wants it there, and two
// copies would drift.
const agents = fileURLToPath(new URL('../../AGENTS.md', import.meta.url));
const agentsOut = fileURLToPath(new URL('../public/AGENTS.md', import.meta.url));
await cp(agents, agentsOut);
console.log('  docs: synced AGENTS.md from the repository root');
