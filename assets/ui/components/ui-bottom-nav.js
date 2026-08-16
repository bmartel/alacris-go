// <ui-bottom-nav> — the Material navigation bar.
//
//   <ui-bottom-nav value=${route} @change=${(e) => route(e.detail.value)}>
//     <ui-nav-item value="home" icon="home" label="Home"></ui-nav-item>
//     <ui-nav-item value="search" icon="search" label="Search"></ui-nav-item>
//   </ui-bottom-nav>
//
// @prop  {string} value='' — the selected item's `value`
// @prop  {string} label='' — accessible name of the <nav>
// @event change — a destination was chosen; detail: { value }
// @slot  (default) — <ui-nav-item> children
// @part  bar — the <nav> container
// @vars  see `t` below (`themeVars.names`)
//
// The host is `display: block` and flows with the page — apps that want the
// bar pinned to the viewport position it themselves (position: fixed; bottom:
// 0; left: 0; right: 0). Selection reflects down: every `value` write (or
// slotchange) sets `selected` on the children. Keyboard: one Tab stop, arrow
// keys rove between items (rovingTabindex, horizontal).

import { define, html, css, vars, effect, onCleanup } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { rovingTabindex } from '../util/keys.js';
import './ui-nav-item.js';

const t = vars('ui-bottom-nav', {
  height: '80px',
  bg: sys.color.surfaceContainer,
  shadow: 'none',
});

const styles = css`
  :host { display: block; }
  .bar {
    display: flex;
    align-items: stretch;
    justify-content: space-around;
    block-size: ${t.height};
    background: ${t.bg};
    box-shadow: ${t.shadow};
  }
  ::slotted(ui-nav-item) { flex: 1; max-inline-size: 168px; }
`;

define('ui-bottom-nav', {
  props: { value: '', label: '' },
  styles: [base, styles],
  setup({ value, label }, host) {
    const roving = rovingTabindex(host, {
      selector: 'ui-nav-item',
      orientation: 'horizontal',
      skip: (el) => el.disabled,
    });
    onCleanup(() => roving.destroy());

    const sync = () => {
      for (const item of host.querySelectorAll('ui-nav-item')) {
        item.selected = item.value === value();
      }
      roving.refresh();
    };
    effect(sync); // re-runs on every `value` write; slotchange calls it too

    const onSelect = (e) => {
      const v = e.detail?.value;
      if (v === undefined || v === value()) return;
      value.set(v);
      host.emit('change', { value: v });
    };
    // Items are light-DOM children; their bubbling select events reach the host.
    host.addEventListener('ui-nav-select', onSelect);
    onCleanup(() => host.removeEventListener('ui-nav-select', onSelect));

    return html`
      <nav class="bar" part="bar" aria-label=${() => label() || null}>
        <slot @slotchange=${sync}></slot>
      </nav>`;
  },
});

export const tag = 'ui-bottom-nav';
export const themeVars = t;
