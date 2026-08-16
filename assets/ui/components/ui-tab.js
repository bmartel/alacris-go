// <ui-tab> — one tab inside <ui-tabs>.
//
// The host element carries the `tab` semantics (role, aria-selected,
// focusability) so <ui-tabs>' roving tabindex can manage it directly in the
// light DOM. `selected` is written by the parent <ui-tabs>; do not set it
// yourself — set the tabs' `value` instead.
//
// @prop  {string}  value=''       — REQUIRED; matched against the tabs' value
// @prop  {string}  icon=''        — leading registry icon
// @prop  {boolean} disabled=false
// @prop  {boolean} selected=false — managed by the parent <ui-tabs>
// @event ui-tab-select — internal; detail: { value } (consumed by <ui-tabs>)
// @slot  (default) — label
// @part  control — the styled tab surface
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, effect } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base, focusRingOn } from './base.js';
import { ripple } from '../motion/ripple.js';
import './ui-icon.js';

const t = vars('ui-tab', {
  height: '48px',
  padInline: sys.space(4),
  gap: sys.space(2),
  font: sys.type.titleSm,
  tracking: sys.tracking.titleSm,
  fg: sys.color.onSurfaceVariant,
  selectedFg: sys.color.primary,
});

const styles = css`
  :host {
    display: inline-flex;
    outline: none;
  }
  :host([aria-disabled="true"]) { pointer-events: none; }
  ${focusRingOn(':host')}
  :host(:focus-visible) { outline-offset: -2px; }
  .control {
    position: relative;
    isolation: isolate;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: ${t.gap};
    inline-size: 100%;
    block-size: calc(${t.height} + var(--ui-density, 0) * 4px);
    padding-inline: ${t.padInline};
    font: ${t.font};
    letter-spacing: ${t.tracking};
    color: ${t.fg};
    cursor: pointer;
    user-select: none;
    white-space: nowrap;
    transition: color ${sys.duration.short4} ${sys.easing.standard};
    --ui-icon-size: 1.5rem;
  }
  .inner {
    display: inline-flex;
    align-items: center;
    gap: inherit;
  }
  .selected { color: ${t.selectedFg}; }
  .layer {
    position: absolute; inset: 0; z-index: -1;
    background: currentColor; opacity: 0;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  .control:hover .layer { opacity: ${sys.state.hover}; }
  :host(:focus-visible) .layer { opacity: ${sys.state.focus}; }
  .control:active .layer { opacity: ${sys.state.pressed}; }
  :host([aria-disabled="true"]) .control {
    cursor: default;
    color: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContent} * 100%), transparent);
  }
`;

define('ui-tab', {
  props: { value: '', icon: '', disabled: false, selected: false },
  styles: [base, styles],
  setup({ value, icon, disabled, selected }, host) {
    host.setAttribute('role', 'tab');
    effect(() => host.setAttribute('aria-selected', selected() ? 'true' : 'false'));
    effect(() => {
      if (disabled()) host.setAttribute('aria-disabled', 'true');
      else host.removeAttribute('aria-disabled');
    });

    const select = () => {
      if (disabled()) return;
      host.emit('ui-tab-select', { value: value() });
    };
    host.addEventListener('click', select);
    host.addEventListener('keydown', (e) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        select();
      }
    });

    return html`
      <div part="control" class=${() => `control${selected() ? ' selected' : ''}`}
           ref=${(el) => ripple(el, { disabled })}>
        <span class="layer" aria-hidden="true"></span>
        <span class="inner">
          ${() => (icon() ? html`<ui-icon name=${icon}></ui-icon>` : null)}
          <slot></slot>
        </span>
      </div>`;
  },
});

export const tag = 'ui-tab';
export const themeVars = t;
