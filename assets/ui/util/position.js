// Anchored positioning for popups (menu, select, tooltip, autocomplete).
//
// The panel is `position: fixed`, measured, placed relative to the anchor's
// viewport rect, flipped to the side with more room when it would overflow,
// and sized to that side so it never has to be dragged over the anchor.
// `autoUpdate` keeps it glued through scroll and resize.
// ~90 lines instead of a positioning dependency, because the components only
// need the four sides, alignment, flip and shift.

const MAIN = { top: 'top', bottom: 'top', left: 'left', right: 'left' };
const OPPOSITE = { top: 'bottom', bottom: 'top', left: 'right', right: 'left' };
const ORIGIN = {
  top: 'bottom center',
  bottom: 'top center',
  left: 'center right',
  right: 'center left',
};

/**
 * position(panel, anchor, { placement = 'bottom-start', offset = 4,
 *                           flip = true, matchWidth = false, padding = 8 })
 *
 * placement: side[-alignment] — side: top|bottom|left|right,
 * alignment: start|center|end (start = aligned to the anchor's leading edge).
 * Writes `left`/`top` on the panel and returns { placement } (the side may
 * have flipped). Uses layout size (`offsetWidth`/`offsetHeight`) so an enter
 * transform does not shrink the first measurement.
 */
export function position(panel, anchor, opts = {}) {
  const { placement = 'bottom-start', offset = 4, flip = true, matchWidth = false, padding = 8 } = opts;
  const a = anchor.getBoundingClientRect();
  if (matchWidth) panel.style.minInlineSize = `${a.width}px`;

  // Drop a previous constraint so a later scroll/resize can grow the panel.
  panel.style.maxHeight = '';
  panel.style.maxWidth = '';

  const vw = window.innerWidth;
  const vh = window.innerHeight;
  let [side, align = 'start'] = placement.split('-');
  const alongX = () => side === 'top' || side === 'bottom';
  const size = () => ({ w: panel.offsetWidth, h: panel.offsetHeight });

  let { w, h } = size();
  const space = {
    bottom: vh - a.bottom - offset - padding,
    top: a.top - offset - padding,
    right: vw - a.right - offset - padding,
    left: a.left - offset - padding,
  };
  const needed = () => (alongX() ? h : w);
  if (flip && space[side] < needed() && space[OPPOSITE[side]] > space[side]) {
    side = OPPOSITE[side];
  }

  if (alongX()) panel.style.maxHeight = `${Math.max(0, space[side])}px`;
  else panel.style.maxWidth = `${Math.max(0, space[side])}px`;
  ({ w, h } = size());

  let x, y;
  if (alongX()) {
    y = side === 'bottom' ? a.bottom + offset : a.top - offset - h;
    x = align === 'start' ? a.left : align === 'end' ? a.right - w : a.left + a.width / 2 - w / 2;
  } else {
    x = side === 'right' ? a.right + offset : a.left - offset - w;
    y = align === 'start' ? a.top : align === 'end' ? a.bottom - h : a.top + a.height / 2 - h / 2;
  }

  // Shift on the cross axis only — never drag the panel across the anchor.
  if (alongX()) x = Math.max(padding, Math.min(x, vw - w - padding));
  else y = Math.max(padding, Math.min(y, vh - h - padding));

  panel.style.left = `${x}px`;
  panel.style.top = `${y}px`;
  panel.style.transformOrigin = ORIGIN[side];
  return { placement: align ? `${side}-${align}` : side, [MAIN[side]]: true };
}

/**
 * Keep a panel positioned while it is open. Returns a stop function.
 * Repositions on scroll anywhere (capture), resize, and anchor size change.
 * One rAF pass catches the first layout after a `ref` fires pre-insertion.
 */
export function autoUpdate(panel, anchor, opts) {
  const run = () => {
    if (!panel.isConnected) return;
    position(panel, anchor, opts);
  };
  run();
  const raf = typeof requestAnimationFrame === 'function' ? requestAnimationFrame(run) : 0;
  window.addEventListener('scroll', run, { capture: true, passive: true });
  window.addEventListener('resize', run, { passive: true });
  const ro = typeof ResizeObserver === 'function' ? new ResizeObserver(run) : null;
  ro?.observe(anchor);
  ro?.observe(panel);
  return () => {
    if (raf) cancelAnimationFrame(raf);
    window.removeEventListener('scroll', run, { capture: true });
    window.removeEventListener('resize', run);
    ro?.disconnect();
  };
}
