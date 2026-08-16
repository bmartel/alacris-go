// <ui-tabs> — Material tabs: a tab bar with an animated active indicator.
//
//   <ui-tabs value="one" label="Demo tabs" @change=${(e) => ...}>
//     <ui-tab value="one" icon="home">One</ui-tab>
//     <ui-tab value="two">Two</ui-tab>
//     <ui-tab-panel slot="panels" value="one">…</ui-tab-panel>
//     <ui-tab-panel slot="panels" value="two">…</ui-tab-panel>
//   </ui-tabs>
//
// Tabs go in the default slot, panels in the "panels" slot. The parent
// reflects `selected` onto each <ui-tab> and `active` onto each
// <ui-tab-panel>, and wires aria-controls/aria-labelledby ids between them.
// Activation is AUTOMATIC per the ARIA APG: arrow keys move focus AND select
// (one Tab stop for the whole bar via roving tabindex).
//
// @prop  {string} value='' — the selected tab's value
// @prop  {string} variant='primary' — primary | secondary
// @prop  {string} label='' — accessible name for the tablist
// @event change — user selected a tab; detail: { value }
// @slot  (default) — <ui-tab> children
// @slot  panels    — <ui-tab-panel> children
// @part  tablist, indicator
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, signal, effect, onCleanup } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { rovingTabindex } from '../util/keys.js';
import './ui-tab.js';
import './ui-tab-panel.js';

const t = vars('ui-tabs', {
  indicator: sys.color.primary,
  indicatorHeight: '3px',
  divider: sys.color.outlineVariant,
});

const styles = css`
  :host { display: block; position: relative; }
  .tablist {
    position: relative;
    display: flex;
    align-items: stretch;
    border-block-end: 1px solid ${t.divider};
  }
  .indicator {
    position: absolute;
    inset-block-end: 0;
    inset-inline-start: 0;
    block-size: ${t.indicatorHeight};
    background: ${t.indicator};
    border-start-start-radius: ${t.indicatorHeight};
    border-start-end-radius: ${t.indicatorHeight};
    pointer-events: none;
    transition: transform ${sys.duration.medium2} ${sys.easing.emphasized},
                width ${sys.duration.medium2} ${sys.easing.emphasized},
                opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  .secondary .indicator {
    block-size: 2px;
    border-radius: 0;
    background: ${sys.color.onSurface};
  }
`;

let uid = 0;

define('ui-tabs', {
  props: { value: '', variant: 'primary', label: '' },
  styles: [base, styles],
  setup({ value, variant, label }, host) {
    const id = ++uid;
    const rev = signal(0); // bumped on slotchange so syncs re-run
    const bump = () => rev.update((n) => n + 1);

    const tabsOf = () => [...host.querySelectorAll('ui-tab')];
    const panelsOf = () => [...host.querySelectorAll('ui-tab-panel')];

    const select = (v) => {
      if (v === undefined || v === value.peek()) return;
      value.set(v);
      host.emit('change', { value: v });
    };

    // Selection made by a tab (click / Enter / Space).
    host.addEventListener('ui-tab-select', (e) => {
      e.stopPropagation();
      select(e.detail.value);
    });

    // Automatic activation: arrows move focus and select.
    const roving = rovingTabindex(host, {
      selector: 'ui-tab',
      orientation: 'horizontal',
      skip: (el) => el.disabled,
      onMove: (el) => select(el.value),
    });
    onCleanup(() => roving.destroy());

    // Reflect selection down and keep the ARIA wiring current.
    effect(() => {
      const v = value();
      rev();
      const tabs = tabsOf();
      const panels = panelsOf();
      let hasStop = false;
      for (const tab of tabs) {
        const on = tab.value === v;
        tab.selected = on;
        tab.tabIndex = on && !tab.disabled ? 0 : -1;
        if (on && !tab.disabled) hasStop = true;
        if (!tab.id) tab.id = `ui-tab-${id}-${tab.value}`;
      }
      if (!hasStop) {
        const first = tabs.find((el) => !el.disabled);
        if (first) first.tabIndex = 0;
      }
      for (const panel of panels) {
        panel.active = panel.value === v;
        if (!panel.id) panel.id = `ui-tab-panel-${id}-${panel.value}`;
        const tab = tabs.find((el) => el.value === panel.value);
        if (tab) {
          tab.setAttribute('aria-controls', panel.id);
          panel.setAttribute('aria-labelledby', tab.id);
        }
      }
    });

    // Animated indicator: measure the selected tab in the host's coordinate
    // space. Zero sizes (simulated DOM, display:none) hide the bar instead of
    // animating to garbage.
    const ix = signal(0);
    const iw = signal(0);
    let tablistEl = null;
    const measure = () => {
      const tab = tabsOf().find((el) => el.selected);
      if (!tab || !tablistEl) { iw.set(0); return; }
      if (variant.peek() === 'primary') {
        const inner = tab.shadowRoot?.querySelector('.inner');
        const listRect = tablistEl.getBoundingClientRect();
        const r = (inner || tab).getBoundingClientRect();
        if (!r.width) { iw.set(0); return; }
        ix.set(r.left - listRect.left);
        iw.set(r.width);
      } else {
        const w = tab.offsetWidth || 0;
        if (!w) { iw.set(0); return; }
        ix.set(tab.offsetLeft || 0);
        iw.set(w);
      }
    };
    effect(() => {
      value();
      rev();
      queueMicrotask(measure);
    });
    effect(() => {
      window.addEventListener('resize', measure, { passive: true });
      return () => window.removeEventListener('resize', measure);
    });

    return html`
      <div class=${() => `tablist ${variant()}`} part="tablist" role="tablist" aria-label=${() => label() || null}
           ref=${(el) => (tablistEl = el)}>
        <slot @slotchange=${bump}></slot>
        <span class="indicator" part="indicator" aria-hidden="true"
              style=${() => ({
                transform: `translateX(${ix()}px)`,
                width: `${iw()}px`,
                opacity: iw() ? '1' : '0',
              })}></span>
      </div>
      <slot name="panels" @slotchange=${bump}></slot>`;
  },
});

export const tag = 'ui-tabs';
export const themeVars = t;
