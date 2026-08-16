// <ui-list> — a Material list container for <ui-list-item> children.
//
// @prop  {string} label='' — accessible name for the list
// @slot  (default) — <ui-list-item> elements (and <ui-divider>s)
// @vars  --ui-list-bg, --ui-list-pad-block
//
// role="list" on the host; items carry role="listitem".

import { define, html, css, vars, effect } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';

const t = vars('ui-list', {
  bg: 'transparent',
  padBlock: sys.space(2),
});

define('ui-list', {
  props: { label: '' },
  styles: [base, css`
    :host {
      display: block;
      background: ${t.bg};
      padding-block: ${t.padBlock};
    }
  `],
  setup({ label }, host) {
    host.setAttribute('role', 'list');
    effect(() => {
      if (label()) host.setAttribute('aria-label', label());
      else host.removeAttribute('aria-label');
    });
    return html`<slot></slot>`;
  },
});

export const tag = 'ui-list';
export const themeVars = t;
