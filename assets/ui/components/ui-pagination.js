// <ui-pagination> — page navigation with sibling/boundary windows.
//
//   <ui-pagination count="10" page=${page} @change=${(e) => page(e.detail.page)}>
//   </ui-pagination>
//
// @prop  {number}  page=1       — current page (1-based)
// @prop  {number}  count=1      — total pages
// @prop  {number}  siblings=1   — pages shown on each side of the current page
// @prop  {number}  boundaries=1 — pages always shown at each end
// @prop  {boolean} disabled=false
// @prop  {string}  label='Pagination' — accessible name of the <nav>
// @event change — a page was chosen (numbers, prev, next); detail: { page }
// @part  nav, list
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, computed } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base, focusRingOn } from './base.js';
import { ripple } from '../motion/ripple.js';
import './ui-icon.js';

const t = vars('ui-pagination', {
  size: '40px',
  radius: sys.radius.full,
  fg: sys.color.onSurfaceVariant,
  currentBg: sys.color.primary,
  currentFg: sys.color.onPrimary,
  font: sys.type.labelLg,
  tracking: sys.tracking.labelLg,
});

const styles = css`
  :host { display: block; }
  .list {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: ${sys.space(1)};
  }
  .item {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-inline-size: calc(${t.size} + var(--ui-density, 0) * 4px);
    block-size: calc(${t.size} + var(--ui-density, 0) * 4px);
    border-radius: ${t.radius};
    font: ${t.font};
    letter-spacing: ${t.tracking};
    color: ${t.fg};
  }
  button.item {
    position: relative;
    isolation: isolate;
    border: none;
    background: transparent;
    padding: 0;
    cursor: pointer;
    user-select: none;
    --ui-icon-size: 1.25rem;
  }
  ${focusRingOn('button.item')}
  .layer {
    position: absolute; inset: 0; z-index: -1;
    border-radius: inherit;
    background: currentColor;
    opacity: 0;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  button.item:hover .layer { opacity: ${sys.state.hover}; }
  button.item:focus-visible .layer { opacity: ${sys.state.focus}; }
  button.item:active .layer { opacity: ${sys.state.pressed}; }
  button.item.current { background: ${t.currentBg}; color: ${t.currentFg}; }
  button.item:disabled {
    cursor: default;
    pointer-events: none;
    color: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContent} * 100%), transparent);
  }
  button.item.current:disabled {
    background: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContainer} * 100%), transparent);
  }
`;

const range = (start, end) => {
  const out = [];
  for (let i = start; i <= end; i++) out.push(i);
  return out;
};

define('ui-pagination', {
  props: { page: 1, count: 1, siblings: 1, boundaries: 1, disabled: false, label: 'Pagination' },
  styles: [base, styles],
  setup({ page, count, siblings, boundaries, disabled, label }, host) {
    const total = () => Math.max(1, Math.floor(count()) || 1);
    const current = () => Math.min(Math.max(1, Math.floor(page()) || 1), total());

    // The MUI usePagination windowing: boundary pages at both ends, a sibling
    // window around the current page, and ellipses where the gaps are >1.
    const items = computed(() => {
      const c = total();
      const p = current();
      const sib = Math.max(0, siblings());
      const bnd = Math.max(0, boundaries());

      const startPages = range(1, Math.min(bnd, c));
      const endPages = range(Math.max(c - bnd + 1, bnd + 1), c);

      const siblingsStart = Math.max(Math.min(p - sib, c - bnd - sib * 2 - 1), bnd + 2);
      const siblingsEnd = Math.min(
        Math.max(p + sib, bnd + sib * 2 + 2),
        endPages.length > 0 ? endPages[0] - 2 : c - 1,
      );

      return [
        'prev',
        ...startPages,
        ...(siblingsStart > bnd + 2 ? ['start-ellipsis'] : bnd + 1 < c - bnd ? [bnd + 1] : []),
        ...range(siblingsStart, siblingsEnd),
        ...(siblingsEnd < c - bnd - 1 ? ['end-ellipsis'] : c - bnd > bnd ? [c - bnd] : []),
        ...endPages,
        'next',
      ];
    });

    const setPage = (n) => {
      const clamped = Math.min(Math.max(1, n), total());
      if (clamped === current()) return;
      page.set(clamped);
      host.emit('change', { page: clamped });
    };

    return html`
      <nav part="nav" aria-label=${label}>
        <div class="list" part="list">
          ${() => {
            const dis = disabled();
            const p = current();
            const c = total();
            return items().map((it) => {
              if (it === 'prev' || it === 'next') {
                const next = it === 'next';
                return html`
                  <button type="button" class="item"
                          aria-label=${next ? 'Next page' : 'Previous page'}
                          ?disabled=${dis || (next ? p >= c : p <= 1)}
                          @click=${() => setPage(next ? current() + 1 : current() - 1)}
                          ref=${(el) => ripple(el, { disabled, centered: true })}>
                    <span class="layer" aria-hidden="true"></span>
                    <ui-icon name=${next ? 'chevron-right' : 'chevron-left'}></ui-icon>
                  </button>`;
              }
              if (typeof it === 'string') {
                return html`<span class="item ellipsis" aria-hidden="true">…</span>`;
              }
              const isCurrent = it === p;
              return html`
                <button type="button" class=${isCurrent ? 'item current' : 'item'}
                        aria-label=${`Page ${it}`}
                        aria-current=${isCurrent ? 'page' : null}
                        ?disabled=${dis}
                        @click=${() => setPage(it)}
                        ref=${(el) => ripple(el, { disabled, centered: true })}>
                  <span class="layer" aria-hidden="true"></span>
                  ${it}
                </button>`;
            });
          }}
        </div>
      </nav>`;
  },
});

export const tag = 'ui-pagination';
export const themeVars = t;
