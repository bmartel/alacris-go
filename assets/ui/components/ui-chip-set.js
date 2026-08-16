// <ui-chip-set> — a wrapping row of <ui-chip>s with roving-tabindex focus.
//
// When any slotted chip is a filter chip the set is a listbox
// (aria-multiselectable mirrors `multi`); otherwise it is a plain group.
// Single-select coordination: with `multi=false`, selecting a filter chip
// deselects its siblings and the set emits `change` with the selected chip's
// `value`; deselecting the active chip emits `change` with ''.
//
// @prop  {string}  label='' — accessible name for the set
// @prop  {boolean} multi=false — allow several filter chips selected at once
// @event change — single-select mode only; detail: { value }
// @slot  (default) — <ui-chip> children
// @vars  --ui-chip-set-gap

import { define, html, css, vars, signal, effect, onCleanup } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { rovingTabindex } from '../util/keys.js';
import './ui-chip.js';

const t = vars('ui-chip-set', {
  gap: sys.space(2),
});

const styles = css`
  :host {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: ${t.gap};
  }
`;

define('ui-chip-set', {
  props: { label: '', multi: false },
  styles: [base, styles],
  setup({ label, multi }, host) {
    const hasFilter = signal(false);

    effect(() => {
      if (hasFilter()) {
        host.setAttribute('role', 'listbox');
        host.setAttribute('aria-multiselectable', String(multi()));
      } else {
        host.setAttribute('role', 'group');
        host.removeAttribute('aria-multiselectable');
      }
    });
    effect(() => {
      if (label()) host.setAttribute('aria-label', label());
      else host.removeAttribute('aria-label');
    });

    const roving = rovingTabindex(host, {
      selector: 'ui-chip',
      orientation: 'both',
      skip: (el) => !!el.disabled || el.hasAttribute('disabled'),
    });
    onCleanup(() => roving.destroy());

    const chips = () => [...host.querySelectorAll('ui-chip')];
    const chipValue = (c) => {
      const v = c.value;
      return (v === undefined || v === null || v === '') ? (c.getAttribute('value') || '') : String(v);
    };
    const scan = () => {
      hasFilter.set(chips().some((c) => (c.variant ?? c.getAttribute('variant')) === 'filter'));
      roving.refresh();
    };

    // Single-select coordination over bubbling chip `change` events.
    host.addEventListener('change', (e) => {
      const chip = e.target;
      if (!(chip instanceof Element) || chip.tagName !== 'UI-CHIP') return;
      if (multi()) return;
      e.stopPropagation();
      if (e.detail?.selected) {
        for (const c of chips()) if (c !== chip) c.selected = false;
        host.emit('change', { value: chipValue(chip) });
      } else {
        host.emit('change', { value: '' });
      }
    });

    return html`<slot ref=${(el) => { el.addEventListener('slotchange', scan); scan(); }}></slot>`;
  },
});

export const tag = 'ui-chip-set';
export const themeVars = t;
