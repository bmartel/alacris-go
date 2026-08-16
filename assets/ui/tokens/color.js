// Color engine — dependency-free tonal palettes in OKLCH.
//
// Material's color system is built on tonal palettes: thirteen-plus tones of a
// single hue/chroma, indexed by perceptual lightness 0–100, with semantic roles
// (primary, on-primary, surface-container, …) mapped onto specific tones per
// scheme. Google derives palettes in HCT; we get an equivalent result with
// OKLab/OKLCH: hold hue and chroma from the seed, target each tone's CIE
// luminance, and gamut-map by walking chroma down until the color fits sRGB.
//
// Everything here is pure math on hex strings — no DOM, usable in a worker or
// at build time to pregenerate a static theme.

// ---------------------------------------------------------------- sRGB <-> hex

const clamp01 = (x) => (x < 0 ? 0 : x > 1 ? 1 : x);

export function hexToRgb(hex) {
  let h = hex.replace('#', '').trim();
  if (h.length === 3) h = h[0] + h[0] + h[1] + h[1] + h[2] + h[2];
  const n = parseInt(h, 16);
  if (h.length !== 6 || Number.isNaN(n)) throw new Error(`invalid hex color: ${hex}`);
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
}

export function rgbToHex([r, g, b]) {
  const to = (v) => Math.round(clamp01(v / 255) * 255).toString(16).padStart(2, '0');
  return '#' + to(r) + to(g) + to(b);
}

const srgbToLinear = (c) => {
  const v = c / 255;
  return v <= 0.04045 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4);
};

const linearToSrgb = (v) => {
  const c = v <= 0.0031308 ? v * 12.92 : 1.055 * Math.pow(v, 1 / 2.4) - 0.055;
  return clamp01(c) * 255;
};

// ---------------------------------------------------------------- OKLab/OKLCH

function linearToOklab(r, g, b) {
  const l = Math.cbrt(0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b);
  const m = Math.cbrt(0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b);
  const s = Math.cbrt(0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b);
  return [
    0.2104542553 * l + 0.793617785 * m - 0.0040720468 * s,
    1.9779984951 * l - 2.428592205 * m + 0.4505937099 * s,
    0.0259040371 * l + 0.7827717662 * m - 0.808675766 * s,
  ];
}

function oklabToLinear(L, a, b) {
  const l = (L + 0.3963377774 * a + 0.2158037573 * b) ** 3;
  const m = (L - 0.1055613458 * a - 0.0638541728 * b) ** 3;
  const s = (L - 0.0894841775 * a - 1.291485548 * b) ** 3;
  return [
    4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s,
    -1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s,
    -0.0041960863 * l - 0.7034186147 * m + 1.7076147010 * s,
  ];
}

/** hex → { l, c, h } (OKLCH; h in degrees, 0 for achromatic). */
export function oklchFromHex(hex) {
  const [r, g, b] = hexToRgb(hex);
  const [L, a, bb] = linearToOklab(srgbToLinear(r), srgbToLinear(g), srgbToLinear(b));
  const c = Math.hypot(a, bb);
  const h = c < 1e-6 ? 0 : ((Math.atan2(bb, a) * 180) / Math.PI + 360) % 360;
  return { l: L, c, h };
}

const inGamut = ([r, g, b]) =>
  r >= -1e-4 && r <= 1 + 1e-4 && g >= -1e-4 && g <= 1 + 1e-4 && b >= -1e-4 && b <= 1 + 1e-4;

/**
 * { l, c, h } → hex. If the color is outside sRGB, chroma is reduced (binary
 * search) until it fits — hue and lightness are preserved, which is what keeps
 * a tonal ramp looking like one hue.
 */
export function hexFromOklch({ l, c, h }) {
  const rad = (h * Math.PI) / 180;
  const attempt = (cc) => oklabToLinear(l, cc * Math.cos(rad), cc * Math.sin(rad));
  let rgb = attempt(c);
  if (!inGamut(rgb)) {
    let lo = 0, hi = c;
    for (let i = 0; i < 20; i++) {
      const mid = (lo + hi) / 2;
      if (inGamut(attempt(mid))) lo = mid;
      else hi = mid;
    }
    rgb = attempt(lo);
  }
  return rgbToHex(rgb.map(linearToSrgb));
}

// ------------------------------------------------------------------- tones

// Material tone T is CIE L* — convert to luminance Y, and for near-neutral
// colors OKLab lightness is exactly cbrt(Y), which anchors the ramp so that
// tone 100 is white, tone 0 is black, and tone 50 is mid-gray to the eye.
const toneToOklabL = (tone) => {
  const L = tone;
  const Y = L > 8 ? Math.pow((L + 16) / 116, 3) : L / 903.2962962;
  return Math.cbrt(Y);
};

/** The tone steps Material's schemes draw from. */
export const TONES = [0, 4, 6, 10, 12, 17, 20, 22, 24, 30, 40, 50, 60, 70, 80, 87, 90, 92, 94, 95, 96, 98, 99, 100];

/**
 * A full tonal palette from one hue/chroma.
 * Returns { 0: '#000000', …, 100: '#ffffff' } over TONES.
 */
export function tonalPalette(hue, chroma) {
  const out = {};
  for (const tone of TONES) {
    out[tone] =
      tone === 0 ? '#000000'
      : tone === 100 ? '#ffffff'
      : hexFromOklch({ l: toneToOklabL(tone), c: chroma, h: hue });
  }
  return out;
}

// -------------------------------------------------------------- key palettes

// Fixed seeds for the status hues (Material error, plus the MUI-style
// success/warning/info set). Each can be overridden via createTheme colors.
export const DEFAULT_SEED = '#e8ad18';
const STATUS_SEEDS = { error: '#b3261e', success: '#1e8e3e', warning: '#e37400', info: '#0b57d0' };

// Chroma floors/targets in OKLCH units (roughly HCT chroma / 350).
const MIN_VIVID = 0.09;

/**
 * Build the key tonal palettes from a seed (or explicit per-palette seeds).
 *
 *   makePalettes({ seed: '#e8ad18' })
 *   makePalettes({ colors: { primary: '#0b57d0', tertiary: '#00695c' } })
 *
 * Returns { primary, secondary, tertiary, neutral, neutralVariant,
 *           error, success, warning, info } — each a tonal palette.
 */
export function makePalettes({ seed = DEFAULT_SEED, colors = {} } = {}) {
  const base = oklchFromHex(colors.primary || seed);
  const c = Math.max(base.c, MIN_VIVID);
  const from = (name, fallbackHue, fallbackChroma) => {
    if (colors[name]) {
      const k = oklchFromHex(colors[name]);
      return tonalPalette(k.h, Math.max(k.c, name.startsWith('neutral') ? k.c : MIN_VIVID * 0.6));
    }
    return tonalPalette(fallbackHue, fallbackChroma);
  };
  return {
    primary: from('primary', base.h, c),
    secondary: from('secondary', base.h, c / 3),
    tertiary: from('tertiary', (base.h + 60) % 360, c / 2),
    neutral: from('neutral', base.h, 0.012),
    neutralVariant: from('neutralVariant', base.h, 0.024),
    error: from('error', ...seedHC(colors.error || STATUS_SEEDS.error)),
    success: from('success', ...seedHC(colors.success || STATUS_SEEDS.success)),
    warning: from('warning', ...seedHC(colors.warning || STATUS_SEEDS.warning)),
    info: from('info', ...seedHC(colors.info || STATUS_SEEDS.info)),
  };
}

const seedHC = (hex) => {
  const k = oklchFromHex(hex);
  return [k.h, Math.max(k.c, MIN_VIVID)];
};

// ------------------------------------------------------------------ schemes

// Role → [palette, lightTone, darkTone]. This is the Material scheme mapping,
// extended with success/warning/info role sets.
const ROLES = {
  primary: ['primary', 40, 80],
  onPrimary: ['primary', 100, 20],
  primaryContainer: ['primary', 90, 30],
  onPrimaryContainer: ['primary', 10, 90],
  secondary: ['secondary', 40, 80],
  onSecondary: ['secondary', 100, 20],
  secondaryContainer: ['secondary', 90, 30],
  onSecondaryContainer: ['secondary', 10, 90],
  tertiary: ['tertiary', 40, 80],
  onTertiary: ['tertiary', 100, 20],
  tertiaryContainer: ['tertiary', 90, 30],
  onTertiaryContainer: ['tertiary', 10, 90],
  error: ['error', 40, 80],
  onError: ['error', 100, 20],
  errorContainer: ['error', 90, 30],
  onErrorContainer: ['error', 10, 90],
  success: ['success', 40, 80],
  onSuccess: ['success', 100, 20],
  successContainer: ['success', 90, 30],
  onSuccessContainer: ['success', 10, 90],
  warning: ['warning', 40, 80],
  onWarning: ['warning', 100, 20],
  warningContainer: ['warning', 90, 30],
  onWarningContainer: ['warning', 10, 90],
  info: ['info', 40, 80],
  onInfo: ['info', 100, 20],
  infoContainer: ['info', 90, 30],
  onInfoContainer: ['info', 10, 90],
  surface: ['neutral', 98, 6],
  surfaceDim: ['neutral', 87, 6],
  surfaceBright: ['neutral', 98, 24],
  surfaceContainerLowest: ['neutral', 100, 4],
  surfaceContainerLow: ['neutral', 96, 10],
  surfaceContainer: ['neutral', 94, 12],
  surfaceContainerHigh: ['neutral', 92, 17],
  surfaceContainerHighest: ['neutral', 90, 22],
  onSurface: ['neutral', 10, 90],
  surfaceVariant: ['neutralVariant', 90, 30],
  onSurfaceVariant: ['neutralVariant', 30, 80],
  outline: ['neutralVariant', 50, 60],
  outlineVariant: ['neutralVariant', 80, 30],
  inverseSurface: ['neutral', 20, 90],
  inverseOnSurface: ['neutral', 95, 20],
  inversePrimary: ['primary', 80, 40],
  scrim: ['neutral', 0, 0],
  shadow: ['neutral', 0, 0],
};

export const COLOR_ROLES = Object.keys(ROLES);

/** Map palettes to the semantic color roles for one scheme. */
export function makeScheme(palettes, scheme /* 'light' | 'dark' */) {
  const dark = scheme === 'dark';
  const out = {};
  for (const role of COLOR_ROLES) {
    const [palette, light, darkTone] = ROLES[role];
    out[role] = palettes[palette][dark ? darkTone : light];
  }
  return out;
}
