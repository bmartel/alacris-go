// <ui-autocomplete> — a combobox with a filtering text input.
//
//   <ui-autocomplete label="Fruit" options=${['Apple', 'Banana']}
//                    @change=${(e) => pick(e.detail.value)}></ui-autocomplete>
//
// Typing filters the options case-insensitively and opens a listbox panel
// while there are matches. ArrowUp/Down move the active option, Enter commits
// the active option (or, with `freeSolo`, the raw text), Escape closes, blur
// commits an exact label match — or the raw text when `freeSolo` — and
// otherwise reverts to the last committed value.
//
// @prop  {string}  label=''
// @prop  {string}  value=''         — the committed value
// @prop  {Array}   options=[]       — strings or { value, label } objects
//                                     (JSON attribute or property)
// @prop  {string}  variant='filled' — filled | outlined
// @prop  {boolean} disabled=false
// @prop  {boolean} required=false
// @prop  {string}  name=''          — form participation
// @prop  {string}  placeholder=''
// @prop  {boolean} freeSolo=false   — allow values not present in options
// @event input  — every keystroke; detail: { value } (the raw text)
// @event change — committed; detail: { value }
// @part  field, input, label, panel
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, computed, signal, effect, onCleanup, each } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { formBind } from '../util/form.js';
import { escapeLayer } from '../util/keys.js';
import { presence } from '../motion/presence.js';
import { fx } from '../motion/animate.js';
import { autoUpdate } from '../util/position.js';
import './ui-icon.js';

const t = vars('ui-autocomplete', {
  bg: sys.color.surfaceContainerHighest,
  fg: sys.color.onSurface,
  labelFg: sys.color.onSurfaceVariant,
  accent: sys.color.primary,
  outlineColor: sys.color.outline,
  radius: sys.radius.xs,
  font: sys.type.bodyLg,
  height: '56px',
  panelBg: sys.color.surfaceContainer,
});

const styles = css`
  :host { display: block; inline-size: 240px; }
  .root { display: block; position: relative; }
  .field {
    position: relative;
    display: flex;
    align-items: center;
    gap: ${sys.space(2)};
    min-block-size: calc(${t.height} + var(--ui-density, 0) * 4px);
    padding-inline: ${sys.space(4)};
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
  .filled.focused .field::after { block-size: 2px; background: ${t.accent}; }
  .filled .field::before {
    content: '';
    position: absolute; inset: 0; pointer-events: none;
    background: ${sys.color.onSurface};
    opacity: 0;
    border-radius: inherit;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  .filled:hover:not(.focused):not(.disabled) .field::before {
    opacity: ${sys.state.hover};
  }
  .filled:hover:not(.focused):not(.disabled) .field::after {
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
  .outlined.focused fieldset { border-width: 2px; border-color: ${t.accent}; }
  .root:hover:not(.focused) fieldset { border-color: ${sys.color.onSurface}; }

  .label {
    position: absolute;
    inset-inline-start: ${sys.space(4)};
    inset-block-start: 50%;
    translate: 0 -50%;
    color: ${t.labelFg};
    pointer-events: none;
    transform-origin: 0 50%;
    transition: inset-block-start ${sys.duration.short3} ${sys.easing.standard},
                inset-inline-start ${sys.duration.short3} ${sys.easing.standard},
                translate ${sys.duration.short3} ${sys.easing.standard},
                scale ${sys.duration.short3} ${sys.easing.standard},
                color ${sys.duration.short2} ${sys.easing.standard};
  }
  .filled.floating .label { translate: 0 calc(-50% - 16px); scale: 0.75; }
  .outlined.floating .label {
    inset-block-start: 0;
    translate: 0 -50%;
    scale: 0.75;
  }
  .focused .label { color: ${t.accent}; }

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
  input::placeholder { color: ${t.labelFg}; opacity: 0; transition: opacity ${sys.duration.short2} linear; }
  .floating input::placeholder { opacity: 1; }

  .arrow {
    color: ${t.labelFg};
    transition: rotate ${sys.duration.short3} ${sys.easing.standard};
  }
  .expanded .arrow { rotate: 180deg; }

  .panel {
    position: fixed;
    z-index: ${sys.z.popup};
    min-inline-size: 112px;
    max-block-size: 40vh;
    overflow: auto;
    padding-block: ${sys.space(2)};
    background: ${t.panelBg};
    border-radius: ${sys.radius.xs};
    box-shadow: ${sys.elevation[2]};
    transform-origin: top center;
  }
  .opt {
    position: relative;
    isolation: isolate;
    display: flex;
    align-items: center;
    min-block-size: calc(48px + var(--ui-density, 0) * 4px);
    padding-inline: ${sys.space(4)};
    font: ${t.font};
    letter-spacing: ${sys.tracking.bodyLg};
    color: ${t.fg};
    cursor: pointer;
    user-select: none;
  }
  .opt .layer {
    position: absolute; inset: 0; z-index: -1;
    background: currentColor; opacity: 0;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  .opt:hover .layer { opacity: ${sys.state.hover}; }
  .opt.active .layer { opacity: ${sys.state.focus}; }
  .opt:active .layer { opacity: ${sys.state.pressed}; }
  .opt[aria-selected="true"] { background: ${sys.color.secondaryContainer}; color: ${sys.color.onSecondaryContainer}; }

  .disabled { opacity: ${sys.state.disabledContent}; pointer-events: none; }
`;

define('ui-autocomplete', {
  formAssociated: true,
  props: {
    label: '', value: '', options: [], variant: 'filled', disabled: false,
    required: false, name: '', placeholder: '', freeSolo: false,
  },
  styles: [base, styles],
  setup(p, host) {
    const { label, value, options, variant, disabled, required, name, placeholder, freeSolo } = p;
    formBind(host, { name, value, disabled });

    const query = signal('');
    const focused = signal(false);
    const open = signal(false);
    const activeIndex = signal(-1);
    let fieldEl = null;
    let stopAuto = null;

    const opts = computed(() => (options() || []).map((o) =>
      typeof o === 'string'
        ? { value: o, label: o }
        : { value: String(o.value), label: o.label != null ? String(o.label) : String(o.value) }));

    const labelFor = (v) => opts().find((o) => o.value === v)?.label ?? v;

    // The committed value drives the visible text (also picks up label text
    // arriving late through `options`).
    effect(() => query.set(labelFor(value())));

    const filtered = computed(() => {
      const q = query().trim().toLowerCase();
      return q ? opts().filter((o) => o.label.toLowerCase().includes(q)) : opts();
    });
    // New matches invalidate the keyboard-active option.
    effect(() => { filtered(); activeIndex.set(-1); });

    const activeOpt = computed(() => filtered()[activeIndex()] ?? null);
    const showPanel = computed(() => open() && !disabled() && filtered().length > 0);

    const floating = computed(() => focused() || query() !== '' || placeholder() !== '');
    const cls = computed(() =>
      ['root', variant(), floating() && 'floating', focused() && 'focused',
       showPanel() && 'expanded', disabled() && 'disabled', label() && 'has-label']
        .filter(Boolean).join(' '));

    const commit = (v) => {
      const text = labelFor(v);
      if (v !== value()) {
        value.set(v);
        host.emit('change', { value: v });
      }
      query.set(text);
      open.set(false);
    };

    let isComposing = false;
    const onCompositionStart = () => { isComposing = true; };
    const onCompositionEnd = (e) => {
      isComposing = false;
      onInput(e);
    };

    const onInput = (e) => {
      e.stopPropagation();
      if (isComposing) return;
      query.set(e.target.value);
      open.set(true);
      host.emit('input', { value: e.target.value });
    };
    const move = (delta) => {
      const list = filtered();
      if (!list.length) return;
      const i = activeIndex();
      activeIndex.set(i < 0 ? (delta > 0 ? 0 : list.length - 1) : (i + delta + list.length) % list.length);
    };
    const onKeydown = (e) => {
      switch (e.key) {
        case 'ArrowDown': e.preventDefault(); if (!open()) open.set(true); move(1); break;
        case 'ArrowUp': e.preventDefault(); if (!open()) open.set(true); move(-1); break;
        case 'Enter':
          if (showPanel() && activeOpt()) { e.preventDefault(); commit(activeOpt().value); }
          else if (freeSolo()) commit(query().trim());
          break;
      }
    };
    // The panel owns Escape while it is up, so that closing it inside a
    // dialog does not close the dialog as well.
    effect(() => {
      if (!showPanel()) return;
      return escapeLayer(() => open.set(false));
    });

    const onFocus = () => { focused.set(true); open.set(true); };
    const onBlur = () => {
      focused.set(false);
      open.set(false);
      const raw = query().trim();
      if (freeSolo()) { commit(raw); return; }
      const match = opts().find((o) => o.label.toLowerCase() === raw.toLowerCase());
      if (match) commit(match.value);
      else query.set(labelFor(value())); // revert to the committed value
    };

    effect(() => {
      if (!showPanel() && stopAuto) { stopAuto(); stopAuto = null; }
    });
    onCleanup(() => stopAuto?.());

    const panelRef = (el) => {
      stopAuto?.();
      stopAuto = autoUpdate(el, fieldEl, { placement: 'bottom-start', matchWidth: true, offset: 4 });
    };
    const panelView = () => html`
      <div class="panel" part="panel" role="listbox" id="panel"
           aria-label=${() => label() || null} ref=${panelRef}>
        ${each(
          () => filtered(),
          (o) => html`
            <div role="option"
                 class=${() => ({ opt: true, active: activeOpt()?.value === o().value })}
                 id=${() => (activeOpt()?.value === o().value ? 'ui-ac-active' : null)}
                 aria-selected=${() => String(value() === o().value)}
                 @pointerdown=${(e) => e.preventDefault()}
                 @click=${() => commit(o().value)}>
              <span class="layer" aria-hidden="true"></span>
              ${() => o().label}
            </div>`,
          (o) => o.value,
        )}
      </div>`;

    return html`
      <div class=${cls}>
        <label class="field" part="field" ref=${(el) => (fieldEl = el)}>
          ${() => (variant() === 'outlined'
            ? html`<fieldset aria-hidden="true"><legend><span>${label}${() => (required() ? ' *' : '')}</span></legend></fieldset>`
            : null)}
          ${() => (label() ? html`<span class="label" part="label">${label}${() => (required() ? ' *' : '')}</span>` : null)}
          <input part="input" role="combobox" aria-autocomplete="list"
                 aria-haspopup="listbox" aria-controls="panel"
                 aria-expanded=${() => String(showPanel())}
                 aria-activedescendant=${() => (showPanel() && activeOpt() ? 'ui-ac-active' : null)}
                 .value=${query}
                 placeholder=${() => placeholder() || null}
                 ?disabled=${disabled} ?required=${required}
                 @compositionstart=${onCompositionStart}
                 @compositionend=${onCompositionEnd}
                 @input=${onInput} @change=${(e) => e.stopPropagation()} @keydown=${onKeydown}
                 @focus=${onFocus} @blur=${onBlur}>
          <ui-icon class="arrow" name="arrow-drop-down"></ui-icon>
        </label>
        ${presence(showPanel, panelView, {
          enter: fx.scaleIn,
          exit: fx.scaleOut,
          enterDuration: 'short4',
          exitDuration: 'short2',
          enterEasing: 'emphasizedDecelerate',
          exitEasing: 'emphasizedAccelerate',
        })}
      </div>`;
  },
});

export const tag = 'ui-autocomplete';
export const themeVars = t;
