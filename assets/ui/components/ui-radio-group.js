// <ui-radio-group> — owns a set of slotted <ui-radio>s: one selected value,
// one tab stop, arrow keys move AND select (ARIA APG radio group pattern).
//
//   <ui-radio-group name="size" value=${size} @change=${(e) => size.set(e.detail.value)}>
//     <ui-radio value="s" label="Small"></ui-radio>
//     <ui-radio value="m" label="Medium"></ui-radio>
//   </ui-radio-group>
//
// @prop  {string}  value=''
// @prop  {string}  name=''  — form participation (submits `value`)
// @prop  {string}  label='' — accessible name for the group
// @prop  {boolean} disabled=false — disables every radio
// @prop  {string}  orientation='vertical' — vertical | horizontal (layout only;
//        arrows work on both axes per the APG)
// @event change — detail: { value }
// @slot  (default) — the <ui-radio> children
// @part  group — the layout container
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, effect, onCleanup, untrack } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { rovingTabindex } from '../util/keys.js';
import { formBind } from '../util/form.js';
import './ui-radio.js';

const t = vars('ui-radio-group', {
  gap: sys.space(1),
});

const styles = css`
  :host { display: inline-flex; }
  .group { display: flex; flex-direction: column; gap: ${t.gap}; }
  :host([orientation="horizontal"]) .group { flex-direction: row; gap: ${sys.space(4)}; }
`;

define('ui-radio-group', {
  formAssociated: true,
  props: { value: '', name: '', label: '', disabled: false, orientation: 'vertical' },
  styles: [base, styles],
  setup({ value, name, label, disabled }, host) {
    formBind(host, { name, value, disabled });

    const radios = () => [...host.querySelectorAll('ui-radio')];

    // Reflect the value down: check the matching radio, give it the tab stop.
    const sync = () => {
      const v = value();
      const list = radios();
      const stop = list.find((r) => r.value === v && !r.disabled) || list.find((r) => !r.disabled);
      for (const r of list) {
        r.checked = r.value === v;
        r.tabIndex = r === stop ? 0 : -1;
      }
    };
    effect(sync);
    // … and once after children have upgraded/distributed.
    queueMicrotask(() => untrack(sync));

    // Group-level disabling with remember-restore, so a radio's own
    // `disabled` survives the group toggling.
    let claimed = null;
    effect(() => {
      if (disabled()) {
        if (claimed) return;
        claimed = new Set();
        for (const r of radios()) if (!r.disabled) { r.disabled = true; claimed.add(r); }
      } else if (claimed) {
        for (const r of claimed) r.disabled = false;
        claimed = null;
      }
    });

    const select = (v) => {
      if (disabled() || v === value()) return;
      value.set(v);
      host.emit('change', { value: v });
    };

    // Per the APG, moving with arrows also selects the focused radio.
    const roving = rovingTabindex(host, {
      selector: 'ui-radio',
      orientation: 'both',
      skip: (el) => el.disabled,
      onMove: (el) => select(el.value),
    });
    onCleanup(() => roving.destroy());

    host.addEventListener('ui-radio-select', (e) => {
      e.stopPropagation();
      select(e.detail.value);
    });

    return html`
      <div class="group" part="group" role="radiogroup"
           aria-label=${() => label() || null}
           aria-disabled=${() => (disabled() ? 'true' : null)}>
        <slot @slotchange=${() => untrack(sync)}></slot>
      </div>`;
  },
});

export const tag = 'ui-radio-group';
export const themeVars = t;
