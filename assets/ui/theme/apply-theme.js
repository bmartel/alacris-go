// applyTheme — one constructed stylesheet for the whole design system.
//
// All tokens live on `:root`, so they inherit through every shadow boundary
// and a re-theme is a single `replaceSync` — no component re-renders, no
// sheet re-adoption, every element on the page updates at once.
//
// Scheme selection is layered so the same sheet serves all three modes:
//   - light tokens on `:root` (the default)
//   - dark tokens under `prefers-color-scheme: dark` unless the app pinned
//     light (`<html data-ui-scheme="light">`)
//   - dark tokens whenever the app pinned dark (`data-ui-scheme="dark"`)

import { signal, computed } from '@alacris/core';
import { createTheme } from './create-theme.js';

// `color-scheme` is a real CSS property riding along in the token map (it
// keeps scrollbars and form controls in scheme); everything else gets `--ui-`.
const decl = (name, value) => (name === 'color-scheme' ? name : `--ui-${name}`) + `:${value};`;

const block = (tokens) => {
  let s = '';
  for (const name in tokens) s += decl(name, tokens[name]);
  return s;
};

// Face on `:root` so native text and un-typed markup inherit the system
// plain typeface. Components then set a type-role shorthand on top.
const FACE = 'font-family:var(--ui-font-plain);font-optical-sizing:auto;font-synthesis:none;';

const SCROLLBAR = [
  '@media (hover: hover) and (pointer: fine) {',
  '  :root, html, body {',
  '    scrollbar-width: thin;',
  '    scrollbar-color: var(--ui-scrollbar-thumb) var(--ui-scrollbar-track);',
  '  }',
  '  ::-webkit-scrollbar {',
  '    inline-size: var(--ui-scrollbar-size);',
  '    block-size: var(--ui-scrollbar-size);',
  '  }',
  '  ::-webkit-scrollbar-track {',
  '    background: var(--ui-scrollbar-track);',
  '  }',
  '  ::-webkit-scrollbar-thumb {',
  '    background-color: var(--ui-scrollbar-thumb);',
  '    border-radius: var(--ui-scrollbar-radius);',
  '    border: 2px solid transparent;',
  '    background-clip: content-box;',
  '  }',
  '  ::-webkit-scrollbar-thumb:hover {',
  '    background-color: var(--ui-scrollbar-thumb-hover);',
  '  }',
  '  ::-webkit-scrollbar-thumb:active {',
  '    background-color: var(--ui-scrollbar-thumb-active);',
  '  }',
  '  ::-webkit-scrollbar-corner {',
  '    background: transparent;',
  '  }',
  '  ::-webkit-scrollbar-button {',
  '    display: none;',
  '    inline-size: 0;',
  '    block-size: 0;',
  '  }',
  '}',
].join('\n');

export function themeCss(theme) {
  const { common, schemes } = theme;
  return [
    `:root{${FACE}${block(common)}${block(schemes.light)}}`,
    `:root[data-ui-scheme="dark"]{${block(schemes.dark)}}`,
    `@media (prefers-color-scheme: dark){:root:not([data-ui-scheme="light"]){${block(schemes.dark)}}}`,
    SCROLLBAR,
  ].join('\n');
}

let sheet = null;
let styleEl = null;

/** The currently applied theme (a signal; null until applyTheme runs). */
export const activeTheme = signal(null);

/**
 * Apply a theme to the document. Accepts a theme from `createTheme` or a
 * config object (which is passed through `createTheme` for you). Calling it
 * again rewrites the same stylesheet in place.
 *
 * Faces referenced by the theme are loaded automatically (a Google Fonts
 * stylesheet for presets and `family` names). Pass `loadFonts: false` to
 * skip — useful when the files are self-hosted or already on the page.
 */
export function applyTheme(themeOrConfig = {}) {
  const theme = themeOrConfig.schemes ? themeOrConfig : createTheme(themeOrConfig);
  const text = themeCss(theme);
  if (!sheet && !styleEl) {
    if (document.adoptedStyleSheets && typeof CSSStyleSheet.prototype.replaceSync === 'function') {
      sheet = new CSSStyleSheet();
      document.adoptedStyleSheets = [...document.adoptedStyleSheets, sheet];
    } else {
      styleEl = document.createElement('style');
      document.head.append(styleEl);
    }
  }
  if (sheet) sheet.replaceSync(text);
  else styleEl.textContent = text;
  loadThemeFonts(themeOrConfig.loadFonts === false ? { fonts: { href: null } } : theme);
  activeTheme.set(theme);
  return theme;
}

const FONT_ATTR = 'data-ui-font';
const PRECONNECT_ATTR = 'data-ui-font-preconnect';

/**
 * Sync the document's typeface stylesheet to `theme.fonts.href`. A null href
 * removes a previously injected link. Safe to call repeatedly; one `<link>`
 * is reused in place. Constructed stylesheets cannot `@import`, so this is a
 * real element — the same one `themeCss` consumers add by hand for static CSS.
 */
export function loadThemeFonts(theme, doc = document) {
  const head = doc?.head;
  if (!head) return;
  const href = theme?.fonts?.href || null;
  const existing = head.querySelector(`link[${FONT_ATTR}]`);
  if (!href) {
    existing?.remove();
    return;
  }
  if (/fonts\.googleapis\.com|fonts\.gstatic\.com/.test(href)) ensurePreconnect(head);
  if (existing) {
    if (existing.getAttribute('href') !== href) existing.setAttribute('href', href);
    return;
  }
  const link = doc.createElement('link');
  link.rel = 'stylesheet';
  link.setAttribute('href', href);
  link.setAttribute(FONT_ATTR, '');
  head.append(link);
}

function ensurePreconnect(head) {
  if (head.querySelector(`link[${PRECONNECT_ATTR}]`)) return;
  for (const href of ['https://fonts.googleapis.com', 'https://fonts.gstatic.com']) {
    const l = head.ownerDocument.createElement('link');
    l.rel = 'preconnect';
    l.href = href;
    if (href.includes('gstatic')) l.crossOrigin = 'anonymous';
    l.setAttribute(PRECONNECT_ATTR, '');
    head.append(l);
  }
}

// ------------------------------------------------------------------ scheme

/** 'light' | 'dark' | 'auto' — what the app asked for. */
export const schemePreference = signal('auto');

const prefersDark =
  typeof matchMedia === 'function' ? matchMedia('(prefers-color-scheme: dark)') : null;
const osDark = signal(!!prefersDark?.matches);
prefersDark?.addEventListener?.('change', (e) => osDark.set(e.matches));

/** The scheme actually in effect right now: 'light' | 'dark'. */
export const scheme = computed(() => {
  const pref = schemePreference();
  return pref === 'auto' ? (osDark() ? 'dark' : 'light') : pref;
});

/** Pin the scheme, or return to following the OS with 'auto'. */
export function setScheme(pref /* 'light' | 'dark' | 'auto' */) {
  schemePreference.set(pref);
  const el = document.documentElement;
  if (pref === 'auto') el.removeAttribute('data-ui-scheme');
  else el.setAttribute('data-ui-scheme', pref);
}

/** Flip between light and dark (leaves 'auto' by pinning the opposite). */
export function toggleScheme() {
  setScheme(scheme() === 'dark' ? 'light' : 'dark');
}
