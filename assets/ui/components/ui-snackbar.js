// <ui-snackbar> — a transient bottom-center message, plus `showSnackbar()`,
// an imperative service that queues messages through one shared instance.
//
// Declarative (parent owns the state):
//
//   <ui-snackbar open=${open} message="Draft saved" action="Undo"
//                @action=${undo} @close=${() => open(false)}></ui-snackbar>
//
// Imperative (fire and forget; FIFO — one visible at a time):
//
//   const { close, closed } = showSnackbar('Message archived', { action: 'Undo' });
//
// The component requests closing by emitting `close` with a reason — the
// PARENT flips `open`. `closed` fires after the exit animation finishes.
//
// @prop  {boolean} open=false
// @prop  {string}  message=''
// @prop  {string}  action=''      — label for a trailing text action button
// @prop  {number}  duration=4000  — auto-dismiss after ms; 0 = sticky
// @prop  {boolean} closeButton=false — trailing close icon button
// @event action — action button pressed
// @event close  — detail: { reason: 'timeout' | 'action' | 'close' | 'method' }
// @event opened — enter animation finished
// @event closed — exit animation finished, DOM removed
// @part  surface, message, action, close
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, effect } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base, focusRingOn } from './base.js';
import { presence } from '../motion/presence.js';
import { fx } from '../motion/animate.js';
import { ripple } from '../motion/ripple.js';
import './ui-icon.js';

const t = vars('ui-snackbar', {
  bg: sys.color.inverseSurface,
  fg: sys.color.inverseOnSurface,
  actionFg: sys.color.inversePrimary,
  radius: sys.radius.xs,
  minWidth: '288px',
  maxWidth: '560px',
  elevation: sys.elevation[3],
});

const styles = css`
  :host { display: contents; }
  .region {
    position: fixed;
    inset-inline: 0;
    inset-block-end: ${sys.space(4)};
    z-index: ${sys.z.snackbar};
    display: flex;
    justify-content: center;
    padding-inline: ${sys.space(4)};
    pointer-events: none;
  }
  .surface {
    pointer-events: auto;
    display: flex;
    align-items: center;
    gap: ${sys.space(1)};
    min-inline-size: min(${t.minWidth}, 100%);
    max-inline-size: ${t.maxWidth};
    min-block-size: 48px;
    padding-block: ${sys.space(1)};
    padding-inline: ${sys.space(4)} ${sys.space(2)};
    background: ${t.bg};
    color: ${t.fg};
    border-radius: ${t.radius};
    box-shadow: ${t.elevation};
    font: ${sys.type.bodyMd};
    letter-spacing: ${sys.tracking.bodyMd};
  }
  .message { flex: 1; padding-block: ${sys.space(2)}; }
  .surface:not(.trailing) { padding-inline-end: ${sys.space(4)}; }

  .action, .close {
    position: relative;
    isolation: isolate;
    flex: none;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: none;
    background: transparent;
    cursor: pointer;
    border-radius: ${sys.radius.full};
  }
  ${focusRingOn('.action')}
  ${focusRingOn('.close')}
  .layer {
    position: absolute; inset: 0; z-index: -1;
    border-radius: inherit; background: currentColor; opacity: 0;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  .action:hover .layer, .close:hover .layer { opacity: ${sys.state.hover}; }
  .action:focus-visible .layer, .close:focus-visible .layer { opacity: ${sys.state.focus}; }
  .action:active .layer, .close:active .layer { opacity: ${sys.state.pressed}; }

  .action {
    block-size: 36px;
    padding-inline: ${sys.space(2)};
    color: ${t.actionFg};
    font: ${sys.type.labelLg};
    letter-spacing: ${sys.tracking.labelLg};
    white-space: nowrap;
  }
  .close {
    inline-size: 32px;
    block-size: 32px;
    padding: 0;
    color: inherit;
    --ui-icon-size: 18px;
  }
`;

define('ui-snackbar', {
  props: { open: false, message: '', action: '', duration: 4000, closeButton: false },
  styles: [base, styles],
  setup({ open, message, action, duration, closeButton }, host) {
    const requestClose = (reason) => host.emit('close', { reason });

    // Auto-dismiss while open; the cleanup clears the pending timer whenever
    // `open`/`duration` change or the element goes away.
    effect(() => {
      if (!open()) return;
      const ms = duration();
      if (!(ms > 0)) return;
      const id = setTimeout(() => requestClose('timeout'), ms);
      return () => clearTimeout(id);
    });

    const view = () => html`
      <div class="region">
        <div part="surface" role="status" aria-live="polite"
             class=${() => `surface${action() || closeButton() ? ' trailing' : ''}`}>
          <span class="message" part="message">${message}</span>
          ${() =>
            action()
              ? html`<button class="action" part="action" type="button"
                        @click=${() => { host.emit('action'); requestClose('action'); }}
                        ref=${(el) => ripple(el)}>
                    <span class="layer" aria-hidden="true"></span>${action}
                  </button>`
              : null}
          ${() =>
            closeButton()
              ? html`<button class="close" part="close" type="button" aria-label="Dismiss"
                        @click=${() => requestClose('close')}
                        ref=${(el) => ripple(el, { centered: true })}>
                    <span class="layer" aria-hidden="true"></span>
                    <ui-icon name="close"></ui-icon>
                  </button>`
              : null}
        </div>
      </div>`;

    return html`${presence(open, view, {
      enter: fx.slideInUp,
      exit: fx.slideOutDown,
      exitDuration: 'short4',
      target: '.surface',
      onEntered: () => host.emit('opened'),
      onExited: () => host.emit('closed'),
    })}`;
  },
});

export const tag = 'ui-snackbar';
export const themeVars = t;

// --------------------------------------------------------------- service
//
// One shared light-DOM <ui-snackbar> appended to <body>; messages queue FIFO
// and the next one shows only after the previous fully exited.

let serviceEl = null; // singleton host
let current = null;   // entry on screen (or animating out)
const queue = [];

function serviceHost() {
  if (!serviceEl || !serviceEl.isConnected) {
    serviceEl = document.createElement('ui-snackbar');
    // The service is the "parent": it owns the open state.
    serviceEl.addEventListener('close', () => { serviceEl.open = false; });
    serviceEl.addEventListener('closed', () => {
      const done = current;
      current = null;
      done?.resolve();
      next();
    });
    document.body.append(serviceEl);
  }
  return serviceEl;
}

function next() {
  if (current || !queue.length) return;
  current = queue.shift();
  const el = serviceHost();
  el.message = current.message;
  el.action = current.action;
  el.duration = current.duration;
  el.closeButton = current.closeButton;
  el.open = true;
}

/**
 * showSnackbar(message, { action, duration, closeButton })
 *
 * Returns { close, closed }: `close()` dismisses this snackbar (or removes it
 * from the queue if it has not shown yet); `closed` resolves once it has fully
 * left the screen.
 */
export function showSnackbar(message, { action = '', duration = 4000, closeButton = false } = {}) {
  let resolve;
  const closed = new Promise((r) => { resolve = r; });
  const entry = { message, action, duration, closeButton, resolve };
  queue.push(entry);
  next();
  return {
    closed,
    close() {
      if (current === entry) {
        serviceEl.open = false; // exit animation → 'closed' → resolve + next
      } else {
        const i = queue.indexOf(entry);
        if (i > -1) { queue.splice(i, 1); entry.resolve(); }
      }
    },
  };
}
