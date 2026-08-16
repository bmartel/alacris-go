// withFlip — animate a list reorder with the FLIP technique.
//
// Alacris's `each` moves the DOM nodes that changed position, synchronously.
// Wrap the mutation and every child that moved glides to its new place;
// children that appeared fade in. (Removed children are already gone by the
// time we can look — exits inside a hot list are a deliberate non-goal; use
// `presence` for individually dismissable items.)
//
//   withFlip(listEl, () => state.rows.sort(byName));
//
// Keyed by node identity, which is exactly what `each` keeps stable.

import { animate, fx } from './animate.js';

/**
 * withFlip(container, mutate, { duration, easing, stagger })
 * Returns whatever `mutate` returns.
 */
export function withFlip(container, mutate, opts = {}) {
  const before = new Map();
  for (const el of container.children) before.set(el, el.getBoundingClientRect());

  const result = mutate();

  let i = 0;
  for (const el of container.children) {
    const was = before.get(el);
    const delay = opts.stagger ? i++ * opts.stagger : 0;
    if (!was) {
      animate(el, fx.fadeIn, { duration: opts.duration ?? 'medium1', delay, fill: 'backwards' });
      continue;
    }
    const now = el.getBoundingClientRect();
    const dx = was.left - now.left;
    const dy = was.top - now.top;
    if (dx || dy) {
      animate(el, [{ transform: `translate(${dx}px, ${dy}px)` }, { transform: 'none' }], {
        duration: opts.duration ?? 'medium3',
        easing: opts.easing ?? 'emphasized',
        delay,
        fill: 'backwards',
      });
    }
  }
  return result;
}
