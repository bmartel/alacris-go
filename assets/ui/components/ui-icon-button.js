// <ui-icon-button> — a 40px icon button, optionally a toggle.
//
// @prop  {string}  icon=''          — registry icon name (or slot an <ui-icon>)
// @prop  {string}  selectedIcon=''  — icon while selected (toggle mode; defaults to `icon`)
// @prop  {string}  label=''         — REQUIRED accessible name
// @prop  {string}  variant='standard' — standard | filled | tonal | outlined
// @prop  {boolean} toggle=false     — makes it a pressed-state toggle
// @prop  {boolean} selected=false   — toggle state
// @prop  {boolean} disabled=false
// @event change — toggle flipped; detail: { selected }
// @slot  (default) — custom icon content when `icon` is empty
// @part  control — the <button>
// @vars  see `t` below

import { define, html, css, vars, computed } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base, focusRingOn } from './base.js';
import { ripple } from '../motion/ripple.js';
import './ui-icon.js';

const t = vars('ui-icon-button', {
  size: '40px',
  radius: sys.radius.full,
  fg: sys.color.onSurfaceVariant,
  selectedFg: sys.color.primary,
  filledBg: sys.color.primary,
  filledFg: sys.color.onPrimary,
  tonalBg: sys.color.secondaryContainer,
  tonalFg: sys.color.onSecondaryContainer,
  outlineColor: sys.color.outline,
});

const styles = css`
  :host { display: inline-flex; vertical-align: middle; }
  .control {
    position: relative;
    isolation: isolate;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    inline-size: calc(${t.size} + var(--ui-density, 0) * 4px);
    block-size: calc(${t.size} + var(--ui-density, 0) * 4px);
    border: none;
    border-radius: ${t.radius};
    background: transparent;
    color: ${t.fg};
    cursor: pointer;
    user-select: none;
    transition: background-color ${sys.duration.short4} ${sys.easing.standard},
                color ${sys.duration.short4} ${sys.easing.standard};
  }
  ${focusRingOn('.control')}
  .layer {
    position: absolute; inset: 0; z-index: -1;
    border-radius: inherit; background: currentColor; opacity: 0;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  .control:hover .layer { opacity: ${sys.state.hover}; }
  .control:focus-visible .layer { opacity: ${sys.state.focus}; }
  .control:active .layer { opacity: ${sys.state.pressed}; }

  .standard[aria-pressed="true"] { color: ${t.selectedFg}; }
  .filled { background: ${t.filledBg}; color: ${t.filledFg}; }
  .filled:not([aria-pressed="true"]).toggle { background: ${sys.color.surfaceContainerHighest}; color: ${t.selectedFg}; }
  .tonal { background: ${t.tonalBg}; color: ${t.tonalFg}; }
  .tonal:not([aria-pressed="true"]).toggle { background: ${sys.color.surfaceContainerHighest}; color: ${sys.color.onSurfaceVariant}; }
  .outlined { border: 1px solid ${t.outlineColor}; }
  .outlined[aria-pressed="true"] { background: ${sys.color.inverseSurface}; color: ${sys.color.inverseOnSurface}; border-color: transparent; }

  .control:disabled {
    cursor: default;
    pointer-events: none;
    color: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContent} * 100%), transparent);
  }
  .filled:disabled, .tonal:disabled {
    background: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContainer} * 100%), transparent);
  }
  .outlined:disabled {
    border-color: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContainer} * 100%), transparent);
  }
`;

define('ui-icon-button', {
  props: {
    icon: '', selectedIcon: '', label: '',
    variant: 'standard', toggle: false, selected: false, disabled: false,
  },
  styles: [base, styles],
  setup({ icon, selectedIcon, label, variant, toggle, selected, disabled }, host) {
    const cls = computed(() => `control ${variant()}${toggle() ? ' toggle' : ''}`);
    const shownIcon = computed(() => (toggle() && selected() && selectedIcon()) || icon());

    const onClick = () => {
      if (!toggle()) return;
      selected.set(!selected());
      host.emit('change', { selected: selected() });
    };

    return html`
      <button part="control" class=${cls} type="button" ?disabled=${disabled}
              aria-label=${() => label() || icon() || null}
              aria-pressed=${() => (toggle() ? String(selected()) : null)}
              @click=${onClick}
              ref=${(el) => ripple(el, { disabled, centered: true })}>
        <span class="layer" aria-hidden="true"></span>
        ${() => (shownIcon() ? html`<ui-icon name=${shownIcon}></ui-icon>` : html`<slot></slot>`)}
      </button>`;
  },
});

export const tag = 'ui-icon-button';
export const themeVars = t;
