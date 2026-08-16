// presence — enter/exit animation for conditional templates.
//
// Alacris removes a conditional subtree the instant its condition turns false,
// which is correct — and too soon for an exit animation. `presence` gives a
// condition a lifecycle: mount + enter animation when it becomes truthy, exit
// animation *then* unmount when it becomes falsy.
//
//   setup({ open }) {
//     return html`
//       <div class="host">
//         ${presence(open, () => html`<div class="sheet">…</div>`, {
//           enter: fx.slideInUp,
//           exit: fx.slideOutDown,
//         })}
//       </div>`;
//   }
//
// The returned node is a `display: contents` anchor, so it is invisible to
// layout. Call `presence` in a child position during `setup` (or under an
// active `root()`), because it creates an effect that must be owned.

import { effect, onCleanup, render } from '@alacris/core';
import { animate, fx } from './animate.js';

/**
 * presence(when, view, opts)
 *
 *   when  — signal/thunk; truthy shows the view
 *   view  — () => template (rebuilt fresh on every entry)
 *   opts  — enter, exit: keyframes or false to disable (default fade)
 *           enterDuration/exitDuration, enterEasing/exitEasing: token names
 *           target: CSS selector for the element to animate (default: the
 *                   view's first element)
 *           onEntered: called after the enter animation finishes
 *           onExited: called after the exit animation finishes
 */
export function presence(when, view, opts = {}) {
  const anchor = document.createElement('span');
  anchor.style.display = 'contents';

  const enterFx = opts.enter === false ? null : opts.enter || fx.fadeIn;
  const exitFx = opts.exit === false ? null : opts.exit || fx.fadeOut;
  const target = () => (opts.target ? anchor.querySelector(opts.target) : anchor.firstElementChild);

  let dispose = null; // active render
  let exiting = null; // in-flight exit animation

  effect(() => {
    const show = !!when();
    if (show) {
      // Re-entering while the exit plays: cut it short and remount fresh.
      if (exiting) {
        exiting.cancel();
        exiting = null;
        dispose?.();
        dispose = null;
      }
      if (!dispose) {
        dispose = render(view(), anchor);
        const el = target();
        if (el && enterFx) {
          const anim = animate(el, enterFx, {
            duration: opts.enterDuration ?? 'medium2',
            easing: opts.enterEasing ?? 'emphasizedDecelerate',
          });
          anim.finished.then(() => {
            try { anim.cancel(); } catch {}
            opts.onEntered?.();
          }, () => {});
        } else if (opts.onEntered) {
          opts.onEntered();
        }
      }
    } else if (dispose && !exiting) {
      const el = target();
      const done = () => {
        if (!exiting) return; // re-entry already tore this down
        exiting = null;
        dispose?.();
        dispose = null;
        opts.onExited?.();
      };
      if (el && exitFx) {
        exiting = animate(el, exitFx, {
          duration: opts.exitDuration ?? 'short4',
          easing: opts.exitEasing ?? 'emphasizedAccelerate',
        });
        exiting.finished.then(done, () => {});
      } else {
        dispose();
        dispose = null;
      }
    }
  });

  onCleanup(() => {
    exiting?.cancel();
    exiting = null;
    dispose?.();
    dispose = null;
  });

  return anchor;
}
