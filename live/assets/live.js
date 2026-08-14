// alacris live — the browser half of server-driven props.
//
// Down: the server sends {o:'p', i:<element id>, k:<prop>, v:<value>} and this
// writes el[prop] = value. That is the whole trick. A prop is a signal, so one
// property write becomes one DOM update, with no HTML on the wire and nothing
// to diff or morph. Focus, scroll position and typed text are untouched
// because no node is replaced.
//
// Up: an element carrying data-ala-on="greet:say-hello" has its CustomEvents
// forwarded to the named server action. alacris events are composed and
// bubbling, so one listener per event type on the document reaches every
// element, including ones that do not exist yet.

// The page id says which of this browser's pages is talking. It is not a
// secret and never has to be: the capability is an HttpOnly cookie the server
// set, which this script cannot read and does not need to. The browser
// attaches it, and the two together are what authorise a request.
const script = document.querySelector('script[data-page][data-endpoint]');
const page = script?.dataset.page;
const endpoint = script?.dataset.endpoint;

// Recovery is on unless the page opts out with data-recover="off"
// (alacris.Config.NoRecover). See recover() for what it does and why.
const recoverEnabled = script?.dataset.recover !== 'off';

// Trusted Types blocks assigning a string to innerHTML. Only SetHTML needs it,
// and the markup is the server's own, so a pass-through policy is honest about
// what is happening rather than pretending to sanitize.
let policy = null;
if (window.trustedTypes?.createPolicy) {
  try {
    policy = window.trustedTypes.createPolicy('alacris-live', { createHTML: (s) => s });
  } catch {
    // Another page already registered the name, or the CSP forbids it.
  }
}
const asHTML = (s) => (policy ? policy.createHTML(s) : s);

// The connection lifecycle, announced as 'alacris:live' window events so a
// page can show its own indicator: state is 'open', 'closed' (stream down;
// the browser retries transient drops itself), 'recovering' (probing after a
// permanent failure), 'reloading' (session gone; a fresh render is coming),
// or 'error' (an action failed; carries action and status or error).
const announce = (state, detail) =>
  window.dispatchEvent(new CustomEvent('alacris:live', { detail: { state, ...detail } }));

// The keys that would turn a state write into script execution. The server
// refuses to send them; refusing to apply them keeps that true even for a
// frame that reached apply() some other way — the docs site drives apply()
// directly, and a consumer copying that pattern should not inherit a sink.
const blockedProps = new Set(['innerHTML', 'outerHTML', '__proto__', 'constructor', 'prototype']);
const blockedAttr = (k) => /^on[a-z]+$/i.test(k) || String(k).toLowerCase() === 'srcdoc';

/** Apply one patch. */
function apply(p) {
  const el = p.i ? document.getElementById(p.i) : document.documentElement;
  if (!el) return; // a patch for an element this page does not have is not an error

  switch (p.o) {
    case 'p':
      // The prop name is the JavaScript one from define(), not the attribute:
      // define() puts an accessor for it on the element's prototype, and
      // writing it sets the signal directly. Values assigned before the
      // element upgrades are replayed by define()'s constructor.
      if (blockedProps.has(p.k)) {
        console.warn('alacris live: refusing to set "%s"; it is not a component prop', p.k);
        break;
      }
      el[p.k] = p.v;
      break;

    case 'a':
      if (blockedAttr(p.k)) {
        console.warn('alacris live: refusing to set attribute "%s"; it executes rather than describes', p.k);
        break;
      }
      if (p.v === null || p.v === undefined) el.removeAttribute(p.k);
      else el.setAttribute(p.k, String(p.v));
      break;

    case 'h':
      // The default slot has an empty name, which the encoder leaves out of
      // the frame entirely, so it arrives as undefined rather than "".
      setSlot(el, p.k ?? '', p.v ?? '');
      break;

    case 'c':
      el.classList.toggle(p.k, !!p.v);
      break;

    case 'x':
      location.reload();
      break;
  }
}

/** Replace the light-DOM children assigned to one slot. */
function setSlot(el, slot, html) {
  for (const node of [...el.childNodes]) {
    // Text goes to the default slot and cannot carry a slot attribute of its
    // own, so it has to be considered here — server-rendered slot content is
    // often bare text, and skipping it appends instead of replacing.
    const assigned = node.nodeType === 1 ? node.getAttribute('slot') || '' : '';
    if (assigned === slot && (node.nodeType === 1 || node.nodeType === 3)) node.remove();
  }

  const tpl = document.createElement('template');
  tpl.innerHTML = asHTML(html);
  if (slot) {
    for (const node of [...tpl.content.children]) node.setAttribute('slot', slot);
    // Bare text cannot be assigned to a named slot; it would silently land in
    // the default one.
    for (const node of [...tpl.content.childNodes]) {
      if (node.nodeType === 3 && node.textContent.trim()) {
        console.warn('alacris live: text sent to slot "%s" needs an element around it', slot);
        break;
      }
    }
  }
  el.append(tpl.content);
}

/** Post one action to the server. */
function send(action, id, detail) {
  return fetch(endpoint, {
    method: 'POST',
    // application/json is not a content type a cross-site form can send, which
    // is one of the things keeping a hostile page from forging an action.
    headers: { 'content-type': 'application/json' },
    // include, not same-origin: a page served from a different origin than the
    // live endpoint still has to send the cookie. The server only accepts a
    // cross-origin request from an origin it was told to trust.
    credentials: 'include',
    body: JSON.stringify({ p: page, a: action, i: id, d: detail ?? null }),
  }).then(
    async (r) => {
      if (r.ok) return;
      announce('error', { action, status: r.status });
      // "unknown session" means this page's session no longer exists — the
      // server restarted or the session expired. The stream will hit the same
      // wall; start recovering now rather than waiting for it. A 404 for an
      // unknown *action* has a different body and is a developer error, not a
      // dead session.
      if (r.status === 404 && (await r.text()).startsWith('unknown session')) recover();
    },
    (error) => announce('error', { action, error: String(error) }),
  );
}

let source = null;
let recovering = false;

/** The stream URL for this page. */
function streamURL() {
  const url = new URL(endpoint, location.href);
  url.searchParams.set('p', page);
  return url;
}

/** Open the patch stream. EventSource retries transient drops on its own. */
function connect() {
  source = new EventSource(streamURL(), { withCredentials: true });
  source.onopen = () => announce('open');
  source.onerror = () => {
    // CONNECTING means the browser is already retrying — a dropped
    // connection, a heartbeat gap — and needs no help. CLOSED means the
    // server answered and refused: the spec makes that failure permanent, so
    // if anyone is going to get this page updating again, it is us.
    if (source.readyState === EventSource.CLOSED) {
      if (recoverEnabled) recover();
      else announce('closed');
    } else {
      announce('closed');
    }
  };
  source.onmessage = (e) => {
    let patches;
    try {
      patches = JSON.parse(e.data);
    } catch {
      return;
    }
    if (!Array.isArray(patches)) return;
    for (const p of patches) {
      // One malformed patch must not take the rest of the frame with it.
      try {
        apply(p);
      } catch (err) {
        console.warn('alacris live: a patch failed to apply', p, err);
      }
    }
  };
}

/**
 * Get the page updating again after a permanent stream failure.
 *
 * Sessions live in the server's memory, so a restart or an expiry leaves this
 * page holding an id the server has never heard of — and an EventSource that
 * got a 404 will not retry. Probe the endpoint until the server answers:
 *
 *   - 200: the session still exists (the failure was something else) — attach
 *     a fresh EventSource and carry on.
 *   - 404: the server is up and the session is gone. The only way back to a
 *     correct page is a fresh render, so reload — once. State the server held
 *     for this page is gone either way; a reload is the honest outcome.
 *   - unreachable, or a 5xx: the server is restarting or a proxy is holding
 *     the fort. Keep probing, with backoff — this is the window where a dev
 *     server is recompiling, and waiting it out is the whole point.
 *   - any other status: something is misconfigured in a way probing will not
 *     fix. Announce closed and stop.
 */
async function recover() {
  if (recovering || !recoverEnabled) return;
  recovering = true;
  source?.close();
  announce('recovering');

  let delay = 500;
  for (;;) {
    let status = 0;
    try {
      const ctrl = new AbortController();
      const r = await fetch(streamURL(), {
        credentials: 'include',
        headers: { accept: 'text/event-stream' },
        signal: ctrl.signal,
      });
      status = r.status;
      // Only the status matters; do not hold a second stream open.
      ctrl.abort();
    } catch {
      // Unreachable: keep probing.
    }
    if (status === 200) {
      recovering = false;
      connect();
      return;
    }
    if (status === 404) {
      reloadOnce();
      return;
    }
    if (status !== 0 && status < 500) {
      announce('closed');
      return;
    }
    await new Promise((r) => setTimeout(r, delay + Math.random() * 250));
    delay = Math.min(delay * 2, 15_000);
  }
}

/**
 * Reload for a fresh session — unless this page already did that a moment
 * ago, which means fresh sessions are dying too and reloading again would
 * loop. sessionStorage is per-tab, which is exactly the scope a page id has.
 */
function reloadOnce() {
  const KEY = 'alacris-live-reloaded';
  let last = 0;
  try {
    last = Number(sessionStorage.getItem(KEY)) || 0;
  } catch {
    // Storage can be denied; recovering is still worth attempting.
  }
  if (Date.now() - last < 10_000) {
    announce('closed');
    return;
  }
  try {
    sessionStorage.setItem(KEY, String(Date.now()));
  } catch {}
  announce('reloading');
  location.reload();
}

const wired = new Set();

/** Listen for one event type, once, for the whole document. */
function wire(type) {
  if (wired.has(type)) return;
  wired.add(type);
  document.addEventListener(type, (e) => {
    // composedPath crosses shadow boundaries, which is where the event
    // started; the element carrying data-ala-on is somewhere along it.
    for (const node of e.composedPath()) {
      const spec = node.dataset?.alaOn;
      if (!spec) continue;
      for (const pair of spec.split(/\s+/)) {
        const split = pair.indexOf(':');
        if (split > 0 && pair.slice(0, split) === type) {
          send(pair.slice(split + 1), node.id || '', e.detail);
          return;
        }
      }
    }
  });
}

/** Wire every event type declared anywhere under root. */
function scan(root) {
  if (root.nodeType !== 1 && root.nodeType !== 9) return;
  const found = root.querySelectorAll?.('[data-ala-on]') ?? [];
  const all = root.matches?.('[data-ala-on]') ? [root, ...found] : found;
  for (const el of all) {
    for (const pair of el.dataset.alaOn.split(/\s+/)) {
      const split = pair.indexOf(':');
      if (split > 0) wire(pair.slice(0, split));
    }
  }
}

function start() {
  scan(document);
  new MutationObserver((records) => {
    for (const r of records) {
      if (r.type === 'attributes') scan(r.target);
      else for (const node of r.addedNodes) scan(node);
    }
  }).observe(document.documentElement, {
    subtree: true,
    childList: true,
    attributeFilter: ['data-ala-on'],
  });

  // withCredentials sends the cookie, which is what actually authorises this.
  connect();

  window.addEventListener('pagehide', () => source?.close());
}

if (page && endpoint) {
  start();
} else {
  console.warn('alacris live: no page id on the page; the live client did nothing');
}

// apply and send are exported so a page can drive the real client directly —
// the documentation site feeds apply the same frames a Go server would send,
// rather than reimplementing it and letting the two drift.
export { send, apply };
