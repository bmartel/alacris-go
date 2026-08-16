// <ui-time-picker> — Material time picker: a text-field-style control that
// opens an hour/minute chooser. Value is a 24-hour `HH:mm` string.
//
//   <ui-time-picker label="Alarm" value=${time}
//                   @change=${(e) => time(e.detail.value)}></ui-time-picker>
//
// The keyboard icon in the panel toggles between the analog dial and the
// digital hour/minute grids (MD3 input-method toggle). Hour and minute
// faces crossfade; the clock hand rotates with the motion tokens.
//
// @prop  {string}  label=''
// @prop  {string}  value=''         — 24-hour HH:mm, or '' for none
// @prop  {string}  variant='filled' — filled | outlined
// @prop  {string}  view='clock'     — clock | input (dial vs digital grid)
// @prop  {string}  hourCycle='12'   — 12 | 24
// @prop  {number}  minuteStep=5     — minute choices (1, 5, or 15 typical)
// @prop  {string}  locale=''        — BCP 47 tag; empty uses the runtime locale
// @prop  {boolean} disabled=false
// @prop  {boolean} required=false
// @prop  {string}  name=''          — form participation
// @prop  {string}  placeholder=''
// @event change — committed; detail: { value }
// @event input  — field keystroke; detail: { value } (the raw text)
// @event open   — panel visible (after the enter animation)
// @event close  — panel removed (after the exit animation)
// @part  field, input, label, panel, dial
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, computed, signal, effect, onCleanup, each } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { formBind } from '../util/form.js';
import { presence } from '../motion/presence.js';
import { fx } from '../motion/animate.js';
import { autoUpdate } from '../util/position.js';
import { rovingTabindex } from '../util/keys.js';
import './ui-icon-button.js';

const TIME = /^(\d{1,2}):(\d{2})$/;
const pad = (n) => String(n).padStart(2, '0');
const toValue = (h, m) => `${pad(((h % 24) + 24) % 24)}:${pad(((m % 60) + 60) % 60)}`;
const parseTime = (s) => {
  const m = TIME.exec((s || '').trim());
  if (!m) return null;
  const h = +m[1];
  const min = +m[2];
  if (h > 23 || min > 59) return null;
  return { h, m: min };
};
const loc = (locale) => locale || undefined;
const formatTime = (value, locale, hourCycle) => {
  const t = parseTime(value);
  if (!t) return '';
  const d = new Date(2026, 0, 1, t.h, t.m);
  return new Intl.DateTimeFormat(loc(locale), {
    hour: 'numeric',
    minute: '2-digit',
    hourCycle: hourCycle === '24' ? 'h23' : 'h12',
  }).format(d);
};
const nowValue = () => {
  const n = new Date();
  return toValue(n.getHours(), n.getMinutes());
};

const t = vars('ui-time-picker', {
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
  digitBg: sys.color.surfaceContainerHighest,
  selectedBg: sys.color.primaryContainer,
  selectedFg: sys.color.onPrimaryContainer,
});

const styles = css`
  :host { display: block; inline-size: 240px; }
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
    z-index: ${sys.z.modal};
    display: flex;
    flex-direction: column;
    gap: ${sys.space(3)};
    padding: ${sys.space(4)};
    min-inline-size: 320px;
    background: ${t.panelBg};
    border-radius: ${t.panelRadius};
    box-shadow: ${sys.elevation[3]};
    color: ${t.fg};
    overflow: auto;
  }
  .face {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: ${sys.space(1)};
  }
  .digit {
    min-inline-size: 72px;
    block-size: 64px;
    margin: 0;
    padding: 0;
    border: none;
    outline: none;
    appearance: none;
    border-radius: ${sys.radius.sm};
    background: ${t.digitBg};
    color: ${t.fg};
    font: ${sys.type.displaySm};
    letter-spacing: ${sys.tracking.displaySm};
    cursor: pointer;
    transition: background-color ${sys.duration.short4} ${sys.easing.standard},
                color ${sys.duration.short4} ${sys.easing.standard};
  }
  .digit.active { background: ${t.selectedBg}; color: ${t.selectedFg}; }
  .digit:focus-visible { outline: ${sys.focus.ring}; outline-offset: 2px; }
  .colon {
    font: ${sys.type.displaySm};
    color: ${t.fg};
    padding-inline: calc(${sys.space(1)} / 2);
  }
  .period {
    display: flex;
    flex-direction: column;
    margin-inline-start: ${sys.space(2)};
    border: 1px solid ${t.outlineColor};
    border-radius: ${sys.radius.xs};
    overflow: hidden;
  }
  .period button {
    min-inline-size: 48px;
    block-size: 32px;
    margin: 0;
    padding: 0;
    border: none;
    outline: none;
    appearance: none;
    background: transparent;
    color: ${t.labelFg};
    font: ${sys.type.labelMd};
    letter-spacing: ${sys.tracking.labelMd};
    cursor: pointer;
    transition: background-color ${sys.duration.short4} ${sys.easing.standard},
                color ${sys.duration.short4} ${sys.easing.standard};
  }
  .period button.active { background: ${t.selectedBg}; color: ${t.selectedFg}; }
  .period button:focus-visible { outline: ${sys.focus.ring}; outline-offset: -2px; }

  .grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: ${sys.space(1)};
    transition: opacity ${sys.duration.short4} ${sys.easing.standard};
  }
  .grid.off {
    opacity: 0;
    pointer-events: none;
  }
  .choice {
    position: relative;
    isolation: isolate;
    block-size: 40px;
    margin: 0;
    padding: 0;
    border: none;
    outline: none;
    appearance: none;
    background: transparent;
    border-radius: ${sys.radius.full};
    font: ${sys.type.bodyMd};
    letter-spacing: ${sys.tracking.bodyMd};
    color: ${t.fg};
    cursor: pointer;
    transition: background-color ${sys.duration.short4} ${sys.easing.standard},
                color ${sys.duration.short4} ${sys.easing.standard};
  }
  .choice .layer {
    position: absolute; inset: 0; z-index: -1;
    border-radius: inherit; background: currentColor; opacity: 0;
  }
  .choice:hover .layer { opacity: ${sys.state.hover}; }
  .choice:active .layer { opacity: ${sys.state.pressed}; }
  .choice.active { background: ${t.accent}; color: ${t.onAccent}; }
  .choice:focus-visible { outline: ${sys.focus.ring}; outline-offset: -2px; }

  .dial {
    position: relative;
    inline-size: 256px;
    block-size: 256px;
    margin-inline: auto;
    border-radius: 50%;
    background: ${t.digitBg};
  }
  .hand {
    position: absolute;
    inset-inline-start: 50%;
    inset-block-start: 50%;
    inline-size: 2px;
    block-size: 108px;
    margin-inline-start: -1px;
    margin-block-start: -108px;
    background: ${t.accent};
    transform-origin: center bottom;
    pointer-events: none;
    z-index: 0;
    transition: transform ${sys.duration.medium4} ${sys.easing.emphasized};
  }
  .hand::after {
    content: '';
    position: absolute;
    inset-inline-start: 50%;
    inset-block-start: 0;
    inline-size: 40px;
    block-size: 40px;
    margin-inline-start: -20px;
    margin-block-start: -20px;
    border-radius: 50%;
    background: ${t.accent};
  }
  .hub {
    position: absolute;
    inset: 0;
    margin: auto;
    inline-size: 8px;
    block-size: 8px;
    border-radius: 50%;
    background: ${t.accent};
    pointer-events: none;
    z-index: 2;
  }
  .tick {
    position: absolute;
    inset: 0;
    z-index: 1;
    margin: auto;
    inline-size: 40px;
    block-size: 40px;
    border: none;
    outline: none;
    appearance: none;
    border-radius: 50%;
    background: transparent;
    color: ${t.fg};
    font: ${sys.type.bodyMd};
    letter-spacing: ${sys.tracking.bodyMd};
    cursor: pointer;
    transform: rotate(var(--a)) translateY(-108px) rotate(calc(-1 * var(--a)));
    transition: color ${sys.duration.short4} ${sys.easing.standard};
  }
  .tick.inner {
    transform: rotate(var(--a)) translateY(-68px) rotate(calc(-1 * var(--a)));
    font: ${sys.type.bodySm};
    letter-spacing: ${sys.tracking.bodySm};
  }
  .tick.active { color: ${t.onAccent}; }
  .tick:focus-visible { outline: ${sys.focus.ring}; outline-offset: 2px; }
  .ticks {
    position: absolute;
    inset: 0;
    opacity: 0;
    pointer-events: none;
    scale: 0.85;
    transition: opacity ${sys.duration.short4} ${sys.easing.standard},
                scale ${sys.duration.medium2} ${sys.easing.emphasizedDecelerate};
  }
  .ticks.on {
    opacity: 1;
    pointer-events: auto;
    scale: 1;
  }
  .pane { display: grid; }
  .grids { display: grid; min-inline-size: 100%; }
  .grids > .grid { grid-area: 1 / 1; }
  .pane > * { grid-area: 1 / 1; }
  .pane .dial, .pane .grids {
    transition: opacity ${sys.duration.medium2} ${sys.easing.emphasizedDecelerate},
                scale ${sys.duration.medium2} ${sys.easing.emphasizedDecelerate};
  }
  .pane .dial.off, .pane .grids.off {
    opacity: 0;
    scale: 0.96;
    pointer-events: none;
  }
  .switch {
    display: flex;
    justify-content: flex-end;
    padding-block-start: ${sys.space(1)};
  }

  .disabled { opacity: ${sys.state.disabledContent}; pointer-events: none; }
`;

define('ui-time-picker', {
  formAssociated: true,
  props: {
    label: '', value: '', variant: 'filled', view: 'clock', hourCycle: '12', minuteStep: 5,
    locale: '', disabled: false, required: false, name: '', placeholder: '',
  },
  styles: [base, styles],
  setup(p, host) {
    const { label, value, variant, view, hourCycle, minuteStep, locale, disabled, required, name, placeholder } = p;
    formBind(host, { name, value, disabled });

    const open = signal(false);
    const focused = signal(false);
    const text = signal('');
    const selecting = signal('hour'); // 'hour' | 'minute'
    const draft = signal(value() || nowValue());
    const face = signal(view() === 'input' ? 'input' : 'clock');
    let fieldEl = null;
    let stopAuto = null;

    effect(() => {
      const v = value();
      text.set(formatTime(v, locale(), hourCycle()) || v);
    });

    const floating = computed(() => focused() || text() !== '' || placeholder() !== '' || open());
    const cls = computed(() =>
      ['root', variant(), floating() && 'floating', open() && 'open',
       disabled() && 'disabled', label() && 'has-label'].filter(Boolean).join(' '));

    const parsed = computed(() => parseTime(draft()) || { h: 0, m: 0 });
    const is12 = computed(() => String(hourCycle()) !== '24');
    const hour12 = computed(() => {
      const h = parsed().h % 12;
      return h === 0 ? 12 : h;
    });
    const period = computed(() => (parsed().h < 12 ? 'AM' : 'PM'));
    const hourChoices = computed(() =>
      is12() ? [12, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11] : Array.from({ length: 24 }, (_, i) => i));
    const minuteChoices = computed(() => {
      const step = Math.max(1, minuteStep() || 5);
      const list = [];
      for (let m = 0; m < 60; m += step) list.push(m);
      const cur = parsed().m;
      if (!list.includes(cur)) list.push(cur);
      return list.sort((a, b) => a - b);
    });
    const clockHours = [12, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11];
    const innerHours = computed(() => (is12() ? [] : [0, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23]));
    const clockMinutes = [0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55];
    const isClock = computed(() => face() !== 'input');
    const handDeg = computed(() => {
      if (selecting() === 'minute') return parsed().m * 6;
      return ((is12() ? hour12() : parsed().h) % 12) * 30;
    });
    const hourActive = (h) => (is12() ? h === hour12() : h === parsed().h);

    const commit = (next, { close = true } = {}) => {
      if (next !== value()) {
        value.set(next);
        host.emit('change', { value: next });
      }
      text.set(formatTime(next, locale(), hourCycle()) || next);
      if (close) closePanel();
    };

    const setDraft = (h, m) => draft.set(toValue(h, m));

    const pickHour = (display) => {
      const { m } = parsed();
      let h;
      if (is12()) {
        const pm = period() === 'PM';
        h = display === 12 ? (pm ? 12 : 0) : display + (pm ? 12 : 0);
      } else {
        h = display;
      }
      setDraft(h, m);
      selecting.set('minute');
    };
    const pickMinute = (m) => commit(toValue(parsed().h, m));
    const setPeriod = (next) => {
      let { h, m } = parsed();
      if (next === 'AM' && h >= 12) h -= 12;
      if (next === 'PM' && h < 12) h += 12;
      setDraft(h, m);
    };

    const openPanel = () => {
      if (disabled() || open()) return;
      draft.set(parseTime(value()) ? value() : nowValue());
      selecting.set('hour');
      face.set(view() === 'input' ? 'input' : 'clock');
      open.set(true);
    };
    const closePanel = () => open.set(false);
    const toggle = (e) => {
      e?.stopPropagation();
      open() ? closePanel() : openPanel();
    };

    const onInput = (e) => {
      text.set(e.target.value);
      host.emit('input', { value: e.target.value });
    };
    const onBlur = () => {
      focused.set(false);
      let parsedVal = parseTime(text());
      if (!parsedVal) {
        const d = new Date(`1970-01-01T${text()}`);
        if (!Number.isNaN(d.getTime())) parsedVal = { h: d.getHours(), m: d.getMinutes() };
      }
      if (!parsedVal && text().trim() === '') {
        if (value()) commit('', { close: false });
        return;
      }
      if (parsedVal) commit(toValue(parsedVal.h, parsedVal.m), { close: false });
      else text.set(formatTime(value(), locale(), hourCycle()) || value());
    };
    const onKeydown = (e) => {
      if (e.key === 'ArrowDown' && e.altKey) { e.preventDefault(); openPanel(); }
      else if (e.key === 'F4') { e.preventDefault(); toggle(); }
      else if (e.key === 'Enter') {
        e.preventDefault();
        const p0 = parseTime(text());
        if (p0) commit(toValue(p0.h, p0.m));
      } else if (e.key === 'Escape' && open()) {
        e.preventDefault();
        closePanel();
      }
    };

    effect(() => {
      if (!open()) return;
      const onDoc = (e) => {
        if (e.composedPath().includes(host)) return;
        commit(draft());
      };
      const onEsc = (e) => { if (e.key === 'Escape') closePanel(); };
      document.addEventListener('pointerdown', onDoc);
      document.addEventListener('keydown', onEsc, true);
      return () => {
        document.removeEventListener('pointerdown', onDoc);
        document.removeEventListener('keydown', onEsc, true);
      };
    });
    // One tab stop per clock face / digital grid; arrows rove. Do not steal
    // focus from the field — only retarget the tab stop onto the current value.
    effect(() => {
      if (!open()) return;
      isClock();
      selecting();
      let roving = null;
      let cancelled = false;
      queueMicrotask(() => {
        if (cancelled || !open.peek()) return;
        const root = host.shadowRoot;
        if (!root) return;
        if (isClock.peek()) {
          const ticks = root.querySelector('.ticks.on');
          if (!ticks) return;
          const sel = selecting.peek() === 'hour' ? '[data-hour]' : '[data-minute]';
          roving = rovingTabindex(ticks, { selector: sel, orientation: 'both' });
          const current = ticks.querySelector('.tick.active') || ticks.querySelector(sel);
          if (current) roving.activate(current);
        } else {
          const grid = root.querySelector('.grid:not(.off)');
          if (!grid) return;
          roving = rovingTabindex(grid, { selector: '[role=option]', orientation: 'both' });
          const current = grid.querySelector('[aria-selected="true"]') || grid.querySelector('[role=option]');
          if (current) roving.activate(current);
        }
      });
      return () => {
        cancelled = true;
        roving?.destroy();
      };
    });
    effect(() => {
      if (!open() && stopAuto) { stopAuto(); stopAuto = null; }
    });
    onCleanup(() => stopAuto?.());

    const panelRef = (el) => {
      stopAuto?.();
      stopAuto = autoUpdate(el, fieldEl, { placement: 'bottom-start', offset: 4 });
    };

    const panelView = () => html`
      <div class="panel" part="panel" role="dialog" aria-label=${() => label() || 'Choose time'}
           ref=${panelRef}>
        <div class="face">
          <button type="button" class=${() => ({ digit: true, active: selecting() === 'hour' })}
                  aria-label="Hours" @click=${() => selecting.set('hour')}>
            ${() => pad(is12() ? hour12() : parsed().h)}
          </button>
          <span class="colon" aria-hidden="true">:</span>
          <button type="button" class=${() => ({ digit: true, active: selecting() === 'minute' })}
                  aria-label="Minutes" @click=${() => selecting.set('minute')}>
            ${() => pad(parsed().m)}
          </button>
          ${() => (is12()
            ? html`<div class="period">
                <button type="button" class=${() => ({ active: period() === 'AM' })}
                        @click=${() => setPeriod('AM')}>AM</button>
                <button type="button" class=${() => ({ active: period() === 'PM' })}
                        @click=${() => setPeriod('PM')}>PM</button>
              </div>`
            : null)}
        </div>
        <div class="pane">
          <div class=${() => `dial${isClock() ? '' : ' off'}`} part="dial">
              <div class="hand" aria-hidden="true"
                   style=${() => ({ transform: `rotate(${handDeg()}deg)` })}></div>
              <div class="hub" aria-hidden="true"></div>
              <div class=${() => `ticks${selecting() === 'hour' ? ' on' : ''}`}>
                ${clockHours.map((h) => html`
                    <button type="button" class=${() => ({ tick: true, active: hourActive(h) })}
                            style=${{ '--a': ((h % 12) * 30) + 'deg' }}
                            data-hour=${h}
                            aria-label=${h + ' hours'}
                            tabindex="-1"
                            @click=${() => pickHour(h)}>${h}</button>`)}
                ${() => innerHours().map((h) => html`
                    <button type="button" class=${() => ({ tick: true, inner: true, active: hourActive(h) })}
                            style=${{ '--a': ((h % 12) * 30) + 'deg' }}
                            data-hour=${h}
                            aria-label=${h + ' hours'}
                            tabindex="-1"
                            @click=${() => pickHour(h)}>${pad(h)}</button>`)}
              </div>
              <div class=${() => `ticks${selecting() === 'minute' ? ' on' : ''}`}>
                ${clockMinutes.map((m) => html`
                    <button type="button" class=${() => ({ tick: true, active: m === parsed().m })}
                            style=${{ '--a': (m * 6) + 'deg' }}
                            data-minute=${m}
                            aria-label=${m + ' minutes'}
                            tabindex="-1"
                            @click=${() => pickMinute(m)}>${pad(m)}</button>`)}
              </div>
          </div>
          <div class=${() => `grids${isClock() ? ' off' : ''}`}>
        <div class=${() => `grid${selecting() === 'hour' ? '' : ' off'}`}
             role="listbox" aria-label="Hours">
          ${each(
            () => hourChoices(),
            (h) => html`
              <button type="button" role="option" class=${() => ({
                choice: true,
                active: is12() ? h() === hour12() : h() === parsed().h,
              })}
              data-hour=${() => h()}
              tabindex="-1"
              aria-selected=${() => String(is12() ? h() === hour12() : h() === parsed().h)}
              @click=${() => pickHour(h())}>
                <span class="layer" aria-hidden="true"></span>
                ${() => pad(h())}
              </button>`,
            (h) => 'h' + h,
          )}
        </div>
        <div class=${() => `grid${selecting() === 'minute' ? '' : ' off'}`}
             role="listbox" aria-label="Minutes">
          ${each(
            () => minuteChoices(),
            (m) => html`
              <button type="button" role="option" class=${() => ({
                choice: true,
                active: m() === parsed().m,
              })}
              data-minute=${() => m()}
              tabindex="-1"
              aria-selected=${() => String(m() === parsed().m)}
              @click=${() => pickMinute(m())}>
                <span class="layer" aria-hidden="true"></span>
                ${() => pad(m())}
              </button>`,
            (m) => 'm' + m,
          )}
        </div>
          </div>
        </div>
        <div class="switch">
          <ui-icon-button
            icon=${() => (isClock() ? 'keyboard' : 'clock')}
            label=${() => (isClock() ? 'Switch to text input' : 'Switch to clock')}
            @click=${(e) => { e.stopPropagation(); face.set(isClock() ? 'input' : 'clock'); }}></ui-icon-button>
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
                 aria-label=${() => (label() ? null : (placeholder() || 'Time'))}
                 aria-haspopup="dialog" aria-expanded=${() => String(open())}
                 autocomplete="off"
                 @input=${onInput} @focus=${() => focused.set(true)} @blur=${onBlur}
                 @keydown=${onKeydown}>
          <ui-icon-button icon="clock" label=${() => (open() ? 'Close time picker' : 'Open time picker')}
                          @click=${toggle}></ui-icon-button>
        </div>
        ${presence(open, panelView, {
          enter: fx.scaleIn,
          exit: fx.scaleOut,
          enterDuration: 'short4',
          exitDuration: 'short4',
          onEntered: () => host.emit('open'),
          onExited: () => host.emit('close'),
        })}
      </div>`;
  },
});

export const tag = 'ui-time-picker';
export const themeVars = t;
