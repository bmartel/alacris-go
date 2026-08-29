// <ui-chip> — Material chip: assist, filter, input, and suggestion.
//
// The HOST is the interactive element (focusable, role="button" — or
// role="option" with aria-selected for filter chips) so that <ui-chip-set>'s
// roving tabindex can move focus across plain light-DOM chips.
//
// Filter chips toggle `selected` on click/Enter/Space and emit `change`;
// assist/suggestion/input chips just let the native click bubble. A
// dismissible chip animates the host collapsing, then emits `dismiss` — the
// PARENT owns the list and removes the chip from the DOM; the chip never
// removes itself.
//
// @prop  {string}  variant='assist' — assist | filter | input | suggestion
// @prop  {boolean} selected=false   — filter chips only
// @prop  {boolean} disabled=false
// @prop  {string}  icon=''          — leading icon name (check replaces it while
//                                     a filter chip is selected)
// @prop  {boolean} dismissible=false — trailing remove button
// @prop  {string}  value=''         — identity within a <ui-chip-set>
// @event change  — filter chip toggled; detail: { selected }
// @event dismiss — remove requested (after the collapse animation)
// @slot  (default) — the chip label
// @part  control — the chip surface
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, computed, effect } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base, focusRingOn } from './base.js';
import { ripple } from '../motion/ripple.js';
import { animate, settled, fx } from '../motion/animate.js';
import { presence } from '../motion/presence.js';
import './ui-icon.js';
import './ui-icon-button.js';

const t = vars('ui-chip', {
  height: '32px',
  radius: sys.radius.sm,
  fg: sys.color.onSurfaceVariant,
  labelFg: sys.color.onSurface,
  outlineColor: sys.color.outlineVariant,
  selectedBg: sys.color.secondaryContainer,
  selectedFg: sys.color.onSecondaryContainer,
  font: sys.type.labelLg,
});

const styles = css`
  :host {
    display: inline-flex;
    vertical-align: middle;
    border-radius: ${t.radius};
    outline: none;
  }
  ${focusRingOn(':host')}
  .control {
    position: relative;
    isolation: isolate;
    display: inline-flex;
    align-items: center;
    gap: 0;
    block-size: calc(${t.height} + var(--ui-density, 0) * 4px);
    padding-inline: ${sys.space(3)};
    border: 1px solid ${t.outlineColor};
    border-radius: ${t.radius};
    background: transparent;
    color: ${t.fg};
    font: ${t.font};
    letter-spacing: ${sys.tracking.labelLg};
    cursor: pointer;
    user-select: none;
    white-space: nowrap;
    --ui-icon-size: 1.125rem;
    transition: background-color ${sys.duration.short2} ${sys.easing.standard},
                border-color ${sys.duration.short2} ${sys.easing.standard},
                padding-inline-start ${sys.duration.short4} ${sys.easing.emphasized};
  }
  .selected .control { padding-inline-start: ${sys.space(2)}; }
  .lead {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    inline-size: 0;
    min-inline-size: 0;
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
  .layer {
    position: absolute; inset: 0; z-index: -1;
    border-radius: inherit;
    background: currentColor;
    opacity: 0;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  .control:hover .layer { opacity: ${sys.state.hover}; }
  :host(:focus-visible) .layer { opacity: ${sys.state.focus}; }
  .control:active .layer { opacity: ${sys.state.pressed}; }

  .label { color: ${t.labelFg}; }
  .with-lead .control { padding-inline-start: ${sys.space(2)}; }
  .with-dismiss .control { padding-inline-end: ${sys.space(2)}; }
  .selected .control, :host([aria-selected="true"]) .control {
    background: ${t.selectedBg};
    border-color: transparent;
    color: ${t.selectedFg};
  }
  .selected .label, :host([aria-selected="true"]) .label { color: ${t.selectedFg}; }

  .dismiss {
    margin-inline-start: ${sys.space(2)};
    --ui-icon-button-size: 18px;
    --ui-icon-size: 1.125rem;
  }

  :host([aria-disabled="true"]) { pointer-events: none; }
  :host([aria-disabled="true"]) .control {
    cursor: default;
    color: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContent} * 100%), transparent);
    border-color: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContainer} * 100%), transparent);
  }
  :host([aria-disabled="true"]) .label {
    color: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContent} * 100%), transparent);
  }
`;

define('ui-chip', {
  props: {
    variant: 'assist', selected: false, disabled: false,
    icon: '', dismissible: false, value: '',
  },
  styles: [base, styles],
  setup({ variant, selected, disabled, icon, dismissible }, host) {
    // The host is the control: role, selection, disabled and focusability.
    effect(() => {
      if (variant() === 'filter') {
        host.setAttribute('role', 'option');
        host.setAttribute('aria-selected', String(selected()));
      } else {
        host.setAttribute('role', 'button');
        host.removeAttribute('aria-selected');
      }
    });
    effect(() => {
      if (disabled()) {
        host.setAttribute('aria-disabled', 'true');
        host.tabIndex = -1;
      } else {
        host.removeAttribute('aria-disabled');
        if (host.tabIndex < 0) host.tabIndex = 0;
      }
    });

    const checkWhen = computed(() => variant() === 'filter' && selected());
    const rootCls = computed(() =>
      [
        checkWhen() ? 'selected' : '',
        icon() ? 'with-lead' : '',
        dismissible() ? 'with-dismiss' : '',
      ].filter(Boolean).join(' '));

    const activate = () => {
      if (disabled()) return;
      if (variant() !== 'filter') return; // assist/suggestion/input: native click is the event
      selected.set(!selected());
      host.emit('change', { selected: selected() });
    };
    host.addEventListener('keydown', (e) => {
      if (e.key !== 'Enter' && e.key !== ' ') return;
      if (e.target !== host) return; // let the dismiss button handle its own keys
      e.preventDefault();
      activate();
      if (variant() !== 'filter') host.click();
    });

    const dismiss = (e) => {
      e.stopPropagation();
      if (disabled()) return;
      host.inert = true;
      const w = host.getBoundingClientRect().width;
      const anim = animate(host, [
        { inlineSize: `${w}px`, opacity: 1 },
        { inlineSize: '0px', opacity: 0 },
      ], { duration: 'short4', easing: 'emphasizedAccelerate' });
      settled(anim).then(() => host.emit('dismiss'));
    };

    // Filter check: replaces the leading icon while selected, scaling in/out
    // inside a width-animated slot so the chip grows instead of jumping.
    const lead = presence(checkWhen, () => html`<ui-icon name="check"></ui-icon>`, {
      enter: fx.scaleIn,
      exit: fx.scaleOut,
      enterDuration: 'short3',
      exitDuration: 'short2',
    });
    const showLead = computed(() => variant() === 'filter' || !!icon());

    return html`
      <span class=${rootCls}>
        <span class="control" part="control" @click=${activate}
              ref=${(el) => ripple(el, { disabled })}>
          <span class="layer" aria-hidden="true"></span>
          ${() => (showLead()
            ? html`<span class=${() => `lead${checkWhen() || icon() ? ' on' : ''}`}>
                ${lead}
                ${() => (icon() && !checkWhen() ? html`<ui-icon name=${icon}></ui-icon>` : null)}
              </span>`
            : null)}
          <span class="label"><slot></slot></span>
          ${() => (dismissible()
            ? html`<ui-icon-button class="dismiss" icon="close" label="Remove"
                                   ?disabled=${disabled} @click=${dismiss}></ui-icon-button>`
            : null)}
        </span>
      </span>`;
  },
});

export const tag = 'ui-chip';
export const themeVars = t;
