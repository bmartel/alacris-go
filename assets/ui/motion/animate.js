// Motion — one ergonomic layer over the Web Animations API.
//
// Principles:
//   - Durations and easings come from the motion tokens, so JS-driven motion
//     obeys the theme's `motion.scale` exactly like CSS transitions do —
//     `animate(el, fx.fadeIn, { duration: 'medium2' })` resolves the token at
//     call time.
//   - `prefers-reduced-motion` is honored everywhere: animations jump to their
//     end state instead of playing. CSS transitions get the same guard from
//     `base.js`.
//   - No timeouts, no rAF bookkeeping: WAAPI runs off the main thread where
//     the browser can manage it, and `.finished` sequences the rest.

import { DURATIONS, EASINGS } from '../tokens/system.js';

const camelToToken = (s) => s.replace(/([A-Z]|\d+$)/g, (c) => '-' + c.toLowerCase());

/** True when the user asked for reduced motion (checked live). */
export const prefersReducedMotion = () =>
  typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches;

/**
 * Resolve a duration token ('medium2', 'extraLong1') to milliseconds, reading
 * the live CSS token so a theme's motion scale applies. Numbers pass through.
 */
export function duration(key) {
  if (typeof key === 'number') return key;
  const token = camelToToken(key);
  if (typeof getComputedStyle === 'function') {
    const raw = getComputedStyle(document.documentElement).getPropertyValue(`--ui-duration-${token}`);
    const ms = parseFloat(raw);
    if (!Number.isNaN(ms)) return ms;
  }
  return DURATIONS[token] ?? 200;
}

/** Resolve an easing token ('standard', 'emphasizedDecelerate') to its curve. */
export const easing = (key) => EASINGS[camelToToken(key)] || key;

/**
 * animate(el, keyframes, { duration: 'short4'|ms, easing: 'standard'|curve,
 *                          fill: 'both', ...WAAPI options })
 *
 * Returns the Animation. Under reduced motion the animation still runs (so
 * `.finished` chains keep working) but with ~zero duration — the element
 * simply lands in its final state.
 */
export function animate(el, keyframes, opts = {}) {
  const { duration: d = 'short4', easing: e = 'standard', fill = 'both', ...rest } = opts;
  // Environments without WAAPI (simulated DOMs in tests) get an
  // already-finished stand-in so `.finished` sequencing still works.
  if (typeof el?.animate !== 'function') {
    return { finished: Promise.resolve(), cancel() {}, finish() {}, play() {}, pause() {} };
  }
  const ms = prefersReducedMotion() ? 1 : duration(d);
  const run = () => {
    const frames = typeof keyframes === 'function' ? keyframes(el) : keyframes;
    const anim = el.animate(frames, { duration: ms, easing: easing(e), fill, ...rest });
    anim.finished.catch(() => {}); // cancellation is normal control flow, not an error
    // Hidden/occluded tabs freeze the animation timeline, which would leave
    // `.finished` chains (presence unmounts, `closed` events, snackbar queues)
    // waiting forever. setTimeout still ticks there, so force-finish on the
    // wall clock; in a visible tab the animation wins the race and this no-ops.
    const delay = Number(rest.delay) || 0;
    const guard = setTimeout(() => {
      try { if (anim.playState === 'running' || anim.pending) anim.finish(); } catch { /* infinite/cancelled */ }
    }, ms + delay + 100);
    anim.finished.then(() => clearTimeout(guard), () => clearTimeout(guard));
    return anim;
  };
  if (el.isConnected) return run();

  // A `ref` fires while the element is still in its render fragment, before
  // insertion. An animation created on a disconnected subtree can wedge with
  // its first keyframe applied but its timeline never advancing — so start on
  // the next frame, once the element is in the document. The facade keeps
  // `.finished`/`.cancel` working either way. If rAF never fires (hidden tab),
  // the wall-clock fallback settles without animating — the element simply
  // rests in its CSS end state.
  let anim = null;
  let cancelled = false;
  let started = false;
  let settle;
  const finished = new Promise((resolve) => (settle = resolve));
  const start = () => {
    if (cancelled || started) return;
    started = true;
    if (!el.isConnected) return settle();
    anim = run();
    anim.finished.then(settle, () => settle());
  };
  requestAnimationFrame(start);
  const fallback = setTimeout(() => (started ? null : (started = true, settle())), ms + 200);
  finished.then(() => clearTimeout(fallback));
  return {
    finished,
    cancel() { cancelled = true; anim?.cancel(); settle(); },
    finish() { anim?.finish(); },
    play() { anim?.play(); },
    pause() { anim?.pause(); },
  };
}

/** Await an animation without throwing when it is cancelled mid-flight. */
export const settled = (anim) => anim.finished.catch(() => {});

// ------------------------------------------------------------------ presets
//
// Keyframe presets tuned to the Material motion spec. Enter presets pair with
// 'emphasizedDecelerate', exits with 'emphasizedAccelerate' — `animate`'s
// defaults are fine for simple fades.

export const fx = {
  fadeIn: [{ opacity: 0 }, { opacity: 1 }],
  fadeOut: [{ opacity: 1 }, { opacity: 0 }],
  // 0.8 is the MD3 container-transform start scale (dialogs, menus).
  scaleIn: [{ opacity: 0, transform: 'scale(0.8)' }, { opacity: 1, transform: 'scale(1)' }],
  scaleOut: [{ opacity: 1, transform: 'scale(1)' }, { opacity: 0, transform: 'scale(0.8)' }],
  slideInUp: [{ opacity: 0, transform: 'translateY(16px)' }, { opacity: 1, transform: 'translateY(0)' }],
  slideOutDown: [{ opacity: 1, transform: 'translateY(0)' }, { opacity: 0, transform: 'translateY(16px)' }],
  slideInDown: [{ opacity: 0, transform: 'translateY(-16px)' }, { opacity: 1, transform: 'translateY(0)' }],
  slideOutUp: [{ opacity: 1, transform: 'translateY(0)' }, { opacity: 0, transform: 'translateY(-16px)' }],
  expandDown: (el) => {
    const h = el?.scrollHeight || 0;
    return [
      { blockSize: '0px', opacity: 0, overflow: 'hidden' },
      { blockSize: `${h}px`, opacity: 1, overflow: 'hidden' },
    ];
  },
  collapseUp: (el) => {
    const h = el?.offsetHeight || el?.scrollHeight || 0;
    return [
      { blockSize: `${h}px`, opacity: 1, overflow: 'hidden' },
      { blockSize: '0px', opacity: 0, overflow: 'hidden' },
    ];
  },
  slideInLeft: [{ transform: 'translateX(-100%)' }, { transform: 'translateX(0)' }],
  slideOutLeft: [{ transform: 'translateX(0)' }, { transform: 'translateX(-100%)' }],
  slideInRight: [{ transform: 'translateX(100%)' }, { transform: 'translateX(0)' }],
  slideOutRight: [{ transform: 'translateX(0)' }, { transform: 'translateX(100%)' }],
  // Bottom-sheet cover: the panel travels its full height, not a 16px nudge.
  sheetIn: [{ transform: 'translateY(100%)' }, { transform: 'translateY(0)' }],
  sheetOut: [{ transform: 'translateY(0)' }, { transform: 'translateY(100%)' }],
  collapse: [{ blockSize: 'var(--ui-collapse-size, auto)', opacity: 1 }, { blockSize: '0', opacity: 0 }],
};
