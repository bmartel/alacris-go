// <ui-breadcrumbs> — a navigation trail of slotted links.
//
//   <ui-breadcrumbs separator-icon="chevron-right">
//     <a href="/">Home</a>
//     <a href="/library">Library</a>
//     <span aria-current="page">Data</span>
//   </ui-breadcrumbs>
//
// @prop  {string} separator='/'     — separator text between items
// @prop  {string} separatorIcon=''  — registry icon name; overrides `separator`
// @prop  {string} label='Breadcrumb' — accessible name of the <nav>
// @slot  (default) — the trail items. Children MUST be elements (<a>, <span>,
//                    <ui-button variant="text">, …), never bare text nodes —
//                    separators are drawn with ::slotted(*)::before, which only
//                    exists on element children. Mark the current page with
//                    aria-current="page".
// @part  nav, list
// @vars  see `t` below (`themeVars.names`)
//
// Separators are rendered as generated content on every item but the first:
// text mode sets the `content` from a private custom property; icon mode masks
// a currentColor box with the icon's path, so it themes like any glyph.

import { define, html, css, vars, effect } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { iconPath, iconsVersion } from '../util/icons.js';

const t = vars('ui-breadcrumbs', {
  fg: sys.color.onSurfaceVariant,
  currentFg: sys.color.onSurface,
  separatorFg: sys.color.onSurfaceVariant,
  gap: sys.space(2),
  font: sys.type.bodyMd,
  tracking: sys.tracking.bodyMd,
});

const styles = css`
  :host { display: block; }
  nav {
    font: ${t.font};
    letter-spacing: ${t.tracking};
    color: ${t.fg};
  }
  .list {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: ${t.gap};
  }
  ::slotted(*) { display: inline-flex; align-items: center; color: inherit; }
  ::slotted(a) { color: inherit; text-decoration: none; }
  ::slotted(a:hover) { color: ${t.currentFg}; text-decoration: underline; }
  ::slotted([aria-current="page"]) { color: ${t.currentFg}; }
  ::slotted(*:not(:first-child))::before {
    content: var(--_ui-breadcrumbs-sep, "/");
    display: inline-block;
    flex: none;
    margin-inline-end: ${t.gap};
    color: ${t.separatorFg};
    inline-size: var(--_ui-breadcrumbs-sep-size, auto);
    block-size: var(--_ui-breadcrumbs-sep-size, auto);
    background: var(--_ui-breadcrumbs-sep-bg, none);
    -webkit-mask: var(--_ui-breadcrumbs-sep-mask, none) center / contain no-repeat;
    mask: var(--_ui-breadcrumbs-sep-mask, none) center / contain no-repeat;
  }
`;

define('ui-breadcrumbs', {
  props: { separator: '/', separatorIcon: '', label: 'Breadcrumb' },
  styles: [base, styles],
  setup({ separator, separatorIcon, label }, host) {
    effect(() => {
      iconsVersion(); // pick up icons registered after first render
      const d = separatorIcon() && iconPath(separatorIcon());
      const s = host.style;
      if (d) {
        const svgMarkup =
          `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="${d}"/></svg>`;
        s.setProperty('--_ui-breadcrumbs-sep', '""');
        s.setProperty('--_ui-breadcrumbs-sep-mask', `url("data:image/svg+xml,${encodeURIComponent(svgMarkup)}")`);
        s.setProperty('--_ui-breadcrumbs-sep-bg', 'currentColor');
        s.setProperty('--_ui-breadcrumbs-sep-size', '1em');
      } else {
        s.setProperty('--_ui-breadcrumbs-sep', JSON.stringify(separator()));
        s.removeProperty('--_ui-breadcrumbs-sep-mask');
        s.removeProperty('--_ui-breadcrumbs-sep-bg');
        s.removeProperty('--_ui-breadcrumbs-sep-size');
      }
    });

    return html`
      <nav part="nav" aria-label=${label}>
        <div class="list" part="list"><slot></slot></div>
      </nav>`;
  },
});

export const tag = 'ui-breadcrumbs';
export const themeVars = t;
