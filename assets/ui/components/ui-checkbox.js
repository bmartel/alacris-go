// <ui-checkbox> — the Material checkbox with indeterminate support.
//
// The interactive element is a native <button role="checkbox"> sized to a
// 40px touch target; the visible 18px box sits centered inside it. Clicking
// clears `indeterminate` and toggles `checked`. The check/dash is the MD3
// 2px stroke mark in the 18dp icon, not the generic 24dp `check` glyph.
//
// @prop  {boolean} checked=false
// @prop  {boolean} indeterminate=false — aria-checked="mixed", shows a dash
// @prop  {boolean} disabled=false
// @prop  {string}  label=''  — visible label; also the accessible name.
//                               Empty leaves the control unnamed (authoring error).
// @prop  {string}  name=''   — form field name (submits `value` while checked)
// @prop  {string}  value='on'
// @event change — detail: { checked, indeterminate: false }
// @part  control — the <button role="checkbox"> (the 40px target)
// @part  box     — the visible 18px box
// @part  label   — the visible label span (omit `label` and this is absent)
// @vars  see `t` below (`themeVars.names`)

import { define, html, svg, css, vars, computed } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base, focusRingOn } from './base.js';
import { ripple } from '../motion/ripple.js';
import { presence } from '../motion/presence.js';
import { fx } from '../motion/animate.js';
import { formBind } from '../util/form.js';

const t = vars('ui-checkbox', {
  target: '40px',
  boxSize: '18px',
  radius: sys.radius.xs,
  outlineColor: sys.color.onSurfaceVariant,
  bg: sys.color.primary,
  markFg: sys.color.onPrimary,
  labelFg: sys.color.onSurface,
});

const styles = css`
  :host { display: inline-flex; }
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
    inline-size: max(${t.boxSize}, calc(${t.target} + var(--ui-density, 0) * 4px));
    block-size: max(${t.boxSize}, calc(${t.target} + var(--ui-density, 0) * 4px));
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

  .box {
    position: relative;
    inline-size: ${t.boxSize};
    block-size: ${t.boxSize};
    border-radius: ${t.radius};
    display: grid;
    place-items: center;
    overflow: hidden;
    color: ${t.markFg};
    box-shadow: inset 0 0 0 2px ${t.outlineColor};
    transition: background-color ${sys.duration.short2} ${sys.easing.standard},
                box-shadow ${sys.duration.short2} ${sys.easing.standard};
  }
  .box svg {
    display: block;
    inline-size: ${t.boxSize};
    block-size: ${t.boxSize};
  }
  .control[aria-checked="true"],
  .control[aria-checked="mixed"] { color: ${t.bg}; }
  .control[aria-checked="true"] .box,
  .control[aria-checked="mixed"] .box {
    background: ${t.bg};
    box-shadow: none;
  }

  .control:disabled { cursor: default; pointer-events: none; }
  .control:disabled .box {
    box-shadow: inset 0 0 0 2px color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContent} * 100%), transparent);
  }
  .control:disabled[aria-checked="true"] .box,
  .control:disabled[aria-checked="mixed"] .box {
    background: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContent} * 100%), transparent);
    box-shadow: none;
    color: ${sys.color.surface};
  }
`;

define('ui-checkbox', {
  formAssociated: true,
  props: { checked: false, indeterminate: false, disabled: false, label: '', name: '', value: 'on' },
  styles: [base, styles],
  setup({ checked, indeterminate, disabled, label, name, value }, host) {
    formBind(host, { name, value, checked, disabled });

    const state = computed(() => (indeterminate() ? 'mixed' : String(checked())));
    const marked = computed(() => checked() || indeterminate());

    const onClick = () => {
      if (disabled()) return;
      indeterminate.set(false);
      checked.set(!checked());
      host.emit('change', { checked: checked(), indeterminate: false });
    };

    return html`
      <label class=${() => (disabled() ? 'disabled' : null)}>
        <button part="control" class="control" type="button" role="checkbox"
                aria-checked=${state} ?disabled=${disabled}
                aria-labelledby=${() => (label() ? 'label' : null)}
                @click=${onClick}
                ref=${(el) => ripple(el, { disabled, centered: true })}>
          <span class="layer" aria-hidden="true"></span>
          <span part="box" class="box" aria-hidden="true">
            ${presence(marked, () => svg`
                <svg viewBox="0 0 18 18" aria-hidden="true">
                  <path fill="none" stroke="currentColor" stroke-width="2"
                        stroke-linecap="round" stroke-linejoin="round"
                        d=${() => (indeterminate() ? 'M4 9h10' : 'M4 9.8l3.2 3.2 7.2-7.2')}></path>
                </svg>`, {
              enter: fx.scaleIn,
              exit: fx.scaleOut,
              enterDuration: 'short4',
              exitDuration: 'short2',
            })}
          </span>
        </button>
        ${() => (label() ? html`<span id="label" part="label">${label}</span>` : null)}
      </label>`;
  },
});

export const tag = 'ui-checkbox';
export const themeVars = t;
