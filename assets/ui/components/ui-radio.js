// <ui-radio> — one Material radio button, managed by <ui-radio-group>.
//
// The interactive element is a native <button role="radio"> on a 40px touch
// target. The radio never checks itself: it emits `ui-radio-select` and the
// owning group sets `checked` back down and roves the host's tabindex (the
// host forwards focus to the inner button).
//
// @prop  {string}  value=''      — REQUIRED identity within the group
// @prop  {boolean} checked=false — set by the owning group
// @prop  {boolean} disabled=false
// @prop  {string}  label=''      — visible label; also the accessible name.
//                                   Empty leaves the control unnamed (authoring error).
// @event ui-radio-select — pressed; detail: { value } (consumed by ui-radio-group)
// @part  control — the <button role="radio"> (the 40px target)
// @part  circle  — the visible 20px ring
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base, focusRingOn } from './base.js';
import { ripple } from '../motion/ripple.js';
import { presence } from '../motion/presence.js';
import { fx } from '../motion/animate.js';

const t = vars('ui-radio', {
  target: '40px',
  circleSize: '20px',
  outlineColor: sys.color.onSurfaceVariant,
  checkedColor: sys.color.primary,
  labelFg: sys.color.onSurface,
});

const styles = css`
  :host { display: inline-flex; outline: none; }
  label {
    display: inline-flex;
    align-items: center;
    cursor: pointer;
    font: ${sys.type.bodyMd};
    letter-spacing: ${sys.tracking.bodyMd};
    color: ${t.labelFg};
    user-select: none;
  }
  label.disabled {
    cursor: default;
    pointer-events: none;
    color: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContent} * 100%), transparent);
  }
  .control {
    position: relative;
    isolation: isolate;
    flex: none;
    display: grid;
    place-items: center;
    inline-size: max(${t.circleSize}, calc(${t.target} + var(--ui-density, 0) * 4px));
    block-size: max(${t.circleSize}, calc(${t.target} + var(--ui-density, 0) * 4px));
    padding: 0;
    border: none;
    border-radius: ${sys.radius.full};
    background: transparent;
    color: ${t.outlineColor};
    cursor: pointer;
  }
  ${focusRingOn('.control')}
  .layer {
    position: absolute; inset: 0; z-index: -1;
    border-radius: inherit;
    background: currentColor;
    opacity: 0;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  .control:hover .layer { opacity: ${sys.state.hover}; }
  .control:focus-visible .layer { opacity: ${sys.state.focus}; }
  .control:active .layer { opacity: ${sys.state.pressed}; }

  .circle {
    inline-size: ${t.circleSize};
    block-size: ${t.circleSize};
    border: 2px solid ${t.outlineColor};
    border-radius: ${sys.radius.full};
    display: grid;
    place-items: center;
    transition: border-color ${sys.duration.short2} ${sys.easing.standard};
  }
  .control[aria-checked="true"] { color: ${t.checkedColor}; }
  .control[aria-checked="true"] .circle { border-color: ${t.checkedColor}; }
  .dot {
    inline-size: 10px;
    block-size: 10px;
    border-radius: ${sys.radius.full};
    background: ${t.checkedColor};
  }

  .control:disabled { cursor: default; pointer-events: none; }
  .control:disabled .circle {
    border-color: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContent} * 100%), transparent);
  }
  .control:disabled .dot {
    background: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContent} * 100%), transparent);
  }
`;

define('ui-radio', {
  props: { value: '', checked: false, disabled: false, label: '' },
  styles: [base, styles],
  setup({ value, checked, disabled, label }, host) {
    let btn = null;

    // The group roves tabindex on the HOST; forward its focus to the button
    // so keyboard interaction lands on the real control.
    host.focus = (opts) => btn?.focus(opts);
    host.addEventListener('focus', () => btn?.focus());

    const onClick = () => {
      if (disabled()) return;
      host.emit('ui-radio-select', { value: value() });
    };

    return html`
      <label class=${() => (disabled() ? 'disabled' : null)}>
        <button part="control" class="control" type="button" role="radio"
                tabindex="-1"
                aria-checked=${() => String(checked())} ?disabled=${disabled}
                aria-labelledby=${() => (label() ? 'label' : null)}
                @click=${onClick}
                ref=${(el) => { btn = el; ripple(el, { disabled, centered: true }); }}>
          <span class="layer" aria-hidden="true"></span>
          <span part="circle" class="circle" aria-hidden="true">
            ${presence(checked, () => html`<span class="dot"></span>`, {
              enter: fx.scaleIn,
              exit: fx.scaleOut,
              enterDuration: 'short4',
              exitDuration: 'short2',
            })}
          </span>
        </button>
        ${() => (label() ? html`<span id="label">${label}</span>` : null)}
      </label>`;
  },
});

export const tag = 'ui-radio';
export const themeVars = t;
