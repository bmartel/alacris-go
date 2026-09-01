// <ui-drawer> — a navigation drawer, modal or standard.
//
//   <ui-drawer open=${open} @close=${() => open(false)}>…nav content…</ui-drawer>
//   <ui-drawer variant="standard" anchor="start" open=${open}>…</ui-drawer>
//
// Modal: a fixed overlay — scrim plus a full-height panel that slides in from
// the anchor side. Focus is trapped and page scroll locked while open. The
// PARENT owns `open`: Escape and scrim clicks emit `close` with a reason and
// the parent flips the signal. `opened`/`closed` fire after the enter/exit
// animations settle.
// Standard: an in-flow panel; the host animates its inline size open/closed —
// no scrim, no trap.
//
// @prop  {boolean} open=false
// @prop  {string}  variant='modal' — modal | standard
// @prop  {string}  anchor='start'  — start | end (which edge it slides from)
// @prop  {boolean} persistent=false — Escape/scrim/swipe do not request closing
// @prop  {boolean} swipable=true   — swipe in anchor direction to dismiss
// @prop  {string}  label=''        — accessible name; falls back to "Navigation"
// @event close  — modal dismissed; detail: { reason: 'esc' | 'scrim' | 'swipe' }
// @event opened — modal enter animation finished
// @event closed — modal exit animation finished, DOM removed
// @slot  (default) — drawer content
// @part  surface, scrim
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, effect, onCleanup } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { presence } from '../motion/presence.js';
import { animate, fx, releaseFill } from '../motion/animate.js';
import { createSwipeTracker, rubberBand } from '../motion/gesture.js';
import { focusTrap, scrollLock } from '../util/focus.js';

const t = vars('ui-drawer', {
  bg: sys.color.surfaceContainerLow,
  stdBg: sys.color.surface,
  fg: sys.color.onSurface,
  width: 'min(360px, 80vw)',
  radius: sys.radius.lg,
  pad: sys.space(3),
  scrim: `color-mix(in srgb, ${sys.color.scrim} 32%, transparent)`,
});

const styles = css`
  :host { display: block; inline-size: fit-content; }
  .overlay {
    position: fixed;
    inset: 0;
    z-index: ${sys.z.drawer};
  }
  .scrim { position: absolute; inset: 0; background: ${t.scrim}; }
  .surface {
    position: absolute;
    inset-block: 0;
    inline-size: ${t.width};
    background: ${t.bg};
    color: ${t.fg};
    padding: ${t.pad};
    overflow-y: auto;
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
  .std {
    inline-size: 0;
    block-size: 100%;
    overflow: hidden;
    background: ${t.stdBg};
    color: ${t.fg};
    transition: inline-size ${sys.duration.medium2} ${sys.easing.emphasized};
  }
  .std.open { inline-size: ${t.width}; }
  .std.start {
    border-start-end-radius: ${t.radius};
    border-end-end-radius: ${t.radius};
  }
  .std.end {
    border-start-start-radius: ${t.radius};
    border-end-start-radius: ${t.radius};
  }
  .std .inner { padding: ${t.pad}; inline-size: ${t.width}; }
`;

define('ui-drawer', {
  props: { open: false, variant: 'modal', anchor: 'start', persistent: false, swipable: true, label: '' },
  styles: [base, styles],
  setup({ open, variant, anchor, persistent, swipable, label }, host) {
    let releaseTrap = null;
    let unlock = null;
    let surfaceEl = null;
    let scrimEl = null;
    let tracker = null;

    const requestClose = (reason) => {
      if (reason !== 'method' && persistent()) return;
      host.emit('close', { reason });
    };

    const slideIn = () => (anchor.peek() === 'start' ? fx.slideInLeft : fx.slideInRight);
    const slideOut = () => (anchor.peek() === 'start' ? fx.slideOutLeft : fx.slideOutRight);

    // Escape must work wherever focus is, so listen at the document while
    // the modal drawer is open.
    const onDocKeydown = (e) => {
      if (e.key === 'Escape') requestClose('esc');
    };

    // Trap focus + lock scroll exactly while the modal drawer is open.
    let closingViaSwipe = false;
    let prevActive = null;
    effect(() => {
      if (open() && variant() === 'modal') {
        closingViaSwipe = false;
        prevActive = document.activeElement;
        document.addEventListener('keydown', onDocKeydown, true);
        if (!unlock) unlock = scrollLock();
        queueMicrotask(() => {
          if (open.peek() && !releaseTrap) releaseTrap = focusTrap(host, { restore: prevActive });
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
    onCleanup(() => {
      document.removeEventListener('keydown', onDocKeydown, true);
      releaseTrap?.();
      unlock?.();
      tracker?.destroy();
    });

    // The panel slides out while the presence overlay (scrim included) fades.
    effect(() => {
      if (!open() && surfaceEl?.isConnected && !closingViaSwipe) {
        animate(surfaceEl, slideOut(), { duration: 'short4', easing: 'emphasizedAccelerate' });
      }
    });

    const surfaceRef = (el) => {
      surfaceEl = el;
      releaseFill(animate(el, slideIn(), { duration: 'medium2', easing: 'emphasizedDecelerate' }));

      tracker?.destroy();
      tracker = createSwipeTracker(el, {
        axis: 'x',
        threshold: 8,
        filter(e) {
          if (!swipable() || persistent() || variant() !== 'modal') return false;
          return true;
        },
        onStart() {
          el.getAnimations?.()?.forEach((a) => a.cancel());
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
            closingViaSwipe = true;
            const targetTransform = isEnd ? 'translateX(100%)' : 'translateX(-100%)';
            const remaining = Math.max(0, w - Math.abs(dx));
            const ms = Math.min(300, Math.max(120, Math.round(remaining / (Math.max(Math.abs(vx), 0.8)))));
            if (scrimEl) {
              animate(scrimEl, fx.fadeOut, { duration: ms, easing: 'emphasizedAccelerate' });
            }
            animate(el, [
              { transform: el.style.transform || `translateX(${dx}px)` },
              { transform: targetTransform },
            ], { duration: ms, easing: 'emphasizedAccelerate', fill: 'forwards' });
            requestClose('swipe');
          } else {
            if (scrimEl) {
              const scrimSnap = animate(scrimEl, [{ opacity: scrimEl.style.opacity || '0.5' }, { opacity: 1 }], {
                duration: 'short4', easing: 'emphasizedDecelerate',
              });
              scrimSnap.finished.then(() => {
                try { scrimSnap.cancel(); } catch {}
                if (scrimEl) scrimEl.style.opacity = '';
              });
            }
            const snapAnim = animate(el, [
              { transform: el.style.transform || `translateX(${dx}px)` },
              { transform: 'translateX(0)' },
            ], { duration: 'short4', easing: 'emphasizedDecelerate' });
            snapAnim.finished.then(() => {
              try { snapAnim.cancel(); } catch {}
              if (el) el.style.transform = '';
            });
          }
        },
      });
    };

    const overlay = () => html`
      <div class="overlay">
        <div class="scrim" part="scrim" aria-hidden="true"
             ref=${(el) => { scrimEl = el; }}
             @click=${() => requestClose('scrim')}></div>
        <aside class=${() => `surface ${anchor()}`} part="surface" role="dialog" aria-modal="true"
               aria-label=${() => label() || 'Navigation'} tabindex="-1" ref=${surfaceRef}>
          <slot></slot>
        </aside>
      </div>`;

    return html`
      ${() =>
        variant() === 'standard'
          ? html`<aside class=${() => `std ${anchor()}${open() ? ' open' : ''}`} ?inert=${() => !open()} part="surface"
                        aria-label=${() => label() || 'Navigation'}>
              <div class="inner"><slot></slot></div>
            </aside>`
          : null}
      ${presence(() => open() && variant() === 'modal', overlay, {
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

export const tag = 'ui-drawer';
export const themeVars = t;
