// `sys` — how components reference system tokens.
//
// Every value is a `var(--ui-…)` string for interpolation into `css`
// templates. Components never write raw `var(--ui-…)` literals; going through
// `sys` keeps the token names in one place and makes a typo a visible
// `undefined` in the stylesheet instead of a silently-inert custom property.
//
//   css`:host { color: ${sys.color.onSurface}; font: ${sys.type.bodyMd}; }`

import { COLOR_ROLES } from './color.js';
import { TYPE_ROLES } from './typography.js';
import { RADIUS_KEYS, ELEVATION_LEVELS, DURATIONS, EASINGS } from './system.js';

const kebab = (s) => s.replace(/[A-Z]/g, (c) => '-' + c.toLowerCase());
const camel = (s) => s.replace(/-([a-z0-9])/g, (_, c) => c.toUpperCase());
const v = (name) => `var(--ui-${name})`;

const color = {};
for (const role of COLOR_ROLES) color[role] = v(`color-${kebab(role)}`);

const radius = {};
for (const k of RADIUS_KEYS) radius[k] = v(`radius-${k}`);

const elevation = {};
for (const lvl of ELEVATION_LEVELS) elevation[lvl] = v(`elevation-${lvl}`);

const duration = {};
for (const k in DURATIONS) duration[camel(k)] = v(`duration-${k}`);

const easing = {};
for (const k in EASINGS) easing[camel(k)] = v(`easing-${k}`);

const type = {};
const tracking = {};
for (const role of TYPE_ROLES) {
  type[camel(role)] = v(`type-${role}`);
  tracking[camel(role)] = v(`type-${role}-tracking`);
}

export const sys = {
  color,
  radius,
  elevation,
  duration,
  easing,
  type,
  tracking,
  font: { brand: v('font-brand'), plain: v('font-plain'), code: v('font-code') },
  space: (n) => v(`space-${n}`),
  state: {
    hover: v('state-hover'),
    focus: v('state-focus'),
    pressed: v('state-pressed'),
    dragged: v('state-dragged'),
    disabledContent: v('state-disabled-content'),
    disabledContainer: v('state-disabled-container'),
  },
  focus: {
    ring: v('focus-ring'),
    ringWidth: v('focus-ring-width'),
    ringOffset: v('focus-ring-offset'),
    ringColor: v('focus-ring-color'),
  },
  z: {
    appBar: v('z-app-bar'),
    drawer: v('z-drawer'),
    modal: v('z-modal'),
    popup: v('z-popup'),
    snackbar: v('z-snackbar'),
    tooltip: v('z-tooltip'),
  },
  density: 'var(--ui-density, 0)',
};

/** `font` + `letter-spacing` for a type role, ready to drop into a rule. */
export const typeRule = (role) =>
  `font: ${type[camel(role)] || type[role]}; letter-spacing: ${tracking[camel(role)] || tracking[role]};`;
