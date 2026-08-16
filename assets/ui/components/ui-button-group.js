// <ui-button-group> — joins slotted <ui-button>s into one segmented bar.
//
// Slotted buttons lose their own rounding (their `--ui-button-radius` is
// zeroed); the group container carries the full radius and clips, and a 1px
// gap lets the container's divider color show between segments — the Material
// connected button group.
//
// @prop  {string} label='' — accessible name for the group
// @slot  (default) — the <ui-button> children
// @part  group — the clipping container
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import './ui-button.js';

const t = vars('ui-button-group', {
  radius: sys.radius.full,
  divider: sys.color.outlineVariant,
});

const styles = css`
  :host { display: inline-flex; vertical-align: middle; }
  .group {
    display: inline-flex;
    align-items: stretch;
    gap: 1px;
    background: ${t.divider};
    border-radius: ${t.radius};
    overflow: hidden;
  }
  ::slotted(ui-button) { --ui-button-radius: 0; }
`;

define('ui-button-group', {
  props: { label: '' },
  styles: [base, styles],
  setup({ label }) {
    return html`
      <div class="group" part="group" role="group"
           aria-label=${() => label() || null}>
        <slot></slot>
      </div>`;
  },
});

export const tag = 'ui-button-group';
export const themeVars = t;
