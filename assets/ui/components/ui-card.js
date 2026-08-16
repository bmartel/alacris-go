// <ui-card> — a Material card surface.
//
// @prop  {string}  variant='elevated' — elevated | filled | outlined
// @prop  {boolean} interactive=false  — hover elevation + ripple + button semantics
// @event action — interactive card activated (click/Enter/Space)
// @slot  (default) — card body (compose freely; padding is yours via parts/vars)
// @slot  media     — full-bleed media at the top
// @part  container — the card surface
// @part  body      — padded wrapper around the default slot
// @vars  see `t` below

import { define, html, css, vars, computed } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base, focusRingOn } from './base.js';
import { ripple } from '../motion/ripple.js';

const t = vars('ui-card', {
  radius: sys.radius.md,
  bg: sys.color.surfaceContainerLow,
  fg: sys.color.onSurface,
  filledBg: sys.color.surfaceContainerHighest,
  outlineColor: sys.color.outlineVariant,
  padding: sys.space(4),
});

const styles = css`
  :host { display: block; }
  .container {
    position: relative;
    isolation: isolate;
    display: flex;
    flex-direction: column;
    border-radius: ${t.radius};
    color: ${t.fg};
    overflow: hidden;
    transition: box-shadow ${sys.duration.short4} ${sys.easing.standard};
  }
  .elevated { background: ${t.bg}; box-shadow: ${sys.elevation[1]}; }
  .filled { background: ${t.filledBg}; }
  .outlined { background: ${sys.color.surface}; border: 1px solid ${t.outlineColor}; }
  .body { padding: ${t.padding}; }
  .interactive { cursor: pointer; }
  ${focusRingOn('.interactive')}
  .interactive .layer {
    position: absolute; inset: 0; z-index: 1;
    background: currentColor; opacity: 0; pointer-events: none;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  .interactive:hover .layer { opacity: ${sys.state.hover}; }
  .interactive:focus-visible .layer { opacity: ${sys.state.focus}; }
  .interactive:active .layer { opacity: ${sys.state.pressed}; }
  .interactive.elevated:hover { box-shadow: ${sys.elevation[2]}; }
  .interactive.elevated:active { box-shadow: ${sys.elevation[1]}; }
  .interactive.outlined:hover, .interactive.filled:hover { box-shadow: ${sys.elevation[1]}; }
  .interactive.outlined:active, .interactive.filled:active { box-shadow: none; }
`;

define('ui-card', {
  props: { variant: 'elevated', interactive: false },
  styles: [base, styles],
  setup({ variant, interactive }, host) {
    const cls = computed(() => `container ${variant()}${interactive() ? ' interactive' : ''}`);

    const activate = () => interactive() && host.emit('action');
    const onKeydown = (e) => {
      if (interactive() && (e.key === 'Enter' || e.key === ' ')) {
        e.preventDefault();
        activate();
      }
    };

    return html`
      <div part="container" class=${cls}
           role=${() => (interactive() ? 'button' : null)}
           tabindex=${() => (interactive() ? '0' : null)}
           @click=${activate} @keydown=${onKeydown}
           ref=${(el) => ripple(el, { disabled: () => !interactive() })}>
        <span class="layer" aria-hidden="true"></span>
        <slot name="media"></slot>
        <div class="body" part="body"><slot></slot></div>
      </div>`;
  },
});

export const tag = 'ui-card';
export const themeVars = t;
