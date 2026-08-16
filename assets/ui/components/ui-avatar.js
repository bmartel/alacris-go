// <ui-avatar> — a circular avatar: image, initials, or icon.
//
// Fallback chain: `src` image → initials from `name` (first + last word) →
// `icon` → slotted content. A broken image (error event) falls back to the
// initials automatically. Initials scale with the avatar (~40% of its size).
//
// @prop  {string} src=''   — image URL (object-fit: cover)
// @prop  {string} name=''  — person's name; drives initials and the aria-label fallback
// @prop  {string} icon=''  — registry icon shown when there is no image and no name
// @prop  {string} size=''  — CSS length overriding the 40px default (sets --ui-avatar-size)
// @prop  {string} label='' — accessible name; falls back to `name`, else decorative
// @slot  (default) — custom content when src/name/icon are all empty
// @vars  --ui-avatar-size, --ui-avatar-bg, --ui-avatar-fg, --ui-avatar-radius

import { define, html, css, vars, computed, effect, signal } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import './ui-icon.js';

const t = vars('ui-avatar', {
  size: '40px',
  bg: sys.color.primaryContainer,
  fg: sys.color.onPrimaryContainer,
  radius: sys.radius.full,
});

const styles = css`
  :host {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex: none;
    inline-size: ${t.size};
    block-size: ${t.size};
    border-radius: ${t.radius};
    background: ${t.bg};
    color: ${t.fg};
    overflow: hidden;
    vertical-align: middle;
    user-select: none;
    /* em-based scaling: initials are 1em of this. */
    font-size: calc(${t.size} * 0.4);
    --ui-icon-size: calc(${t.size} * 0.6);
  }
  img {
    inline-size: 100%;
    block-size: 100%;
    object-fit: cover;
    display: block;
  }
  .initials {
    font: ${sys.type.titleMd};
    font-size: 1em;
    letter-spacing: normal;
    line-height: 1;
  }
`;

define('ui-avatar', {
  props: { src: '', name: '', icon: '', size: '', label: '' },
  styles: [base, styles],
  setup({ src, name, icon, size, label }, host) {
    const broken = signal(false);
    effect(() => { src(); broken.set(false); }); // new source, new chance

    const initials = computed(() => {
      const words = name().trim().split(/\s+/).filter(Boolean);
      if (!words.length) return '';
      const first = words[0][0];
      const last = words.length > 1 ? words[words.length - 1][0] : '';
      return (first + last).toUpperCase();
    });

    effect(() => {
      const l = label() || name();
      if (l) {
        host.setAttribute('role', 'img');
        host.setAttribute('aria-label', l);
        host.removeAttribute('aria-hidden');
      } else {
        host.removeAttribute('role');
        host.removeAttribute('aria-label');
        host.setAttribute('aria-hidden', 'true');
      }
    });

    effect(() => {
      if (size()) host.style.setProperty('--ui-avatar-size', size());
      else host.style.removeProperty('--ui-avatar-size');
    });

    return html`${() => {
      if (src() && !broken()) {
        return html`<img src=${src} alt="" @error=${() => broken.set(true)}>`;
      }
      if (initials()) return html`<span class="initials" aria-hidden="true">${initials}</span>`;
      if (icon()) return html`<ui-icon name=${icon}></ui-icon>`;
      return html`<slot></slot>`;
    }}`;
  },
});

export const tag = 'ui-avatar';
export const themeVars = t;
