// <ui-toggle-group> — Material segmented buttons: an outlined container that
// owns the selection of its slotted <ui-toggle-button>s.
//
//   <ui-toggle-group value=${align} @change=${(e) => align.set(e.detail.value)}>
//     <ui-toggle-button value="left">Left</ui-toggle-button>
//     <ui-toggle-button value="right">Right</ui-toggle-button>
//   </ui-toggle-group>
//
// Single-select by default (`value` is the selected string, '' for none;
// pressing the selected segment deselects it). With `multi`, `value` is an
// array (a JSON array string works as an attribute). Buttons are natural tab
// stops — no roving tabindex, per the toolbar-of-toggle-buttons pattern.
//
// @prop  {string|array} value=''   — selected value, or array when `multi`
// @prop  {boolean} multi=false     — multiple segments may be selected
// @prop  {boolean} disabled=false  — disables every segment
// @prop  {string}  label=''        — accessible name for the group
// @event change — selection changed; detail: { value } (string, or array when multi)
// @slot  (default) — the <ui-toggle-button> children
// @part  group — the outlined clipping container
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, effect, untrack } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import './ui-toggle-button.js';

const t = vars('ui-toggle-group', {
  radius: sys.radius.full,
  outlineColor: sys.color.outline,
});

const styles = css`
  :host { display: inline-flex; vertical-align: middle; }
  .group {
    display: inline-flex;
    align-items: stretch;
    gap: 1px;
    border: 1px solid ${t.outlineColor};
    background: ${t.outlineColor};
    border-radius: ${t.radius};
    overflow: hidden;
  }
  ::slotted(ui-toggle-button) { --ui-toggle-button-radius: 0; flex: 1; }
`;

define('ui-toggle-group', {
  props: { value: '', multi: false, disabled: false, label: '' },
  styles: [base, styles],
  setup({ value, multi, disabled, label }, host) {
    const buttons = () => [...host.querySelectorAll('ui-toggle-button')];

    /** Current selection, normalized to an array of values. */
    const selection = () => {
      const v = value();
      if (Array.isArray(v)) return v;
      if (typeof v === 'string' && v.trim().startsWith('[')) {
        try { return JSON.parse(v); } catch { return []; }
      }
      return v == null || v === '' ? [] : [v];
    };

    // Reflect the selection down whenever value changes …
    const sync = () => {
      const sel = selection();
      for (const b of buttons()) b.selected = sel.includes(b.value);
    };
    effect(sync);
    // … and once after children have upgraded/distributed.
    queueMicrotask(() => untrack(sync));

    // Group-level disabling: remember which children we disabled so their own
    // `disabled` survives the round trip.
    let claimed = null;
    effect(() => {
      if (disabled()) {
        if (claimed) return;
        claimed = new Set();
        for (const b of buttons()) if (!b.disabled) { b.disabled = true; claimed.add(b); }
      } else if (claimed) {
        for (const b of claimed) b.disabled = false;
        claimed = null;
      }
    });

    host.addEventListener('ui-toggle', (e) => {
      e.stopPropagation();
      if (disabled()) return;
      const v = e.detail.value;
      let next;
      if (multi()) {
        const sel = selection();
        next = sel.includes(v) ? sel.filter((x) => x !== v) : [...sel, v];
      } else {
        next = selection().includes(v) ? '' : v;
      }
      value.set(next);
      host.emit('change', { value: next });
    });

    return html`
      <div class="group" part="group" role="group"
           aria-label=${() => label() || null}
           aria-disabled=${() => (disabled() ? 'true' : null)}>
        <slot @slotchange=${() => untrack(sync)}></slot>
      </div>`;
  },
});

export const tag = 'ui-toggle-group';
export const themeVars = t;
