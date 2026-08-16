// <ui-spinner> — Material circular progress.
//
//   <ui-spinner label="Loading"></ui-spinner>                 indeterminate
//   <ui-spinner label="Upload" value=${pct}></ui-spinner>     determinate
//   <ui-spinner label="Loading" size="24px"></ui-spinner>
//
// @prop  {number} value=-1 — current value; -1 (any negative) = indeterminate
// @prop  {number} max=100  — value scale
// @prop  {string} size=''  — CSS length; overrides --ui-spinner-size (default 48px)
// @prop  {string} label='' — accessible name (aria-label on the progressbar)
// @part  progress — the progressbar wrapper
// @vars  see `t` below (`themeVars.names`)
//
// One SVG circle in a 44-unit viewBox with a 4-unit stroke, so the stroke
// scales with `size`. Determinate progress binds stroke-dashoffset from the
// value; indeterminate is the classic rotate + dash-grow loop as CSS
// keyframes, its cycle derived from the motion tokens.

import { define, html, svg, css, vars, computed, effect } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';

const t = vars('ui-spinner', {
  size: '48px',
  color: sys.color.primary,
  trackColor: 'transparent',
});

// r=20 in a 44-unit viewBox; the arc length everything is dashed against.
const R = 20;
const C = 2 * Math.PI * R; // ≈ 125.66

const styles = css`
  :host {
    display: inline-flex;
    inline-size: ${t.size};
    block-size: ${t.size};
    flex: none;
    color: ${t.color};
  }
  .progress, svg { inline-size: 100%; block-size: 100%; }
  svg { transform: rotate(-90deg); } /* start determinate sweep at 12 o'clock */
  circle { fill: none; stroke-width: 4; }
  .track { stroke: ${t.trackColor}; }
  .arc {
    stroke: currentColor;
    stroke-linecap: round;
    transition: stroke-dashoffset ${sys.duration.short4} ${sys.easing.standard};
  }

  @keyframes ui-spinner-rotate {
    100% { transform: rotate(360deg); }
  }
  @keyframes ui-spinner-dash {
    0%   { stroke-dasharray: 1, 200; stroke-dashoffset: 0; }
    50%  { stroke-dasharray: 90, 200; stroke-dashoffset: -35; }
    100% { stroke-dasharray: 90, 200; stroke-dashoffset: -124; }
  }
  .indeterminate svg {
    transform: none;
    animation: ui-spinner-rotate calc(${sys.duration.extraLong4} * 1.4) ${sys.easing.linear} infinite;
  }
  .indeterminate .arc {
    animation: ui-spinner-dash calc(${sys.duration.extraLong4} * 1.4) ${sys.easing.standard} infinite;
  }
`;

define('ui-spinner', {
  props: { value: -1, max: 100, size: '', label: '' },
  styles: [base, styles],
  setup({ value, max, size, label }, host) {
    const indeterminate = computed(() => value() < 0);
    const clamped = computed(() => Math.min(Math.max(value(), 0), max() || 100));
    const offset = computed(() => C * (1 - clamped() / (max() || 100)));

    effect(() => {
      if (size()) host.style.setProperty('--ui-spinner-size', size());
      else host.style.removeProperty('--ui-spinner-size');
    });

    return html`
      <div part="progress"
           class=${() => `progress${indeterminate() ? ' indeterminate' : ''}`}
           role="progressbar"
           aria-label=${() => label() || null}
           aria-valuemin="0"
           aria-valuemax=${max}
           aria-valuenow=${() => (indeterminate() ? null : String(clamped()))}>
        ${svg`
          <svg viewBox="0 0 44 44" aria-hidden="true">
            <circle class="track" cx="22" cy="22" r="20"></circle>
            <circle class="arc" cx="22" cy="22" r="20"
                    stroke-dasharray=${() => (indeterminate() ? null : `${C}`)}
                    stroke-dashoffset=${() => (indeterminate() ? null : String(offset()))}></circle>
          </svg>`}
      </div>`;
  },
});

export const tag = 'ui-spinner';
export const themeVars = t;
