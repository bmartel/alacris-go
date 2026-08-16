// <ui-table-footer> — pagination and row-count for <ui-table>.
// Composes <ui-pagination> for the page window and a rows-per-page <select>.
//
//   <ui-table-footer page=${page} page-size="10" row-count=${n}
//                    @page=${(e) => page(e.detail.page)}></ui-table-footer>
//
// @prop  {number}  page=1
// @prop  {number}  pageSize=0       — 0 hides the pager and shows a total only
// @prop  {number}  rowCount=0
// @prop  {Array}   pageSizeOptions=[] — default 5, 10, 25
// @prop  {boolean} disabled=false
// @prop  {string}  label='Table pagination'
// @event page — page or page-size changed; detail: { page, pageSize }
// @part  footer, range
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, computed } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base, focusRingOn } from './base.js';
import './ui-pagination.js';
import './ui-icon.js';

const t = vars('ui-table-footer', {
  height: '52px',
  fg: sys.color.onSurfaceVariant,
  borderColor: sys.color.outlineVariant,
});

const styles = css`
  :host { display: block; }
  .footer {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    flex-wrap: wrap;
    gap: ${sys.space(4)};
    min-block-size: ${t.height};
    padding-inline: ${sys.space(4)};
    border-block-start: 1px solid ${t.borderColor};
    color: ${t.fg};
    font: ${sys.type.bodySm};
    letter-spacing: ${sys.tracking.bodySm};
  }
  .page-size {
    display: inline-flex;
    align-items: center;
    gap: ${sys.space(2)};
  }
  .select-wrap {
    position: relative;
    display: inline-flex;
    align-items: center;
    --ui-icon-size: 1.25rem;
  }
  .page-size select {
    appearance: none;
    margin: 0;
    padding: ${sys.space(1)} ${sys.space(6)} ${sys.space(1)} ${sys.space(2)};
    border: none;
    border-radius: ${sys.radius.xs};
    background: ${sys.color.surfaceContainerHighest};
    color: ${sys.color.onSurface};
    font: ${sys.type.bodyMd};
    letter-spacing: ${sys.tracking.bodyMd};
    cursor: pointer;
  }
  .select-wrap ui-icon {
    position: absolute;
    inset-inline-end: ${sys.space(1)};
    pointer-events: none;
    color: ${sys.color.onSurfaceVariant};
  }
  ${focusRingOn('.page-size select')}
  .range { font: ${sys.type.bodySm}; letter-spacing: ${sys.tracking.bodySm}; }
  ui-pagination { flex: none; }
`;

define('ui-table-footer', {
  props: {
    page: 1,
    pageSize: 0,
    rowCount: 0,
    pageSizeOptions: [],
    disabled: false,
    label: 'Table pagination',
  },
  styles: [base, styles],
  setup({ page, pageSize, rowCount, pageSizeOptions, disabled, label }, host) {
    const size = computed(() => Math.max(0, Math.floor(pageSize()) || 0));
    const total = computed(() => Math.max(0, Math.floor(rowCount()) || 0));
    const pages = computed(() => {
      const s = size();
      if (!s) return 1;
      return Math.max(1, Math.ceil(total() / s));
    });
    const current = computed(() =>
      Math.min(Math.max(1, Math.floor(page()) || 1), pages()));
    const choices = computed(() => {
      const o = pageSizeOptions();
      return o && o.length ? o : [5, 10, 25];
    });
    const rangeText = computed(() => {
      const n = total();
      const s = size();
      if (!s) return `Total rows: ${n}`;
      if (!n) return '0 of 0';
      const start = (current() - 1) * s + 1;
      const end = Math.min(current() * s, n);
      return `${start}–${end} of ${n}`;
    });

    const emitPage = (nextPage, nextSize = size()) => {
      host.emit('page', { page: nextPage, pageSize: nextSize });
    };

    return html`
      <div class="footer" part="footer">
        ${() => (size() ? html`
          <label class="page-size">
            Rows per page
            <span class="select-wrap">
              <select aria-label="Rows per page" ?disabled=${disabled}
                      .value=${() => String(size())}
                      @change=${(e) => emitPage(1, +e.target.value)}>
                ${() => choices().map((n) => html`<option value=${n}>${n}</option>`)}
              </select>
              <ui-icon name="arrow-drop-down"></ui-icon>
            </span>
          </label>` : null)}
        <span class="range" part="range">${rangeText}</span>
        ${() => (size() ? html`
          <ui-pagination page=${current} count=${pages} disabled=${disabled}
                         label=${label}
                         @change=${(e) => { e.stopPropagation(); emitPage(e.detail.page); }}>
          </ui-pagination>` : null)}
      </div>`;
  },
});

export const tag = 'ui-table-footer';
export const themeVars = t;
