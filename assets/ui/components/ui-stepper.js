// <ui-stepper> — a horizontal stepper of <ui-step> children.
//
//   <ui-stepper active=${active}>
//     <ui-step label="Cart"></ui-step>
//     <ui-step label="Shipping" optional-text="Optional"></ui-step>
//     <ui-step label="Payment"></ui-step>
//   </ui-stepper>
//
// @prop  {number} active=0 — index of the active step; steps before it become
//                            `completed`, steps after it `upcoming`
// @slot  (default) — <ui-step> children
// @part  row — the flex row wrapping the slot
// @vars  see `t` below (`themeVars.names`)
//
// Presentation only — it emits nothing; wiring next/back navigation to
// `active` is application logic. On slotchange (and every `active` write) it
// assigns each child's `index`, `state`, a `data-state` attribute (used by the
// connector CSS), `role="listitem"`, and `aria-current="step"` on the active
// step. Connector lines between steps turn primary once the segment before a
// step is completed.

import { define, html, css, vars, effect } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import './ui-step.js';

const t = vars('ui-stepper', {
  connector: sys.color.outlineVariant,
  connectorActive: sys.color.primary,
});

const styles = css`
  :host { display: block; }
  .row {
    display: flex;
    align-items: flex-start;
  }
  ::slotted(ui-step) {
    flex: 1;
    min-inline-size: 0;
    position: relative;
  }
  ::slotted(ui-step:not(:first-child))::before {
    content: "";
    position: absolute;
    inset-block-start: 12px;
    inset-inline-start: calc(-50% + 20px);
    inset-inline-end: calc(50% + 20px);
    block-size: 1px;
    background: ${t.connector};
    transition: background-color ${sys.duration.short4} ${sys.easing.standard};
  }
  ::slotted(ui-step[data-state="completed"]:not(:first-child))::before,
  ::slotted(ui-step[data-state="active"]:not(:first-child))::before {
    background: ${t.connectorActive};
  }
`;

define('ui-stepper', {
  props: { active: 0 },
  styles: [base, styles],
  setup({ active }, host) {
    const sync = () => {
      const a = active();
      const steps = host.querySelectorAll('ui-step');
      let i = 0;
      for (const el of steps) {
        const s = i < a ? 'completed' : i === a ? 'active' : 'upcoming';
        el.index = i;
        el.state = s;
        el.setAttribute('data-state', s);
        el.setAttribute('role', 'listitem');
        if (s === 'active') el.setAttribute('aria-current', 'step');
        else el.removeAttribute('aria-current');
        i++;
      }
    };

    effect(sync); // re-runs on every `active` write

    return html`<div class="row" part="row" role="list"><slot @slotchange=${sync}></slot></div>`;
  },
});

export const tag = 'ui-stepper';
export const themeVars = t;
