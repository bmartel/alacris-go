// <ui-date-picker> — Material date picker: a text-field-style control that
// opens a calendar. Docked (default) commits on day click; modal confirms
// with OK / Cancel.
//
//   <ui-date-picker label="Event" value=${date}
//                   @change=${(e) => date(e.detail.value)}></ui-date-picker>
//
// `value` is an ISO date string (YYYY-MM-DD), or '' for none. Typing an
// ISO or locale-formatted date into the field commits on blur / Enter.
// Set `range` to pick a start and end; `change` then reports
// `{ start, end, value }` where `value` is `start/end`.
//
// @prop  {string}  label=''
// @prop  {string}  value=''         — ISO date (YYYY-MM-DD); range: start/end
// @prop  {boolean} range=false      — pick a start and end date
// @prop  {string}  start=''         — range start ISO
// @prop  {string}  end=''           — range end ISO
// @prop  {string}  variant='filled' — filled | outlined
// @prop  {string}  presentation='docked' — docked | modal
// @prop  {string}  min=''           — inclusive ISO lower bound
// @prop  {string}  max=''           — inclusive ISO upper bound
// @prop  {string}  locale=''        — BCP 47 tag; empty uses the runtime locale
// @prop  {boolean} disabled=false
// @prop  {boolean} required=false
// @prop  {string}  name=''          — form participation
// @prop  {string}  placeholder=''
// @event change — committed; detail: { value } or { start, end, value } when range
// @event input  — field keystroke; detail: { value } (the raw text)
// @event open   — calendar visible (after the enter animation); does not bubble
// @event close  — calendar removed (after the exit animation); does not bubble
// @part  field, input, label, panel, day
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, computed, signal, effect, onCleanup } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { formBind } from '../util/form.js';
import { presence } from '../motion/presence.js';
import { animate, fx } from '../motion/animate.js';
import { autoUpdate } from '../util/position.js';
import { escapeLayer } from '../util/keys.js';
import { popupEvent } from '../util/popup.js';
import { focusTrap, scrollLock } from '../util/focus.js';
import './ui-icon-button.js';
import './ui-button.js';

const ISO = /^(\d{4})-(\d{2})-(\d{2})$/;

const pad = (n) => String(n).padStart(2, '0');
const toISO = (d) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
const parseISO = (s) => {
  const m = ISO.exec(s || '');
  if (!m) return null;
  const d = new Date(+m[1], +m[2] - 1, +m[3]);
  return (d.getFullYear() === +m[1] && d.getMonth() === +m[2] - 1 && d.getDate() === +m[3]) ? d : null;
};
const loc = (locale) => locale || undefined;
const formatDate = (iso, locale) => {
  const d = parseISO(iso);
  return d ? new Intl.DateTimeFormat(loc(locale), { dateStyle: 'medium' }).format(d) : '';
};
const formatRange = (start, end, locale) => {
  const a = formatDate(start, locale);
  const b = formatDate(end, locale);
  if (a && b) return `${a} – ${b}`;
  return a || b || '';
};
const monthTitle = (d, locale) =>
  new Intl.DateTimeFormat(loc(locale), { month: 'long', year: 'numeric' }).format(d);
const weekdays = (locale) => {
  const fmt = new Intl.DateTimeFormat(loc(locale), { weekday: 'narrow' });
  return Array.from({ length: 7 }, (_, i) => fmt.format(new Date(2026, 7, 9 + i)));
};
const parseTyped = (text) => {
  const t = (text || '').trim();
  if (!t) return '';
  if (parseISO(t)) return t;
  const d = new Date(t);
  return Number.isNaN(d.getTime()) ? null : toISO(d);
};
const startOfMonth = (d) => new Date(d.getFullYear(), d.getMonth(), 1);
const todayISO = () => toISO(new Date());
const monthCells = (view, selected, min, max, today, rangeStart = '', rangeEnd = '') => {
  const y = view.getFullYear();
  const m = view.getMonth();
  const start = new Date(y, m, 1 - new Date(y, m, 1).getDay());
  const cells = [];
  for (let i = 0; i < 42; i++) {
    const d = new Date(start.getFullYear(), start.getMonth(), start.getDate() + i);
    const iso = toISO(d);
    const isStart = !!rangeStart && iso === rangeStart;
    const isEnd = !!rangeEnd && iso === rangeEnd;
    cells.push({
      iso,
      day: d.getDate(),
      inMonth: d.getMonth() === m,
      selected: isStart || isEnd || (!rangeStart && iso === selected),
      rangeStart: isStart,
      rangeEnd: isEnd,
      inRange: !!(rangeStart && rangeEnd && iso >= rangeStart && iso <= rangeEnd),
      today: iso === today,
      disabled: !!(min && iso < min) || !!(max && iso > max),
    });
  }
  return cells;
};

const t = vars('ui-date-picker', {
  bg: sys.color.surfaceContainerHighest,
  fg: sys.color.onSurface,
  labelFg: sys.color.onSurfaceVariant,
  accent: sys.color.primary,
  onAccent: sys.color.onPrimary,
  outlineColor: sys.color.outline,
  radius: sys.radius.xs,
  font: sys.type.bodyLg,
  height: '56px',
  panelBg: sys.color.surfaceContainerHigh,
  panelRadius: sys.radius.xl,
  dayRadius: sys.radius.full,
  todayFg: sys.color.primary,
  mutedFg: sys.color.onSurfaceVariant,
  rangeBg: sys.color.secondaryContainer,
  rangeFg: sys.color.onSecondaryContainer,
  scrim: `color-mix(in srgb, ${sys.color.scrim} 32%, transparent)`,
});

const styles = css`
  :host { display: block; inline-size: 240px; }
  :host([range]) { inline-size: 320px; }
  .range { inline-size: 100%; }
  .range .field { min-inline-size: 320px; }
  .root { display: block; position: relative; }
  .field {
    position: relative;
    display: flex;
    align-items: center;
    gap: ${sys.space(1)};
    min-block-size: calc(${t.height} + var(--ui-density, 0) * 4px);
    padding-inline: ${sys.space(4)} ${sys.space(1)};
    border-radius: ${t.radius};
    font: ${t.font};
    letter-spacing: ${sys.tracking.bodyLg};
    color: ${t.fg};
    cursor: text;
    --ui-icon-size: 1.5rem;
  }
  .filled .field {
    background: ${t.bg};
    border-start-start-radius: ${t.radius};
    border-start-end-radius: ${t.radius};
    border-end-start-radius: 0;
    border-end-end-radius: 0;
  }
  .filled .field::after {
    content: '';
    position: absolute;
    inset-inline: 0;
    inset-block-end: 0;
    block-size: 1px;
    background: ${sys.color.onSurfaceVariant};
    transition: block-size ${sys.duration.short2} ${sys.easing.standard},
                background-color ${sys.duration.short2} ${sys.easing.standard};
  }
  .filled:focus-within .field::after,
  .filled.open .field::after { block-size: 2px; background: ${t.accent}; }
  .filled .field::before {
    content: '';
    position: absolute; inset: 0; pointer-events: none;
    background: ${sys.color.onSurface};
    opacity: 0;
    border-radius: inherit;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  .filled:hover:not(:focus-within):not(.open):not(.disabled) .field::before {
    opacity: ${sys.state.hover};
  }
  .filled:hover:not(:focus-within):not(.open):not(.disabled) .field::after {
    background: ${sys.color.onSurface};
  }

  fieldset {
    position: absolute;
    inset: -6px 0 0;
    margin: 0;
    padding: 0 calc(${sys.space(3)} - 2px);
    min-inline-size: 0;
    box-sizing: border-box;
    appearance: none;
    border: 1px solid ${t.outlineColor};
    border-radius: ${t.radius};
    pointer-events: none;
    transition: border-color ${sys.duration.short2} ${sys.easing.standard},
                border-width ${sys.duration.short2} ${sys.easing.standard};
  }
  legend {
    float: unset;
    display: block;
    width: max-content;
    padding: 0;
    margin: 0;
    margin-inline-start: 0;
    white-space: nowrap;
    overflow: hidden;
    font: ${t.font};
    font-size: 0.75em;
    letter-spacing: calc(${sys.tracking.bodyLg} * 0.75);
    visibility: hidden;
    max-inline-size: 0.01px;
    height: 12px;
    line-height: 12px;
    transition: max-inline-size ${sys.duration.short2} ${sys.easing.standard};
  }
  legend span {
    padding-inline: ${sys.space(1)} calc(${sys.space(1)} - 0.5px);
    display: inline-block;
    opacity: 0;
    visibility: visible;
  }
  .outlined.floating legend {
    max-inline-size: 100%;
    transition: max-inline-size ${sys.duration.short3} ${sys.easing.standard};
  }
  .outlined:focus-within fieldset,
  .outlined.open fieldset { border-width: 2px; border-color: ${t.accent}; }
  .root:hover:not(:focus-within):not(.open) fieldset { border-color: ${sys.color.onSurface}; }

  .label {
    position: absolute;
    inset-inline-start: ${sys.space(4)};
    inset-block-start: 50%;
    translate: 0 -50%;
    color: ${t.labelFg};
    pointer-events: none;
    transform-origin: 0 50%;
    transition: translate ${sys.duration.short3} ${sys.easing.standard},
                scale ${sys.duration.short3} ${sys.easing.standard},
                color ${sys.duration.short2} ${sys.easing.standard};
  }
  .filled.floating .label { translate: 0 calc(-50% - 16px); scale: 0.75; }
  .outlined.floating .label {
    translate: 0 calc(-50% - (${t.height} + var(--ui-density, 0) * 4px) / 2);
    scale: 0.75;
  }
  :focus-within .label, .open .label { color: ${t.accent}; }

  input {
    flex: 1;
    min-inline-size: 0;
    margin: 0;
    border: none;
    outline: none;
    appearance: none;
    background: transparent;
    font: inherit;
    letter-spacing: inherit;
    color: inherit;
    padding: 0;
  }
  .filled.has-label input { padding-block-start: 18px; }
  input::placeholder { color: ${t.labelFg}; opacity: 0; }
  .floating input::placeholder { opacity: 1; }

  .panel {
    position: fixed;
    z-index: ${sys.z.popup};
    padding: ${sys.space(3)} ${sys.space(3)} ${sys.space(4)};
    background: ${t.panelBg};
    border-radius: ${t.panelRadius};
    box-shadow: ${sys.elevation[3]};
    color: ${t.fg};
    overflow: auto;
  }
  .overlay {
    position: fixed;
    inset: 0;
    z-index: ${sys.z.popup};
    display: grid;
    place-items: center;
  }
  .scrim { position: absolute; inset: 0; background: ${t.scrim}; }
  @keyframes ui-date-picker-in {
    from { transform: scale(0.8); }
    to { transform: none; }
  }
  .modal-surface {
    animation: ui-date-picker-in ${sys.duration.medium2} ${sys.easing.emphasizedDecelerate};
    position: relative;
    display: flex;
    flex-direction: column;
    gap: ${sys.space(2)};
    padding: ${sys.space(6)} ${sys.space(3)} ${sys.space(3)};
    background: ${t.panelBg};
    border-radius: ${t.panelRadius};
    box-shadow: ${sys.elevation[3]};
    color: ${t.fg};
    min-inline-size: min(360px, calc(100vw - 48px));
    max-inline-size: calc(100vw - 48px);
  }
  .headline {
    padding-inline: ${sys.space(3)};
    font: ${sys.type.labelMd};
    letter-spacing: ${sys.tracking.labelMd};
    color: ${t.mutedFg};
  }
  .picked {
    padding-inline: ${sys.space(3)};
    padding-block-end: ${sys.space(2)};
    font: ${sys.type.headlineMd};
    letter-spacing: ${sys.tracking.headlineMd};
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    gap: ${sys.space(2)};
    padding-inline: ${sys.space(1)};
  }

  .cal-header {
    display: flex;
    align-items: center;
    gap: ${sys.space(1)};
    padding-inline: ${sys.space(2)};
    min-block-size: 48px;
  }
  .month {
    flex: 1;
    font: ${sys.type.titleSm};
    letter-spacing: ${sys.tracking.titleSm};
  }
  .weekdays, .cal-row {
    display: grid;
    grid-template-columns: repeat(7, 40px);
    justify-content: center;
  }
  .days {
    display: flex;
    flex-direction: column;
    align-items: center;
  }
  .weekday {
    display: grid;
    place-items: center;
    block-size: 40px;
    font: ${sys.type.labelSm};
    letter-spacing: ${sys.tracking.labelSm};
    color: ${t.mutedFg};
  }
  .day {
    position: relative;
    isolation: isolate;
    display: grid;
    place-items: center;
    inline-size: 40px;
    block-size: 40px;
    margin: 0;
    padding: 0;
    border: none;
    outline: none;
    appearance: none;
    background: transparent;
    font: ${sys.type.bodySm};
    letter-spacing: ${sys.tracking.bodySm};
    color: ${t.fg};
    cursor: pointer;
  }
  .day .text {
    position: relative;
    z-index: 2;
  }
  .day .layer {
    position: absolute; inset: 0; z-index: 3;
    border-radius: ${t.dayRadius}; background: currentColor; opacity: 0;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  .day:hover .layer { opacity: ${sys.state.hover}; }
  .day:active .layer { opacity: ${sys.state.pressed}; }
  .day:focus-visible { outline: ${sys.focus.ring}; outline-offset: -2px; }
  .day.outside { color: ${t.mutedFg}; }

  /* In-range connector track */
  .day.in-range { color: ${t.rangeFg}; }
  .day.in-range::before {
    content: '';
    position: absolute;
    inset: 0;
    background: ${t.rangeBg};
    z-index: 0;
    pointer-events: none;
  }
  .day.in-range.range-start::before {
    inset-inline-start: 50%;
    inset-inline-end: 0;
  }
  .day.in-range.range-end::before {
    inset-inline-start: 0;
    inset-inline-end: 50%;
  }
  .day.in-range:nth-child(7n + 1)::before {
    border-start-start-radius: ${t.dayRadius};
    border-end-start-radius: ${t.dayRadius};
  }
  .day.in-range:nth-child(7n)::before {
    border-start-end-radius: ${t.dayRadius};
    border-end-end-radius: ${t.dayRadius};
  }
  .day.in-range.range-start.range-end::before { display: none; }

  /* Selected circular badge */
  .day.selected {
    color: ${t.onAccent};
    font-weight: 500;
  }
  .day.selected::after {
    content: '';
    position: absolute;
    inset: 0;
    border-radius: ${t.dayRadius};
    background: ${t.accent};
    z-index: 1;
    pointer-events: none;
    transition: background-color ${sys.duration.short4} ${sys.easing.standard};
  }

  /* Today outline indicator */
  .day.today:not(.selected) {
    color: ${t.todayFg};
  }
  .day.today:not(.selected)::after {
    content: '';
    position: absolute;
    inset: 0;
    border-radius: ${t.dayRadius};
    border: 1px solid ${t.todayFg};
    box-sizing: border-box;
    z-index: 1;
    pointer-events: none;
  }

  .day:disabled {
    color: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContent} * 100%), transparent);
    cursor: default;
    pointer-events: none;
  }

  .disabled { opacity: ${sys.state.disabledContent}; pointer-events: none; }
`;

define('ui-date-picker', {
  formAssociated: true,
  props: {
    label: '', value: '', range: false, start: '', end: '',
    variant: 'filled', presentation: 'docked',
    min: '', max: '', locale: '', disabled: false, required: false,
    name: '', placeholder: '',
  },
  styles: [base, styles],
  setup(p, host) {
    const { label, value, range, start, end, variant, presentation, min, max, locale, disabled, required, name, placeholder } = p;
    formBind(host, { name, value, disabled });

    const open = signal(false);
    const focused = signal(false);
    const text = signal('');
    const viewMonth = signal(startOfMonth(parseISO(value() || start()) || new Date()));
    const draft = signal(value());
    const draftStart = signal(start());
    const draftEnd = signal(end());
    const rangeAnchor = signal('');
    let fieldEl = null;
    let modalSurfaceEl = null;
    let stopAuto = null;
    let releaseTrap = null;
    let unlock = null;

    effect(() => {
      if (range()) {
        const next = [start(), end()].filter(Boolean).join('/');
        if (value.peek() !== next) value.set(next);
        text.set(formatRange(start(), end(), locale()) || next);
        const d = parseISO(start());
        if (d && !open()) viewMonth.set(startOfMonth(d));
        return;
      }
      const v = value();
      text.set(formatDate(v, locale()) || v);
      const d = parseISO(v);
      if (d && !open()) viewMonth.set(startOfMonth(d));
    });

    const floating = computed(() => focused() || text() !== '' || placeholder() !== '' || open());
    const cls = computed(() =>
      ['root', variant(), range() && 'range', floating() && 'floating', open() && 'open',
       disabled() && 'disabled', label() && 'has-label'].filter(Boolean).join(' '));
    const selected = computed(() => (presentation() === 'modal' && open() ? draft() : value()));
    const liveStart = computed(() => (range() && open() ? draftStart() : start()));
    const liveEnd = computed(() => (range() && open() ? draftEnd() : end()));
    const cells = computed(() =>
      monthCells(viewMonth(), selected(), min(), max(), todayISO(),
        range() ? liveStart() : '', range() ? liveEnd() : ''));
    const title = computed(() => monthTitle(viewMonth(), locale()));
    const heads = computed(() => weekdays(locale()));
    const pickedLabel = computed(() => range()
      ? (formatRange(liveStart(), liveEnd(), locale()) || 'Selected dates')
      : (formatDate(selected(), locale()) || 'Selected date'));

    const outOfRange = (iso) => !!(min() && iso < min()) || !!(max() && iso > max());

    const commit = (iso, { close = true } = {}) => {
      if (iso !== value()) {
        value.set(iso);
        host.emit('change', { value: iso });
      }
      text.set(formatDate(iso, locale()) || iso);
      if (close) closePanel();
    };
    const commitRange = (s, e, { close = true } = {}) => {
      start.set(s);
      end.set(e);
      const joined = [s, e].filter(Boolean).join('/');
      value.set(joined);
      host.emit('change', { start: s, end: e, value: joined });
      text.set(formatRange(s, e, locale()) || joined);
      if (close) closePanel();
    };

    let lastPicked = null;
    const pick = (iso) => {
      if (!iso || outOfRange(iso) || iso === lastPicked) return;
      lastPicked = iso;
      setTimeout(() => { lastPicked = null; }, 0);
      if (range()) {
        const anchor = rangeAnchor();
        if (!anchor || iso === anchor) {
          rangeAnchor.set(iso);
          draftStart.set(iso);
          draftEnd.set('');
          text.set(formatDate(iso, locale()) || iso);
          return;
        }
        let a = anchor;
        let b = iso;
        if (b < a) { a = iso; b = anchor; }
        rangeAnchor.set('');
        draftStart.set(a);
        draftEnd.set(b);
        if (presentation() !== 'modal') commitRange(a, b);
        return;
      }
      if (presentation() === 'modal') { draft.set(iso); return; }
      commit(iso);
    };

    const openPanel = () => {
      if (disabled() || open()) return;
      const d = parseISO(range() ? start() : value()) || new Date();
      viewMonth.set(startOfMonth(d));
      draft.set(value());
      draftStart.set(start());
      draftEnd.set(end());
      rangeAnchor.set(range() && start() && !end() ? start() : '');
      open.set(true);
    };
    const closePanel = () => open.set(false);
    const toggle = (e) => {
      e?.stopPropagation();
      open() ? closePanel() : openPanel();
    };

    const shiftMonth = (delta) => {
      const v = viewMonth();
      viewMonth.set(new Date(v.getFullYear(), v.getMonth() + delta, 1));
    };

    const onInput = (e) => {
      text.set(e.target.value);
      host.emit('input', { value: e.target.value });
    };
    const onBlur = () => {
      focused.set(false);
      if (range()) {
        text.set(formatRange(start(), end(), locale()) || value());
        return;
      }
      const parsed = parseTyped(text());
      if (parsed === null) {
        text.set(formatDate(value(), locale()) || value());
        return;
      }
      if (parsed !== value() && !outOfRange(parsed)) commit(parsed, { close: false });
      else text.set(formatDate(value(), locale()) || value());
    };
    const onKeydown = (e) => {
      if (e.key === 'ArrowDown' && e.altKey) { e.preventDefault(); openPanel(); }
      else if (e.key === 'F4') { e.preventDefault(); toggle(); }
      else if (e.key === 'Enter') {
        e.preventDefault();
        const parsed = parseTyped(text());
        if (parsed !== null && !outOfRange(parsed)) commit(parsed);
      } else if (e.key === 'Escape' && open()) {
        e.preventDefault();
        closePanel();
      }
    };

    effect(() => {
      if (!open()) return;
      const onDoc = (e) => {
        if (e.composedPath().includes(host)) return;
        closePanel();
      };
      // Capture at the document is not early enough: a dialog registers the
      // same way when it opens, so it is already listening by the time this
      // panel does and one Escape closes both.
      const releaseEsc = escapeLayer(closePanel);
      document.addEventListener('pointerdown', onDoc);
      return () => {
        document.removeEventListener('pointerdown', onDoc);
        releaseEsc();
      };
    });
    effect(() => {
      if (open() && presentation() === 'modal') {
        unlock = scrollLock();
        queueMicrotask(() => { if (open() && !releaseTrap) releaseTrap = focusTrap(host); });
      } else {
        if (modalSurfaceEl?.isConnected) {
          animate(modalSurfaceEl, fx.scaleOut, { duration: 'short4', easing: 'emphasizedAccelerate' });
        }
        releaseTrap?.(); releaseTrap = null;
        unlock?.(); unlock = null;
      }
    });
    effect(() => {
      if (!open() && stopAuto) { stopAuto(); stopAuto = null; }
    });
    onCleanup(() => { stopAuto?.(); releaseTrap?.(); unlock?.(); });

    const dayClassFrom = (cell) => [
      'day',
      !cell.inMonth && 'outside',
      cell.selected && 'selected',
      cell.today && 'today',
      cell.inRange && 'in-range',
      cell.rangeStart && 'range-start',
      cell.rangeEnd && 'range-end',
    ].filter(Boolean).join(' ');
    const dayButton = (cell) => html`
      <button type="button" part="day" role="gridcell"
              class=${dayClassFrom(cell)}
              data-iso=${cell.iso}
              aria-selected=${cell.selected ? 'true' : 'false'}
              ?disabled=${cell.disabled}
              @click=${() => pick(cell.iso)}>
        <span class="layer" aria-hidden="true"></span>
        <span class="text">${cell.day}</span>
      </button>`;
    const calGrid = () => {
      const all = cells();
      const rows = [];
      for (let r = 0; r < all.length; r += 7) {
        rows.push(html`<div class="cal-row" role="row">${all.slice(r, r + 7).map(dayButton)}</div>`);
      }
      return html`
        <div class="cal" role="grid" aria-label=${() => title()}>
          <div class="weekdays" role="row">${() => heads().map((w) => html`<span class="weekday" role="columnheader">${w}</span>`)}</div>
          <div class="days">${rows}</div>
        </div>`;
    };

    // Presence mounts the grid in a nested owner. A setup-level paint keeps
    // selected / in-range classes in sync even if a row binding does not.
    effect(() => {
      const a = range() ? liveStart() : '';
      const b = range() ? liveEnd() : '';
      const sel = selected();
      const rng = range();
      if (!open()) return;
      const paint = () => {
        const root = host.shadowRoot;
        if (!root) return;
        for (const btn of root.querySelectorAll('.day')) {
          const iso = btn.getAttribute('data-iso');
          if (!iso) continue;
          const isStart = !!(rng && a && iso === a);
          const isEnd = !!(rng && b && iso === b);
          const isSel = !!(isStart || isEnd || (!rng && iso === sel) || (!a && iso === sel));
          btn.classList.toggle('selected', isSel);
          btn.classList.toggle('in-range', !!(a && b && iso >= a && iso <= b));
          btn.classList.toggle('range-start', isStart);
          btn.classList.toggle('range-end', isEnd);
          btn.setAttribute('aria-selected', isSel ? 'true' : 'false');
        }
      };
      paint();
      queueMicrotask(paint);
    });

    const panelRef = (el) => {
      stopAuto?.();
      if (presentation() !== 'modal') {
        stopAuto = autoUpdate(el, fieldEl, { placement: 'bottom-start', offset: 4 });
      }
    };

    const dockedView = () => html`
      <div class="panel" part="panel" role="dialog" aria-label=${() => label() || 'Choose date'}
           ref=${panelRef}>
        <div class="cal-header">
          <span class="month">${title}</span>
          <ui-icon-button icon="chevron-left" label="Previous month" @click=${() => shiftMonth(-1)}></ui-icon-button>
          <ui-icon-button icon="chevron-right" label="Next month" @click=${() => shiftMonth(1)}></ui-icon-button>
        </div>
        ${calGrid}
      </div>`;

    const modalView = () => html`
      <div class="overlay">
        <div class="scrim" aria-hidden="true" @click=${closePanel}></div>
        <div class="modal-surface" part="panel" role="dialog" aria-modal="true"
             aria-label=${() => label() || 'Choose date'}
             ref=${(el) => (modalSurfaceEl = el)}>
          <div class="headline">${() => (range() ? 'Select dates' : 'Select date')}</div>
          <div class="picked">${pickedLabel}</div>
          <div class="cal-header">
            <span class="month">${title}</span>
            <ui-icon-button icon="chevron-left" label="Previous month" @click=${() => shiftMonth(-1)}></ui-icon-button>
            <ui-icon-button icon="chevron-right" label="Next month" @click=${() => shiftMonth(1)}></ui-icon-button>
          </div>
          ${calGrid}
          <div class="actions">
            <ui-button variant="text" @click=${closePanel}>Cancel</ui-button>
            <ui-button variant="text" @click=${() => range() ? commitRange(draftStart(), draftEnd()) : commit(draft())}>OK</ui-button>
          </div>
        </div>
      </div>`;

    return html`
      <div class=${cls}>
        <div class="field" part="field" ref=${(el) => (fieldEl = el)}>
          ${() => (variant() === 'outlined'
            ? html`<fieldset aria-hidden="true"><legend><span>${label}${() => (required() ? ' *' : '')}</span></legend></fieldset>`
            : null)}
          ${() => (label() ? html`<span class="label" part="label" id="field-label">${label}${() => (required() ? ' *' : '')}</span>` : null)}
          <input part="input" .value=${text}
                 placeholder=${() => placeholder() || null}
                 ?disabled=${disabled} ?required=${required}
                 aria-labelledby=${() => (label() ? 'field-label' : null)}
                 aria-label=${() => (label() ? null : (placeholder() || 'Date'))}
                 aria-haspopup="dialog" aria-expanded=${() => String(open())}
                 autocomplete="off"
                 @input=${onInput} @focus=${() => focused.set(true)} @blur=${onBlur}
                 @keydown=${onKeydown}>
          <ui-icon-button icon="calendar" label=${() => (open() ? 'Close calendar' : 'Open calendar')}
                          @click=${toggle}></ui-icon-button>
        </div>
        ${presence(() => open() && presentation() !== 'modal', dockedView, {
          enter: fx.scaleIn,
          exit: fx.scaleOut,
          enterDuration: 'short4',
          exitDuration: 'short4',
          onEntered: () => host.emit('open', null, popupEvent),
          onExited: () => host.emit('close', null, popupEvent),
        })}
        ${presence(() => open() && presentation() === 'modal', modalView, {
          enter: fx.fadeIn,
          exit: fx.fadeOut,
          enterDuration: 'medium2',
          exitDuration: 'short4',
          onEntered: () => host.emit('open', null, popupEvent),
          onExited: () => host.emit('close', null, popupEvent),
        })}
      </div>`;
  },
});

export const tag = 'ui-date-picker';
export const themeVars = t;
