// <ui-split-button> — a Material split button: a primary action plus a
// connected chevron that opens related actions.
//
//   <ui-split-button variant="filled" @click=${save}>
//     Save
//     <ui-menu-item slot="menu" value="draft">Save draft</ui-menu-item>
//     <ui-menu-item slot="menu" value="copy">Save a copy</ui-menu-item>
//   </ui-split-button>
//
// The leading segment is the primary action (native click bubbles). Choosing
// a menu item emits `select` with that item's value.
//
// @prop  {string}  variant='filled' — filled | tonal | outlined | elevated
// @prop  {boolean} disabled=false
// @event (native click bubbles from the leading segment)
// @event select — a menu action was chosen; detail: { value }
// @slot  (default) — leading-segment label
// @slot  icon      — leading icon on the primary action
// @slot  menu      — <ui-menu-item> children
// @part  group, action, chevron
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, computed, onCleanup } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base, focusRingOn } from './base.js';
import { ripple } from '../motion/ripple.js';
import './ui-icon.js';
import './ui-menu.js';
import './ui-menu-item.js';

const t = vars('ui-split-button', {
  height: '40px',
  padInline: '20px',
  radius: sys.radius.full,
  font: sys.type.labelLg,
  tracking: sys.tracking.labelLg,
  filledBg: sys.color.primary,
  filledFg: sys.color.onPrimary,
  tonalBg: sys.color.secondaryContainer,
  tonalFg: sys.color.onSecondaryContainer,
  outlinedFg: sys.color.primary,
  outlineColor: sys.color.outline,
  elevatedBg: sys.color.surfaceContainerLow,
  elevatedFg: sys.color.primary,
  divider: sys.color.outlineVariant,
});

const styles = css`
  :host { display: inline-flex; vertical-align: middle; }
  .group {
    display: inline-flex;
    align-items: stretch;
    block-size: calc(${t.height} + var(--ui-density, 0) * 4px);
    border-radius: ${t.radius};
    overflow: hidden;
    font: ${t.font};
    letter-spacing: ${t.tracking};
  }
  .action, .chevron {
    position: relative;
    isolation: isolate;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    margin: 0;
    border: none;
    background: transparent;
    color: inherit;
    cursor: pointer;
    user-select: none;
  }
  ${focusRingOn('.action')}
  ${focusRingOn('.chevron')}
  .layer {
    position: absolute; inset: 0; z-index: -1;
    background: currentColor; opacity: 0;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  .action:hover .layer, .chevron:hover .layer { opacity: ${sys.state.hover}; }
  .action:focus-visible .layer, .chevron:focus-visible .layer { opacity: ${sys.state.focus}; }
  .action:active .layer, .chevron:active .layer { opacity: ${sys.state.pressed}; }

  .action {
    gap: ${sys.space(2)};
    padding-inline: ${t.padInline} ${sys.space(3)};
    --ui-icon-size: 1.125rem;
  }
  .chevron {
    padding-inline: ${sys.space(2)} ${sys.space(3)};
    border-inline-start: 1px solid ${t.divider};
    --ui-icon-size: 1.5rem;
  }

  .filled { background: ${t.filledBg}; color: ${t.filledFg}; }
  .tonal { background: ${t.tonalBg}; color: ${t.tonalFg}; }
  .outlined {
    color: ${t.outlinedFg};
    box-shadow: inset 0 0 0 1px ${t.outlineColor};
    background: transparent;
  }
  .elevated {
    background: ${t.elevatedBg};
    color: ${t.elevatedFg};
    box-shadow: ${sys.elevation[1]};
  }

  .disabled { pointer-events: none; }
  .filled.disabled, .tonal.disabled, .elevated.disabled {
    background: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContainer} * 100%), transparent);
    color: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContent} * 100%), transparent);
    box-shadow: none;
  }
  .outlined.disabled {
    color: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContent} * 100%), transparent);
    box-shadow: inset 0 0 0 1px color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContainer} * 100%), transparent);
  }
  .group ui-menu { display: contents; }
`;

define('ui-split-button', {
  props: { variant: 'filled', disabled: false },
  styles: [base, styles],
  setup({ variant, disabled }, host) {
    const cls = computed(() =>
      `group ${variant()}${disabled() ? ' disabled' : ''}`);

    const onSelect = (e) => {
      e.stopPropagation();
      host.emit('select', { value: e.detail.value });
      const menu = host.shadowRoot?.querySelector('ui-menu');
      if (menu) menu.open = false;
    };
    host.addEventListener('ui-menu-select', onSelect);
    onCleanup(() => host.removeEventListener('ui-menu-select', onSelect));

    return html`
      <div class=${cls} part="group" role="group">
        <button part="action" class="action" type="button" ?disabled=${disabled}
                ref=${(el) => ripple(el, { disabled })}>
          <span class="layer" aria-hidden="true"></span>
          <slot name="icon"></slot>
          <slot></slot>
        </button>
        <ui-menu>
          <button slot="anchor" part="chevron" class="chevron" type="button"
                  ?disabled=${disabled} aria-label="More actions"
                  ref=${(el) => ripple(el, { disabled })}>
            <span class="layer" aria-hidden="true"></span>
            <ui-icon name="arrow-drop-down"></ui-icon>
          </button>
          <slot name="menu"></slot>
        </ui-menu>
      </div>`;
  },
});

export const tag = 'ui-split-button';
export const themeVars = t;
