// <ui-loading-indicator> — the Material loading indicator: morphing dots,
// distinct from determinate <ui-progress> / <ui-spinner>.
//
//   <ui-loading-indicator label="Loading"></ui-loading-indicator>
//   <ui-loading-indicator variant="contained"></ui-loading-indicator>
//
// Always indeterminate. `contained` draws the dots on a tonal pill.
//
// @prop  {string} variant='uncontained' — uncontained | contained
// @prop  {string} label='Loading'       — accessible name
// @part  track, dot
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, computed } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';

const t = vars('ui-loading-indicator', {
  size: '10px',
  gap: sys.space(2),
  dot: sys.color.primary,
  containedBg: sys.color.secondaryContainer,
  containedPad: sys.space(3),
  containedRadius: sys.radius.full,
});

const styles = css`
  :host { display: inline-flex; vertical-align: middle; }
  .track {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: ${t.gap};
  }
  .contained {
    padding: ${t.containedPad} ${sys.space(4)};
    background: ${t.containedBg};
    border-radius: ${t.containedRadius};
  }
  .dot {
    inline-size: ${t.size};
    block-size: ${t.size};
    border-radius: ${sys.radius.full};
    background: ${t.dot};
    animation: ui-loading-dot ${sys.duration.extraLong2} ${sys.easing.standard} infinite;
  }
  .dot:nth-child(2) { animation-delay: calc(${sys.duration.short4} * 0.5); }
  .dot:nth-child(3) { animation-delay: ${sys.duration.short4}; }
  .dot:nth-child(4) { animation-delay: calc(${sys.duration.short4} * 1.5); }
  @keyframes ui-loading-dot {
    0%, 80%, 100% { transform: scale(0.55); opacity: 0.4; }
    40% { transform: scale(1); opacity: 1; }
  }
`;

define('ui-loading-indicator', {
  props: { variant: 'uncontained', label: 'Loading' },
  styles: [base, styles],
  setup({ variant, label }) {
    const cls = computed(() =>
      `track${variant() === 'contained' ? ' contained' : ''}`);
    return html`
      <div class=${cls} part="track" role="progressbar" aria-busy="true"
           aria-label=${() => label() || 'Loading'}>
        <span class="dot" part="dot" aria-hidden="true"></span>
        <span class="dot" part="dot" aria-hidden="true"></span>
        <span class="dot" part="dot" aria-hidden="true"></span>
        <span class="dot" part="dot" aria-hidden="true"></span>
      </div>`;
  },
});

export const tag = 'ui-loading-indicator';
export const themeVars = t;
