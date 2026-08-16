// <ui-option> — one choice inside <ui-select>.
//
// The host itself is the option (role="option"): it lives in the select's
// light DOM and is projected into the select's listbox panel. The select
// drives `selected`/`active` (via their attributes) and handles activation;
// the option only renders itself and its state layer.
//
// @prop  {string}  value=''       — the value this option contributes (required)
// @prop  {boolean} disabled=false
// @prop  {boolean} selected=false — set by the owning select; mirrors aria-selected
// @prop  {boolean} active=false   — set by the owning select while keyboard-active
// @slot  (default) — the visible label (also used by the select's field text)
// @part  control — the option row
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, effect } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { ripple } from '../motion/ripple.js';

const t = vars('ui-option', {
  fg: sys.color.onSurface,
  selectedBg: sys.color.secondaryContainer,
  selectedFg: sys.color.onSecondaryContainer,
  font: sys.type.bodyLg,
  height: '48px',
});

let uid = 0;

const styles = css`
  :host { display: block; cursor: pointer; user-select: none; }
  .control {
    position: relative;
    isolation: isolate;
    display: flex;
    align-items: center;
    gap: ${sys.space(3)};
    min-block-size: calc(${t.height} + var(--ui-density, 0) * 4px);
    padding-inline: ${sys.space(4)};
    font: ${t.font};
    letter-spacing: ${sys.tracking.bodyLg};
    color: ${t.fg};
  }
  .layer {
    position: absolute; inset: 0; z-index: -1;
    background: currentColor; opacity: 0;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  :host(:hover) .layer { opacity: ${sys.state.hover}; }
  :host([data-active]) .layer { opacity: ${sys.state.focus}; }
  :host(:active) .layer { opacity: ${sys.state.pressed}; }
  :host([aria-selected="true"]) .control {
    background: ${t.selectedBg};
    color: ${t.selectedFg};
  }
  :host([aria-disabled="true"]) { pointer-events: none; cursor: default; }
  :host([aria-disabled="true"]) .control {
    color: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContent} * 100%), transparent);
  }
`;

define('ui-option', {
  props: { value: '', disabled: false, selected: false, active: false },
  styles: [base, styles],
  setup({ disabled, selected, active }, host) {
    if (!host.id) host.id = `ui-option-${++uid}`;
    host.setAttribute('role', 'option');
    effect(() => host.setAttribute('aria-selected', String(selected())));
    effect(() => {
      if (disabled()) host.setAttribute('aria-disabled', 'true');
      else host.removeAttribute('aria-disabled');
    });
    effect(() => host.toggleAttribute('data-active', active()));

    return html`
      <div class="control" part="control" ref=${(el) => ripple(el, { disabled })}>
        <span class="layer" aria-hidden="true"></span>
        <slot></slot>
      </div>`;
  },
});

export const tag = 'ui-option';
export const themeVars = t;
