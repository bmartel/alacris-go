// <ui-fab> — the Material floating action button, regular and extended.
//
// A non-empty `label` renders the extended FAB (icon + text) and is always the
// accessible name, extended or not.
//
// @prop  {string}  icon=''           — registry icon name (or slot custom content)
// @prop  {string}  label=''          — extended-FAB text; always used as aria-label
//                                      (falls back to the icon name when empty)
// @prop  {string}  variant='primary' — primary | secondary | tertiary | surface
// @prop  {string}  size='md'         — sm (40px) | md (56px) | lg (96px)
// @prop  {boolean} disabled=false
// @event (native click bubbles; no custom event)
// @slot  (default) — custom icon content when `icon` is empty
// @part  control   — the <button>
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, computed } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base, focusRingOn } from './base.js';
import { ripple } from '../motion/ripple.js';
import './ui-icon.js';

const t = vars('ui-fab', {
  sizeSm: '40px',
  sizeMd: '56px',
  sizeLg: '96px',
  radiusSm: sys.radius.md,
  radiusMd: sys.radius.lg,
  radiusLg: sys.radius.xl,
  font: sys.type.labelLg,
  tracking: sys.tracking.labelLg,
  primaryBg: sys.color.primaryContainer,
  primaryFg: sys.color.onPrimaryContainer,
  secondaryBg: sys.color.secondaryContainer,
  secondaryFg: sys.color.onSecondaryContainer,
  tertiaryBg: sys.color.tertiaryContainer,
  tertiaryFg: sys.color.onTertiaryContainer,
  surfaceBg: sys.color.surfaceContainerHigh,
  surfaceFg: sys.color.primary,
});

const styles = css`
  :host { display: inline-flex; vertical-align: middle; }
  .control {
    position: relative;
    isolation: isolate;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: ${sys.space(3)};
    block-size: calc(var(--_size) + var(--ui-density, 0) * 4px);
    min-inline-size: calc(var(--_size) + var(--ui-density, 0) * 4px);
    padding: 0;
    border: none;
    border-radius: var(--_radius);
    font: ${t.font};
    letter-spacing: ${t.tracking};
    cursor: pointer;
    user-select: none;
    white-space: nowrap;
    box-shadow: ${sys.elevation[3]};
    transition: box-shadow ${sys.duration.short4} ${sys.easing.standard};
    --ui-icon-size: var(--_icon);
  }
  .control:hover { box-shadow: ${sys.elevation[4]}; }
  .control:active { box-shadow: ${sys.elevation[3]}; }
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

  .sm { --_size: ${t.sizeSm}; --_radius: ${t.radiusSm}; --_icon: 1.5rem; }
  .md { --_size: ${t.sizeMd}; --_radius: ${t.radiusMd}; --_icon: 1.5rem; }
  .lg { --_size: ${t.sizeLg}; --_radius: ${t.radiusLg}; --_icon: 2.25rem; }
  .extended { padding-inline: ${sys.space(4)} 20px; --_icon: 1.5rem; }

  .primary { background: ${t.primaryBg}; color: ${t.primaryFg}; }
  .secondary { background: ${t.secondaryBg}; color: ${t.secondaryFg}; }
  .tertiary { background: ${t.tertiaryBg}; color: ${t.tertiaryFg}; }
  .surface { background: ${t.surfaceBg}; color: ${t.surfaceFg}; }

  .control:disabled {
    cursor: default;
    pointer-events: none;
    box-shadow: none;
    color: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContent} * 100%), transparent);
    background: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContainer} * 100%), transparent);
  }
`;

define('ui-fab', {
  props: { icon: '', label: '', variant: 'primary', size: 'md', disabled: false },
  styles: [base, styles],
  setup({ icon, label, variant, size, disabled }, host) {
    const cls = computed(() =>
      `control ${variant()} ${size()}${label() ? ' extended' : ''}`);

    return html`
      <button part="control" class=${cls} type="button" ?disabled=${disabled}
              aria-label=${() => label() || icon() || null}
              ref=${(el) => ripple(el, { disabled })}>
        <span class="layer" aria-hidden="true"></span>
        ${() => (icon() ? html`<ui-icon name=${icon}></ui-icon>` : html`<slot></slot>`)}
        ${() => (label() ? html`<span class="text">${label}</span>` : null)}
      </button>`;
  },
});

export const tag = 'ui-fab';
export const themeVars = t;
