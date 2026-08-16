// <ui-progress> — Material linear progress.
//
//   <ui-progress label="Upload" value=${pct}></ui-progress>   determinate
//   <ui-progress label="Loading"></ui-progress>               indeterminate
//
// @prop  {number} value=-1 — current value; -1 (any negative) = indeterminate
// @prop  {number} max=100  — value scale
// @prop  {string} label='' — accessible name (aria-label on the progressbar)
// @part  track — the rail
// @part  bar   — the active indicator
// @vars  see `t` below (`themeVars.names`)
//
// The determinate width rides a custom property bound from the template
// (`--ui-progress-pct`), so a value change is one property write. The
// indeterminate mode is the Material two-bar translate/scale loop, written as
// CSS keyframes; its cycle length derives from the motion tokens so a theme's
// motion scale slows or stops it with everything else.

import { define, html, css, vars, computed } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';

const t = vars('ui-progress', {
  height: '4px',
  radius: sys.radius.full,
  trackBg: sys.color.surfaceContainerHighest,
  barBg: sys.color.primary,
});

const styles = css`
  :host { display: block; inline-size: 100%; }
  .track {
    position: relative;
    block-size: ${t.height};
    background: ${t.trackBg};
    border-radius: ${t.radius};
    overflow: hidden;
  }
  .bar {
    position: absolute;
    inset-block: 0;
    inset-inline-start: 0;
    inline-size: 100%;
    background: ${t.barBg};
    border-radius: inherit;
  }
  .track:not(.indeterminate) .bar1 {
    inline-size: var(--ui-progress-pct, 0%);
    transition: inline-size ${sys.duration.short4} ${sys.easing.standard};
  }
  .bar2 { display: none; }

  @keyframes ui-progress-i1 {
    0%   { transform: translateX(-100%) scaleX(0.5); }
    30%  { transform: translateX(-10%) scaleX(0.7); }
    60%  { transform: translateX(100%) scaleX(0.35); }
    100% { transform: translateX(100%) scaleX(0.35); }
  }
  @keyframes ui-progress-i2 {
    0%   { transform: translateX(-100%) scaleX(0.6); }
    40%  { transform: translateX(-100%) scaleX(0.6); }
    70%  { transform: translateX(15%) scaleX(0.5); }
    100% { transform: translateX(101%) scaleX(0.2); }
  }
  .indeterminate .bar { transform: translateX(-100%); transform-origin: left center; }
  .indeterminate .bar1 {
    animation: ui-progress-i1 calc(${sys.duration.extraLong4} * 2) ${sys.easing.linear} infinite;
  }
  .indeterminate .bar2 {
    display: block;
    animation: ui-progress-i2 calc(${sys.duration.extraLong4} * 2) ${sys.easing.linear} infinite;
  }
`;

define('ui-progress', {
  props: { value: -1, max: 100, label: '' },
  styles: [base, styles],
  setup({ value, max, label }) {
    const indeterminate = computed(() => value() < 0);
    const clamped = computed(() => Math.min(Math.max(value(), 0), max() || 100));
    // Rounded so float noise (55.00000000000001) never reaches the style.
    const pct = computed(() => Math.round((clamped() / (max() || 100)) * 10000) / 100);

    return html`
      <div part="track"
           class=${() => `track${indeterminate() ? ' indeterminate' : ''}`}
           role="progressbar"
           aria-label=${() => label() || null}
           aria-valuemin="0"
           aria-valuemax=${max}
           aria-valuenow=${() => (indeterminate() ? null : String(clamped()))}
           style=${() => (indeterminate() ? null : { '--ui-progress-pct': pct() + '%' })}>
        <div class="bar bar1" part="bar"></div>
        <div class="bar bar2" aria-hidden="true"></div>
      </div>`;
  },
});

export const tag = 'ui-progress';
export const themeVars = t;
