// <ui-divider> — a 1px rule separating content, horizontal or vertical.
//
// Props reflect to host attributes so the styling is pure CSS on :host.
//
// @prop  {string}  orientation='horizontal' — horizontal | vertical
// @prop  {boolean} inset=false  — indented from the start edge (16px)
// @prop  {boolean} middle=false — indented from both edges
// @vars  --ui-divider-color, --ui-divider-thickness
//
// role="separator" with aria-orientation.

import { define, html, css, vars, effect } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';

const t = vars('ui-divider', {
  color: sys.color.outlineVariant,
  thickness: '1px',
});

const styles = css`
  :host {
    display: block;
    flex: none;
    align-self: stretch;
    background: ${t.color};
    block-size: ${t.thickness};
    margin: 0;
  }
  :host([inset]) { margin-inline-start: ${sys.space(4)}; }
  :host([middle]) { margin-inline: ${sys.space(4)}; }
  :host([orientation="vertical"]) {
    inline-size: ${t.thickness};
    block-size: auto;
    min-block-size: 1em;
    margin-inline: 0;
  }
  :host([orientation="vertical"][inset]) { margin-inline: 0; margin-block-start: ${sys.space(4)}; }
  :host([orientation="vertical"][middle]) { margin-inline: 0; margin-block: ${sys.space(4)}; }
`;

define('ui-divider', {
  props: { orientation: 'horizontal', inset: false, middle: false },
  styles: [base, styles],
  setup({ orientation, inset, middle }, host) {
    host.setAttribute('role', 'separator');
    effect(() => {
      host.setAttribute('orientation', orientation());
      host.setAttribute('aria-orientation', orientation());
    });
    effect(() => host.toggleAttribute('inset', inset()));
    effect(() => host.toggleAttribute('middle', middle()));
    return html``;
  },
});

export const tag = 'ui-divider';
export const themeVars = t;
