// Typography tokens — the Material type scale.
//
// Fifteen roles (display/headline/title/body/label × lg/md/sm), each emitted
// as granular tokens plus a `font` shorthand so a component can write
// `font: var(--ui-type-body-md)` and get weight, size, line-height and family
// in one declaration. Letter-spacing cannot ride the shorthand, so tracking is
// its own token.
//
// Faces are system tokens (`--ui-font-brand/plain/code`). The default is
// Google Sans Flex — the typeface Google's own Material 3 surfaces use —
// with Google Sans as the installed-on-device fallback. Swap the whole
// product with a preset name, a single `family`, or explicit stacks; every
// component follows because they consume `sys.type.*` / `sys.font.*` only.

const SYSTEM_SANS = "system-ui, -apple-system, 'Segoe UI', sans-serif";
const SYSTEM_MONO = "ui-monospace, 'SF Mono', Menlo, Consolas, monospace";
const GF = 'https://fonts.googleapis.com/css2?';

const sansStack = (...names) =>
  names.map((n) => `'${n}'`).join(', ') + ', ' + SYSTEM_SANS;

/** Google Fonts CSS2 queries keyed by the CSS family name they load. */
const GF_QUERY = {
  'Google Sans Flex': 'Google+Sans+Flex:opsz,wght@6..144,400..700',
  'Google Sans': 'Google+Sans:wght@400;500;700',
  Roboto: 'Roboto:wght@400;500;700',
  'Roboto Flex': 'Roboto+Flex:opsz,wght@8..144,400..700',
};

/**
 * Named typeface packages. A preset is the one-liner for `createTheme`:
 * `typography: 'google-sans'` (or `{ preset: 'google-sans' }`).
 */
export const FONT_PRESETS = {
  'google-sans-flex': {
    brand: sansStack('Google Sans Flex', 'Google Sans'),
    plain: sansStack('Google Sans Flex', 'Google Sans'),
    href: GF + 'family=Google+Sans+Flex:opsz,wght@6..144,400..700&display=swap',
  },
  'google-sans': {
    brand: sansStack('Google Sans', 'Google Sans Flex'),
    plain: sansStack('Google Sans', 'Google Sans Flex'),
    href: GF + 'family=Google+Sans:wght@400;500;700&display=swap',
  },
  roboto: {
    brand: sansStack('Roboto'),
    plain: sansStack('Roboto'),
    href: GF + 'family=Roboto:wght@400;500;700&display=swap',
  },
  system: {
    brand: SYSTEM_SANS,
    plain: SYSTEM_SANS,
    href: null,
  },
};

export const DEFAULT_FONT_PRESET = 'google-sans-flex';

/** Default stacks — what `typographyTokens()` emits with no config. */
export const FONT_STACKS = {
  brand: FONT_PRESETS[DEFAULT_FONT_PRESET].brand,
  plain: FONT_PRESETS[DEFAULT_FONT_PRESET].plain,
  code: SYSTEM_MONO,
};

// role: [sizePx, lineHeightPx, weight, trackingPx, family]
const SCALE = {
  'display-lg': [57, 64, 400, -0.25, 'brand'],
  'display-md': [45, 52, 400, 0, 'brand'],
  'display-sm': [36, 44, 400, 0, 'brand'],
  'headline-lg': [32, 40, 400, 0, 'brand'],
  'headline-md': [28, 36, 400, 0, 'brand'],
  'headline-sm': [24, 32, 400, 0, 'brand'],
  'title-lg': [22, 28, 400, 0, 'brand'],
  'title-md': [16, 24, 500, 0.15, 'plain'],
  'title-sm': [14, 20, 500, 0.1, 'plain'],
  'body-lg': [16, 24, 400, 0.5, 'plain'],
  'body-md': [14, 20, 400, 0.25, 'plain'],
  'body-sm': [12, 16, 400, 0.4, 'plain'],
  'label-lg': [14, 20, 500, 0.1, 'plain'],
  'label-md': [12, 16, 500, 0.5, 'plain'],
  'label-sm': [11, 16, 500, 0.5, 'plain'],
};

export const TYPE_ROLES = Object.keys(SCALE);

const rem = (px) => `${+(px / 16).toFixed(4)}rem`;

const GENERIC = new Set([
  'serif', 'sans-serif', 'monospace', 'cursive', 'fantasy', 'system-ui',
  'ui-sans-serif', 'ui-serif', 'ui-monospace', 'ui-rounded', 'emoji', 'math',
  'fangsong', '-apple-system', 'blinkmacsystemfont', 'segoe ui', 'arial',
  'helvetica neue', 'helvetica',
]);

const stripQuote = (s) => s.replace(/^['"]|['"]$/g, '');

/** First family name in a CSS font-family stack. */
export function firstFamily(stack) {
  return stripQuote((stack || '').split(',')[0].trim());
}

/**
 * A CSS stack from either a family name (`'Inter'`) or an already-complete
 * stack (`Inter, system-ui, sans-serif`). Names are quoted; stacks pass through.
 */
function asStack(value, fallback) {
  if (!value) return null;
  const v = String(value).trim();
  if (v.includes(',')) return v;
  return `'${stripQuote(v)}', ${fallback}`;
}

/** Google Fonts stylesheet URL for the given CSS family names / stacks. */
export function googleFontsHref(names) {
  const families = [];
  const seen = new Set();
  for (const n of names) {
    const fam = firstFamily(n);
    if (!fam || GENERIC.has(fam.toLowerCase()) || seen.has(fam)) continue;
    seen.add(fam);
    families.push(fam);
  }
  if (!families.length) return null;
  const params = families.map((fam) => 'family=' + (GF_QUERY[fam] || `${fam.replace(/ /g, '+')}:wght@400;500;700`));
  return GF + params.join('&') + '&display=swap';
}

/**
 * Normalize `createTheme({ typography })` into stacks + a stylesheet href.
 *
 * Accepts:
 *   'google-sans-flex' | 'google-sans' | 'roboto' | 'system'
 *   { preset, family, brand, plain, code, scale, load }
 *
 * `family` sets brand and plain together. `load` is a Google Fonts (or any)
 * stylesheet URL, `true` (default — derive from the faces), or `false`
 * (tokens only; you host the files).
 */
export function resolveTypography(config = {}) {
  if (typeof config === 'string') config = { preset: config };

  const presetName = config.preset
    || (config.family || config.brand || config.plain ? null : DEFAULT_FONT_PRESET);
  if (presetName && !FONT_PRESETS[presetName]) {
    throw new Error(
      `Unknown font preset "${presetName}". Use one of: ${Object.keys(FONT_PRESETS).join(', ')}`);
  }
  const preset = presetName ? FONT_PRESETS[presetName] : null;
  const familyStack = asStack(config.family, SYSTEM_SANS);

  const brand = asStack(config.brand, SYSTEM_SANS) || familyStack || preset?.brand || FONT_STACKS.brand;
  const plain = asStack(config.plain, SYSTEM_SANS) || familyStack || preset?.plain || FONT_STACKS.plain;
  const code = asStack(config.code, SYSTEM_MONO) || FONT_STACKS.code;
  const scale = config.scale ?? 1;

  let href = null;
  if (config.load === false) href = null;
  else if (typeof config.load === 'string') href = config.load;
  else href = preset?.href ?? googleFontsHref([brand, plain]);

  return { brand, plain, code, scale, href, preset: presetName };
}

/**
 * Build the typography token map.
 *
 * config: a preset name, or { preset, family, brand, plain, code, scale, load }
 */
export function typographyTokens(config = {}) {
  const { brand, plain, code, scale } = resolveTypography(config);
  const out = {
    'font-brand': brand,
    'font-plain': plain,
    'font-code': code,
  };
  for (const role of TYPE_ROLES) {
    const [size, line, weight, tracking, family] = SCALE[role];
    const fam = `var(--ui-font-${family})`;
    out[`type-${role}-size`] = rem(size * scale);
    out[`type-${role}-line`] = rem(line * scale);
    out[`type-${role}-weight`] = String(weight);
    out[`type-${role}-tracking`] = tracking ? `${tracking * scale}px` : '0';
    out[`type-${role}-font`] = fam;
    out[`type-${role}`] =
      `var(--ui-type-${role}-weight) var(--ui-type-${role}-size) / var(--ui-type-${role}-line) var(--ui-type-${role}-font)`;
  }
  return out;
}
