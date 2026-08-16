// <ui-tooltip> — a plain (or rich) tooltip on hover/focus around its target.
//
//   <ui-tooltip text="Save changes"><ui-button>Save</ui-button></ui-tooltip>
//
// Shows after `delay` ms on pointerenter (timer cancelled on leave) and
// immediately on focusin; hides on pointerleave, focusout, and Escape. The
// panel is position: fixed, anchored to the host with position()/autoUpdate,
// flipping to the opposite side when it would overflow the viewport.
//
// a11y: `aria-describedby` cannot point across shadow boundaries, so the
// association is not programmatic — the panel carries role="tooltip" and the
// target keeps its own accessible name. Give the target a matching
// label/aria-label when the tooltip is the only description.
//
// @prop  {string}  text=''      — plain tooltip content
// @prop  {string}  position='top' — top | bottom | left | right
// @prop  {number}  delay=500    — hover show delay (ms); focus shows instantly
// @prop  {boolean} rich=false   — renders the `content` slot in a surface
//                                 panel instead of the text pill
// @slot  (default) — the target
// @slot  content   — rich tooltip content (title/body/actions) when `rich`
// @part  panel — the tooltip surface
// @vars  see `t` below

import { define, html, css, vars, effect, onCleanup, signal } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { presence } from '../motion/presence.js';
import { fx } from '../motion/animate.js';
import { autoUpdate } from '../util/position.js';

const t = vars('ui-tooltip', {
  bg: sys.color.inverseSurface,
  fg: sys.color.inverseOnSurface,
  font: sys.type.labelSm,
  radius: sys.radius.xs,
  maxWidth: '200px',
  richBg: sys.color.surfaceContainer,
  richFg: sys.color.onSurfaceVariant,
  richRadius: sys.radius.md,
  richMaxWidth: '320px',
});

const styles = css`
  :host { display: inline-block; }
  .panel {
    position: fixed;
    inset-inline-start: 0;
    inset-block-start: 0;
    z-index: ${sys.z.tooltip};
    inline-size: max-content;
  }
  .plain {
    background: ${t.bg};
    color: ${t.fg};
    font: ${t.font};
    letter-spacing: ${sys.tracking.labelSm};
    border-radius: ${t.radius};
    padding: ${sys.space(1)} ${sys.space(2)};
    max-inline-size: ${t.maxWidth};
    pointer-events: none;
  }
  .rich {
    background: ${t.richBg};
    color: ${t.richFg};
    font: ${sys.type.bodyMd};
    letter-spacing: ${sys.tracking.bodyMd};
    border-radius: ${t.richRadius};
    box-shadow: ${sys.elevation[2]};
    padding: ${sys.space(3)};
    max-inline-size: ${t.richMaxWidth};
  }
`;

define('ui-tooltip', {
  props: { text: '', position: 'top', delay: 500, rich: false },
  styles: [base, styles],
  setup({ text, position: pos, delay, rich }, host) {
    const open = signal(false);
    let timer = 0;
    let stopAuto = null;

    const show = () => open.set(true);
    const hide = () => {
      clearTimeout(timer);
      timer = 0;
      open.set(false);
    };

    host.addEventListener('pointerenter', () => {
      clearTimeout(timer);
      const d = Math.max(0, delay());
      if (d) timer = setTimeout(show, d);
      else show();
    });
    host.addEventListener('pointerleave', hide);
    host.addEventListener('focusin', show); // no delay on keyboard focus
    host.addEventListener('focusout', hide);

    // Escape hides even while focus is elsewhere on the page.
    effect(() => {
      if (!open()) return;
      const onKey = (e) => { if (e.key === 'Escape') hide(); };
      document.addEventListener('keydown', onKey, true);
      return () => document.removeEventListener('keydown', onKey, true);
    });

    // Anchor tracking lives exactly as long as the panel does.
    effect(() => {
      if (!open()) {
        stopAuto?.();
        stopAuto = null;
      }
    });
    onCleanup(() => {
      clearTimeout(timer);
      stopAuto?.();
      stopAuto = null;
    });

    const anchor = (el) => {
      stopAuto?.();
      stopAuto = autoUpdate(el, host, {
        placement: `${pos()}-center`,
        offset: 8,
      });
    };

    const view = () => html`
      <div part="panel" role="tooltip"
           class=${() => `panel ${rich() ? 'rich' : 'plain'}`}
           ref=${anchor}>
        ${() => (rich() ? html`<slot name="content"></slot>` : html`${text}`)}
      </div>`;

    return html`
      <slot></slot>
      ${presence(open, view, {
        enter: fx.fadeIn,
        exit: fx.fadeOut,
        enterDuration: 'short2',
        exitDuration: 'short2',
      })}`;
  },
});

export const tag = 'ui-tooltip';
export const themeVars = t;
