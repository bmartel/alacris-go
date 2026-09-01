// <ui-swipe-row> — an Android-style horizontal swipe-to-reveal row container.
//
//   <ui-swipe-row>
//     <ui-icon-button slot="end" icon="delete" label="Delete" danger></ui-icon-button>
//     <ui-icon-button slot="end" icon="archive" label="Archive"></ui-icon-button>
//     <ui-list-item headline="Swipeable message" supporting="Swipe left to reveal actions"></ui-list-item>
//   </ui-swipe-row>
//
// @prop  {string}  open=''         — '' | 'start' | 'end'
// @prop  {boolean} disabled=false  — disables gesture tracking
// @prop  {number}  threshold=0.4   — fraction of action width to snap open
// @prop  {boolean} fullSwipe=false — swiping past 65% width triggers action
// @event open   — detail: { side: 'start' | 'end' }
// @event close  — detail: {}
// @event action — detail: { side: 'start' | 'end' }
// @slot  (default) — primary row content (e.g. <ui-list-item>, <ui-card>)
// @slot  start — actions revealed when swiping right
// @slot  end   — actions revealed when swiping left
// @part  container, content, start-actions, end-actions
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, effect, onCleanup, signal } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { animate } from '../motion/animate.js';
import { createSwipeTracker, rubberBand } from '../motion/gesture.js';

const t = vars('ui-swipe-row', {
  bg: sys.color.surface,
  actionStartBg: sys.color.primaryContainer,
  actionStartFg: sys.color.onPrimaryContainer,
  actionEndBg: sys.color.errorContainer,
  actionEndFg: sys.color.onErrorContainer,
});

const styles = css`
  :host { display: block; overflow: hidden; isolation: isolate; }
  .container {
    position: relative;
    inline-size: 100%;
    overflow: hidden;
    user-select: none;
    touch-action: pan-y;
  }
  .actions {
    position: absolute;
    inset-block: 0;
    display: flex;
    align-items: center;
    z-index: 0;
  }
  .actions.start {
    inset-inline-start: 0;
    background: ${t.actionStartBg};
    color: ${t.actionStartFg};
  }
  .actions.end {
    inset-inline-end: 0;
    background: ${t.actionEndBg};
    color: ${t.actionEndFg};
  }
  .actions:not(.has) { display: none; }
  .content {
    position: relative;
    z-index: 1;
    background: ${t.bg};
    inline-size: 100%;
    touch-action: pan-y;
  }
  .disabled .content, .disabled .container { touch-action: auto; }
`;

define('ui-swipe-row', {
  props: { open: '', disabled: false, threshold: 0.4, fullSwipe: false },
  styles: [base, styles],
  setup({ open, disabled, threshold, fullSwipe }, host) {
    let containerEl = null;
    let contentEl = null;
    let startActionsEl = null;
    let endActionsEl = null;
    let tracker = null;
    let currentOpen = '';
    let startOffset = 0;

    const hasStart = signal(false);
    const hasEnd = signal(false);

    const getStartWidth = () => (startActionsEl?.offsetWidth || 0);
    const getEndWidth = () => (endActionsEl?.offsetWidth || 0);

    const snapTo = (x, ms = 200, easing = 'emphasizedDecelerate') => {
      if (!contentEl) return Promise.resolve();
      const currentTransform = contentEl.style.transform || 'translateX(0px)';
      const targetTransform = `translateX(${x}px)`;
      const anim = animate(contentEl, [
        { transform: currentTransform },
        { transform: targetTransform },
      ], { duration: ms, easing });
      return anim.finished.then(() => {
        try { anim.cancel(); } catch {}
        if (contentEl) contentEl.style.transform = targetTransform;
      });
    };

    const setOpenState = (side, emitEvent = true) => {
      currentOpen = side;
      if (side === 'end' && hasEnd.peek()) {
        const w = getEndWidth();
        snapTo(-w);
        if (emitEvent) host.emit('open', { side: 'end' });
      } else if (side === 'start' && hasStart.peek()) {
        const w = getStartWidth();
        snapTo(w);
        if (emitEvent) host.emit('open', { side: 'start' });
      } else {
        currentOpen = '';
        snapTo(0);
        if (emitEvent) host.emit('close');
      }
    };

    effect(() => {
      const target = open();
      if (target !== currentOpen) {
        setOpenState(target, false);
      }
    });

    const onDocClick = (e) => {
      if (currentOpen && !host.contains(e.target)) {
        setOpenState('');
      }
    };

    document.addEventListener('click', onDocClick);
    onCleanup(() => {
      document.removeEventListener('click', onDocClick);
      tracker?.destroy();
    });

    const onContentClick = (e) => {
      if (currentOpen !== '') {
        e.preventDefault();
        e.stopPropagation();
        setOpenState('');
      }
    };

    const hasSlot = (el, set) => {
      const sync = () => set(el.assignedElements().length > 0);
      el.addEventListener('slotchange', sync);
      sync();
    };

    const initTracker = (el) => {
      contentEl = el;
      tracker?.destroy();

      tracker = createSwipeTracker(el, {
        axis: 'x',
        threshold: 8,
        filter(e) {
          if (disabled()) return false;
          return true;
        },
        onStart() {
          el.getAnimations?.()?.forEach((a) => a.cancel());
          if (currentOpen === 'end') startOffset = -getEndWidth();
          else if (currentOpen === 'start') startOffset = getStartWidth();
          else startOffset = 0;
          el.style.transition = 'none';
        },
        onMove({ dx }) {
          const totalDx = startOffset + dx;
          const startW = getStartWidth();
          const endW = getEndWidth();
          let effectiveDx = totalDx;

          if (totalDx > 0) {
            if (!hasStart.peek()) effectiveDx = rubberBand(totalDx, 0.15);
            else if (totalDx > startW && !fullSwipe()) {
              effectiveDx = startW + rubberBand(totalDx - startW, 0.25);
            }
          } else if (totalDx < 0) {
            if (!hasEnd.peek()) effectiveDx = rubberBand(totalDx, 0.15);
            else if (-totalDx > endW && !fullSwipe()) {
              effectiveDx = -endW - rubberBand(-totalDx - endW, 0.25);
            }
          }

          el.style.transform = `translateX(${effectiveDx}px)`;
        },
        onEnd({ dx, vx, cancelled }) {
          if (cancelled) {
            setOpenState(currentOpen, false);
            return;
          }

          const totalDx = startOffset + dx;
          const containerW = containerEl?.offsetWidth || 300;
          const startW = getStartWidth() || 80;
          const endW = getEndWidth() || 80;
          const thresh = Number(threshold()) || 0.4;

          // Check full-swipe action
          if (fullSwipe() && Math.abs(totalDx) > containerW * 0.65) {
            const side = totalDx > 0 ? 'start' : 'end';
            const targetX = totalDx > 0 ? containerW : -containerW;
            snapTo(targetX, 150, 'emphasizedAccelerate').then(() => {
              host.emit('action', { side });
              setOpenState('');
            });
            return;
          }

          // Check reveal / snap
          if (totalDx < 0 && hasEnd.peek()) {
            if (vx < -0.4 || -totalDx > endW * thresh) {
              setOpenState('end');
              return;
            }
          } else if (totalDx > 0 && hasStart.peek()) {
            if (vx > 0.4 || totalDx > startW * thresh) {
              setOpenState('start');
              return;
            }
          }

          setOpenState('');
        },
      });
    };

    return html`
      <div class=${() => `container${disabled() ? ' disabled' : ''}`} part="container"
           ref=${(el) => { containerEl = el; }}>
        <div class=${() => `actions start${hasStart() ? ' has' : ''}`} part="start-actions"
             ref=${(el) => { startActionsEl = el; }}>
          <slot name="start" ref=${(el) => hasSlot(el, hasStart.set)}></slot>
        </div>
        <div class=${() => `actions end${hasEnd() ? ' has' : ''}`} part="end-actions"
             ref=${(el) => { endActionsEl = el; }}>
          <slot name="end" ref=${(el) => hasSlot(el, hasEnd.set)}></slot>
        </div>
        <div class="content" part="content"
             ref=${initTracker}
             @click=${onContentClick}>
          <slot></slot>
        </div>
      </div>`;
  },
});

export const tag = 'ui-swipe-row';
export const themeVars = t;
