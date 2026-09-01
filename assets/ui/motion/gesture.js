// Gesture — fluid pointer drag and velocity tracking for Material gestures.
//
// Features:
//   - Pointer event coordination with pointer capture and directional locking.
//   - Rolling-window velocity calculation (px/ms) for natural inertia flicks.
//   - Resistance / rubber-band calculation for out-of-bounds dragging.
//   - Safe cleanup when elements disconnect.

/**
 * Calculate velocity from a rolling history of pointer positions.
 * @param {Array<{ x: number, y: number, t: number }>} history
 * @param {number} now
 * @param {number} windowMs
 * @returns {{ vx: number, vy: number }} in px/ms
 */
export function calculateVelocity(history, now = Date.now(), windowMs = 120) {
  if (!history || history.length < 2) return { vx: 0, vy: 0 };
  const cutoff = now - windowMs;
  const recent = history.filter((p) => p.t >= cutoff);
  if (recent.length < 2) return { vx: 0, vy: 0 };

  const first = recent[0];
  const last = recent[recent.length - 1];
  const dt = Math.max(1, last.t - first.t);
  return {
    vx: (last.x - first.x) / dt,
    vy: (last.y - first.y) / dt,
  };
}

/**
 * Apply logarithmic / exponential rubber-band damping.
 * @param {number} delta Current displacement
 * @param {number} factor Damping factor
 * @returns {number} Damped displacement
 */
export function rubberBand(delta, factor = 0.25) {
  return delta * factor;
}

/**
 * createSwipeTracker(element, options)
 *
 * Tracks 1D or 2D drag gestures with directional locking and velocity calculation.
 *
 * Options:
 *   - axis: 'x' | 'y' | 'both' (default 'both')
 *   - threshold: minimum drag distance before locking (default 6)
 *   - filter: (e) => boolean (return false to ignore pointerdown)
 *   - onStart: ({ x, y, event }) => void
 *   - onMove: ({ dx, dy, x, y, vx, vy, event }) => void
 *   - onEnd: ({ dx, dy, vx, vy, event, cancelled }) => void
 *
 * Returns: { destroy() }
 */
export function createSwipeTracker(element, opts = {}) {
  const {
    axis = 'both',
    threshold = 6,
    filter = null,
    onStart = null,
    onMove = null,
    onEnd = null,
  } = opts;

  let activePointerId = null;
  let startX = 0;
  let startY = 0;
  let isTracking = false;
  let isLocked = false;
  let history = [];

  const onPointerDown = (e) => {
    if (activePointerId !== null) return;
    if (e.button !== 0 && e.isPrimary === false) return;
    if (typeof filter === 'function' && !filter(e)) return;

    activePointerId = e.pointerId;
    startX = e.clientX;
    startY = e.clientY;
    isTracking = true;
    isLocked = false;
    history = [{ x: e.clientX, y: e.clientY, t: Date.now() }];

    element.addEventListener('pointermove', onPointerMove);
    element.addEventListener('pointerup', onPointerUp);
    element.addEventListener('pointercancel', onPointerCancel);
  };

  const onPointerMove = (e) => {
    if (!isTracking || e.pointerId !== activePointerId) return;

    const currentX = e.clientX;
    const currentY = e.clientY;
    const dx = currentX - startX;
    const dy = currentY - startY;
    const now = Date.now();

    history.push({ x: currentX, y: currentY, t: now });
    if (history.length > 20) history.shift();

    if (!isLocked) {
      const absX = Math.abs(dx);
      const absY = Math.abs(dy);

      if (axis === 'x') {
        if (absY > absX && absY > threshold) {
          // Vertical movement dominates -> cancel horizontal swipe
          cleanupListeners();
          return;
        }
        if (absX >= threshold) {
          isLocked = true;
          try { element.setPointerCapture(activePointerId); } catch {}
          onStart?.({ x: currentX, y: currentY, event: e });
        }
      } else if (axis === 'y') {
        if (absX > absY && absX > threshold) {
          // Horizontal movement dominates -> cancel vertical swipe
          cleanupListeners();
          return;
        }
        if (absY >= threshold) {
          isLocked = true;
          try { element.setPointerCapture(activePointerId); } catch {}
          onStart?.({ x: currentX, y: currentY, event: e });
        }
      } else {
        if (absX >= threshold || absY >= threshold) {
          isLocked = true;
          try { element.setPointerCapture(activePointerId); } catch {}
          onStart?.({ x: currentX, y: currentY, event: e });
        }
      }
    }

    if (isLocked) {
      if (e.cancelable) e.preventDefault();
      const { vx, vy } = calculateVelocity(history, now);
      onMove?.({ dx, dy, x: currentX, y: currentY, vx, vy, event: e });
    }
  };

  const onPointerUp = (e) => {
    if (!isTracking || e.pointerId !== activePointerId) return;
    const dx = e.clientX - startX;
    const dy = e.clientY - startY;
    const now = Date.now();
    history.push({ x: e.clientX, y: e.clientY, t: now });
    const { vx, vy } = calculateVelocity(history, now);
    const wasLocked = isLocked;

    cleanupListeners();

    if (wasLocked) {
      onEnd?.({ dx, dy, vx, vy, event: e, cancelled: false });
    }
  };

  const onPointerCancel = (e) => {
    if (!isTracking || e.pointerId !== activePointerId) return;
    const dx = e.clientX - startX;
    const dy = e.clientY - startY;
    const now = Date.now();
    const { vx, vy } = calculateVelocity(history, now);
    const wasLocked = isLocked;

    cleanupListeners();

    if (wasLocked) {
      onEnd?.({ dx, dy, vx, vy, event: e, cancelled: true });
    }
  };

  const cleanupListeners = () => {
    if (activePointerId !== null) {
      try { element.releasePointerCapture(activePointerId); } catch {}
    }
    element.removeEventListener('pointermove', onPointerMove);
    element.removeEventListener('pointerup', onPointerUp);
    element.removeEventListener('pointercancel', onPointerCancel);
    activePointerId = null;
    isTracking = false;
    isLocked = false;
    history = [];
  };

  element.addEventListener('pointerdown', onPointerDown);

  return {
    destroy() {
      cleanupListeners();
      element.removeEventListener('pointerdown', onPointerDown);
    },
  };
}
