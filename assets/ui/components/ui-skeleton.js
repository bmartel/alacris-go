// <ui-skeleton> — a loading placeholder shape.
//
//   <ui-skeleton variant="circular" width="40px" height="40px"></ui-skeleton>
//   <ui-skeleton width="60%"></ui-skeleton>
//   <ui-skeleton variant="rectangular" height="120px" animation="wave"></ui-skeleton>
//
// Always aria-hidden: a skeleton is decorative. Announce loading state on the
// region that will receive the content (aria-busy), not on the placeholder.
//
// @prop  {string} variant='text'    — text | circular | rectangular
// @prop  {string} width=''          — CSS length; defaults to 100%
// @prop  {string} height=''         — CSS length; text defaults to 1em
// @prop  {string} animation='pulse' — pulse | wave | none
// @part  shape — the placeholder element
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';

const t = vars('ui-skeleton', {
  bg: `color-mix(in srgb, ${sys.color.onSurface} 11%, transparent)`,
  wave: `color-mix(in srgb, ${sys.color.onSurface} 7%, transparent)`,
  radius: sys.radius.xs,
});

const styles = css`
  :host { display: block; }
  .shape {
    position: relative;
    overflow: hidden;
    inline-size: 100%;
    background: ${t.bg};
  }
  .text {
    block-size: 1em;
    border-radius: ${t.radius};
    transform: scale(1, 0.6); /* mimic a line of text within its line-height */
    transform-origin: 0 60%;
  }
  .circular {
    border-radius: ${sys.radius.full};
    inline-size: 40px;
    block-size: 40px;
  }
  .rectangular { block-size: 1.2em; }

  @keyframes ui-skeleton-pulse {
    0%   { opacity: 1; }
    50%  { opacity: 0.5; }
    100% { opacity: 1; }
  }
  .pulse {
    animation: ui-skeleton-pulse calc(${sys.duration.extraLong4} * 2) ${sys.easing.standard} infinite;
  }

  @keyframes ui-skeleton-wave {
    0%   { transform: translateX(-100%); }
    100% { transform: translateX(100%); }
  }
  .wave::after {
    content: '';
    position: absolute;
    inset: 0;
    transform: translateX(-100%);
    background: linear-gradient(90deg, transparent, ${t.wave}, transparent);
    animation: ui-skeleton-wave calc(${sys.duration.extraLong4} * 1.6) ${sys.easing.linear} infinite;
  }
`;

define('ui-skeleton', {
  props: { variant: 'text', width: '', height: '', animation: 'pulse' },
  styles: [base, styles],
  setup({ variant, width, height, animation }, host) {
    host.setAttribute('aria-hidden', 'true');

    const cls = () =>
      `shape ${variant()}${animation() !== 'none' ? ` ${animation()}` : ''}`;
    const dims = () => ({
      width: width() || null,
      height: height() || null,
    });

    return html`<div part="shape" class=${cls} style=${dims}></div>`;
  },
});

export const tag = 'ui-skeleton';
export const themeVars = t;
