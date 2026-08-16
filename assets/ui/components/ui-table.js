// <ui-table> — Material data-table / DataGrid.
//
// Markup mode — pass a native <table> (adopted into the shadow so cells can
// be themed). `data-sortable` / `data-numeric` decorate headers.
//
//   <ui-table label="Nutrition">
//     <table>…</table>
//   </ui-table>
//
// Data mode — `columns` + `rows`. The grid is `role="table"` (Alacris cannot
// bind inside native <tr>). Pipeline (filter → sort → group → paginate →
// aggregate) lives in `util/table.js`. Chrome is <ui-table-toolbar> and
// <ui-table-footer>, composed from search, menus, checkboxes, and pagination.
//
//   <ui-table headline="Trades" columns=${cols} rows=${rows}
//             selectable="multiple" group-by="commodity"
//             quick-filter column-menu density-menu csv-export
//             page-size="10"></ui-table>
//
// Column objects: { key, label, numeric, sortable, width, hidden, align,
//   aggregate: 'sum'|'avg'|'min'|'max'|'count', render(row, col), sortValue(row) }
//
// @prop  {string}  label=''
// @prop  {string}  headline=''
// @prop  {string}  supporting=''
// @prop  {string}  variant='outlined' — outlined | standard
// @prop  {string}  density='standard' — compact | standard | comfortable
// @prop  {boolean} dense=false        — alias for density="compact"
// @prop  {boolean} stickyHeader=false
// @prop  {boolean} stickyFirst=false
// @prop  {boolean} striped=false
// @prop  {boolean} loading=false
// @prop  {string}  maxHeight=''
// @prop  {string}  selectable='none'  — none | single | multiple
// @prop  {Array}   selected=[]
// @prop  {Array}   columns=[]
// @prop  {Array}   rows=[]
// @prop  {object}  getRowId=null      — (row, index) => id
// @prop  {string}  sortBy=''
// @prop  {string}  sortDir='asc'      — asc | desc
// @prop  {string}  sortMode='client'  — client | server
// @prop  {string}  filter=''
// @prop  {string}  groupBy=''         — column key to group rows
// @prop  {Array}   expandedGroups=[]  — empty means all groups expanded
// @prop  {Array}   hiddenColumns=[]
// @prop  {number}  page=1
// @prop  {number}  pageSize=0         — 0 shows every row
// @prop  {number}  rowCount=0
// @prop  {string}  paginationMode='client' — client | server
// @prop  {Array}   pageSizeOptions=[]
// @prop  {boolean} quickFilter=false
// @prop  {boolean} columnMenu=false
// @prop  {boolean} densityMenu=false
// @prop  {boolean} csvExport=false
// @prop  {string}  csvFileName='table.csv'
// @prop  {string}  emptyText='No results'
// @event change            — selection; detail: { selected }
// @event sort              — detail: { key, dir }
// @event page              — detail: { page, pageSize }
// @event filter            — detail: { value }
// @event density           — detail: { density }
// @event column-visibility — detail: { hidden }
// @event group             — a group was toggled; detail: { key, expanded, expandedGroups }
// @event row-click         — detail: { id, row }
// @event export            — CSV produced; detail: { format, filename }
// @slot  (default) — native <table> (markup mode)
// @slot  toolbar   — replaces the default <ui-table-toolbar>
// @slot  headline, supporting, actions — projected into the default toolbar
// @slot  footer    — replaces the default <ui-table-footer>
// @slot  empty
// @part  container, table
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, computed, effect, each, signal } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base, focusRingOn } from './base.js';
import { ripple } from '../motion/ripple.js';
import {
  processTable, makeIdOf, formatAgg, toCsv, downloadText, compare,
} from '../util/table.js';
import './ui-table-toolbar.js';
import './ui-table-footer.js';
import './ui-checkbox.js';
import './ui-icon.js';
import './ui-progress.js';

const t = vars('ui-table', {
  rowHeight: '52px',
  denseRowHeight: '40px',
  comfortableRowHeight: '56px',
  checkCol: '52px',
  borderColor: sys.color.outlineVariant,
  headerFg: sys.color.onSurfaceVariant,
  headerBg: sys.color.surface,
  fg: sys.color.onSurface,
  bg: sys.color.surface,
  hoverBg: sys.color.surfaceContainerLow,
  selectedBg: `color-mix(in srgb, ${sys.color.primary} 8%, ${sys.color.surface})`,
  stripeBg: sys.color.surfaceContainerLow,
  groupBg: sys.color.surfaceContainerLow,
  aggFg: sys.color.primary,
  radius: sys.radius.md,
});

const styles = css`
  :host { display: block; }
  .root {
    display: flex;
    flex-direction: column;
    position: relative;
    background: ${t.bg};
    color: ${t.fg};
  }
  .root.outlined {
    border: 1px solid ${t.borderColor};
    border-radius: ${t.radius};
    overflow: hidden;
  }
  .loading {
    position: absolute;
    inset-inline: 0;
    inset-block-start: 0;
    z-index: 2;
  }
  .scroller { overflow: auto; max-block-size: var(--ui-table-max-height, none); }

  /* Markup mode is a real <table>. Data mode is display:table — Alacris
     cannot put bindings as children of <tr>/<tbody> (the parser drops them). */
  table, .grid {
    inline-size: 100%;
    border-collapse: collapse;
    color: ${t.fg};
  }
  .grid { display: table; }
  .thead, .tbody, .tfoot { display: table-row-group; }
  .tr { display: table-row; }
  caption {
    position: absolute;
    inline-size: 1px; block-size: 1px; overflow: hidden;
    clip: rect(0 0 0 0); clip-path: inset(50%); white-space: nowrap;
  }
  tr, .tr { block-size: calc(${t.rowHeight} + var(--ui-density, 0) * 4px); }
  :host(.compact) tr, :host(.compact) .tr,
  :host([dense]) tr, :host([dense]) .tr {
    block-size: calc(${t.denseRowHeight} + var(--ui-density, 0) * 4px);
  }
  :host(.comfortable) tr, :host(.comfortable) .tr {
    block-size: calc(${t.comfortableRowHeight} + var(--ui-density, 0) * 4px);
  }
  th, td, .th, .td {
    display: table-cell;
    padding-inline: ${sys.space(4)};
    border-block-end: 1px solid ${t.borderColor};
    white-space: nowrap;
    vertical-align: middle;
  }
  tbody tr:last-child td, .tbody .tr:last-child .td { border-block-end: none; }
  th, .th {
    font: ${sys.type.labelLg};
    letter-spacing: ${sys.tracking.labelLg};
    color: ${t.headerFg};
    text-align: start;
    background: ${t.headerBg};
  }
  td, .td {
    font: ${sys.type.bodyMd};
    letter-spacing: ${sys.tracking.bodyMd};
  }
  .numeric, [data-numeric] { text-align: end; font-variant-numeric: tabular-nums; }
  .center { text-align: center; }
  .end { text-align: end; }
  .agg-label {
    display: block;
    font: ${sys.type.labelSm};
    letter-spacing: ${sys.tracking.labelSm};
    color: ${sys.color.onSurfaceVariant};
    text-transform: lowercase;
  }

  tbody tr, .tbody .tr {
    transition: background-color ${sys.duration.short2} ${sys.easing.standard};
  }
  tbody tr:hover, .tbody .tr:hover { background: ${t.hoverBg}; }
  tbody tr[aria-selected="true"], .tbody .tr[aria-selected="true"] { background: ${t.selectedBg}; }
  tbody tr[aria-selected="true"]:hover, .tbody .tr[aria-selected="true"]:hover {
    background: color-mix(in srgb, ${sys.color.primary} 12%, ${sys.color.surface});
  }
  :host([striped]) tbody tr:nth-child(even):not([aria-selected="true"]):not(:hover),
  :host([striped]) .tbody .tr:nth-child(even):not(.group):not(.total):not([aria-selected="true"]):not(:hover) {
    background: ${t.stripeBg};
  }
  .tr.group { background: ${t.groupBg}; font: ${sys.type.labelLg}; }
  .tr.group:hover { background: ${t.hoverBg}; }
  .tr.total { font: ${sys.type.labelLg}; color: ${t.aggFg}; }
  .agg { color: ${t.aggFg}; font-variant-numeric: tabular-nums; }

  :host([sticky-header]) thead th, :host([sticky-header]) .thead .th {
    position: sticky; inset-block-start: 0; z-index: 1; background: ${t.headerBg};
  }
  :host([sticky-first]) .pin {
    position: sticky; inset-inline-start: 0; z-index: 1; background: ${t.headerBg};
  }
  :host([sticky-first]) .td.pin { background: ${t.bg}; }
  :host([sticky-first]) .tbody .tr:hover .td.pin { background: ${t.hoverBg}; }
  :host([sticky-first]) .tbody .tr[aria-selected="true"] .td.pin { background: ${t.selectedBg}; }
  :host([sticky-first]) .pin-data { inset-inline-start: ${t.checkCol}; }
  :host([sticky-first]) .pin { box-shadow: 4px 0 8px rgb(var(--ui-shadow-rgb) / 0.08); }

  .check {
    inline-size: ${t.checkCol};
    min-inline-size: ${t.checkCol};
    padding-inline: ${sys.space(1)};
    text-align: center;
  }
  .check ui-checkbox::part(label) {
    position: absolute; inline-size: 1px; block-size: 1px; overflow: hidden;
    clip: rect(0 0 0 0); clip-path: inset(50%); white-space: nowrap;
  }

  button.sort, button.group-toggle {
    position: relative; isolation: isolate;
    display: inline-flex; align-items: center; gap: ${sys.space(1)};
    margin: 0; padding: ${sys.space(1)} ${sys.space(2)};
    border: none; border-radius: ${sys.radius.xs};
    background: transparent; color: inherit; font: inherit;
    letter-spacing: inherit; cursor: pointer;
    --ui-icon-size: 1.125rem;
  }
  button.sort { margin-inline: calc(${sys.space(2)} * -1); }
  ${focusRingOn('button.sort')}
  ${focusRingOn('button.group-toggle')}
  button.sort .layer, button.group-toggle .layer {
    position: absolute; inset: 0; z-index: -1;
    border-radius: inherit; background: currentColor; opacity: 0;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  button.sort:hover .layer, button.group-toggle:hover .layer { opacity: ${sys.state.hover}; }
  button.sort:focus-visible .layer, button.group-toggle:focus-visible .layer { opacity: ${sys.state.focus}; }
  button.sort:active .layer, button.group-toggle:active .layer { opacity: ${sys.state.pressed}; }
  button.sort ui-icon {
    opacity: 0;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard},
                transform ${sys.duration.short4} ${sys.easing.standard};
  }
  .th:hover button.sort ui-icon,
  th:hover button.sort ui-icon,
  button.sort:focus-visible ui-icon,
  [aria-sort="ascending"] button.sort ui-icon,
  [aria-sort="descending"] button.sort ui-icon { opacity: 1; }
  [aria-sort="descending"] button.sort ui-icon { transform: rotate(180deg); }

  .empty {
    text-align: center; white-space: normal;
    color: ${sys.color.onSurfaceVariant};
    font: ${sys.type.bodyMd}; letter-spacing: ${sys.tracking.bodyMd};
    padding-block: ${sys.space(6)};
  }
  .tr.empty-row, .tr.empty-row .empty { display: block; }
  .tr.empty-row .empty { inline-size: 100%; }

  :host([loading]) .scroller { pointer-events: none; opacity: 0.72; }
`;

const cellVal = (tr, idx) => {
  const td = (tr.cells && tr.cells[idx]) || tr.children[idx];
  if (!td) return '';
  if (td.dataset.sort != null) return td.dataset.sort;
  const text = td.textContent.trim();
  const n = +text.replace(/,/g, '');
  return text !== '' && !Number.isNaN(n) ? n : text;
};

const colAlign = (col) => {
  if (col.numeric) return 'numeric';
  if (col.align === 'center') return 'center';
  if (col.align === 'end') return 'end';
  return '';
};

define('ui-table', {
  props: {
    label: '',
    headline: '',
    supporting: '',
    variant: 'outlined',
    density: 'standard',
    dense: false,
    stickyHeader: false,
    stickyFirst: false,
    striped: false,
    loading: false,
    maxHeight: '',
    selectable: 'none',
    selected: [],
    columns: [],
    rows: [],
    getRowId: null,
    sortBy: '',
    sortDir: 'asc',
    sortMode: 'client',
    filter: '',
    groupBy: '',
    expandedGroups: [],
    hiddenColumns: [],
    page: 1,
    pageSize: 0,
    rowCount: 0,
    paginationMode: 'client',
    pageSizeOptions: [],
    quickFilter: false,
    columnMenu: false,
    densityMenu: false,
    csvExport: false,
    csvFileName: 'table.csv',
    emptyText: 'No results',
  },
  styles: [base, styles],
  setup(p, host) {
    const {
      label, headline, supporting, variant, density, dense, stickyHeader, stickyFirst,
      striped, loading, maxHeight, selectable, selected, columns, rows, getRowId,
      sortBy, sortDir, sortMode, filter, groupBy, expandedGroups, hiddenColumns,
      page, pageSize, rowCount, paginationMode, pageSizeOptions,
      quickFilter, columnMenu, densityMenu, csvExport, csvFileName, emptyText,
    } = p;

    const resolvedDensity = computed(() =>
      dense() && density() === 'standard' ? 'compact' : (density() || 'standard'));

    effect(() => host.toggleAttribute('dense', dense()));
    effect(() => {
      host.classList.toggle('compact', resolvedDensity() === 'compact');
      host.classList.toggle('comfortable', resolvedDensity() === 'comfortable');
    });
    effect(() => host.toggleAttribute('sticky-header', stickyHeader()));
    effect(() => host.toggleAttribute('sticky-first', stickyFirst()));
    effect(() => host.toggleAttribute('striped', striped()));
    effect(() => host.toggleAttribute('loading', loading()));
    effect(() => host.setAttribute('variant', variant() || 'outlined'));
    effect(() => {
      const s = selectable() || 'none';
      if (s === 'none') host.removeAttribute('selectable');
      else host.setAttribute('selectable', s);
    });
    effect(() => {
      if (maxHeight()) host.style.setProperty('--ui-table-max-height', maxHeight());
      else host.style.removeProperty('--ui-table-max-height');
    });

    const idOf = makeIdOf(null);
    const idOfLive = (row, i) => {
      const fn = getRowId();
      return typeof fn === 'function' ? fn(row, i) : idOf(row, i);
    };

    const dataMode = computed(() =>
      (columns() && columns().length > 0) || (rows() && rows().length > 0));

    const expanded = computed(() => {
      const v = expandedGroups();
      return v && v.length ? new Set(v) : null;
    });

    const model = computed(() => processTable({
      rows: rows() || [],
      columns: columns() || [],
      hiddenColumns: hiddenColumns() || [],
      filter: filter(),
      sortBy: sortBy(),
      sortDir: sortDir(),
      sortMode: sortMode(),
      groupBy: groupBy(),
      expanded: expanded(),
      page: page(),
      pageSize: pageSize(),
      paginationMode: paginationMode(),
      rowCount: rowCount(),
      idOf: idOfLive,
    }));

    const cols = computed(() => model().cols);
    const visible = computed(() => model().visible);
    const list = computed(() => model().list);
    const totals = computed(() => model().totals);
    const filteredCount = computed(() => model().filteredCount);
    const hasTotals = computed(() => Object.keys(totals()).length > 0 && list().length > 0);

    const multi = computed(() => selectable() === 'multiple');
    const picking = computed(() => selectable() === 'multiple' || selectable() === 'single');
    const selectedSet = computed(() => new Set(selected() || []));
    const pageIds = computed(() =>
      visible().filter((it) => it.type === 'row').map((it) => it.id));
    const allOnPage = computed(() => {
      const ids = pageIds();
      return ids.length > 0 && ids.every((id) => selectedSet().has(id));
    });
    const someOnPage = computed(() => {
      const ids = pageIds();
      const n = ids.filter((id) => selectedSet().has(id)).length;
      return n > 0 && n < ids.length;
    });

    const emitSelected = (next) => {
      selected.set(next);
      host.emit('change', { selected: next });
    };
    const toggleId = (id) => {
      if (selectable() === 'single') {
        emitSelected(selectedSet().has(id) ? [] : [id]);
        return;
      }
      const next = new Set(selected() || []);
      if (next.has(id)) next.delete(id); else next.add(id);
      emitSelected([...next]);
    };
    const toggleAll = () => {
      if (allOnPage()) {
        const drop = new Set(pageIds());
        emitSelected((selected() || []).filter((id) => !drop.has(id)));
      } else {
        const next = new Set(selected() || []);
        for (const id of pageIds()) next.add(id);
        emitSelected([...next]);
      }
    };

    const applySort = (key) => {
      let dir;
      if (sortBy() !== key) dir = 'asc';
      else if (sortDir() === 'asc') dir = 'desc';
      else dir = 'none';
      sortBy.set(dir === 'none' ? '' : key);
      if (dir !== 'none') sortDir.set(dir);
      host.emit('sort', { key: dir === 'none' ? '' : key, dir });
    };

    const toggleGroup = (key) => {
      const groups = model().items.filter((it) => it.type === 'group').map((g) => g.key);
      const current = expanded();
      const set = new Set(current ? current : groups);
      const open = !set.has(key);
      if (open) set.add(key); else set.delete(key);
      expandedGroups.set([...set]);
      host.emit('group', { key, expanded: open, expandedGroups: [...set] });
    };

    const firstKey = computed(() => cols()[0]?.key);
    const cellContent = (row, col) => {
      if (typeof col.render === 'function') return col.render(row, col);
      const v = row == null ? '' : row[col.key];
      return v == null ? '' : v;
    };
    const rowName = (row) => {
      const c = cols()[0];
      if (!c) return String(idOfLive(row, 0));
      const v = row == null ? '' : row[c.key];
      return v == null || v === '' ? String(idOfLive(row, 0)) : String(v);
    };

    const pinClass = (col, pickingOn) => {
      const pin = stickyFirst() && col.key === firstKey();
      return pin ? (pickingOn ? 'pin pin-data' : 'pin') : '';
    };

    const adopted = signal(null);
    const adopt = (slot) => {
      if (dataMode.peek()) return;
      const wrap = slot.parentNode;
      for (const el of slot.assignedElements()) {
        if (el.localName === 'table') {
          wrap.append(el);
          decorateMarkup(el);
          adopted.set(el);
        }
      }
    };

    const decorateMarkup = (table) => {
      table.querySelectorAll('th[data-numeric], td[data-numeric]').forEach((c) =>
        c.classList.add('numeric'));
      table.querySelectorAll('thead th[data-numeric], thead th.numeric').forEach((th) => {
        const i = [...th.parentElement.children].indexOf(th);
        for (const tr of table.querySelectorAll('tbody > tr')) {
          tr.children[i]?.classList.add('numeric');
        }
      });
      table.querySelectorAll('thead th[data-sortable]').forEach((th) => {
        if (th.dataset.uiEnhanced) return;
        th.dataset.uiEnhanced = '1';
        const key = th.dataset.key || String([...th.parentElement.children].indexOf(th));
        const name = th.textContent.trim();
        const btn = document.createElement('button');
        btn.type = 'button';
        btn.className = 'sort';
        btn.setAttribute('aria-label', `Sort by ${name}`);
        while (th.firstChild) btn.append(th.firstChild);
        const icon = document.createElement('ui-icon');
        icon.setAttribute('name', 'arrow-upward');
        btn.append(icon);
        const layer = document.createElement('span');
        layer.className = 'layer';
        layer.setAttribute('aria-hidden', 'true');
        btn.prepend(layer);
        th.append(btn);
        ripple(btn);
        btn.addEventListener('click', () => {
          applySort(key);
          table.querySelectorAll('thead th[data-sortable]').forEach((h) => {
            const k = h.dataset.key || String([...h.parentElement.children].indexOf(h));
            if (k === sortBy()) h.setAttribute('aria-sort', sortDir() === 'desc' ? 'descending' : 'ascending');
            else h.removeAttribute('aria-sort');
          });
          if (!sortBy() || sortMode() === 'server') return;
          const tbody = table.querySelector('tbody');
          if (!tbody) return;
          const idx = [...th.parentElement.children].indexOf(th);
          const dir = sortDir() === 'desc' ? -1 : 1;
          const bodyRows = [...tbody.querySelectorAll('tr')];
          bodyRows.sort((a, b) => dir * compare(cellVal(a, idx), cellVal(b, idx)));
          for (const r of bodyRows) tbody.appendChild(r);
        });
      });
    };

    effect(() => {
      const table = adopted();
      if (!table) return;
      const name = label();
      if (name && !table.caption) table.setAttribute('aria-label', name);
      else if (!name) table.removeAttribute('aria-label');
    });

    const tableAriaLabel = computed(() => {
      const n = (selected() || []).length;
      if (picking() && n) return `${n} selected`;
      return headline() || label() || null;
    });

    const onRowClick = (e, row, id) => {
      const path = e.composedPath();
      if (path.some((n) => n.localName === 'button' || n.localName === 'a' ||
          n.localName === 'input' || n.localName === 'ui-checkbox' ||
          n.localName === 'ui-icon-button' || n.localName === 'select')) {
        return;
      }
      host.emit('row-click', { id, row });
      if (picking()) toggleId(id);
    };

    const empty = computed(() => dataMode() && visible().length === 0 && !loading());
    const showFooter = computed(() => dataMode() && (pageSize() > 0 || filteredCount() >= 0));

    const headerCell = (col) => html`
      <div role="columnheader"
           class=${() => ['th', colAlign(col()), pinClass(col(), picking())].filter(Boolean).join(' ')}
           style=${() => (col().width ? { width: col().width } : null)}
           aria-sort=${() => {
             const c = col();
             if (!c.sortable) return null;
             if (sortBy() !== c.key) return 'none';
             return sortDir() === 'desc' ? 'descending' : 'ascending';
           }}>
        ${() => {
          const c = col();
          const labelEl = c.sortable
            ? html`<button type="button" class="sort"
                      aria-label=${() => sortBy() !== c.key
                        ? `Sort by ${c.label}`
                        : `${c.label}, sorted ${sortDir() === 'desc' ? 'descending' : 'ascending'}`}
                      @click=${() => applySort(c.key)}
                      ref=${(el) => ripple(el)}>
                    <span class="layer" aria-hidden="true"></span>
                    ${c.label}
                    <ui-icon name="arrow-upward"></ui-icon>
                  </button>`
            : c.label;
          return c.aggregate
            ? html`<div>${labelEl}<span class="agg-label">${c.aggregate}</span></div>`
            : labelEl;
        }}
      </div>`;

    const bodyCell = (item, col) => {
      const c = col();
      const it = item();
      let content;
      let agg = false;
      if (it.type === 'group') {
        if (c.key === groupBy()) {
          content = html`
            <button type="button" class="group-toggle"
                    aria-expanded=${String(it.expanded)}
                    aria-label=${`${it.expanded ? 'Collapse' : 'Expand'} ${it.label}`}
                    @click=${(e) => { e.stopPropagation(); toggleGroup(it.key); }}
                    ref=${(el) => ripple(el)}>
              <span class="layer" aria-hidden="true"></span>
              <ui-icon name=${it.expanded ? 'expand-more' : 'chevron-right'}></ui-icon>
              ${it.label} (${it.count})
            </button>`;
        } else if (c.aggregate) {
          content = formatAgg(it.aggregates[c.key]);
          agg = true;
        } else content = '';
      } else {
        content = cellContent(it.row, c);
      }
      return html`
        <div role=${it.type === 'group' ? 'rowheader' : 'cell'}
             class=${() => ['td', colAlign(c), pinClass(c, picking()), agg ? 'agg' : '', it.depth ? 'nested' : '']
               .filter(Boolean).join(' ')}>
          ${content}
        </div>`;
    };

    const dataTable = () => html`
      <div class="grid" part="table" role="table"
           aria-label=${tableAriaLabel}
           aria-busy=${() => (loading() ? 'true' : null)}
           aria-rowcount=${() => (pageSize() ? 1 + filteredCount() + (hasTotals() ? 1 : 0) : null)}
           aria-colcount=${() => cols().length + (picking() ? 1 : 0)}>
        <div class="thead" role="rowgroup">
          <div class="tr" role="row" aria-rowindex="1">
            ${() => (picking() ? html`
              <div class=${() => (stickyFirst() ? 'th check pin' : 'th check')} role="columnheader">
                ${() => (multi() ? html`
                  <ui-checkbox label="Select all rows on this page"
                               checked=${allOnPage} indeterminate=${someOnPage}
                               disabled=${loading}
                               @change=${(e) => { e.stopPropagation(); toggleAll(); }}>
                  </ui-checkbox>` : null)}
              </div>` : null)}
            ${each(() => cols(), headerCell, (c) => c.key)}
          </div>
        </div>
        <div class="tbody" role="rowgroup">
          ${() => (empty() ? html`
            <div class="tr empty-row" role="row">
              <div class="td empty" role="cell"><slot name="empty">${emptyText}</slot></div>
            </div>` : null)}
          ${each(
            () => visible(),
            (item) => html`
              <div class=${() => `tr${item().type === 'group' ? ' group' : ''}`}
                   role="row"
                   aria-selected=${() => (item().type === 'row' && picking()
                     ? String(selectedSet().has(item().id)) : null)}
                   aria-expanded=${() => (item().type === 'group' ? String(item().expanded) : null)}
                   @click=${(e) => item().type === 'row' && onRowClick(e, item().row, item().id)}>
                ${() => (picking() ? html`
                  <div class=${() => (stickyFirst() ? 'td check pin' : 'td check')} role="cell"
                       @click=${(e) => e.stopPropagation()}>
                    ${() => (item().type === 'row' ? html`
                      <ui-checkbox label=${() => `Select ${rowName(item().row)}`}
                                   checked=${() => selectedSet().has(item().id)}
                                   disabled=${loading}
                                   @change=${(e) => { e.stopPropagation(); toggleId(item().id); }}>
                      </ui-checkbox>` : null)}
                  </div>` : null)}
                ${each(() => cols(), (col) => bodyCell(item, col), (c) => c.key)}
              </div>`,
            (it) => String(it.id),
          )}
        </div>
        ${() => (hasTotals() ? html`
          <div class="tfoot" role="rowgroup">
            <div class="tr total" role="row">
              ${() => (picking() ? html`<div class="td check" role="cell"></div>` : null)}
              ${each(() => cols(), (col) => html`
                <div role="cell"
                     class=${() => ['td', colAlign(col()), col().aggregate ? 'agg' : ''].filter(Boolean).join(' ')}>
                  ${() => col().key === firstKey()
                    ? 'Total'
                    : (col().aggregate ? formatAgg(totals()[col().key]) : '')}
                </div>`, (c) => c.key)}
            </div>
          </div>` : null)}
      </div>`;

    return html`
      <div class=${() => `root ${variant() || 'outlined'}`} part="container">
        <slot name="toolbar">
          <ui-table-toolbar headline=${headline} supporting=${supporting}
                            selected-count=${() => (selected() || []).length}
                            filter=${filter}
                            quick-filter=${quickFilter}
                            column-menu=${columnMenu}
                            density-menu=${densityMenu}
                            csv-export=${csvExport}
                            columns=${() => (columns() || []).map((c) =>
                              typeof c === 'string' ? { key: c, label: c } : { key: c.key, label: c.label || c.key })}
                            hidden-columns=${hiddenColumns}
                            density=${resolvedDensity}
                            disabled=${loading}
                            @filter=${(e) => { e.stopPropagation(); filter.set(e.detail.value); host.emit('filter', e.detail); }}
                            @density=${(e) => { e.stopPropagation(); density.set(e.detail.density); dense.set(e.detail.density === 'compact'); host.emit('density', e.detail); }}
                            @column-visibility=${(e) => { e.stopPropagation(); hiddenColumns.set(e.detail.hidden); host.emit('column-visibility', e.detail); }}
                            @export=${(e) => {
                              e.stopPropagation();
                              const name = csvFileName() || 'table.csv';
                              downloadText(name, toCsv(list(), cols()));
                              host.emit('export', { format: 'csv', filename: name });
                            }}>
            <slot name="headline" slot="headline"></slot>
            <slot name="supporting" slot="supporting"></slot>
            <slot name="actions" slot="actions"></slot>
          </ui-table-toolbar>
        </slot>
        ${() => (loading() ? html`<ui-progress class="loading" label="Loading"></ui-progress>` : null)}
        <div class="scroller">
          ${() => (dataMode() ? dataTable() : html`
            <slot ref=${(slot) => {
              slot.addEventListener('slotchange', () => adopt(slot));
              adopt(slot);
            }}></slot>`)}
        </div>
        <slot name="footer">
          ${() => (showFooter() ? html`
            <ui-table-footer page=${page} page-size=${pageSize}
                             row-count=${filteredCount}
                             page-size-options=${pageSizeOptions}
                             disabled=${loading}
                             @page=${(e) => {
                               e.stopPropagation();
                               page.set(e.detail.page);
                               if (e.detail.pageSize) pageSize.set(e.detail.pageSize);
                               host.emit('page', e.detail);
                             }}>
            </ui-table-footer>` : null)}
        </slot>
      </div>`;
  },
});

export const tag = 'ui-table';
export const themeVars = t;
