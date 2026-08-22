// <ui-search> — Material search bar.
//
//   <ui-search label="Search mail" value=${q}
//              @input=${(e) => q(e.detail.value)}
//              @submit=${(e) => run(e.detail.value)}></ui-search>
//
// A pill-shaped field with a leading search icon, a trailing clear control
// while there is text, and an optional trailing slot (avatar, voice, …).
// Enter emits `submit`. The field chrome is the focus indicator — the inner
// input has no extra outline.
//
// `presentation="view"` opens a search view: a back control and a suggestions
// list (the default slot) while open. The list overlays the page — it does
// not grow the layout. Typing a query opens the view; clearing the field
// (keyboard or the clear button) closes it back to the pill bar. Focus on an
// empty view still shows recents, but a clear while focused stays on the bar.
// The open surface is one extra-large rounded container (bar + overlay list)
// with a divider between the field and the list.
//
// @prop  {string}  label='Search'   — accessible name (and the floating placeholder)
// @prop  {string}  value=''
// @prop  {string}  placeholder=''   — shown in the field; falls back to `label`
// @prop  {string}  presentation='bar' — bar | view
// @prop  {boolean} open=false       — view / suggestions visibility
// @prop  {boolean} disabled=false
// @prop  {string}  name=''          — form participation
// @event input  — every keystroke; detail: { value }
// @event change — committed (blur/Enter); detail: { value }
// @event submit — Enter pressed; detail: { value }
// @event clear  — the clear affordance was used
// @event open   — suggestions visible (after the enter animation); does not bubble
// @event close  — suggestions removed (after the exit animation); does not bubble
// @slot  leading  — replaces the search icon
// @slot  trailing — after the clear button (avatar, extra actions)
// @slot  (default) — suggestion rows (ui-list-item, …). The panel is a list
//        so those rows keep role="listitem"; it is not a listbox.
// @part  bar, input, panel
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, computed, signal } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { formBind } from '../util/form.js';
import { presence } from '../motion/presence.js';
import { fx } from '../motion/animate.js';
import { popupEvent } from '../util/popup.js';
import './ui-icon.js';
import './ui-icon-button.js';

const t = vars('ui-search', {
  bg: sys.color.surfaceContainerHigh,
  bgActive: sys.color.surfaceContainerHigh,
  fg: sys.color.onSurface,
  placeholderFg: sys.color.onSurfaceVariant,
  radius: sys.radius.xl,
  font: sys.type.bodyLg,
  height: '56px',
  panelBg: sys.color.surfaceContainerHigh,
  panelRadius: sys.radius.xl,
  divider: sys.color.outlineVariant,
});

const styles = css`
  :host { display: block; position: relative; inline-size: min(100%, 720px); }
  .shell {
    position: relative;
    overflow: hidden;
    border-radius: ${t.radius};
    background: ${t.bg};
    box-shadow: none;
    transition: box-shadow ${sys.duration.short4} ${sys.easing.standard},
                background-color ${sys.duration.short2} ${sys.easing.standard};
  }
  .shell:focus-within:not(.open) {
    background: ${t.bgActive};
  }
  .shell.open {
    overflow: visible;
    z-index: ${sys.z.popup};
    background: ${t.panelBg};
    border-start-start-radius: ${t.panelRadius};
    border-start-end-radius: ${t.panelRadius};
    border-end-start-radius: 0;
    border-end-end-radius: 0;
    box-shadow: ${sys.elevation[2]};
  }
  .bar {
    position: relative;
    isolation: isolate;
    display: flex;
    align-items: center;
    gap: ${sys.space(2)};
    min-block-size: calc(${t.height} + var(--ui-density, 0) * 4px);
    padding-inline: ${sys.space(2)};
    border-radius: inherit;
    background: transparent;
    color: ${t.fg};
    font: ${t.font};
    letter-spacing: ${sys.tracking.bodyLg};
    cursor: text;
    --ui-icon-size: 1.5rem;
  }
  .bar::before {
    content: '';
    position: absolute; inset: 0;
    border-radius: inherit;
    background: ${sys.color.onSurface};
    opacity: 0;
    pointer-events: none;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  .bar:hover:not(:focus-within)::before { opacity: ${sys.state.hover}; }
  .shell.open .bar::before { opacity: 0; }
  .lead { color: ${t.placeholderFg}; display: grid; place-items: center; }
  .leads {
    position: relative;
    display: grid;
    place-items: center;
    /* Cap at 48px for the 56px bar; shrink with --ui-search-height so a
       compact field in an app bar can match the 40px icon buttons. */
    inline-size: min(48px, ${t.height});
    block-size: min(48px, ${t.height});
  }
  .leads > * { grid-area: 1 / 1; }
  .lead-slot {
    display: grid;
    place-items: center;
    transform-origin: center center;
    transition: opacity ${sys.duration.short3} ${sys.easing.standard},
                transform ${sys.duration.short4} ${sys.easing.emphasized};
  }
  .lead-search {
    opacity: 1;
    transform: rotate(0deg) scale(1);
  }
  .lead-search.off {
    opacity: 0;
    transform: rotate(-90deg) scale(0.6);
    pointer-events: none;
  }
  .lead-back {
    opacity: 1;
    transform: rotate(0deg) scale(1);
  }
  .lead-back.off {
    opacity: 0;
    transform: rotate(90deg) scale(0.6);
    pointer-events: none;
  }
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
  input::placeholder { color: ${t.placeholderFg}; }
  .panel {
    position: absolute;
    inset-inline: 0;
    inset-block-start: 100%;
    z-index: 1;
    padding-block: ${sys.space(2)};
    background: ${t.panelBg};
    overflow-y: auto;
    overflow-x: hidden;
    box-sizing: border-box;
    max-block-size: min(70vh, 360px);
    border-block-start: 1px solid ${t.divider};
    border-end-start-radius: ${t.panelRadius};
    border-end-end-radius: ${t.panelRadius};
    box-shadow: ${sys.elevation[2]};
  }
  .disabled { opacity: ${sys.state.disabledContent}; pointer-events: none; }
`;

define('ui-search', {
  formAssociated: true,
  props: {
    label: 'Search', value: '', placeholder: '', presentation: 'bar',
    open: false, disabled: false, name: '',
  },
  styles: [base, styles],
  setup({ label, value, placeholder, presentation, open, disabled, name }, host) {
    formBind(host, { name, value, disabled });
    const hasLeading = signal(false);
    let input;
    let skipFocusOpen = false;

    const ph = computed(() => placeholder() || label() || 'Search');
    const isView = computed(() => presentation() === 'view');
    const slotted = () => [...host.children].some((c) => !c.slot);
    const showPanel = computed(() => open() && (isView() || slotted()));
    const hasClear = computed(() => (value() ?? '') !== '' && !disabled());
    const rootCls = computed(() =>
      ['shell', isView() ? 'view' : 'bar-mode', open() && 'open', disabled() && 'disabled']
        .filter(Boolean).join(' '));
    const hasQuery = () => (value() ?? '') !== '';

    const openView = () => {
      if (disabled() || open()) return;
      if (isView() || slotted()) open.set(true);
    };
    const closeView = () => { if (open()) open.set(false); };

    const onInput = (e) => {
      // Native `input` is composed. Stop it so a host listener is not handed
      // UIEvent.detail (0). The CustomEvent we emit carries `{ value }`.
      // Do not bind the host with @input.capture — that runs before this stop
      // and will see the native event.
      e.stopPropagation();
      value.set(e.target.value);
      host.emit('input', { value: value() });
      if (hasQuery()) openView();
      else closeView();
    };
    const commit = (e) => {
      e?.stopPropagation();
      host.emit('change', { value: value() });
    };
    const submit = (e) => {
      if (e.key !== 'Enter') return;
      host.emit('submit', { value: value() });
    };
    const clear = (e) => {
      e?.stopPropagation();
      value.set('');
      host.emit('clear');
      host.emit('input', { value: '' });
      closeView();
      skipFocusOpen = true;
      input?.focus();
      queueMicrotask(() => { skipFocusOpen = false; });
    };
    const onFocus = () => {
      if (skipFocusOpen) return;
      openView();
    };
    const onFocusOut = (e) => {
      const next = e.relatedTarget;
      if (next && (host.contains(next) || host.shadowRoot?.contains(next))) return;
      if (!hasQuery()) closeView();
    };
    const onKeydown = (e) => {
      submit(e);
      if (e.key === 'Escape' && open()) {
        e.preventDefault();
        closeView();
      }
    };

    const panelView = () => html`
      <div class="panel" part="panel" role="list" id="suggestions"
           aria-label=${() => label() || 'Suggestions'}>
        <slot></slot>
      </div>`;

    return html`
      <div class=${rootCls} @focusout=${onFocusOut}>
        <div class=${() => `bar${disabled() ? ' disabled' : ''}`} part="bar"
             @click=${() => input?.focus()}>
          ${() => (isView()
            ? html`<span class="leads">
                <span class=${() => `lead-slot lead-search${open() ? ' off' : ''}`}>
                  <slot name="leading" ref=${(el) => el.addEventListener('slotchange', () => hasLeading.set(el.assignedElements().length > 0))}></slot>
                  ${() => (hasLeading() ? null : html`<ui-icon name="search"></ui-icon>`)}
                </span>
                <span class=${() => `lead-slot lead-back${open() ? '' : ' off'}`}>
                  <ui-icon-button icon="arrow-back" label="Back"
                    @click=${(e) => { e.stopPropagation(); closeView(); }}></ui-icon-button>
                </span>
              </span>`
            : html`<span class="lead">
                <slot name="leading" ref=${(el) => el.addEventListener('slotchange', () => hasLeading.set(el.assignedElements().length > 0))}></slot>
                ${() => (hasLeading() ? null : html`<ui-icon name="search"></ui-icon>`)}
              </span>`)}
          <input part="input" ref=${(el) => (input = el)}
                 .value=${() => value() ?? ''}
                 placeholder=${ph}
                 aria-label=${() => label() || 'Search'}
                 aria-expanded=${() => String(open())}
                 aria-controls=${() => (showPanel() ? 'suggestions' : null)}
                 aria-autocomplete="list"
                 aria-haspopup=${() => (isView() || slotted() ? 'true' : null)}
                 ?disabled=${disabled}
                 @input=${onInput} @change=${commit} @keydown=${onKeydown}
                 @focus=${onFocus}>
          ${presence(hasClear, () => html`<ui-icon-button icon="close" label="Clear" @click=${clear}></ui-icon-button>`, {
            enter: fx.scaleIn,
            exit: fx.scaleOut,
            enterDuration: 'short2',
            exitDuration: 'short2',
            enterEasing: 'emphasizedDecelerate',
            exitEasing: 'emphasizedAccelerate',
          })}
          <slot name="trailing"></slot>
        </div>
        ${presence(showPanel, panelView, {
          enter: fx.expandDown,
          exit: fx.collapseUp,
          enterDuration: 'medium2',
          enterEasing: 'emphasizedDecelerate',
          exitDuration: 'short4',
          exitEasing: 'emphasizedAccelerate',
          onEntered: () => host.emit('open', null, popupEvent),
          onExited: () => host.emit('close', null, popupEvent),
        })}
      </div>`;
  },
});

export const tag = 'ui-search';
export const themeVars = t;
