// <ui-icon> — an icon from the registry, or any slotted SVG.
//
// Names are kebab-case (`arrow-forward`). Underscores are accepted
// (`arrow_forward`). `iconNames()` lists the built-in set; apps add more
// with `registerIcons({ name: 'M…' })`. An unknown name logs a warning
// once and renders a placeholder instead of an empty hole.
//
// @prop  {string} name=''  — registry name; empty renders the slot
// @prop  {string} label='' — accessible name; empty marks the icon decorative
// @prop  {string} size=''  — CSS length; overrides --ui-icon-size for this element
// @slot  (default) — a custom <svg> when no name is given
// @vars  see `t` below (`themeVars.names`)

import { define, html, svg, css, vars, effect } from '@alacris/core';
import { base } from './base.js';
import { iconPath, iconsVersion, warnUnknownIcon } from '../util/icons.js';

// Hollow square — obviously not a real glyph, so a typo is visible.
const MISSING = 'M3 3v18h18V3H3zm2 2h14v14H5V5z';

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
      const n = name();
      if (!n) return html`<slot></slot>`;
      const d = iconPath(n);
      if (!d) warnUnknownIcon(n);
      return svg`<svg viewBox="0 0 24 24" aria-hidden="true" data-icon=${d ? n : 'unknown'}><path d=${d || MISSING}></path></svg>`;
    }}`;
  },
});

export const tag = 'ui-icon';
export const themeVars = t;
