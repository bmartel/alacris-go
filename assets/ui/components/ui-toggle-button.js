// <ui-toggle-button> — one Material segmented button, used inside
// <ui-toggle-group> (the group draws the outlined container and dividers;
// standalone the segment is a flat pill).
//
// Selecting animates a leading check icon in smoothly via width and scale transitions,
// adhering to Material Design 3. When an icon is already present, selecting smoothly
// crossfades and morphs between the custom icon and the checkmark in both directions.
// The button does not own its selection: it emits `ui-toggle` and the group
// (or any parent) sets `selected` back down.
//
// @prop  {string}  value=''       — REQUIRED identity within the group
// @prop  {boolean} selected=false — set by the owning group
// @prop  {boolean} disabled=false
// @prop  {string}  icon=''        — optional leading icon
// @event ui-toggle — pressed; detail: { value } (consumed by ui-toggle-group)
// @slot  (default) — label
// @part  control   — the <button>
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, computed } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base, focusRingOn } from './base.js';
import { ripple } from '../motion/ripple.js';
import './ui-icon.js';

const t = vars('ui-toggle-button', {
  height: '40px',
  radius: sys.radius.full,
  bg: sys.color.surface,
  fg: sys.color.onSurface,
  selectedBg: sys.color.secondaryContainer,
  selectedFg: sys.color.onSecondaryContainer,
  font: sys.type.labelLg,
  tracking: sys.tracking.labelLg,
});

const styles = css`
  :host { display: inline-flex; vertical-align: middle; flex: 1; }
  .control {
    position: relative;
    isolation: isolate;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    inline-size: 100%;
    block-size: calc(${t.height} + var(--ui-density, 0) * 4px);
    padding-inline: ${sys.space(3)};
    border: none;
    border-radius: ${t.radius};
    background: ${t.bg};
    color: ${t.fg};
    font: ${t.font};
    letter-spacing: ${t.tracking};
    cursor: pointer;
    user-select: none;
    white-space: nowrap;
    transition: background-color ${sys.duration.short4} ${sys.easing.standard},
                color ${sys.duration.short4} ${sys.easing.standard};
    --ui-icon-size: 1.125rem;
  }
  .lead {
    position: relative;
    display: inline-grid;
    place-items: center;
    inline-size: 0;
    min-inline-size: 0;
    block-size: 1.125rem;
    overflow: hidden;
    opacity: 0;
    pointer-events: none;
    transition: inline-size ${sys.duration.short4} ${sys.easing.emphasized},
                margin-inline-end ${sys.duration.short4} ${sys.easing.emphasized},
                opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  .lead.on {
    inline-size: 1.125rem;
    margin-inline-end: ${sys.space(2)};
    opacity: 1;
    pointer-events: auto;
  }
  .lead ui-icon {
    grid-area: 1 / 1;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    transition: opacity ${sys.duration.short3} ${sys.easing.standard},
                scale ${sys.duration.short3} ${sys.easing.emphasized},
                transform ${sys.duration.short3} ${sys.easing.emphasized};
  }
  .lead .check-glyph {
    opacity: 0;
    scale: 0.5;
    pointer-events: none;
  }
  .lead .check-glyph.on {
    opacity: 1;
    scale: 1;
    pointer-events: auto;
  }
  .lead .custom-glyph {
    opacity: 1;
    scale: 1;
    pointer-events: auto;
  }
  .lead .custom-glyph.off {
    opacity: 0;
    scale: 0.5;
    pointer-events: none;
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

  .control[aria-pressed="true"] {
    background: ${t.selectedBg};
    color: ${t.selectedFg};
  }

  .control:disabled {
    cursor: default;
    pointer-events: none;
    color: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContent} * 100%), transparent);
  }
  .control:disabled[aria-pressed="true"] {
    background: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContainer} * 100%), transparent);
  }
`;

define('ui-toggle-button', {
  props: { value: '', selected: false, disabled: false, icon: '' },
  styles: [base, styles],
  setup({ value, selected, disabled, icon }, host) {
    const onClick = () => {
      if (disabled()) return;
      host.emit('ui-toggle', { value: value() });
    };

    const isSelected = () => selected();
    const hasIcon = () => !!icon();
    const showLead = () => isSelected() || hasIcon();

    return html`
      <button part="control" class="control" type="button" ?disabled=${disabled}
              aria-pressed=${() => String(selected())}
              @click=${onClick} ref=${(el) => ripple(el, { disabled })}>
        <span class="layer" aria-hidden="true"></span>
        <span class=${() => `lead${showLead() ? ' on' : ''}`}>
          <ui-icon class=${() => `check-glyph${isSelected() ? ' on' : ''}`}
                   name="check"></ui-icon>
          ${() => (hasIcon()
            ? html`<ui-icon class=${() => `custom-glyph${isSelected() ? ' off' : ''}`}
                           name=${icon}></ui-icon>`
            : null)}
        </span>
        <slot></slot>
      </button>`;
  },
});

export const tag = 'ui-toggle-button';
export const themeVars = t;
