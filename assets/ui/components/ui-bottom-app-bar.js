// <ui-bottom-app-bar> — the Material bottom app bar.
//
//   <ui-bottom-app-bar>
//     <ui-icon-button slot="navigation" icon="menu" label="Menu"></ui-icon-button>
//     <ui-icon-button slot="actions" icon="search" label="Search"></ui-icon-button>
//     <ui-fab slot="fab" icon="add"></ui-fab>
//   </ui-bottom-app-bar>
//
// A surface for a leading navigation icon, trailing actions, and an optional
// FAB. The host flows with the page — pin it with CSS when it should hug the
// viewport bottom. Distinct from <ui-bottom-nav>, which is destination tabs.
//
// @prop  {string}  label='Bottom app bar' — accessible name of the toolbar
// @prop  {string} fabAlign='end' — end | center | start — where the FAB sits
// @slot  navigation — leading icon button
// @slot  actions    — trailing icon buttons
// @slot  fab        — optional <ui-fab>
// @part  bar, navigation, actions, fab
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, computed } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';

const t = vars('ui-bottom-app-bar', {
  height: '80px',
  bg: sys.color.surfaceContainer,
  fg: sys.color.onSurface,
  shadow: sys.elevation[2],
});

const styles = css`
  :host { display: block; }
  .bar {
    position: relative;
    display: flex;
    align-items: center;
    block-size: ${t.height};
    padding-inline: ${sys.space(1)} ${sys.space(4)};
    background: ${t.bg};
    color: ${t.fg};
    box-shadow: ${t.shadow};
  }
  .nav, .actions, .fab {
    display: inline-flex;
    align-items: center;
    gap: ${sys.space(1)};
  }
  .nav { order: 0; }
  .actions { order: 1; margin-inline-start: auto; }
  .fab { order: 2; margin-inline-start: ${sys.space(2)}; }
  .start .fab { order: 1; }
  .start .actions { order: 2; }
  .center .fab {
    order: 0;
    position: absolute;
    inset-inline-start: 50%;
    translate: -50% 0;
    margin: 0;
  }
`;

define('ui-bottom-app-bar', {
  props: { label: 'Bottom app bar', fabAlign: 'end' },
  styles: [base, styles],
  setup({ label, fabAlign }) {
    const cls = computed(() => `bar ${fabAlign() === 'center' ? 'center' : fabAlign() === 'start' ? 'start' : 'end'}`);
    return html`
      <div class=${cls} part="bar" role="toolbar" aria-label=${() => label() || 'Bottom app bar'}>
        <div class="nav" part="navigation"><slot name="navigation"></slot></div>
        <div class="fab" part="fab"><slot name="fab"></slot></div>
        <div class="actions" part="actions"><slot name="actions"></slot></div>
      </div>`;
  },
});

export const tag = 'ui-bottom-app-bar';
export const themeVars = t;
