// <ui-button> — the Material common button, all five variants.
//
// @prop  {string}  variant='filled' — filled | tonal | outlined | text | elevated
// @prop  {boolean} disabled=false
// @prop  {string}  type='button'    — button | submit (submit reaches the enclosing form)
// @prop  {string}  href=''          — renders a link styled as a button
// @prop  {string}  target=''        — link target when href is set
// @event (native click bubbles; no custom event)
// @slot  (default) — label
// @slot  icon      — leading icon
// @slot  trailing  — trailing icon
// @part  control   — the <button>/<a>
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, computed, signal } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base, focusRingOn, stateLayerOn } from './base.js';
import { ripple } from '../motion/ripple.js';

const t = vars('ui-button', {
  height: '40px',
  padInline: sys.space(6),
  gap: sys.space(2),
  radius: sys.radius.full,
  font: sys.type.labelLg,
  tracking: sys.tracking.labelLg,
  filledBg: sys.color.primary,
  filledFg: sys.color.onPrimary,
  tonalBg: sys.color.secondaryContainer,
  tonalFg: sys.color.onSecondaryContainer,
  outlinedFg: sys.color.primary,
  outlineColor: sys.color.outline,
  textFg: sys.color.primary,
  elevatedBg: sys.color.surfaceContainerLow,
  elevatedFg: sys.color.primary,
});

const styles = css`
  :host { display: inline-flex; vertical-align: middle; }
  .control {
    position: relative;
    isolation: isolate;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: ${t.gap};
    block-size: calc(${t.height} + var(--ui-density, 0) * 4px);
    padding-inline: ${t.padInline};
    border: none;
    border-radius: ${t.radius};
    font: ${t.font};
    letter-spacing: ${t.tracking};
    background: transparent;
    color: inherit;
    cursor: pointer;
    user-select: none;
    text-decoration: none;
    white-space: nowrap;
    transition: box-shadow ${sys.duration.short4} ${sys.easing.standard};
    --ui-icon-size: 1.125rem;
  }
  ${focusRingOn('.control')}
  .layer {
    position: absolute; inset: 0; z-index: -1;
    border-radius: inherit;
    background: currentColor;
    opacity: 0;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  ${stateLayerOn('.control')}

  .filled { background: ${t.filledBg}; color: ${t.filledFg}; }
  .filled:hover { box-shadow: ${sys.elevation[1]}; }
  .filled:active { box-shadow: none; }
  .tonal { background: ${t.tonalBg}; color: ${t.tonalFg}; }
  .outlined {
    color: ${t.outlinedFg};
    border: 1px solid ${t.outlineColor};
  }
  .text { color: ${t.textFg}; padding-inline: ${sys.space(3)}; }
  .elevated { background: ${t.elevatedBg}; color: ${t.elevatedFg}; box-shadow: ${sys.elevation[1]}; }
  .elevated:hover { box-shadow: ${sys.elevation[2]}; }
  .elevated:active { box-shadow: ${sys.elevation[1]}; }

  /* Leading icon: 16dp start padding. Trailing icon: 16dp end padding. */
  .with-icon:not(.text) { padding-inline-start: ${sys.space(4)}; }
  .with-trailing:not(.text) { padding-inline-end: ${sys.space(4)}; }
  .text.with-icon { padding-inline-end: ${sys.space(4)}; }
  .text.with-trailing { padding-inline-start: ${sys.space(4)}; }

  .control:disabled, .control[aria-disabled="true"] {
    cursor: default;
    pointer-events: none;
    box-shadow: none;
    color: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContent} * 100%), transparent);
  }
  .filled:disabled, .tonal:disabled, .elevated:disabled,
  .filled[aria-disabled="true"], .tonal[aria-disabled="true"], .elevated[aria-disabled="true"] {
    background: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContainer} * 100%), transparent);
  }
  .outlined:disabled, .outlined[aria-disabled="true"] {
    border-color: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContainer} * 100%), transparent);
  }
`;

define('ui-button', {
  props: { variant: 'filled', disabled: false, type: 'button', href: '', target: '' },
  styles: [base, styles],
  setup({ variant, disabled, type, href, target }, host) {
    const hasIcon = signal(false);
    const hasTrailing = signal(false);
    const onSlot = (sig) => (el) => {
      const sync = () => sig.set(el.assignedElements().length > 0);
      el.addEventListener('slotchange', sync);
      sync();
    };
    const cls = computed(() =>
      `control ${variant()}${hasIcon() ? ' with-icon' : ''}${hasTrailing() ? ' with-trailing' : ''}`);

    const onClick = (e) => {
      if (disabled()) { e.preventDefault(); e.stopPropagation(); return; }
      // A shadow-DOM button cannot submit the host's form; bridge it.
      if (!href() && type() === 'submit') host.closest('form')?.requestSubmit();
    };

    const inner = html`
      <span class="layer" aria-hidden="true"></span>
      <slot name="icon" ref=${onSlot(hasIcon)}></slot>
      <slot></slot>
      <slot name="trailing" ref=${onSlot(hasTrailing)}></slot>`;

    return html`${() =>
      href()
        ? html`<a part="control" class=${cls} href=${href} target=${() => target() || null}
                  aria-disabled=${() => (disabled() ? 'true' : null)} tabindex=${() => (disabled() ? '-1' : null)}
                  @click=${onClick} ref=${(el) => ripple(el, { disabled })}>${inner}</a>`
        : html`<button part="control" class=${cls} type="button" ?disabled=${disabled}
                  @click=${onClick} ref=${(el) => ripple(el, { disabled })}>${inner}</button>`}`;
  },
});

export const tag = 'ui-button';
export const themeVars = t;
