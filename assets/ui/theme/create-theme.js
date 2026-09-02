// createTheme — one config object in, complete token maps out.
//
// A theme is data: `{ config, palettes, common, schemes, fonts }`
// where `common` and each scheme are flat maps of token name (without the
// `--ui-` prefix) → CSS value, and `fonts` is `{ href, preset }` for the
// typeface stylesheet `applyTheme` injects. `applyTheme` turns that into one
// document-level stylesheet; nothing here touches the DOM, so themes can be
// built, diffed, serialized, or generated ahead of time.

import { makePalettes, makeScheme, COLOR_ROLES } from '../tokens/color.js';
import { typographyTokens, resolveTypography } from '../tokens/typography.js';
import {
  shapeTokens, elevationTokens, motionTokens, spacingTokens,
  stateTokens, focusTokens, zTokens, densityTokens, scrollbarTokens,
} from '../tokens/system.js';

const kebab = (s) => s.replace(/[A-Z]/g, (c) => '-' + c.toLowerCase());

/**
 * createTheme({
 *   seed: '#e8ad18',               // one color → a whole scheme, or:
 *   colors: { primary, secondary, tertiary, neutral, neutralVariant,
 *             error, success, warning, info },   // explicit palette seeds
 *   typography: 'google-sans-flex' | 'google-sans' | 'roboto' | 'system'
 *               | { preset, family, brand, plain, code, scale, load },
 *   shape: { radius },             // multiplier: 0 = square, 2 = extra round
 *   motion: { scale },             // 0 = instant, 1 = default
 *   density: 0,                    // 0 … -2 (each step: -4px control height)
 *   overrides: {                   // raw token overrides, applied last
 *     common: { 'radius-md': '10px' },
 *     light:  { 'color-primary': '#0b57d0' },
 *     dark:   { 'color-surface': '#101014' },
 *   },
 * })
 */
export function createTheme(config = {}) {
  const palettes = makePalettes(config);
  const type = resolveTypography(config.typography);

  const common = {
    ...typographyTokens(type),
    ...shapeTokens(config.shape),
    ...elevationTokens(),
    ...motionTokens(config.motion),
    ...spacingTokens(),
    ...stateTokens(),
    ...focusTokens(),
    ...zTokens(),
    ...densityTokens(config),
    ...scrollbarTokens(),
    ...(config.overrides?.common || {}),
  };

  const schemeTokens = (scheme) => {
    const roles = makeScheme(palettes, scheme);
    const out = {};
    for (const role of COLOR_ROLES) out[`color-${kebab(role)}`] = roles[role];
    // Deepen shadows on dark surfaces, keep native widgets in scheme.
    out['shadow-rgb'] = '0 0 0';
    out['color-scheme'] = scheme;
    return { ...out, ...(config.overrides?.[scheme] || {}) };
  };

  return {
    config,
    palettes,
    common,
    schemes: { light: schemeTokens('light'), dark: schemeTokens('dark') },
    fonts: { href: type.href, preset: type.preset },
  };
}
