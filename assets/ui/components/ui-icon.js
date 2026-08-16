// <ui-icon> — an icon from the registry, or any slotted SVG.
//
// @prop  {string} name=''  — registry name (see util/icons.js); empty renders the slot
// @prop  {string} label='' — accessible name; empty marks the icon decorative
// @prop  {string} size=''  — CSS length; overrides --ui-icon-size for this element
// @slot  (default) — a custom <svg> when no name is given
// @vars  see `t` below (`themeVars.names`)

import { define, html, svg, css, vars, effect } from '@alacris/core';
import { base } from './base.js';
import { iconPath, iconsVersion } from '../util/icons.js';

const t = vars('ui-icon', {
  size: '1.5rem',
});

define('ui-icon', {
  props: { name: '', label: '', size: '' },
  styles: [base, css`
    :host {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      vertical-align: middle;
      inline-size: ${t.size};
      block-size: ${t.size};
      flex: none;
      color: inherit;
    }
    svg, ::slotted(svg) {
      display: block;
      inline-size: 100%;
      block-size: 100%;
      fill: currentColor;
    }
  `],
  setup({ name, label, size }, host) {
    effect(() => {
      if (label()) {
        host.setAttribute('role', 'img');
        host.setAttribute('aria-label', label());
        host.removeAttribute('aria-hidden');
      } else {
        host.removeAttribute('role');
        host.removeAttribute('aria-label');
        host.setAttribute('aria-hidden', 'true');
      }
    });
    effect(() => {
      if (size()) host.style.setProperty('--ui-icon-size', size());
      else host.style.removeProperty('--ui-icon-size');
    });
    return html`${() => {
      iconsVersion(); // re-render if icons register late
      const d = name() && iconPath(name());
      return d
        ? svg`<svg viewBox="0 0 24 24" aria-hidden="true"><path d=${d}></path></svg>`
        : html`<slot></slot>`;
    }}`;
  },
});

export const tag = 'ui-icon';
export const themeVars = t;
