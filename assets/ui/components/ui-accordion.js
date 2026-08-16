// <ui-accordion> — groups <ui-accordion-item> panels.
//
//   <ui-accordion>
//     <ui-accordion-item value="a" headline="First">…</ui-accordion-item>
//     <ui-accordion-item value="b" headline="Second">…</ui-accordion-item>
//   </ui-accordion>
//
// @prop  {boolean} multi=false — allow several panels open at once; when
//                                false, expanding one collapses the others
// @event change — a panel was toggled by the user; detail: { value }
// @slot  (default) — <ui-accordion-item> children
// @vars  see `t` below (`themeVars.names`)
//
// Coordination rides the bubbling `ui-accordion-toggle` event from the items,
// so only user interaction triggers single-open collapse; programmatic
// `expanded` writes on an item are left alone.

import { define, html, css, vars, onCleanup } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import './ui-accordion-item.js';

const t = vars('ui-accordion', {
  dividerColor: sys.color.outlineVariant,
});

const styles = css`
  :host { display: block; }
  ::slotted(ui-accordion-item:not(:first-child)) {
    border-block-start: 1px solid ${t.dividerColor};
  }
`;

define('ui-accordion', {
  props: { multi: false },
  styles: [base, styles],
  setup({ multi }, host) {
    const onToggle = (e) => {
      const item = e.target;
      if (!multi() && e.detail?.expanded) {
        for (const other of host.querySelectorAll('ui-accordion-item')) {
          if (other !== item && other.expanded) other.expanded = false;
        }
      }
      host.emit('change', { value: e.detail?.value });
    };

    // Listen on the host: items are light-DOM children, so their bubbling
    // toggle events reach it without crossing this shadow root.
    host.addEventListener('ui-accordion-toggle', onToggle);
    onCleanup(() => host.removeEventListener('ui-accordion-toggle', onToggle));

    return html`<slot></slot>`;
  },
});

export const tag = 'ui-accordion';
export const themeVars = t;
