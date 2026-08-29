// Focus management for overlays: trap, restore, and scroll lock.

const FOCUSABLE = [
  'a[href]', 'button:not([disabled])', 'input:not([disabled]):not([type=hidden])',
  'select:not([disabled])', 'textarea:not([disabled])', '[tabindex]:not([tabindex="-1"])',
  'audio[controls]', 'video[controls]', '[contenteditable]:not([contenteditable="false"])',
].join(',');

/**
 * Focusable elements under a host, in a useful order: the host's shadow root
 * first (chrome like close buttons), then its light DOM (slotted content).
 * Custom elements in the light DOM (a slotted <ui-button>) don't match native
 * selectors, so we look one shadow level into each and return the inner
 * focusable — that is the node that actually takes focus.
 */
export function focusables(host) {
  const out = [];
  if (host.shadowRoot) out.push(...host.shadowRoot.querySelectorAll(FOCUSABLE));
  for (const el of host.querySelectorAll('*')) {
    if (el.matches?.(FOCUSABLE)) out.push(el);
    else if (el.shadowRoot) {
      const inner = el.shadowRoot.querySelector(FOCUSABLE);
      if (inner) out.push(inner);
    }
  }
  return out.filter((el) => el.getClientRects === undefined || el.getClientRects().length > 0 || el.offsetParent !== null);
}

/**
 * Trap Tab focus inside `host` and remember what was focused before.
 * Returns a release function that removes the trap and restores focus.
 *
 *   const release = focusTrap(host, { initial: shadowCloseButton });
 */
export function focusTrap(host, { initial, restore } = {}) {
  const previous = restore || document.activeElement;

  const first = initial || focusables(host)[0] || host;
  first.focus?.();

  const onKeydown = (e) => {
    if (e.key !== 'Tab') return;
    const items = focusables(host);
    if (!items.length) { e.preventDefault(); return; }
    const head = items[0];
    const tail = items[items.length - 1];
    // getRootNode().activeElement sees into the shadow root.
    const active = host.shadowRoot?.activeElement || document.activeElement;
    if (e.shiftKey && (active === head || active === host)) {
      e.preventDefault();
      tail.focus();
    } else if (!e.shiftKey && active === tail) {
      e.preventDefault();
      head.focus();
    }
  };

  host.addEventListener('keydown', onKeydown);
  return () => {
    host.removeEventListener('keydown', onKeydown);
    previous?.focus?.();
  };
}

// Ref-counted scroll lock, with scrollbar-width compensation so the page does
// not shift when the bar disappears.
let locks = 0;
let saved = null;

export function scrollLock() {
  if (++locks === 1) {
    const el = document.documentElement;
    const gap = window.innerWidth - el.clientWidth;
    saved = { overflow: el.style.overflow, pad: el.style.paddingInlineEnd };
    el.style.overflow = 'hidden';
    if (gap > 0) el.style.paddingInlineEnd = `${gap}px`;
  }
  let released = false;
  return () => {
    if (released) return;
    released = true;
    if (--locks === 0 && saved) {
      const el = document.documentElement;
      el.style.overflow = saved.overflow;
      el.style.paddingInlineEnd = saved.pad;
      saved = null;
    }
  };
}
