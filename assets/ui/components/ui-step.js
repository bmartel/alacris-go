// <ui-step> — one step inside <ui-stepper>.
//
// @prop  {string} label=''
// @prop  {number} index=0            — 0-based position; assigned by <ui-stepper>
// @prop  {string} state='upcoming'   — upcoming | active | completed; assigned
//                                      by <ui-stepper> from its `active` prop
// @prop  {string} optionalText=''    — small secondary line under the label
// @part  indicator — the 24px circle
// @part  label
// @vars  see `t` below (`themeVars.names`)
//
// Presentation only: the circle shows the 1-based number (or a check when
// completed); <ui-stepper> owns index/state assignment and the connectors.

import { define, html, css, vars } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import './ui-icon.js';

const t = vars('ui-step', {
  indicatorSize: '24px',
  bg: sys.color.surfaceContainerHighest,
  fg: sys.color.onSurfaceVariant,
  activeBg: sys.color.primary,
  activeFg: sys.color.onPrimary,
  labelFg: sys.color.onSurfaceVariant,
  labelFgActive: sys.color.onSurface,
  font: sys.type.titleSm,
  tracking: sys.tracking.titleSm,
  optionalFont: sys.type.bodySm,
  optionalTracking: sys.tracking.bodySm,
});

const styles = css`
  :host {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: ${sys.space(2)};
    padding-inline: ${sys.space(2)};
    text-align: center;
  }
  .indicator {
    display: grid;
    place-items: center;
    inline-size: ${t.indicatorSize};
    block-size: ${t.indicatorSize};
    border-radius: ${sys.radius.full};
    background: ${t.bg};
    color: ${t.fg};
    font: ${sys.type.labelMd};
    letter-spacing: ${sys.tracking.labelMd};
    --ui-icon-size: 16px;
    transition: background-color ${sys.duration.short4} ${sys.easing.standard},
                color ${sys.duration.short4} ${sys.easing.standard};
  }
  .indicator.active, .indicator.completed {
    background: ${t.activeBg};
    color: ${t.activeFg};
  }
  .label {
    font: ${t.font};
    letter-spacing: ${t.tracking};
    color: ${t.labelFg};
    transition: color ${sys.duration.short4} ${sys.easing.standard};
  }
  .label.active { color: ${t.labelFgActive}; }
  .optional {
    font: ${t.optionalFont};
    letter-spacing: ${t.optionalTracking};
    color: ${t.labelFg};
  }
`;

define('ui-step', {
  props: { label: '', index: 0, state: 'upcoming', optionalText: '' },
  styles: [base, styles],
  setup({ label, index, state, optionalText }) {
    return html`
      <span part="indicator" class=${() => `indicator ${state()}`} aria-hidden="true">
        ${() => (state() === 'completed'
          ? html`<ui-icon name="check"></ui-icon>`
          : html`${() => index() + 1}`)}
      </span>
      <span part="label" class=${() => `label ${state()}`}>${label}</span>
      ${() => (optionalText() ? html`<span class="optional">${optionalText}</span>` : null)}`;
  },
});

export const tag = 'ui-step';
export const themeVars = t;
