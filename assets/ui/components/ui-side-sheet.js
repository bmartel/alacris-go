// <ui-side-sheet> — a Material side sheet for complementary content.
//
//   <ui-side-sheet open=${open} @close=${() => open(false)}>
//     <span slot="headline">Filters</span>
//     Sheet body
//     <ui-button slot="actions" variant="text">Apply</ui-button>
//   </ui-side-sheet>
//
// Distinct from <ui-drawer> (navigation) and <ui-sheet> (bottom). Modal
// (default): a scrim plus a panel that slides in from the end edge. Focus is
// trapped and page scroll locked while open. The PARENT owns `open`. Standard:
// an in-flow panel that animates its inline size — no scrim, no trap.
//
// @prop  {boolean} open=false
// @prop  {string}  variant='modal' — modal | standard
// @prop  {string}  anchor='end'    — start | end
// @prop  {boolean} persistent=false — Escape/scrim/swipe do not request closing
// @prop  {boolean} swipable=true   — swipe in anchor direction to dismiss
// @prop  {string}  label=''         — accessible name if no headline slot
// @event close  — detail: { reason: 'esc' | 'scrim' | 'swipe' | 'method' }
// @event opened — enter animation finished
// @event closed — exit animation finished, DOM removed
// @slot  (default) — body content
// @slot  headline
// @slot  actions
// @part  scrim, surface, headline, body, actions
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, effect, onCleanup, signal } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { presence } from '../motion/presence.js';
import { animate, fx, releaseFill } from '../motion/animate.js';
import { createSwipeTracker, rubberBand } from '../motion/gesture.js';
import { focusTrap, scrollLock } from '../util/focus.js';
import './ui-icon-button.js';

const t = vars('ui-side-sheet', {
  bg: sys.color.surfaceContainerLow,
  fg: sys.color.onSurface,
  width: 'min(400px, 90vw)',
  radius: sys.radius.lg,
  scrim: `color-mix(in srgb, ${sys.color.scrim} 32%, transparent)`,
});

const styles = css`
  :host { display: contents; }
  .overlay {
    position: fixed;
    inset: 0;
    z-index: ${sys.z.modal};
  }
  .scrim { position: absolute; inset: 0; background: ${t.scrim}; }
  .surface {
    position: absolute;
    inset-block: 0;
    display: flex;
    flex-direction: column;
    inline-size: ${t.width};
    background: ${t.bg};
    color: ${t.fg};
    box-shadow: ${sys.elevation[1]};
    touch-action: pan-y;
  }
  .surface.start {
    inset-inline-start: 0;
    border-start-end-radius: ${t.radius};
    border-end-end-radius: ${t.radius};
  }
  .surface.end {
    inset-inline-end: 0;
    border-start-start-radius: ${t.radius};
    border-end-start-radius: ${t.radius};
  }
  .header {
    display: flex;
    align-items: center;
    gap: ${sys.space(1)};
    min-block-size: 64px;
    padding-inline: ${sys.space(4)} ${sys.space(2)};
    user-select: none;
  }
  .headline {
    flex: 1;
    font: ${sys.type.titleLg};
    letter-spacing: ${sys.tracking.titleLg};
  }
  .body {
    flex: 1;
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
    display: block;
    inline-size: 0;
    overflow: hidden;
    background: ${t.bg};
    color: ${t.fg};
    transition: inline-size ${sys.duration.medium2} ${sys.easing.emphasized};
  }
  .std.open { inline-size: ${t.width}; }
  .std-inner { inline-size: ${t.width}; }
`;

define('ui-side-sheet', {
  props: { open: false, variant: 'modal', anchor: 'end', persistent: false, swipable: true, label: '' },
  styles: [base, styles],
  setup({ open, variant, anchor, persistent, swipable, label }, host) {
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

    const slideIn = () => (anchor.peek() === 'start' ? fx.slideInLeft : fx.slideInRight);
    const slideOut = () => (anchor.peek() === 'start' ? fx.slideOutLeft : fx.slideOutRight);

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
      }
    });
    effect(() => {
      if (!open() && surfaceEl?.isConnected) {
        animate(surfaceEl, slideOut(), { duration: 'short4', easing: 'emphasizedAccelerate' });
      }
    });
    onCleanup(() => {
      document.removeEventListener('keydown', onDocKeydown, true);
      releaseTrap?.();
      unlock?.();
      tracker?.destroy();
    });

    const surfaceRef = (el) => {
      surfaceEl = el;
      releaseFill(animate(el, slideIn(), { duration: 'medium2', easing: 'emphasizedDecelerate' }));

      tracker?.destroy();
      tracker = createSwipeTracker(el, {
        axis: 'x',
        threshold: 8,
        filter(e) {
          if (!swipable() || persistent() || variant() === 'standard') return false;
          return true;
        },
        onStart() {
          el.style.transition = 'none';
        },
        onMove({ dx }) {
          const isEnd = anchor.peek() === 'end';
          let effectiveDx = dx;
          if (isEnd) {
            if (dx < 0) effectiveDx = rubberBand(dx, 0.2);
          } else {
            if (dx > 0) effectiveDx = rubberBand(dx, 0.2);
          }
          el.style.transform = `translateX(${effectiveDx}px)`;
          const w = el.offsetWidth || 300;
          const progress = Math.min(1, Math.max(0, Math.abs(effectiveDx) / w));
          if (scrimEl) scrimEl.style.opacity = String(1 - progress * 0.7);
        },
        onEnd({ dx, vx, cancelled }) {
          const isEnd = anchor.peek() === 'end';
          const w = el.offsetWidth || 300;
          const dismissDirection = isEnd ? (vx > 0.4 || dx > w * 0.35) : (vx < -0.4 || dx < -w * 0.35);
          const shouldDismiss = !cancelled && dismissDirection;

          if (shouldDismiss) {
            const targetTransform = isEnd ? 'translateX(100%)' : 'translateX(-100%)';
            const remaining = Math.max(0, w - Math.abs(dx));
            const ms = Math.min(300, Math.max(120, Math.round(remaining / (Math.max(Math.abs(vx), 0.8)))));
            if (scrimEl) {
              animate(scrimEl, fx.fadeOut, { duration: ms, easing: 'emphasizedAccelerate' });
            }
            const anim = animate(el, [
              { transform: el.style.transform || `translateX(${dx}px)` },
              { transform: targetTransform },
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
              { transform: el.style.transform || `translateX(${dx}px)` },
              { transform: 'translateX(0)' },
            ], { duration: 'short4', easing: 'emphasizedDecelerate' }).finished.then(() => {
              if (el) el.style.transform = '';
            });
          }
        },
      });
    };

    const hasSlot = (el, set) => {
      const sync = () => set(el.assignedElements().length > 0);
      el.addEventListener('slotchange', sync);
      sync();
    };

    const modalView = () => html`
      <div class="overlay">
        <div class="scrim" part="scrim" aria-hidden="true"
             ref=${(el) => { scrimEl = el; }}
             @click=${() => requestClose('scrim')}></div>
        <aside class=${() => `surface ${anchor()}`} part="surface" role="dialog" aria-modal="true"
               aria-labelledby=${() => (hasHeadline() ? 'headline' : null)}
               aria-label=${() => (hasHeadline() ? null : (label() || 'Side sheet'))}
               tabindex="-1" ref=${surfaceRef}>
          <div class="header">
            <div class="headline" part="headline" id="headline"
                 ref=${(el) => hasSlot(el.querySelector('slot'), (has) => hasHeadline.set(has))}>
              <slot name="headline"></slot>
            </div>
            <ui-icon-button icon="close" label="Close" @click=${() => requestClose('method')}></ui-icon-button>
          </div>
          <div class="body" part="body"><slot></slot></div>
          <div class="actions" part="actions"
               ref=${(el) => hasSlot(el.querySelector('slot'), (has) => el.classList.toggle('has', has))}>
            <slot name="actions"></slot>
          </div>
        </aside>
      </div>`;

    return html`
      ${() =>
        variant() === 'standard'
          ? html`<aside class=${() => `std ${anchor()}${open() ? ' open' : ''}`} ?inert=${() => !open()} part="surface"
                        role="region"
                        aria-labelledby=${() => (hasHeadline() ? 'headline' : null)}
                        aria-label=${() => (hasHeadline() ? null : (label() || 'Side sheet'))}>
              <div class="std-inner">
                <div class="header">
                  <div class="headline" part="headline" id="headline"
                       ref=${(el) => hasSlot(el.querySelector('slot'), (has) => hasHeadline.set(has))}>
                    <slot name="headline"></slot>
                  </div>
                </div>
                <div class="body" part="body"><slot></slot></div>
              </div>
            </aside>`
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

export const tag = 'ui-side-sheet';
export const themeVars = t;

