// The example's components.
//
// Everything reactive lives here, in JavaScript, because setup() runs in the
// browser. Go renders the elements and their props; alacris does the rest.
// Visuals come from Alacris UI (already on the page via Config.UI) so this
// file is composition, not a second design system.
//
// The server owns the board. A component never mutates it: it emits an event,
// the Go handler changes the list, and the new list arrives as one prop write.
// Each lane has its own each(), keyed by id — a card that stays in a lane
// keeps its node; a card that changes lane is the one node that is created
// in the destination.

import { define, html, css, signal, each, effect, batch } from 'alacris';
import { showSnackbar } from '@alacris/ui';

/** Same placement the Go list uses: remove, then insert among the rest. */
function applyMove(list, columns, id, column, index) {
  const item = list.find((c) => c.id === id);
  if (!item) return list;
  const order = {};
  (columns || []).forEach((c, i) => { order[c.id] = i; });
  const rest = list.filter((c) => c.id !== id).map((c) => ({ ...c }));
  const dest = rest.filter((c) => c.column === column)
    .sort((a, b) => a.rank - b.rank || a.id - b.id);
  const others = rest.filter((c) => c.column !== column);
  const at = index < 0 || index > dest.length ? dest.length : index;
  dest.splice(at, 0, { ...item, column });
  dest.forEach((c, i) => { c.rank = i; });
  return [...others, ...dest].sort((a, b) => {
    const d = (order[a.column] ?? 99) - (order[b.column] ?? 99);
    if (d) return d;
    if (a.rank !== b.rank) return a.rank - b.rank;
    return a.id - b.id;
  });
}

function applyMoveColumn(cols, id, index) {
  const next = (cols || []).map((c) => ({ ...c }));
  const from = next.findIndex((c) => c.id === id);
  if (from < 0) return cols;
  const [col] = next.splice(from, 1);
  const at = index < 0 || index > next.length ? next.length : index;
  next.splice(at, 0, col);
  return next;
}

function applyPatch(list, id, patch) {
  return (list || []).map((c) => (c.id === id ? { ...c, ...patch } : c));
}

function sameLabels(a, b) {
  const x = [...(a || [])].sort();
  const y = [...(b || [])].sort();
  return x.length === y.length && x.every((v, i) => v === y[i]);
}

function asWho(who) {
  if (Array.isArray(who)) return who.filter(Boolean);
  return who ? [who] : [];
}

const LABEL_TITLE = { launch: 'Launch', copy: 'Copy', blocked: 'Blocked' };
const SEED_LABELS = ['launch', 'copy', 'blocked'];
const MAX_LABELS = 8;

function slugLabel(s) {
  return String(s || '')
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 24)
    .replace(/-+$/g, '');
}

function labelTitle(id) {
  if (LABEL_TITLE[id]) return LABEL_TITLE[id];
  return String(id || '').replace(/-/g, ' ').replace(/\b[a-z]/g, (c) => c.toUpperCase());
}

// MD3 container / on-container pairs — these are the roles the scheme
// guarantees as a contrast pair. Accent fills (primary, error, …) are not:
// a gold seed makes on-primary white on yellow.
const LABEL_PAIRS = [
  ['primary-container', 'on-primary-container'],
  ['secondary-container', 'on-secondary-container'],
  ['tertiary-container', 'on-tertiary-container'],
  ['error-container', 'on-error-container'],
  ['success-container', 'on-success-container'],
  ['warning-container', 'on-warning-container'],
  ['info-container', 'on-info-container'],
  ['inverse-surface', 'inverse-on-surface'],
];
const SEED_TONE = { launch: 0, copy: 1, blocked: 3 };
const CUSTOM_TONES = [2, 4, 5, 6, 7];

function labelTone(id) {
  if (SEED_TONE[id] != null) return String(SEED_TONE[id]);
  let h = 0;
  for (let i = 0; i < id.length; i++) h = Math.imul(h, 31) + id.charCodeAt(i);
  return String(CUSTOM_TONES[Math.abs(h) % CUSTOM_TONES.length]);
}

const LABEL_TONE_CSS = LABEL_PAIRS.map(([bg, fg], i) => `
    .tags ui-chip[data-tone="${i}"],
    .labels ui-chip[data-tone="${i}"] {
      --ui-chip-fg: var(--ui-color-${fg});
      --ui-chip-label-fg: var(--ui-color-${fg});
      --ui-chip-selected-bg: var(--ui-color-${bg});
      --ui-chip-selected-fg: var(--ui-color-${fg});
      --ui-chip-outline-color: transparent;
    }
    .tags ui-chip[data-tone="${i}"]::part(control),
    .labels ui-chip[data-tone="${i}"]::part(control) {
      background: var(--ui-color-${bg});
      color: var(--ui-color-${fg});
      border-color: transparent;
    }`).join('');

function catalogLabels(list, draft) {
  const seen = new Set();
  const out = [];
  const add = (id) => {
    const s = slugLabel(id);
    if (!s || seen.has(s)) return;
    seen.add(s);
    out.push(s);
  };
  for (const id of SEED_LABELS) add(id);
  for (const it of list || []) {
    for (const lab of it.labels || []) add(lab);
  }
  for (const lab of draft?.labels || []) add(lab);
  return out;
}

/**
 * Multi-select people picker: dismissible input chips plus <ui-autocomplete>.
 *
 * Alacris UI's combobox is single-select; this is the MD3 composition for
 * assigning several people — chips for who is on the card, a filtering
 * autocomplete to add someone else.
 *
 * @prop {string[]} value=[] selected names
 * @prop {string[]} options=[] people that can be assigned
 * @prop {string} label='Members'
 * @prop {string} placeholder='Search people'
 * @fires change {value: string[]} - the selection changed
 * @goname MemberSelect
 */
define('ala-member-select', {
  props: {
    value: [],
    options: ['You', 'Ada Lovelace', 'Ben Linus', 'Cara Moss'],
    label: 'Members',
    placeholder: 'Search people',
  },
  styles: css`
    :host { display: block; inline-size: 100%; }
    .picked {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      margin-block-end: 8px;
    }
    .picked:empty { display: none; margin: 0; }
    .who {
      display: inline-flex;
      align-items: center;
      gap: 8px;
    }
    ui-autocomplete { inline-size: 100%; }
    ui-chip { flex: none; }
  `,
  setup({ value, options, label, placeholder }, host) {
    const selected = () => (Array.isArray(value()) ? value() : []).filter(Boolean);
    const remaining = () => {
      const have = new Set(selected());
      return (options() || []).filter((n) => n && !have.has(n));
    };

    const emitValue = (next) => host.emit('change', { value: next });

    const onPick = (e) => {
      const name = String(e.detail?.value ?? '');
      const field = e.currentTarget;
      queueMicrotask(() => { field.value = ''; });
      if (!name || selected().includes(name)) return;
      emitValue([...selected(), name]);
    };

    const onDismiss = (e) => {
      const name = String(e.currentTarget?.value ?? '');
      if (!name) return;
      emitValue(selected().filter((n) => n !== name));
    };

    return html`
      <div class="picked">
        ${each(
          selected,
          (name) => html`
            <ui-chip variant="input" ?dismissible=${true} value=${() => name()}
              @dismiss=${onDismiss}>
              <span class="who">
                <ui-avatar name=${() => name()} size="24px"></ui-avatar>
                ${() => name()}
              </span>
            </ui-chip>`,
          (n) => n,
        )}
      </div>
      <ui-autocomplete
        label=${label}
        placeholder=${placeholder}
        options=${() => remaining()}
        @change.stop=${onPick}></ui-autocomplete>`;
  },
});

/**
 * The shared board.
 *
 * @prop {string}         title    the board name
 * @prop {go:[]todo.Item} items    the cards, as rendered by the server
 * @prop {go:[]todo.Column} columns the lists, left to right
 * @prop {boolean}        busy     true while the server is working
 * @goimport todo github.com/bmartel/alacris-go/examples/todo/model
 * @fires add        {text: string, column: string}                 - the user added a card
 * @fires move       {id: integer, column: string, index: integer}  - the user dropped a card
 * @fires addlist    {title: string}                                - the user added a list
 * @fires movelist   {id: string, index: integer}                   - the user dropped a list
 * @fires rename     {id: string, title: string}                    - the user renamed a list
 * @fires deletelist {id: string}                                   - the user deleted a list
 * @fires edit       {id: integer, text: string, body: string, who: string[], labels: string[]} - the user edited a card
 * @fires remove     {id: integer}                                  - the user deleted a card
 * @slot empty   - kept in the light DOM for first paint / no-JS
 * @slot members - overlapping avatars; the server rewrites this slot
 * @goname Board
 */
define('ala-board', {
  props: {
    title: 'Ship it',
    items: [],
    columns: [
      { id: 'todo', title: 'To do' },
      { id: 'doing', title: 'Doing' },
      { id: 'done', title: 'Done' },
    ],
    busy: false,
  },
  styles: css`
    :host {
      display: flex;
      flex-direction: column;
      min-block-size: 0;
      block-size: 100%;
      overflow: hidden;
    }
    :host([busy]) { opacity: .6 }
    .chrome {
      display: flex;
      align-items: center;
      gap: var(--ui-space-3, 12px);
      padding: var(--ui-space-3, 12px) var(--ui-space-5, 20px);
      flex: none;
      min-inline-size: 0;
      background: var(--ui-color-surface-container);
    }
    .brand {
      display: flex;
      align-items: center;
      gap: var(--ui-space-2, 8px);
      flex: none;
      min-inline-size: 0;
    }
    .brand h1 {
      margin: 0;
      font: var(--ui-type-title-lg);
      color: var(--ui-color-on-surface);
      letter-spacing: var(--ui-tracking-title-lg, 0);
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }
    .filter {
      flex: 1 1 12rem;
      min-inline-size: 8rem;
      max-inline-size: 22rem;
    }
    .end {
      display: flex;
      align-items: center;
      gap: var(--ui-space-3, 12px);
      margin-inline-start: auto;
      flex: none;
    }
    .members { display: flex; align-items: center; }
    .board {
      display: flex;
      gap: var(--ui-space-3, 12px);
      padding: var(--ui-space-3, 12px) var(--ui-space-5, 20px) var(--ui-space-4, 16px);
      overflow-x: auto;
      overflow-y: hidden;
      flex: 1;
      min-block-size: 0;
      align-items: start;
    }
    .lane {
      display: flex;
      flex-direction: column;
      flex: 1 1 0;
      min-inline-size: min(18rem, 80vw);
      min-block-size: 0;
      max-block-size: 100%;
      overflow: hidden;
      background: var(--ui-color-surface-container);
      border-radius: var(--ui-radius-md, 12px);
      padding: var(--ui-space-2, 8px);
    }
    .lane.over {
      outline: 2px solid var(--ui-color-primary);
      outline-offset: -2px;
    }
    .lane.dragging { visibility: hidden }
    .lane.insert-before { box-shadow: -3px 0 0 0 var(--ui-color-primary) }
    .lane.insert-after { box-shadow: 3px 0 0 0 var(--ui-color-primary) }
    .lane.add {
      flex: 0 0 min(18rem, 80vw);
      max-block-size: none;
      align-self: start;
    }
    .add-list {
      flex: 0 0 min(18rem, 80vw);
      align-self: start;
    }
    .add-list ui-button { inline-size: 100%; }
    .head {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: var(--ui-space-2, 8px);
      padding: var(--ui-space-2, 8px) var(--ui-space-2, 8px) var(--ui-space-1, 4px);
      font: var(--ui-type-title-sm);
      color: var(--ui-color-on-surface-variant);
      flex: none;
      cursor: grab;
      touch-action: none;
      user-select: none;
    }
    .head strong {
      flex: none;
      font: var(--ui-type-label-lg);
      color: var(--ui-color-on-surface);
    }
    .head .name,
    .head .name-edit {
      flex: 1;
      min-inline-size: 0;
      margin: 0;
      padding: 4px 6px;
      border: 0;
      border-radius: 6px;
      background: transparent;
      font: inherit;
      color: var(--ui-color-on-surface);
      text-align: start;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    .head .name { cursor: text; }
    .head .name:hover {
      background: var(--ui-color-surface-container-highest, rgb(0 0 0 / .06));
    }
    .head .name:focus-visible {
      outline: 2px solid var(--ui-color-primary);
      outline-offset: 0;
    }
    .head .name-edit {
      background: var(--ui-color-surface);
      outline: 2px solid var(--ui-color-primary);
      cursor: text;
      user-select: text;
    }
    .head-tools {
      display: flex;
      align-items: center;
      gap: 2px;
      flex: none;
      cursor: default;
    }
    .head-tools ui-menu { flex: none; }
    .cards {
      flex: 1 1 auto;
      min-block-size: 0;
      overflow-y: auto;
      display: flex;
      flex-direction: column;
      gap: var(--ui-space-2, 8px);
      padding: var(--ui-space-1, 4px);
    }
    .cards.drop-end {
      box-shadow: inset 0 -3px 0 var(--ui-color-primary);
    }
    .card {
      position: relative;
      touch-action: none;
      user-select: none;
      cursor: pointer;
    }
    .card.insert-before::before {
      content: '';
      position: absolute;
      inset-inline: 0;
      top: -5px;
      height: 3px;
      border-radius: 2px;
      background: var(--ui-color-primary);
      pointer-events: none;
    }
    .card.dragging { visibility: hidden }
    .card.filtered { display: none }
    .card ui-card { pointer-events: none }
    .ghost {
      position: fixed;
      z-index: 8;
      pointer-events: none;
      rotate: 2deg;
      filter: drop-shadow(0 8px 18px rgb(0 0 0 / .4));
    }
    .ghost.lane-ghost {
      rotate: 1deg;
      background: var(--ui-color-surface-container);
      border-radius: var(--ui-radius-md, 12px);
      padding: var(--ui-space-3, 12px);
      font: var(--ui-type-title-sm);
      color: var(--ui-color-on-surface);
      max-block-size: min(80vh, 24rem);
    }
    .card-title {
      display: block;
      font: var(--ui-type-body-lg);
      color: var(--ui-color-on-surface);
    }
    .tags:empty { display: none }
    .tags {
      display: flex;
      flex-wrap: wrap;
      gap: 6px;
      margin-block-end: 8px;
    }
    .tags ui-chip { --ui-chip-height: 24px; }
    ${LABEL_TONE_CSS}
    .label-create {
      display: flex;
      align-items: flex-end;
      gap: var(--ui-space-2, 8px);
    }
    .label-create ui-text-field { flex: 1; min-inline-size: 0; }
    .label-create ui-button { flex: none; }
    .faces {
      display: flex;
      justify-content: flex-end;
      align-items: center;
      flex: none;
      margin-block-start: 8px;
    }
    .faces:empty { display: none }
    .faces ui-avatar {
      margin-inline-start: -8px;
      box-shadow: 0 0 0 2px var(--ui-color-surface-container-highest, var(--ui-color-surface));
    }
    .faces ui-avatar:first-child { margin-inline-start: 0; }
    .card-excerpt {
      margin: 6px 0 0;
      font: var(--ui-type-body-sm, var(--ui-type-body-md));
      color: var(--ui-color-on-surface-variant);
      display: -webkit-box;
      -webkit-line-clamp: 2;
      -webkit-box-orient: vertical;
      overflow: hidden;
    }
    .composer {
      display: flex;
      flex-direction: column;
      gap: var(--ui-space-2, 8px);
      padding: var(--ui-space-1, 4px);
      flex: none;
      background: var(--ui-color-surface-container);
    }
    .composer ui-text-field { inline-size: 100% }
    .card-dialog { --ui-dialog-width: min(40rem, calc(100vw - 32px)); }
    .card-editor {
      display: flex;
      flex-direction: column;
      gap: var(--ui-space-4, 16px);
      min-inline-size: 0;
    }
    .card-editor ui-text-field,
    .card-editor ui-select,
    .card-editor ala-member-select { inline-size: 100%; }
    .section-title {
      margin: 0;
      display: flex;
      align-items: center;
      gap: var(--ui-space-2, 8px);
      font: var(--ui-type-title-sm);
      color: var(--ui-color-on-surface);
    }
    .hint {
      margin: 0;
      font: var(--ui-type-body-sm, var(--ui-type-body-md));
      color: var(--ui-color-on-surface-variant);
    }
    .danger {
      --ui-button-filled-bg: var(--ui-color-error);
      --ui-button-filled-fg: var(--ui-color-on-error);
    }
    .danger-text { --ui-button-text-fg: var(--ui-color-error); }
    .empty { display: none }
  `,
  setup({ title, items, columns, busy }, host) {
    const adding = signal('');
    const addingList = signal(false);
    const editing = signal('');
    const draft = signal(null);
    const confirm = signal(null);
    const query = signal('');
    const dragging = signal(null);
    const dropAt = signal(null);
    const preview = signal(null);
    const laneDrag = signal(null);
    const laneDrop = signal(null);
    const colPreview = signal(null);
    let ghostEl = null;
    let session = null;
    let laneSession = null;
    let suppressEdit = false;
    let suppressOpen = false;
    let pendingScroll = null;

    const revealAdded = (column, text) => {
      const card = items().filter((c) => c.column === column && c.text === text).at(-1);
      if (!card) return;
      pendingScroll = null;
      const go = () => {
        const el = host.shadowRoot.querySelector(
          `[data-column="${column}"] .card[data-id="${card.id}"]`,
        );
        el?.scrollIntoView({ block: 'nearest', inline: 'nearest', behavior: 'smooth' });
      };
      requestAnimationFrame(() => requestAnimationFrame(go));
    };

    effect(() => {
      items();
      columns();
      preview(null);
      colPreview(null);
      const d = draft();
      if (d && !items().some((c) => c.id === d.id)) draft(null);
      if (pendingScroll) revealAdded(pendingScroll.column, pendingScroll.text);
    });

    effect(() => {
      const d = draft();
      if (!d) return;
      queueMicrotask(() => {
        const set = (sel, val) => {
          const field = host.shadowRoot.querySelector(sel);
          if (!field) return;
          const input = field.shadowRoot?.querySelector('input, textarea');
          if (input) input.value = val;
          field.value = val;
        };
        set('.card-title-field', d.text);
        set('.card-body-field', d.body || '');
      });
    });

    const board = () => preview() ?? items();
    const lists = () => colPreview() ?? columns();
    const inLane = (column) => board().filter((c) => c.column === column);

    const matches = (item) => {
      const q = query().trim().toLowerCase();
      if (!q) return true;
      return item.text.toLowerCase().includes(q)
        || asWho(item.who).some((w) => w.toLowerCase().includes(q))
        || String(item.body || '').toLowerCase().includes(q)
        || (item.labels || []).some((l) => l.includes(q) || labelTitle(l).toLowerCase().includes(q));
    };

    const hit = (clientX, clientY, dragId) => {
      const lanes = [...host.shadowRoot.querySelectorAll('.lane[data-column]')];
      let laneEl = null;
      for (const lane of lanes) {
        const r = lane.getBoundingClientRect();
        if (clientX >= r.left && clientX < r.right) {
          laneEl = lane;
          break;
        }
      }
      if (!laneEl) return null;
      const column = laneEl.dataset.column;
      const cards = [...laneEl.querySelectorAll('.card')].filter(
        (c) => Number(c.dataset.id) !== dragId && !c.classList.contains('filtered'),
      );
      let index = cards.length;
      for (let i = 0; i < cards.length; i++) {
        const r = cards[i].getBoundingClientRect();
        if (clientY < r.top + r.height / 2) {
          index = i;
          break;
        }
      }
      return { column, index };
    };

    const startDrag = (item) => (e) => {
      if (e.button !== 0 || laneSession || draft()) return;
      const origin = e.currentTarget;
      const rect = origin.getBoundingClientRect();
      const id = item().id;
      const originX = e.clientX;
      const originY = e.clientY;
      const ox = e.clientX - rect.left;
      const oy = e.clientY - rect.top;
      let started = false;
      session = null;

      const onMove = (ev) => {
        if (!started) {
          if (Math.hypot(ev.clientX - originX, ev.clientY - originY) < 8) return;
          started = true;
          try { origin.setPointerCapture(ev.pointerId); } catch (_) { /* already released */ }
          session = {
            id,
            text: item().text,
            who: asWho(item().who)[0] || '',
            w: rect.width,
            x: ev.clientX - ox,
            y: ev.clientY - oy,
          };
          dragging(session);
          const fromAt = inLane(item().column).findIndex((c) => c.id === id);
          dropAt({ column: item().column, index: fromAt < 0 ? 0 : fromAt });
        }
        if (started) ev.preventDefault();
        if (ghostEl) {
          ghostEl.style.left = `${ev.clientX - ox}px`;
          ghostEl.style.top = `${ev.clientY - oy}px`;
        }
        const next = hit(ev.clientX, ev.clientY, id);
        const cur = dropAt();
        if (next && (!cur || cur.column !== next.column || cur.index !== next.index)) {
          dropAt(next);
        }
      };

      const onUp = () => {
        window.removeEventListener('pointermove', onMove);
        window.removeEventListener('pointerup', onUp);
        window.removeEventListener('pointercancel', onUp);
        if (started) suppressOpen = true;
        const d = dropAt();
        const drag = session;
        session = null;
        ghostEl = null;
        if (!started || !d || !drag) {
          batch(() => { dragging(null); dropAt(null); });
          return;
        }
        const from = items().find((c) => c.id === drag.id);
        if (!from) {
          batch(() => { dragging(null); dropAt(null); });
          return;
        }
        if (from.column === d.column) {
          const fromAt = items().filter((c) => c.column === d.column).findIndex((c) => c.id === drag.id);
          if (fromAt === d.index) {
            batch(() => { dragging(null); dropAt(null); });
            return;
          }
        }
        batch(() => {
          preview(applyMove(items(), columns(), drag.id, d.column, d.index));
          dragging(null);
          dropAt(null);
        });
        host.emit('move', { id: drag.id, column: d.column, index: d.index });
      };

      window.addEventListener('pointermove', onMove, { passive: false });
      window.addEventListener('pointerup', onUp);
      window.addEventListener('pointercancel', onUp);
    };

    const hitLane = (clientX, dragId) => {
      const lanes = [...host.shadowRoot.querySelectorAll('.lane[data-column]')].filter(
        (el) => el.dataset.column !== dragId,
      );
      let index = lanes.length;
      for (let i = 0; i < lanes.length; i++) {
        const r = lanes[i].getBoundingClientRect();
        if (clientX < r.left + r.width / 2) {
          index = i;
          break;
        }
      }
      return index;
    };

    const startLaneDrag = (col) => (e) => {
      if (e.button !== 0 || session) return;
      if (editing() === col().id) return;
      if (e.target.closest?.('.name-edit, ui-menu, ui-icon-button')) return;
      const origin = e.currentTarget.closest('.lane') || e.currentTarget;
      const rect = origin.getBoundingClientRect();
      const id = col().id;
      const originX = e.clientX;
      const originY = e.clientY;
      const ox = e.clientX - rect.left;
      const oy = e.clientY - rect.top;
      let started = false;
      laneSession = null;

      const onMove = (ev) => {
        if (!started) {
          if (Math.hypot(ev.clientX - originX, ev.clientY - originY) < 8) return;
          started = true;
          try { origin.setPointerCapture(ev.pointerId); } catch (_) { /* already released */ }
          const fromAt = lists().findIndex((c) => c.id === id);
          laneSession = {
            id,
            title: col().title,
            count: inLane(id).length,
            w: rect.width,
            x: ev.clientX - ox,
            y: ev.clientY - oy,
          };
          laneDrag(laneSession);
          laneDrop(fromAt < 0 ? 0 : fromAt);
        }
        if (started) ev.preventDefault();
        if (ghostEl) {
          ghostEl.style.left = `${ev.clientX - ox}px`;
          ghostEl.style.top = `${ev.clientY - oy}px`;
        }
        const next = hitLane(ev.clientX, id);
        if (laneDrop() !== next) laneDrop(next);
      };

      const onUp = () => {
        window.removeEventListener('pointermove', onMove);
        window.removeEventListener('pointerup', onUp);
        window.removeEventListener('pointercancel', onUp);
        if (started) suppressEdit = true;
        const at = laneDrop();
        const drag = laneSession;
        laneSession = null;
        ghostEl = null;
        if (!started || drag == null || at == null) {
          batch(() => { laneDrag(null); laneDrop(null); });
          return;
        }
        const fromAt = columns().findIndex((c) => c.id === drag.id);
        if (fromAt === at) {
          batch(() => { laneDrag(null); laneDrop(null); });
          return;
        }
        batch(() => {
          colPreview(applyMoveColumn(columns(), drag.id, at));
          laneDrag(null);
          laneDrop(null);
        });
        host.emit('movelist', { id: drag.id, index: at });
      };

      window.addEventListener('pointermove', onMove, { passive: false });
      window.addEventListener('pointerup', onUp);
      window.addEventListener('pointercancel', onUp);
    };

    const fieldValue = (rootSel) => {
      const field = host.shadowRoot.querySelector(`${rootSel} ui-text-field`);
      const input = field?.shadowRoot?.querySelector('input');
      return { field, input, text: String(input?.value || field?.value || '').trim() };
    };

    const addTo = (column) => {
      const { field, input, text } = fieldValue(`[data-column="${column}"]`);
      if (!text) return;
      pendingScroll = { column, text };
      host.emit('add', { text, column });
      if (input) input.value = '';
      if (field) field.value = '';
      adding('');
    };

    const addList = () => {
      const { field, input, text } = fieldValue('.lane.add');
      if (!text) return;
      host.emit('addlist', { title: text });
      if (input) input.value = '';
      if (field) field.value = '';
      addingList(false);
    };

    const openRename = (col) => () => {
      if (suppressEdit) {
        suppressEdit = false;
        return;
      }
      editing(col().id);
    };

    const bindRename = (col) => (el) => {
      if (!el || el.dataset.alaBound) return;
      el.dataset.alaBound = '1';
      el.value = col().title;
      el.select();
      let cancelled = false;
      const save = () => {
        if (cancelled) return;
        const title = String(el.value || '').trim();
        const id = col().id;
        const current = lists().find((c) => c.id === id)?.title;
        editing('');
        if (!title || title === current) return;
        colPreview(lists().map((c) => (c.id === id ? { ...c, title } : c)));
        host.emit('rename', { id, title });
      };
      el.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
          e.preventDefault();
          save();
        }
        if (e.key === 'Escape') {
          e.preventDefault();
          cancelled = true;
          editing('');
        }
      });
      el.addEventListener('blur', save);
      el.focus();
    };

    const bindInput = (submit, cancel) => (field) => {
      if (!field) return;
      const attach = () => {
        const input = field.shadowRoot?.querySelector('input');
        if (!input || input.dataset.alaBound) return;
        input.dataset.alaBound = '1';
        input.addEventListener('keydown', (e) => {
          if (e.key === 'Enter') {
            e.preventDefault();
            submit();
          }
          if (e.key === 'Escape') cancel();
        });
        input.focus();
      };
      customElements.whenDefined('ui-text-field').then(() => {
        attach();
        if (!field.shadowRoot?.querySelector('input')) queueMicrotask(attach);
      });
    };

    const openCard = (item) => () => {
      if (suppressOpen) {
        suppressOpen = false;
        return;
      }
      const it = item();
      draft({
        id: it.id,
        text: it.text,
        body: it.body || '',
        who: asWho(it.who),
        column: it.column,
        labels: [...(it.labels || [])],
      });
    };

    const bindDraftField = (key) => (field) => {
      if (!field) return;
      const d = draft();
      if (!d) return;
      const attach = () => {
        const input = field.shadowRoot?.querySelector('input, textarea');
        if (!input) return;
        const token = `${d.id}:${key}`;
        if (input.dataset.alaBound === token) return;
        input.dataset.alaBound = token;
        input.value = d[key] || '';
        if (field.value !== input.value) field.value = input.value;
      };
      customElements.whenDefined('ui-text-field').then(() => {
        attach();
        if (!field.shadowRoot?.querySelector('input, textarea')) queueMicrotask(attach);
      });
    };

    const commitCard = (patch) => {
      const d = draft();
      if (!d) return;
      const next = { ...d, ...patch };
      if (patch.text !== undefined) {
        const t = String(next.text || '').trim();
        if (!t) return;
        next.text = t.slice(0, 200);
      }
      if (patch.body !== undefined) {
        next.body = String(next.body || '').slice(0, 2000);
      }
      draft(next);
      const cur = items().find((c) => c.id === next.id);
      if (
        cur
        && cur.text === next.text
        && (cur.body || '') === (next.body || '')
        && sameLabels(asWho(cur.who), asWho(next.who))
        && sameLabels(cur.labels, next.labels)
      ) {
        return;
      }
      preview(applyPatch(board(), next.id, {
        text: next.text,
        body: next.body,
        who: next.who,
        labels: next.labels,
      }));
      host.emit('edit', {
        id: next.id,
        text: next.text,
        body: next.body || '',
        who: next.who || [],
        labels: next.labels || [],
      });
    };

    const readDraftField = (sel, key) => {
      const field = host.shadowRoot.querySelector(sel);
      const input = field?.shadowRoot?.querySelector('input, textarea');
      const value = String(input?.value ?? field?.value ?? draft()?.[key] ?? '');
      return key === 'text' ? value.trim() : value;
    };

    const closeCard = () => {
      const d = draft();
      if (d) {
        commitCard({
          text: readDraftField('.card-title-field', 'text') || d.text,
          body: readDraftField('.card-body-field', 'body'),
        });
      }
      draft(null);
    };

    // ui-select/ui-menu emit a composed `close` when their panel shuts. That
    // is not the user leaving the editor — only Escape (on this dialog) or
    // the Close button is. Scrim clicks are ignored so a dropdown sitting
    // over the overlay cannot dismiss the card.
    const onCardClose = (e) => {
      if (e.target !== e.currentTarget) return;
      if (e.detail?.reason === 'scrim') return;
      closeCard();
    };

    const onLabelChange = (e) => {
      const id = slugLabel(e.currentTarget?.value);
      if (!id) return;
      const selected = !!e.detail?.selected;
      const d = draft();
      if (!d) return;
      const has = (d.labels || []).includes(id);
      if (selected === has) return;
      if (selected && (d.labels || []).length >= MAX_LABELS) {
        showSnackbar('A card can have 8 labels');
        return;
      }
      const labels = selected
        ? [...(d.labels || []), id]
        : (d.labels || []).filter((x) => x !== id);
      commitCard({ labels });
    };

    const clearNewLabel = () => {
      const field = host.shadowRoot.querySelector('.new-label-field');
      const input = field?.shadowRoot?.querySelector('input');
      if (input) input.value = '';
      if (field) field.value = '';
    };

    const addLabelFromField = () => {
      const field = host.shadowRoot.querySelector('.new-label-field');
      const input = field?.shadowRoot?.querySelector('input');
      const id = slugLabel(input?.value ?? field?.value ?? '');
      if (!id) return;
      const d = draft();
      if (!d) return;
      if ((d.labels || []).includes(id)) {
        clearNewLabel();
        return;
      }
      if ((d.labels || []).length >= MAX_LABELS) {
        showSnackbar('A card can have 8 labels');
        return;
      }
      commitCard({ labels: [...(d.labels || []), id] });
      clearNewLabel();
    };

    const moveDraft = (column) => {
      const d = draft();
      if (!d || !column || column === d.column) return;
      draft({ ...d, column });
      preview(applyMove(board(), lists(), d.id, column, -1));
      host.emit('move', { id: d.id, column, index: -1 });
    };

    const onMembersChange = (e) => {
      commitCard({ who: asWho(e.detail?.value) });
    };

    const onListAction = (col) => (e) => {
      const value = String(e.detail?.value ?? '');
      if (value === 'add') adding(col().id);
      if (value === 'delete') {
        confirm({
          kind: 'list',
          id: col().id,
          title: col().title,
          count: inLane(col().id).length,
        });
      }
    };

    const askDeleteCard = () => {
      const d = draft();
      if (!d) return;
      confirm({ kind: 'card', id: d.id, title: d.text, count: 1 });
    };

    const runConfirm = () => {
      const c = confirm();
      confirm(null);
      if (!c) return;
      if (c.kind === 'list') {
        const d = draft();
        if (d && board().find((it) => it.id === d.id)?.column === c.id) draft(null);
        batch(() => {
          colPreview(lists().filter((x) => x.id !== c.id));
          preview(board().filter((it) => it.column !== c.id));
        });
        host.emit('deletelist', { id: c.id });
        return;
      }
      draft(null);
      preview(board().filter((it) => it.id !== c.id));
      host.emit('remove', { id: c.id });
    };

    const share = async () => {
      try {
        await navigator.clipboard.writeText(location.href);
        showSnackbar('Board link copied');
      } catch {
        showSnackbar('Copy the URL from the address bar');
      }
    };

    const cardClass = (column, item) => () => {
      let cls = 'card';
      const it = item();
      const drag = dragging();
      if (drag && drag.id === it.id) cls += ' dragging';
      if (!matches(it)) cls += ' filtered';
      const d = dropAt();
      if (d && d.column === column && !(drag && drag.id === it.id)) {
        const list = inLane(column).filter((c) => !drag || c.id !== drag.id);
        const idx = list.findIndex((c) => c.id === it.id);
        if (idx === d.index) cls += ' insert-before';
      }
      return cls;
    };

    const cardsClass = (column) => () => {
      const d = dropAt();
      const drag = dragging();
      if (!d || d.column !== column) return 'cards';
      const list = inLane(column).filter((c) => !drag || c.id !== drag.id);
      return d.index >= list.length ? 'cards drop-end' : 'cards';
    };

    const laneClass = (col) => () => {
      let cls = 'lane';
      const id = col().id;
      const drag = laneDrag();
      if (drag && drag.id === id) cls += ' dragging';
      if (!drag && dropAt()?.column === id) cls += ' over';
      const at = laneDrop();
      if (drag && at != null && drag.id !== id) {
        const rest = lists().filter((c) => c.id !== drag.id);
        const idx = rest.findIndex((c) => c.id === id);
        if (idx === at) cls += ' insert-before';
        if (at >= rest.length && idx === rest.length - 1) cls += ' insert-after';
      }
      return cls;
    };

    const lane = (col) => html`
      <section class=${laneClass(col)}
        data-column=${() => col().id}>
        <header class="head" @pointerdown.capture=${startLaneDrag(col)}>
          ${() => (editing() === col().id
            ? html`<input class="name-edit" aria-label="List title" maxlength="80" spellcheck="false"
                @pointerdown=${(e) => e.stopPropagation()}
                ref=${bindRename(col)}>`
            : html`<button type="button" class="name" @click=${openRename(col)}>${col().title}</button>`)}
          <div class="head-tools" @pointerdown=${(e) => e.stopPropagation()}>
            <strong>${() => inLane(col().id).length}</strong>
            <ui-menu placement="bottom-end" @select=${onListAction(col)}>
              <ui-icon-button slot="anchor" icon="more-vert" label="List actions"></ui-icon-button>
              <ui-menu-item value="add" icon="add">Add a card</ui-menu-item>
              <ui-menu-item value="delete" icon="delete" ?danger=${true}>Delete list</ui-menu-item>
            </ui-menu>
          </div>
        </header>
        <div class=${() => cardsClass(col().id)() } role="list">
          ${each(
            () => inLane(col().id),
            (item) => html`
              <div
                class=${() => cardClass(col().id, item)()}
                role="listitem"
                data-id=${() => item().id}
                data-text=${() => item().text}
                @pointerdown.capture=${startDrag(item)}
                @click=${openCard(item)}>
                <ui-card data-text=${() => item().text}>
                  <div class="tags">
                    ${each(
                      () => item().labels || [],
                      (lab) => html`<ui-chip data-label=${() => lab()} data-tone=${() => labelTone(lab())}>${() => labelTitle(lab())}</ui-chip>`,
                      (l) => l,
                    )}
                  </div>
                  <span class="card-title">${() => item().text}</span>
                  ${() => (item().body
                    ? html`<p class="card-excerpt">${item().body}</p>`
                    : '')}
                  <div class="faces">
                    ${each(
                      () => asWho(item().who),
                      (w) => html`<ui-avatar name=${() => w()} size="28px"></ui-avatar>`,
                      (n) => n,
                    )}
                  </div>
                </ui-card>
              </div>`,
            (c) => c.id,
          )}
        </div>
        <div class="composer">
          ${() => (adding() === col().id
            ? html`
                <ui-text-field
                  style="inline-size:100%"
                  label="Add a card"
                  ?disabled=${busy}
                  ref=${bindInput(() => addTo(col().id), () => adding(''))}></ui-text-field>
                <ui-button variant="filled" ?disabled=${busy} @click=${() => addTo(col().id)}>Add</ui-button>`
            : html`<ui-button variant="text" ?disabled=${busy} @click=${() => adding(col().id)}>Add a card</ui-button>`)}
        </div>
      </section>`;

    return html`
      <header class="chrome">
        <div class="brand">
          <ui-icon name="view-column"></ui-icon>
          <h1>${() => title()}</h1>
        </div>
        <ui-search class="filter"
          label="Filter cards"
          placeholder="Filter cards"
          @input=${(e) => query(String(e.detail?.value ?? ''))}
          @clear=${() => query('')}></ui-search>
        <div class="end">
          <div class="members" aria-label="Board members">
            <slot name="members"></slot>
          </div>
          <ui-button variant="filled" @click=${share}>Share</ui-button>
        </div>
      </header>
      <div class="board">
        ${each(() => lists(), (col) => lane(col), (c) => c.id)}
        ${() => (addingList()
          ? html`
              <section class="lane add">
                <div class="composer">
                  <ui-text-field
                    style="inline-size:100%"
                    label="List title"
                    ?disabled=${busy}
                    ref=${bindInput(addList, () => addingList(false))}></ui-text-field>
                  <ui-button variant="filled" ?disabled=${busy} @click=${addList}>Add list</ui-button>
                </div>
              </section>`
          : html`
              <div class="add-list">
                <ui-button variant="tonal" ?disabled=${busy} @click=${() => addingList(true)}>Add another list</ui-button>
              </div>`)}
      </div>
      ${() => {
        const d = dragging();
        if (d) {
          return html`
            <div class="ghost" style=${`width:${d.w}px;left:${d.x}px;top:${d.y}px`}
              ref=${(el) => { ghostEl = el; }}>
              <ui-card>
                <span class="card-title">${d.text}</span>
              </ui-card>
            </div>`;
        }
        const l = laneDrag();
        if (!l) return '';
        return html`
          <div class="ghost lane-ghost" style=${`width:${l.w}px;left:${l.x}px;top:${l.y}px`}
            ref=${(el) => { ghostEl = el; }}>
            ${l.title} · ${l.count}
          </div>`;
      }}
      <ui-dialog class="card-dialog" open=${() => !!draft()} @close=${onCardClose} label="Card">
        <div class="card-editor">
            <ui-text-field
              class="card-title-field"
              label="Title"
              ref=${bindDraftField('text')}
              @change.stop=${(e) => commitCard({ text: String(e.detail?.value ?? '') })}></ui-text-field>
            <ui-select
              label="List"
              value=${() => draft()?.column ?? ''}
              @close=${(e) => e.stopPropagation()}
              @change.stop=${(e) => moveDraft(String(e.detail?.value ?? ''))}>
              ${each(
                () => lists(),
                (col) => html`<ui-option value=${() => col().id}>${() => col().title}</ui-option>`,
                (c) => c.id,
              )}
            </ui-select>
            <h3 class="section-title"><ui-icon name="label"></ui-icon> Labels</h3>
            <p class="hint">Select a label to add or remove it, or create one. Saved as you go.</p>
            <ui-chip-set class="labels" label="Labels" ?multi=${true}>
              ${each(
                () => catalogLabels(board(), draft()),
                (lab) => html`<ui-chip variant="filter" value=${() => lab()}
                  data-tone=${() => labelTone(lab())}
                  ?selected=${() => (draft()?.labels || []).includes(lab())}
                  @change.stop=${onLabelChange}>${() => labelTitle(lab())}</ui-chip>`,
                (l) => l,
              )}
            </ui-chip-set>
            <div class="label-create">
              <ui-text-field class="new-label-field" label="New label" maxlength="24"
                @change.stop=${addLabelFromField}></ui-text-field>
              <ui-button variant="tonal" @click.stop=${addLabelFromField}>Add label</ui-button>
            </div>
            <h3 class="section-title">Description</h3>
            <ui-text-field
              class="card-body-field"
              label="Description"
              type="textarea"
              rows="5"
              ref=${bindDraftField('body')}
              @change.stop=${(e) => commitCard({ body: String(e.detail?.value ?? '') })}></ui-text-field>
            <h3 class="section-title"><ui-icon name="person"></ui-icon> Members</h3>
            <p class="hint">Search to add people. Remove a chip to take them off. Saved as you go.</p>
            <ala-member-select
              value=${() => asWho(draft()?.who)}
              @change.stop=${onMembersChange}></ala-member-select>
        </div>
        <ui-button slot="actions" variant="text" class="danger-text" @click.stop=${askDeleteCard}>Delete</ui-button>
        <ui-button slot="actions" variant="filled" @click.stop=${closeCard}>Close</ui-button>
      </ui-dialog>
      <ui-dialog open=${() => !!confirm()} @close=${() => confirm(null)}>
        <span slot="headline">${() => (confirm()?.kind === 'list' ? 'Delete list?' : 'Delete card?')}</span>
        ${() => {
          const c = confirm();
          if (!c) return '';
          if (c.kind === 'list') {
            if (!c.count) return `${c.title} has no cards. The list will be removed from the board.`;
            if (c.count === 1) return `${c.title} and its card will be removed from the board.`;
            return `${c.title} and its ${c.count} cards will be removed from the board.`;
          }
          return `${c.title} will be removed from the board.`;
        }}
        <ui-button slot="actions" variant="text" @click.stop=${() => confirm(null)}>Cancel</ui-button>
        <ui-button slot="actions" variant="filled" class="danger" @click.stop=${runConfirm}>Delete</ui-button>
      </ui-dialog>
      <span class="empty"><slot name="empty"></slot></span>`;
  },
});
