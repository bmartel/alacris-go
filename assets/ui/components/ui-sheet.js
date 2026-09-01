// <ui-sheet> — a Material bottom sheet.
//
//   <ui-sheet open=${open} @close=${() => open(false)}>
//     <span slot="headline">Title</span>
//     Sheet body
//     <ui-button slot="actions" variant="text">Close</ui-button>
//   </ui-sheet>
//
// Modal (default): a scrim plus a panel that slides up from the bottom.
// Focus is trapped and page scroll locked while open. The PARENT owns
// `open`: Escape and scrim clicks emit `close` with a reason.
// Standard: an in-flow panel that expands from zero height — no scrim,
// no trap.
//
// @prop  {boolean} open=false
// @prop  {string}  variant='modal' — modal | standard
// @prop  {boolean} persistent=false — Escape/scrim/swipe do not request closing
// @prop  {boolean} swipable=true   — swipe down to dismiss
// @prop  {string}  label=''         — accessible name if no headline slot
// @event close  — detail: { reason: 'esc' | 'scrim' | 'swipe' | 'method' }
// @event opened — enter animation finished
// @event closed — exit animation finished, DOM removed
// @slot  (default) — body content
// @slot  headline
// @slot  actions
// @part  scrim, surface, handle, headline, body, actions
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, effect, onCleanup, signal } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { presence } from '../motion/presence.js';
import { animate, fx, releaseFill } from '../motion/animate.js';
import { createSwipeTracker, rubberBand } from '../motion/gesture.js';
import { focusTrap, scrollLock } from '../util/focus.js';

const t = vars('ui-sheet', {
  bg: sys.color.surfaceContainerLow,
  fg: sys.color.onSurface,
  radius: sys.radius.xl,
  width: 'min(640px, 100%)',
  handle: sys.color.outline,
  scrim: `color-mix(in srgb, ${sys.color.scrim} 32%, transparent)`,
});

const styles = css`
  :host { display: contents; }
  .overlay {
    position: fixed;
    inset: 0;
    z-index: ${sys.z.modal};
    display: flex;
    align-items: flex-end;
    justify-content: center;
  }
  .scrim { position: absolute; inset: 0; background: ${t.scrim}; }
  .surface {
    position: relative;
    display: flex;
    flex-direction: column;
    inline-size: ${t.width};
    max-block-size: calc(100vh - 72px);
    background: ${t.bg};
    color: ${t.fg};
    border-start-start-radius: ${t.radius};
    border-start-end-radius: ${t.radius};
    box-shadow: ${sys.elevation[3]};
    touch-action: pan-x;
  }
  .handle {
    flex: none;
    inline-size: 32px;
    block-size: 4px;
    margin: ${sys.space(4)} auto ${sys.space(2)};
    border-radius: ${sys.radius.full};
    background: ${t.handle};
    cursor: grab;
    touch-action: none;
  }
  .headline {
    padding-inline: ${sys.space(6)};
    padding-block-end: ${sys.space(2)};
    font: ${sys.type.titleLg};
    letter-spacing: ${sys.tracking.titleLg};
    user-select: none;
  }
  .headline:not(.has) { display: none; }
  .body {
    overflow: auto;
    padding: ${sys.space(2)} ${sys.space(6)} ${sys.space(6)};
    font: ${sys.type.bodyMd};
    letter-spacing: ${sys.tracking.bodyMd};
    color: ${sys.color.onSurfaceVariant};
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    gap: ${sys.space(2)};
    padding: ${sys.space(2)} ${sys.space(4)} ${sys.space(4)};
  }
  .actions:not(.has) { display: none; }

  .std {
    display: grid;
    grid-template-rows: 0fr;
    background: ${t.bg};
    color: ${t.fg};
    border-start-start-radius: ${t.radius};
    border-start-end-radius: ${t.radius};
    overflow: hidden;
    transition: grid-template-rows ${sys.duration.medium2} ${sys.easing.emphasized};
  }
  .std.open { grid-template-rows: 1fr; }
  .std-inner { min-block-size: 0; overflow: hidden; }
`;

define('ui-sheet', {
  props: { open: false, variant: 'modal', persistent: false, swipable: true, label: '' },
  styles: [base, styles],
  setup({ open, variant, persistent, swipable, label }, host) {
    let releaseTrap = null;
    let unlock = null;
    let surfaceEl = null;
    let scrimEl = null;
    let tracker = null;
    const hasHeadline = signal(false);

    const requestClose = (reason) => {
      if (reason !== 'method' && persistent()) return;
      host.emit('close', { reason });
    };

    const onDocKeydown = (e) => {
      if (e.key === 'Escape') requestClose('esc');
    };

    let prevActive = null;
    effect(() => {
      if (open() && variant() !== 'standard') {
        prevActive = document.activeElement;
        document.addEventListener('keydown', onDocKeydown, true);
        unlock = scrollLock();
        queueMicrotask(() => {
          if (open() && !releaseTrap) releaseTrap = focusTrap(host, { restore: prevActive });
        });
      } else {
        document.removeEventListener('keydown', onDocKeydown, true);
        releaseTrap?.();
        releaseTrap = null;
        prevActive = null;
        unlock?.();
        unlock = null;
        if (surfaceEl?.isConnected) {
          animate(surfaceEl, fx.sheetOut, { duration: 'short4', easing: 'emphasizedAccelerate' });
        }
      }
    });
    onCleanup(() => {
      document.removeEventListener('keydown', onDocKeydown, true);
      releaseTrap?.();
      unlock?.();
      tracker?.destroy();
    });

    const hasSlot = (el, set) => {
      const sync = () => set(el.assignedElements().length > 0);
      el.addEventListener('slotchange', sync);
      sync();
    };

    const surfaceRef = (el) => {
      surfaceEl = el;
      releaseFill(animate(el, fx.sheetIn, { duration: 'medium2', easing: 'emphasizedDecelerate' }));

      tracker?.destroy();
      tracker = createSwipeTracker(el, {
        axis: 'y',
        threshold: 8,
        filter(e) {
          if (!swipable() || persistent() || variant() === 'standard') return false;
          const body = el.querySelector('.body');
          if (body && body.contains(e.target) && body.scrollTop > 0) {
            return false;
          }
          return true;
        },
        onStart() {
          el.style.transition = 'none';
        },
        onMove({ dy }) {
          if (dy < 0) {
            const damped = rubberBand(dy, 0.2);
            el.style.transform = `translateY(${damped}px)`;
            if (scrimEl) scrimEl.style.opacity = '1';
          } else {
            el.style.transform = `translateY(${dy}px)`;
            const h = el.offsetHeight || 300;
            const progress = Math.min(1, Math.max(0, dy / h));
            if (scrimEl) scrimEl.style.opacity = String(1 - progress * 0.7);
          }
        },
        onEnd({ dy, vy, cancelled }) {
          const h = el.offsetHeight || 300;
          const shouldDismiss = !cancelled && (vy > 0.4 || dy > h * 0.35);

          if (shouldDismiss) {
            const remaining = Math.max(0, h - dy);
            const ms = Math.min(300, Math.max(120, Math.round(remaining / (Math.max(vy, 0.8)))));
            if (scrimEl) {
              animate(scrimEl, fx.fadeOut, { duration: ms, easing: 'emphasizedAccelerate' });
            }
            const anim = animate(el, [
              { transform: el.style.transform || `translateY(${dy}px)` },
              { transform: 'translateY(100%)' },
            ], { duration: ms, easing: 'emphasizedAccelerate' });
            anim.finished.then(() => {
              requestClose('swipe');
            });
          } else {
            if (scrimEl) {
              animate(scrimEl, [{ opacity: scrimEl.style.opacity || '0.5' }, { opacity: 1 }], {
                duration: 'short4', easing: 'emphasizedDecelerate',
              }).finished.then(() => { if (scrimEl) scrimEl.style.opacity = ''; });
            }
            animate(el, [
              { transform: el.style.transform || `translateY(${dy}px)` },
              { transform: 'translateY(0)' },
            ], { duration: 'short4', easing: 'emphasizedDecelerate' }).finished.then(() => {
              if (el) el.style.transform = '';
            });
          }
        },
      });
    };

    const modalView = () => html`
      <div class="overlay">
        <div class="scrim" part="scrim" aria-hidden="true"
             ref=${(el) => { scrimEl = el; }}
             @click=${() => requestClose('scrim')}></div>
        <div class="surface" part="surface" role="dialog" aria-modal="true"
             aria-labelledby=${() => (hasHeadline() ? 'headline' : null)}
             aria-label=${() => (hasHeadline() ? null : (label() || 'Sheet'))}
             tabindex="-1"
             ref=${surfaceRef}>
          <div class="handle" part="handle" aria-hidden="true"></div>
          <div class="headline" part="headline" id="headline"
               ref=${(el) => hasSlot(el.querySelector('slot'), (has) => {
                 hasHeadline.set(has);
                 el.classList.toggle('has', has);
               })}>
            <slot name="headline"></slot>
          </div>
          <div class="body" part="body"><slot></slot></div>
          <div class="actions" part="actions"
               ref=${(el) => hasSlot(el.querySelector('slot'), (has) => el.classList.toggle('has', has))}>
            <slot name="actions"></slot>
          </div>
        </div>
      </div>`;

    return html`
      ${() =>
        variant() === 'standard'
          ? html`<div class=${() => `std${open() ? ' open' : ''}`} ?inert=${() => !open()} part="surface" role="region"
                      aria-label=${() => label() || 'Sheet'}>
              <div class="std-inner">
                <div class="handle" part="handle" aria-hidden="true"></div>
                <div class="body" part="body"><slot></slot></div>
              </div>
            </div>`
          : null}
      ${presence(() => open() && variant() !== 'standard', modalView, {
        enter: fx.fadeIn,
        exit: fx.fadeOut,
        exitDuration: 'short4',
        onEntered: () => host.emit('opened'),
        onExited: () => {
          tracker?.destroy();
          host.emit('closed');
        },
      })}`;
  },
});

export const tag = 'ui-sheet';
export const themeVars = t;

