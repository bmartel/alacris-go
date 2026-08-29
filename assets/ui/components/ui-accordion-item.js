// <ui-accordion-item> — one expandable panel inside <ui-accordion>.
//
// @prop  {string}  value=''       — REQUIRED identity of the panel (reported in events)
// @prop  {boolean} expanded=false
// @prop  {boolean} disabled=false
// @prop  {string}  headline=''    — header text
// @event ui-accordion-toggle — header activated; detail: { value, expanded }
// @slot  (default) — panel content
// @part  header  — the header <button>
// @part  content — the collapsible region
// @part  body    — padded wrapper around the default slot
// @vars  see `t` below (`themeVars.names`)
//
// Expand/collapse animates block-size from the measured content height; in
// environments without layout (zero heights) the animation is skipped and the
// state applies instantly. The content region gets `hidden` only AFTER the
// collapse animation completes.

import { define, html, css, vars, effect } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base, focusRingOn } from './base.js';
import { ripple } from '../motion/ripple.js';
import { animate } from '../motion/animate.js';
import './ui-icon.js';

const t = vars('ui-accordion-item', {
  headerMinHeight: '48px',
  padInline: sys.space(4),
  headerFg: sys.color.onSurface,
  iconFg: sys.color.onSurfaceVariant,
  contentFg: sys.color.onSurfaceVariant,
  font: sys.type.titleMd,
  tracking: sys.tracking.titleMd,
  contentFont: sys.type.bodyMd,
  contentTracking: sys.tracking.bodyMd,
});

const styles = css`
  :host { display: block; }
  .heading { margin: 0; font: inherit; }
  .header {
    position: relative;
    isolation: isolate;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: ${sys.space(3)};
    inline-size: 100%;
    min-block-size: calc(${t.headerMinHeight} + var(--ui-density, 0) * 4px);
    padding-inline: ${t.padInline};
    padding-block: ${sys.space(2)};
    border: none;
    background: transparent;
    color: ${t.headerFg};
    font: ${t.font};
    letter-spacing: ${t.tracking};
    text-align: start;
    cursor: pointer;
    user-select: none;
  }
  ${focusRingOn('.header')}
  .layer {
    position: absolute; inset: 0; z-index: -1;
    background: currentColor; opacity: 0;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  .header:hover .layer { opacity: ${sys.state.hover}; }
  .header:focus-visible .layer { opacity: ${sys.state.focus}; }
  .header:active .layer { opacity: ${sys.state.pressed}; }
  .header:disabled {
    cursor: default;
    pointer-events: none;
    color: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContent} * 100%), transparent);
  }
  .chevron {
    color: ${t.iconFg};
    transition: rotate ${sys.duration.short4} ${sys.easing.standard};
  }
  .header:disabled .chevron { color: inherit; }
  .chevron.open { rotate: 180deg; }
  .content { overflow: hidden; }
  .content[hidden] { display: none; }
  .body {
    padding-inline: ${t.padInline};
    padding-block-end: ${sys.space(4)};
    font: ${t.contentFont};
    letter-spacing: ${t.contentTracking};
    color: ${t.contentFg};
  }
`;

define('ui-accordion-item', {
  props: { value: '', expanded: false, disabled: false, headline: '' },
  styles: [base, styles],
  setup({ value, expanded, disabled, headline }, host) {
    let contentEl = null;
    let anim = null;

    // Animate block-size between 0 and the measured content height. Zero
    // heights (no layout, e.g. simulated DOM) skip the animation entirely.
    const apply = (exp) => {
      const el = contentEl;
      anim?.cancel();
      anim = null;
      if (exp) {
        el.hidden = false;
        const h = el.scrollHeight;
        if (!h) return;
        const a = (anim = animate(el, [{ blockSize: '0px' }, { blockSize: `${h}px` }], {
          duration: 'medium2', easing: 'emphasizedDecelerate',
        }));
        // Cancelling after it finishes drops the fill, so block-size clears
        // back to auto (the end keyframe equals the natural height).
        a.finished.then(() => { if (anim === a) { anim = null; a.cancel(); } }, () => {});
      } else {
        const h = el.scrollHeight;
        if (!h) { el.hidden = true; return; }
        const a = (anim = animate(el, [{ blockSize: `${h}px` }, { blockSize: '0px' }], {
          duration: 'short4', easing: 'emphasizedAccelerate',
        }));
        a.finished.then(() => { if (anim === a) { anim = null; a.cancel(); el.hidden = true; } }, () => {});
      }
    };

    effect(() => {
      const exp = expanded();
      if (contentEl) apply(exp); // first run precedes the ref; it syncs below
    });

    const toggle = () => {
      if (disabled()) return;
      expanded.set(!expanded());
      host.emit('ui-accordion-toggle', { value: value(), expanded: expanded() });
    };

    return html`
      <h3 class="heading">
        <button part="header" class="header" type="button" id="header"
                ?disabled=${disabled}
                aria-expanded=${() => String(expanded())}
                aria-controls="content"
                @click=${toggle}
                ref=${(el) => ripple(el, { disabled })}>
          <span class="layer" aria-hidden="true"></span>
          <span class="headline">${headline}</span>
          <ui-icon class=${() => `chevron${expanded() ? ' open' : ''}`} name="expand-more"></ui-icon>
        </button>
      </h3>
      <div part="content" class="content" id="content" role="region" aria-labelledby="header" ?inert=${() => !expanded()}
           ref=${(el) => { contentEl = el; el.hidden = !expanded.peek(); }}>
        <div class="body" part="body"><slot></slot></div>
      </div>`;
  },
});

export const tag = 'ui-accordion-item';
export const themeVars = t;
