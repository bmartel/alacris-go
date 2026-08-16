// Non-color, non-type system tokens: shape, elevation, motion, spacing,
// state layers, focus ring, z-index. Each builder returns a flat map of
// token name (without the leading `--ui-`) → value.

// ------------------------------------------------------------------- shape

const RADII = { none: '0', xs: 4, sm: 8, md: 12, lg: 16, xl: 28, full: 9999 };
export const RADIUS_KEYS = Object.keys(RADII);

/** config: { radius: multiplier } — `radius: 0` squares every corner. */
export function shapeTokens({ radius = 1 } = {}) {
  const out = {};
  for (const k of RADIUS_KEYS) {
    const v = RADII[k];
    out[`radius-${k}`] = v === '0' ? '0' : k === 'full' ? '9999px' : `${Math.round(v * radius)}px`;
  }
  return out;
}

// --------------------------------------------------------------- elevation

// Material's five elevation levels as key+ambient shadow pairs, built on a
// themable shadow color channel so dark schemes can deepen them.
export const ELEVATION_LEVELS = [0, 1, 2, 3, 4, 5];

export function elevationTokens() {
  const c1 = 'rgb(var(--ui-shadow-rgb) / 0.3)';
  const c2 = 'rgb(var(--ui-shadow-rgb) / 0.15)';
  return {
    'shadow-rgb': '0 0 0',
    'elevation-0': 'none',
    'elevation-1': `0 1px 2px ${c1}, 0 1px 3px 1px ${c2}`,
    'elevation-2': `0 1px 2px ${c1}, 0 2px 6px 2px ${c2}`,
    'elevation-3': `0 1px 3px ${c1}, 0 4px 8px 3px ${c2}`,
    'elevation-4': `0 2px 3px ${c1}, 0 6px 10px 4px ${c2}`,
    'elevation-5': `0 4px 4px ${c1}, 0 8px 12px 6px ${c2}`,
  };
}

// ------------------------------------------------------------------ motion

export const DURATIONS = {
  'short-1': 50, 'short-2': 100, 'short-3': 150, 'short-4': 200,
  'medium-1': 250, 'medium-2': 300, 'medium-3': 350, 'medium-4': 400,
  'long-1': 450, 'long-2': 500, 'long-3': 550, 'long-4': 600,
  'extra-long-1': 700, 'extra-long-2': 800, 'extra-long-3': 900, 'extra-long-4': 1000,
};

export const EASINGS = {
  standard: 'cubic-bezier(0.2, 0, 0, 1)',
  'standard-accelerate': 'cubic-bezier(0.3, 0, 1, 1)',
  'standard-decelerate': 'cubic-bezier(0, 0, 0, 1)',
  emphasized: 'cubic-bezier(0.2, 0, 0, 1)',
  'emphasized-accelerate': 'cubic-bezier(0.3, 0, 0.8, 0.15)',
  'emphasized-decelerate': 'cubic-bezier(0.05, 0.7, 0.1, 1)',
  // Switch-handle overshoot — the MD3 track/handle settle (not a generic curve).
  'emphasized-overshoot': 'cubic-bezier(0.175, 0.885, 0.32, 1.275)',
  linear: 'linear',
};

/** config: { scale } — 0 disables durations, 2 doubles them. */
export function motionTokens({ scale = 1 } = {}) {
  const out = {};
  for (const k in DURATIONS) out[`duration-${k}`] = `${Math.round(DURATIONS[k] * scale)}ms`;
  for (const k in EASINGS) out[`easing-${k}`] = EASINGS[k];
  return out;
}

// ----------------------------------------------------------------- spacing

export const SPACE_STEPS = [1, 2, 3, 4, 5, 6, 7, 8, 10, 12, 14, 16, 18, 20, 24];

export function spacingTokens() {
  const out = {};
  for (const s of SPACE_STEPS) out[`space-${s}`] = `${s * 4}px`;
  return out;
}

// ---------------------------------------------------- state / focus / z

export function stateTokens() {
  return {
    'state-hover': '0.08',
    'state-focus': '0.1',
    'state-pressed': '0.1',
    'state-dragged': '0.16',
    'state-disabled-content': '0.38',
    'state-disabled-container': '0.12',
  };
}

export function focusTokens() {
  return {
    'focus-ring-width': '2px',
    'focus-ring-offset': '2px',
    'focus-ring-color': 'var(--ui-color-primary)',
    'focus-ring': 'var(--ui-focus-ring-width) solid var(--ui-focus-ring-color)',
  };
}

export function zTokens() {
  return {
    'z-app-bar': '1100',
    'z-drawer': '1200',
    'z-modal': '1300',
    'z-snackbar': '1400',
    'z-tooltip': '1500',
  };
}

/** config: { density } — 0 (default) … -2; each step removes 4px of control height. */
export function densityTokens({ density = 0 } = {}) {
  return { density: String(Math.max(-2, Math.min(0, density))) };
}
