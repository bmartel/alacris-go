// <ui-menu-item> — one action inside <ui-menu>.
//
// The host carries the `menuitem` semantics so the menu's roving tabindex can
// manage it directly in the light DOM.
//
// @prop  {string}  value=''       — reported in the menu's `select` detail
// @prop  {string}  icon=''        — leading registry icon
// @prop  {boolean} disabled=false
// @prop  {boolean} danger=false   — error color for destructive actions
// @event ui-menu-select — internal; detail: { value } (consumed by <ui-menu>)
// @slot  (default) — label
// @slot  icon      — custom leading icon when `icon` is empty
// @slot  trailing  — trailing hint (keyboard shortcut, badge)
// @part  control — the styled item surface
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, effect } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base, focusRingOn } from './base.js';
import { ripple } from '../motion/ripple.js';
import './ui-icon.js';

const t = vars('ui-menu-item', {
  height: '48px',
  padInline: sys.space(3),
  gap: sys.space(3),
  font: sys.type.bodyLg,
  tracking: sys.tracking.bodyLg,
  fg: sys.color.onSurface,
  iconFg: sys.color.onSurfaceVariant,
  trailingFg: sys.color.onSurfaceVariant,
  dangerFg: sys.color.error,
});

const styles = css`
  :host { display: block; outline: none; }
  :host([aria-disabled="true"]) { pointer-events: none; }
  ${focusRingOn(':host')}
  :host(:focus-visible) { outline-offset: -2px; }
  .control {
    position: relative;
    isolation: isolate;
    display: flex;
    align-items: center;
    gap: ${t.gap};
    min-block-size: calc(${t.height} + var(--ui-density, 0) * 4px);
    padding-inline: ${t.padInline};
    font: ${t.font};
    letter-spacing: ${t.tracking};
    color: ${t.fg};
    cursor: pointer;
    user-select: none;
    white-space: nowrap;
    --ui-icon-size: 1.5rem;
  }
  .danger { color: ${t.dangerFg}; }
  .layer {
    position: absolute; inset: 0; z-index: -1;
    background: currentColor; opacity: 0;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  .control:hover .layer { opacity: ${sys.state.hover}; }
  :host(:focus-visible) .layer { opacity: ${sys.state.focus}; }
  .control:active .layer { opacity: ${sys.state.pressed}; }
  .leading { display: inline-flex; color: ${t.iconFg}; }
  .danger .leading { color: inherit; }
  .label { flex: 1; }
  .trailing {
    display: inline-flex;
    color: ${t.trailingFg};
    font: ${sys.type.labelMd};
    letter-spacing: ${sys.tracking.labelMd};
  }
  :host([aria-disabled="true"]) .control {
    cursor: default;
    color: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContent} * 100%), transparent);
  }
`;

define('ui-menu-item', {
  props: { value: '', icon: '', disabled: false, danger: false },
  styles: [base, styles],
  setup({ value, icon, disabled, danger }, host) {
    host.setAttribute('role', 'menuitem');
    if (!host.hasAttribute('tabindex')) host.tabIndex = -1;
    effect(() => {
      if (disabled()) host.setAttribute('aria-disabled', 'true');
      else host.removeAttribute('aria-disabled');
    });

    const select = () => {
      if (disabled()) return;
      host.emit('ui-menu-select', { value: value() });
    };
    host.addEventListener('click', select);
    host.addEventListener('keydown', (e) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        select();
      }
    });

    return html`
      <div part="control" class=${() => `control${danger() ? ' danger' : ''}`}
           ref=${(el) => ripple(el, { disabled })}>
        <span class="layer" aria-hidden="true"></span>
        <span class="leading" aria-hidden="true">
          ${() => (icon() ? html`<ui-icon name=${icon}></ui-icon>` : html`<slot name="icon"></slot>`)}
        </span>
        <span class="label"><slot></slot></span>
        <span class="trailing"><slot name="trailing"></slot></span>
      </div>`;
  },
});

export const tag = 'ui-menu-item';
export const themeVars = t;
