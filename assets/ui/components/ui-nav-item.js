// <ui-nav-item> — one destination inside <ui-bottom-nav> or <ui-nav-rail>.
//
// @prop  {string}  value=''      — REQUIRED identity of the destination
// @prop  {string}  icon=''       — registry icon name
// @prop  {string}  activeIcon='' — icon while selected (defaults to `icon`)
// @prop  {string}  label=''      — visible label (and the accessible name)
// @prop  {boolean} selected=false — managed by <ui-bottom-nav>
// @prop  {boolean} disabled=false
// @event ui-nav-select — activated; detail: { value }
// @slot  icon — custom icon content when `icon` is empty
// @part  control — the <button>
// @part  pill    — the 56×32 icon container
// @vars  see `t` below (`themeVars.names`)
//
// Focus: the host is the roving tab stop (<ui-bottom-nav> assigns tabindex);
// focus is forwarded to the inner button so Enter/Space activate natively.

import { define, html, css, vars, computed, onCleanup } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { ripple } from '../motion/ripple.js';
import './ui-icon.js';

const t = vars('ui-nav-item', {
  pillBg: sys.color.secondaryContainer,
  iconFg: sys.color.onSurfaceVariant,
  iconFgActive: sys.color.onSecondaryContainer,
  labelFg: sys.color.onSurfaceVariant,
  labelFgActive: sys.color.onSurface,
  font: sys.type.labelMd,
  tracking: sys.tracking.labelMd,
});

const styles = css`
  :host { display: block; }
  :host(:focus-visible) {
    outline: ${sys.focus.ring};
    outline-offset: calc(-1 * ${sys.focus.ringWidth});
    border-radius: ${sys.radius.sm};
  }
  .control {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: ${sys.space(1)};
    inline-size: 100%;
    block-size: 100%;
    min-inline-size: 48px;
    padding-block: ${sys.space(3)};
    border: none;
    background: transparent;
    cursor: pointer;
    user-select: none;
    outline: none;
    color: ${t.iconFg};
  }
  .pill {
    position: relative;
    isolation: isolate;
    display: grid;
    place-items: center;
    inline-size: 56px;
    block-size: 32px;
    border-radius: ${sys.radius.full};
    transition: color ${sys.duration.short4} ${sys.easing.standard};
  }
  .bg {
    position: absolute; inset: 0; z-index: -1;
    border-radius: inherit;
    background: ${t.pillBg};
    opacity: 0;
    scale: 0.6 1;
    transition: opacity ${sys.duration.short4} ${sys.easing.standard},
                scale ${sys.duration.medium2} ${sys.easing.emphasizedDecelerate};
  }
  .layer {
    position: absolute; inset: 0; z-index: -1;
    border-radius: inherit;
    background: currentColor;
    opacity: 0;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  .control:hover .layer { opacity: ${sys.state.hover}; }
  .control:focus-visible .layer, :host(:focus-visible) .layer { opacity: ${sys.state.focus}; }
  .control:active .layer { opacity: ${sys.state.pressed}; }
  .selected .pill { color: ${t.iconFgActive}; }
  .selected .bg { opacity: 1; scale: 1; }
  .label {
    font: ${t.font};
    letter-spacing: ${t.tracking};
    color: ${t.labelFg};
    white-space: nowrap;
    transition: color ${sys.duration.short4} ${sys.easing.standard};
  }
  .selected .label { color: ${t.labelFgActive}; }
  .control:disabled {
    cursor: default;
    pointer-events: none;
    color: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContent} * 100%), transparent);
  }
  .control:disabled .label { color: inherit; }
`;

define('ui-nav-item', {
  props: { value: '', icon: '', activeIcon: '', label: '', selected: false, disabled: false },
  styles: [base, styles],
  setup({ value, icon, activeIcon, label, selected, disabled }, host) {
    let btn = null;
    const forward = () => btn?.focus();
    host.addEventListener('focus', forward);
    onCleanup(() => host.removeEventListener('focus', forward));

    const shownIcon = computed(() => (selected() && activeIcon()) || icon());

    const onClick = () => {
      if (disabled()) return;
      host.emit('ui-nav-select', { value: value() });
    };

    return html`
      <button part="control" class=${() => `control${selected() ? ' selected' : ''}`}
              type="button" tabindex="-1" ?disabled=${disabled}
              aria-label=${() => label() || value()}
              aria-current=${() => (selected() ? 'page' : null)}
              @click=${onClick}
              ref=${(el) => (btn = el)}>
        <span part="pill" class="pill" ref=${(el) => ripple(el, { disabled, centered: true })}>
          <span class="bg" aria-hidden="true"></span>
          <span class="layer" aria-hidden="true"></span>
          ${() => (shownIcon() ? html`<ui-icon name=${shownIcon}></ui-icon>` : html`<slot name="icon"></slot>`)}
        </span>
        ${() => (label() ? html`<span class="label">${label}</span>` : null)}
      </button>`;
  },
});

export const tag = 'ui-nav-item';
export const themeVars = t;
