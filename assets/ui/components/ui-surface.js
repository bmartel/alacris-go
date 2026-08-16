// <ui-surface> — a themable surface (the Paper equivalent).
//
//   <ui-surface elevation="2" radius="lg" bg="surface-container">…</ui-surface>
//
// @prop  {number}  elevation=0 — shadow level 0..5 (--ui-elevation-N)
// @prop  {string}  radius='md' — none | xs | sm | md | lg | xl | full
// @prop  {string}  bg='surface' — any color role name, kebab or camel
//                                 ('primary-container', 'surfaceContainerHigh');
//                                 text color pairs automatically with the
//                                 role's on- counterpart when one exists,
//                                 falling back to on-surface
// @prop  {boolean} outlined=false — 1px outline-variant border
// @slot  (default)
// @vars  (none — every value resolves through the system tokens)

import { define, html, css, effect } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';

const camel = (s) => s.replace(/-([a-z0-9])/g, (_, c) => c.toUpperCase());

define('ui-surface', {
  props: { elevation: 0, radius: 'md', bg: 'surface', outlined: false },
  styles: [base, css`
    :host {
      display: block;
      transition: box-shadow ${sys.duration.short4} ${sys.easing.standard};
    }
  `],
  setup({ elevation, radius, bg, outlined }, host) {
    effect(() => {
      const role = camel(bg());
      host.style.background = sys.color[role] || sys.color.surface;
      const on = sys.color['on' + role[0].toUpperCase() + role.slice(1)];
      host.style.color = on || sys.color.onSurface;
    });
    effect(() => {
      const lvl = Math.max(0, Math.min(5, Math.round(Number(elevation()) || 0)));
      host.style.boxShadow = sys.elevation[lvl];
    });
    effect(() => {
      host.style.borderRadius = sys.radius[radius()] || sys.radius.md;
    });
    effect(() => {
      host.style.border = outlined() ? `1px solid ${sys.color.outlineVariant}` : '';
    });
    return html`<slot></slot>`;
  },
});

export const tag = 'ui-surface';
