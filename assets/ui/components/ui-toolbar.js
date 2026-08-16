// <ui-toolbar> — a Material contextual toolbar: a floating strip of icon
// actions, optionally with an attached FAB.
//
//   <ui-toolbar label="Selection">
//     <ui-icon-button icon="edit" label="Edit"></ui-icon-button>
//     <ui-icon-button icon="delete" label="Delete"></ui-icon-button>
//     <ui-fab slot="fab" icon="add" size="sm"></ui-fab>
//   </ui-toolbar>
//
// @prop  {string} label='' — accessible name for the toolbar
// @slot  (default) — icon buttons and other actions
// @slot  fab       — optional <ui-fab> attached to the end
// @part  bar, actions, fab
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';

const t = vars('ui-toolbar', {
  height: '64px',
  bg: sys.color.surfaceContainer,
  fg: sys.color.onSurface,
  radius: sys.radius.full,
  pad: sys.space(2),
  elevation: sys.elevation[2],
});

const styles = css`
  :host { display: inline-flex; vertical-align: middle; }
  .bar {
    display: inline-flex;
    align-items: center;
    gap: ${sys.space(1)};
    min-block-size: ${t.height};
    padding: ${t.pad};
    background: ${t.bg};
    color: ${t.fg};
    border-radius: ${t.radius};
    box-shadow: ${t.elevation};
  }
  .actions {
    display: inline-flex;
    align-items: center;
    gap: ${sys.space(1)};
    padding-inline: ${sys.space(1)};
  }
  .fab { display: inline-flex; align-items: center; }
`;

define('ui-toolbar', {
  props: { label: '' },
  styles: [base, styles],
  setup({ label }) {
    return html`
      <div class="bar" part="bar" role="toolbar" aria-label=${() => label() || null}>
        <div class="actions" part="actions"><slot></slot></div>
        <div class="fab" part="fab"><slot name="fab"></slot></div>
      </div>`;
  },
});

export const tag = 'ui-toolbar';
export const themeVars = t;
