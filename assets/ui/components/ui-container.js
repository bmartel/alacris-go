// <ui-container> — a centered max-width content wrapper.
//
//   <ui-container size="lg">…</ui-container>
//
// @prop  {string}  size='md'    — sm (640px) | md (960px) | lg (1280px) |
//                                 xl (1536px) | full (no max width)
// @prop  {boolean} gutters=true — inline padding: space(6), space(4) under 600px
// @slot  (default)
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, effect } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';

const t = vars('ui-container', {
  sm: '640px',
  md: '960px',
  lg: '1280px',
  xl: '1536px',
  gutter: sys.space(6),
  gutterNarrow: sys.space(4),
});

const styles = css`
  :host {
    display: block;
    inline-size: 100%;
    margin-inline: auto;
    max-inline-size: var(--_ui-container-max, none);
    padding-inline: var(--_ui-container-pad, 0);
  }
  @media (max-width: 600px) {
    :host { padding-inline: var(--_ui-container-pad-narrow, 0); }
  }
`;

define('ui-container', {
  props: { size: 'md', gutters: true },
  styles: [base, styles],
  setup({ size, gutters }, host) {
    const max = { sm: t.sm, md: t.md, lg: t.lg, xl: t.xl, full: 'none' };
    effect(() => {
      host.style.setProperty('--_ui-container-max', max[size()] || t.md);
    });
    effect(() => {
      if (gutters()) {
        host.style.setProperty('--_ui-container-pad', t.gutter);
        host.style.setProperty('--_ui-container-pad-narrow', t.gutterNarrow);
      } else {
        host.style.removeProperty('--_ui-container-pad');
        host.style.removeProperty('--_ui-container-pad-narrow');
      }
    });
    return html`<slot></slot>`;
  },
});

export const tag = 'ui-container';
export const themeVars = t;
