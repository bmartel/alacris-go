// <ui-nav-rail> — the Material navigation rail.
//
//   <ui-nav-rail value=${route} @change=${(e) => route(e.detail.value)}>
//     <ui-fab slot="fab" icon="add"></ui-fab>
//     <ui-nav-item value="home" icon="home" label="Home"></ui-nav-item>
//     <ui-nav-item value="search" icon="search" label="Search"></ui-nav-item>
//   </ui-nav-rail>
//
// A compact vertical destination list (the large-screen counterpart of
// <ui-bottom-nav>). Reuses <ui-nav-item>. The host flows with the page —
// pin it with position: sticky/fixed yourself.
//
// @prop  {string} value='' — the selected item's `value`
// @prop  {string} label='' — accessible name of the <nav>
// @prop  {string} align='start' — start | center | end (where destinations sit)
// @event change — a destination was chosen; detail: { value }
// @slot  menu — optional leading icon button (typically "menu")
// @slot  fab  — optional FAB above the destinations
// @slot  (default) — <ui-nav-item> children
// @part  rail — the <nav> container
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, effect, onCleanup } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { rovingTabindex } from '../util/keys.js';
import './ui-nav-item.js';

const t = vars('ui-nav-rail', {
  width: '80px',
  bg: sys.color.surface,
  pad: sys.space(3),
});

const styles = css`
  :host { display: block; inline-size: ${t.width}; block-size: 100%; }
  .rail {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: ${sys.space(3)};
    box-sizing: border-box;
    inline-size: ${t.width};
    min-block-size: 100%;
    padding-block: ${t.pad};
    background: ${t.bg};
  }
  .lead, .fab { display: flex; flex-direction: column; align-items: center; gap: ${sys.space(2)}; }
  .lead:not(.has), .fab:not(.has) { display: none; }
  .destinations {
    display: flex;
    flex-direction: column;
    align-items: stretch;
    gap: ${sys.space(1)};
    inline-size: 100%;
    flex: 1;
  }
  .destinations.start { justify-content: flex-start; }
  .destinations.center { justify-content: center; }
  .destinations.end { justify-content: flex-end; }
  ::slotted(ui-nav-item) { inline-size: 100%; }
`;

define('ui-nav-rail', {
  props: { value: '', label: '', align: 'start' },
  styles: [base, styles],
  setup({ value, label, align }, host) {
    const roving = rovingTabindex(host, {
      selector: 'ui-nav-item',
      orientation: 'vertical',
      skip: (el) => el.disabled,
    });
    onCleanup(() => roving.destroy());

    const sync = () => {
      for (const item of host.querySelectorAll('ui-nav-item')) {
        item.selected = item.value === value();
      }
      roving.refresh();
    };
    effect(sync);

    const onSelect = (e) => {
      const v = e.detail?.value;
      if (v === undefined || v === value()) return;
      value.set(v);
      host.emit('change', { value: v });
    };
    host.addEventListener('ui-nav-select', onSelect);
    onCleanup(() => host.removeEventListener('ui-nav-select', onSelect));

    const hasSlot = (el, set) =>
      el.addEventListener('slotchange', () => set(el.assignedElements().length > 0));

    return html`
      <nav class="rail" part="rail" aria-label=${() => label() || null}>
        <div class="lead" ref=${(el) => hasSlot(el.querySelector('slot'), (has) => el.classList.toggle('has', has))}>
          <slot name="menu"></slot>
        </div>
        <div class="fab" ref=${(el) => hasSlot(el.querySelector('slot'), (has) => el.classList.toggle('has', has))}>
          <slot name="fab"></slot>
        </div>
        <div class=${() => `destinations ${align() || 'start'}`}>
          <slot @slotchange=${sync}></slot>
        </div>
      </nav>`;
  },
});

export const tag = 'ui-nav-rail';
export const themeVars = t;
