// <ui-table-toolbar> — DataGrid chrome: title, selection count, quick filter,
// column visibility, density, and CSV export. Built from <ui-search>,
// <ui-icon-button>, <ui-menu>, and <ui-checkbox>.
//
//   <ui-table-toolbar headline="Nutrition" quick-filter column-menu
//                     density-menu csv-export
//                     columns=${cols} hidden-columns=${hidden}
//                     @filter=${…} @density=${…}
//                     @column-visibility=${…} @export=${…}>
//     <ui-button slot="actions">Add</ui-button>
//   </ui-table-toolbar>
//
// @prop  {string}  headline=''
// @prop  {string}  supporting=''
// @prop  {number}  selectedCount=0  — when >0 the bar switches to the
//                                     "N selected" selection state
// @prop  {string}  filter=''        — quick-filter value
// @prop  {boolean} quickFilter=false
// @prop  {boolean} columnMenu=false
// @prop  {boolean} densityMenu=false
// @prop  {boolean} csvExport=false
// @prop  {Array}   columns=[]       — { key, label } (JSON or property)
// @prop  {Array}   hiddenColumns=[]
// @prop  {string}  density='standard' — compact | standard | comfortable
// @prop  {boolean} disabled=false
// @event filter             — quick filter changed; detail: { value }
// @event density            — density chosen; detail: { density }
// @event column-visibility  — a column was toggled; detail: { hidden }
// @event export             — export requested; detail: { format: 'csv' }
// @slot  headline  — replaces the headline text
// @slot  supporting
// @slot  actions   — extra trailing controls (after the built-in tools)
// @part  bar, titles, tools
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, computed, signal, effect, each } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base, focusRingOn } from './base.js';
import './ui-search.js';
import './ui-icon-button.js';
import './ui-menu.js';
import './ui-menu-item.js';
import './ui-checkbox.js';

const t = vars('ui-table-toolbar', {
  height: '64px',
  bg: sys.color.surface,
  selectedBg: sys.color.secondaryContainer,
  selectedFg: sys.color.onSecondaryContainer,
  panelBg: sys.color.surfaceContainer,
  radius: sys.radius.xs,
  elevation: sys.elevation[2],
});

const styles = css`
  :host { display: none; }
  :host(.show) { display: block; }
  .bar {
    display: flex;
    align-items: center;
    gap: ${sys.space(3)};
    min-block-size: ${t.height};
    padding-inline: ${sys.space(4)};
    background: ${t.bg};
  }
  .bar.picking {
    background: ${t.selectedBg};
    color: ${t.selectedFg};
  }
  .titles { flex: 1; min-inline-size: 0; }
  .headline {
    margin: 0;
    font: ${sys.type.titleLg};
    letter-spacing: ${sys.tracking.titleLg};
    color: ${sys.color.onSurface};
  }
  .bar.picking .headline { color: inherit; }
  .supporting {
    margin: 0;
    font: ${sys.type.bodyMd};
    letter-spacing: ${sys.tracking.bodyMd};
    color: ${sys.color.onSurfaceVariant};
  }
  .tools {
    display: flex;
    align-items: center;
    gap: ${sys.space(1)};
    flex: none;
  }
  .quick-filter { inline-size: 220px; }
  .col-wrap { position: relative; }
  .panel {
    position: absolute;
    inset-inline-end: 0;
    inset-block-start: calc(100% + ${sys.space(1)});
    z-index: 3;
    min-inline-size: 220px;
    max-block-size: 320px;
    overflow: auto;
    padding: ${sys.space(2)};
    background: ${t.panelBg};
    border-radius: ${t.radius};
    box-shadow: ${t.elevation};
    display: flex;
    flex-direction: column;
    gap: ${sys.space(1)};
  }
  ${focusRingOn('.panel')}
`;

define('ui-table-toolbar', {
  props: {
    headline: '',
    supporting: '',
    selectedCount: 0,
    filter: '',
    quickFilter: false,
    columnMenu: false,
    densityMenu: false,
    csvExport: false,
    columns: [],
    hiddenColumns: [],
    density: 'standard',
    disabled: false,
  },
  styles: [base, styles],
  setup(p, host) {
    const {
      headline, supporting, selectedCount, filter, quickFilter, columnMenu,
      densityMenu, csvExport, columns, hiddenColumns, density, disabled,
    } = p;

    const colsOpen = signal(false);
    const picking = computed(() => (selectedCount() || 0) > 0);
    const hiddenSet = computed(() => new Set(hiddenColumns() || []));

    const hasChrome = computed(() => !!(
      headline() || supporting() || picking() ||
      quickFilter() || columnMenu() || densityMenu() || csvExport()
    ));
    effect(() => host.classList.toggle('show', hasChrome()));

    const toggleCol = (key) => {
      const next = new Set(hiddenColumns() || []);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      const visible = (columns() || []).filter((c) => c && !next.has(c.key));
      if (!visible.length) return;
      host.emit('column-visibility', { hidden: [...next] });
    };

    effect(() => {
      if (!colsOpen()) return;
      const close = (e) => {
        if (!host.contains(e.target)) colsOpen.set(false);
      };
      document.addEventListener('pointerdown', close);
      return () => document.removeEventListener('pointerdown', close);
    });

    return html`
      <div class=${() => (picking() ? 'bar picking' : 'bar')} part="bar"
           role="toolbar" aria-label=${() => headline() || 'Table toolbar'}>
        <div class="titles" part="titles">
          <div class="headline" id="table-title">
            <slot name="headline">${() =>
              picking() ? `${selectedCount()} selected` : headline()
            }</slot>
          </div>
          <div class="supporting" ?hidden=${picking}>
            <slot name="supporting">${supporting}</slot>
          </div>
        </div>
        <div class="tools" part="tools">
          ${() => (quickFilter() ? html`
            <div class="quick-filter">
              <ui-search label="Search" value=${filter} ?disabled=${disabled}
                         @input=${(e) => host.emit('filter', { value: e.detail.value })}>
              </ui-search>
            </div>` : null)}
          ${() => (columnMenu() ? html`
            <div class="col-wrap">
              <ui-icon-button icon="view-column" label="Show columns"
                              disabled=${disabled}
                              @click=${() => colsOpen.set(!colsOpen())}></ui-icon-button>
              ${() => (colsOpen() ? html`
                <div class="panel" role="group" aria-label="Show columns">
                  ${each(
                    () => columns() || [],
                    (col) => html`
                      <ui-checkbox label=${() => col().label}
                                   checked=${() => !hiddenSet().has(col().key)}
                                   disabled=${disabled}
                                   @change=${(e) => { e.stopPropagation(); toggleCol(col().key); }}>
                      </ui-checkbox>`,
                    (c) => c.key,
                  )}
                </div>` : null)}
            </div>` : null)}
          ${() => (densityMenu() ? html`
            <ui-menu @select=${(e) => host.emit('density', { density: e.detail.value })}>
              <ui-icon-button slot="anchor" icon="table-rows" label="Density"
                              disabled=${disabled}></ui-icon-button>
              <ui-menu-item value="compact">Compact</ui-menu-item>
              <ui-menu-item value="standard">Standard</ui-menu-item>
              <ui-menu-item value="comfortable">Comfortable</ui-menu-item>
            </ui-menu>` : null)}
          ${() => (csvExport() ? html`
            <ui-icon-button icon="download" label="Export CSV" disabled=${disabled}
                            @click=${() => host.emit('export', { format: 'csv' })}>
            </ui-icon-button>` : null)}
          <slot name="actions"></slot>
        </div>
      </div>`;
  },
});

export const tag = 'ui-table-toolbar';
export const themeVars = t;
