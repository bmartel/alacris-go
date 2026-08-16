// ripple — the Material press ripple.
//
// Attach it to any positioned element (the conventions' `.control` recipe sets
// `position: relative`); it appends a clipped overlay and expands a wave from
// the pointer on press, from the center on keyboard activation. Colors ride
// `currentColor` at the pressed-state opacity token, so the ripple is always
// legible on its surface and themes with everything else. Reduced motion gets
// an instant press tint instead of a wave (handled inside `animate`).
//
//   html`<button class="control" ref=${(el) => ripple(el, { disabled })}>…`

import { animate, settled } from './animate.js';

/**
 * ripple(el, { disabled, centered })
 *
 *   disabled — boolean or thunk/signal; suppresses new ripples while truthy
 *   centered — always expand from the center (icon buttons, switches)
 *
 * Idempotent per element. Returns the element for ref-chaining.
 */
export function ripple(el, opts = {}) {
  if (el.__uiRipple) return el;
  el.__uiRipple = 1;

  const holder = document.createElement('span');
  holder.setAttribute('aria-hidden', 'true');
  Object.assign(holder.style, {
    position: 'absolute',
    inset: '0',
    overflow: 'hidden',
    borderRadius: 'inherit',
    pointerEvents: 'none',
  });
  el.append(holder);

  const off = () =>
    (typeof opts.disabled === 'function' ? !!opts.disabled() : !!opts.disabled) || el.disabled;

  let release = null;

  const spawn = (clientX, clientY, centered) => {
    const rect = el.getBoundingClientRect();
    const w = rect.width || 48;
    const h = rect.height || 48;
    const cx = centered ? w / 2 : clientX - rect.left;
    const cy = centered ? h / 2 : clientY - rect.top;
    // Soft-edge padding matches material-web: the wave slightly overruns the
    // container so the edge never looks clipped at rest.
    const radius = Math.hypot(Math.max(cx, w - cx), Math.max(cy, h - cy)) + 10;

    const dot = document.createElement('span');
    Object.assign(dot.style, {
      position: 'absolute',
      left: `${cx - radius}px`,
      top: `${cy - radius}px`,
      width: `${radius * 2}px`,
      height: `${radius * 2}px`,
      borderRadius: '50%',
      background: 'currentColor',
      opacity: 'var(--ui-state-pressed, 0.1)',
      willChange: 'transform, opacity',
    });
    holder.append(dot);

    const pressedAt = typeof performance !== 'undefined' ? performance.now() : Date.now();
    // material-web: grow from 20% of the final size over 450ms (long1) with
    // standard easing; stay visible at least 225ms (short4 + a beat).
    const grow = animate(dot, [{ transform: 'scale(0.2)' }, { transform: 'scale(1)' }], {
      duration: 'long1',
      easing: 'standard',
    });

    let released = false;
    release = () => {
      if (released) return;
      released = true;
      const elapsed = (typeof performance !== 'undefined' ? performance.now() : Date.now()) - pressedAt;
      const rest = Math.max(0, 225 - elapsed);
      const fade = () => {
        const out = animate(dot, [{ opacity: 'var(--ui-state-pressed, 0.1)' }, { opacity: '0' }], {
          duration: 'medium1',
          easing: 'linear',
        });
        settled(out).then(() => dot.remove());
      };
      Promise.all([settled(grow), rest ? new Promise((r) => setTimeout(r, rest)) : null]).then(fade);
    };
  };

  el.addEventListener('pointerdown', (e) => {
    if (off() || e.button !== 0) return;
    spawn(e.clientX, e.clientY, opts.centered);
    const done = () => release?.();
    window.addEventListener('pointerup', done, { once: true });
    window.addEventListener('pointercancel', done, { once: true });
  });

  el.addEventListener('keydown', (e) => {
    if (off() || e.repeat) return;
    if (e.key === 'Enter' || e.key === ' ') {
      spawn(0, 0, true);
      window.addEventListener('keyup', () => release?.(), { once: true });
    }
  });

  return el;
}
