// <ui-select> — Material select: a text-field-style field button that opens a
// listbox of slotted <ui-option>s.
//
//   <ui-select label="Flavor" value=${flavor} @change=${(e) => flavor(e.detail.value)}>
//     <ui-option value="vanilla">Vanilla</ui-option>
//     <ui-option value="mint">Mint</ui-option>
//   </ui-select>
//
// Keyboard (APG select-only combobox): Enter/Space/ArrowDown/ArrowUp open;
// arrows move the active option, Enter/Space selects it, Escape closes the
// panel only — an enclosing dialog keeps its own Escape for a second press,
// typing jumps to the next option starting with that letter. The panel closes
// on outside pointerdown and returns focus to the field.
//
// Past a handful of options the panel gets a filter field, because scrolling
// is not a way to find one entry among several hundred. It takes focus when
// the panel opens, the arrows and Enter work from it, and the options it
// hides are hidden from the keyboard too. The query is dropped when the panel
// closes.
//
// @prop  {string}  label=''
// @prop  {string}  value=''         — the selected option's value
// @prop  {string}  variant='filled' — filled | outlined
// @prop  {boolean} disabled=false
// @prop  {boolean} required=false
// @prop  {string}  name=''          — form participation
// @prop  {string}  placeholder=''   — shown while nothing is selected
// @prop  {string}  search='auto'    — auto | always | never; 'auto' shows the
//                                     filter once there are searchThreshold
//                                     options or more
// @prop  {number}  searchThreshold=8
// @prop  {string}  searchPlaceholder='Search'
// @event change — an option was chosen; detail: { value }
// @event open   — panel enter animation finished
// @event close  — panel exit animation finished
// @slot  (default) — <ui-option> children (projected into the panel)
// @part  control — the field button (role="combobox")
// @part  label, panel
// @vars  see `t` below (`themeVars.names`)
//
// Note: the combobox's aria-activedescendant references option ids in the
// host's light DOM; the options also carry aria-selected for AT that walks
// the composed tree.

import { define, html, css, vars, computed, signal, effect, onCleanup } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { formBind } from '../util/form.js';
import { presence } from '../motion/presence.js';
import { fx } from '../motion/animate.js';
import { autoUpdate } from '../util/position.js';
import { escapeLayer } from '../util/keys.js';
import './ui-icon.js';
import './ui-option.js';

const t = vars('ui-select', {
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
  .field-wrap { position: relative; display: block; }
  .field {
    position: relative;
    display: flex;
    align-items: center;
    gap: ${sys.space(2)};
    inline-size: 100%;
    min-block-size: calc(${t.height} + var(--ui-density, 0) * 4px);
    padding-inline: ${sys.space(4)};
    margin: 0;
    border: none;
    outline: none;
    appearance: none;
    border-radius: ${t.radius};
    background: transparent;
    font: ${t.font};
    letter-spacing: ${sys.tracking.bodyLg};
    text-align: start;
    color: ${t.fg};
    cursor: pointer;
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

  /* Outlined: a fieldset (sibling of the button — never inside it, or the
     UA paints a second inner border) draws the notchable outline. */
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
    padding-inline: ${sys.space(1)} calc(${sys.space(1)} + 3.5px);
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
  .open .label, :focus-within .label { color: ${t.accent}; }

  .value {
    flex: 1;
    min-inline-size: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-block-size: 1lh;
  }
  .filled.has-label .value { padding-block-start: 18px; }
  .value.empty { color: ${t.labelFg}; }

  .arrow {
    color: ${t.labelFg};
    transition: rotate ${sys.duration.short3} ${sys.easing.standard};
  }
  .open .arrow { rotate: 180deg; }

  .panel {
    position: fixed;
    z-index: ${sys.z.modal};
    min-inline-size: 112px;
    max-block-size: 40vh;
    /* The filter field stays put while the options scroll under it, so the
       panel is a column and the list is what overflows. Without this the box
       you are typing into scrolls off the top of the list it is filtering. */
    display: flex;
    flex-direction: column;
    overflow: hidden;
    background: ${t.panelBg};
    border-radius: ${sys.radius.xs};
    box-shadow: ${sys.elevation[2]};
    transform-origin: top center;
  }
  .options { overflow: auto; padding-block: ${sys.space(2)}; }

  .search {
    flex: none;
    display: flex;
    align-items: center;
    gap: ${sys.space(2)};
    padding: ${sys.space(2)} ${sys.space(3)};
    border-block-end: 1px solid ${t.outlineColor};
    color: ${t.labelFg};
  }
  .search input {
    /* A flex item will not shrink below its content, and a text input's
       intrinsic width is wider than a narrow select. */
    min-inline-size: 0;
    flex: 1 1 auto;
    background: transparent;
    border: 0;
    outline: none;
    color: ${t.fg};
    font: ${t.font};
    padding: 0;
  }
  .search input::placeholder { color: ${t.labelFg}; }
  .empty {
    padding: ${sys.space(3)};
    color: ${t.labelFg};
    font: ${t.font};
  }

  .disabled { opacity: ${sys.state.disabledContent}; pointer-events: none; }
`;

define('ui-select', {
  formAssociated: true,
  props: {
    label: '', value: '', variant: 'filled', disabled: false, required: false,
    name: '', placeholder: '',
    // 'auto' shows the filter once there are more options than a person will
    // read down; 'always' and 'never' say so outright.
    search: 'auto', searchThreshold: 8, searchPlaceholder: 'Search',
  },
  styles: [base, styles],
  setup(p, host) {
    const { label, value, variant, disabled, required, name, placeholder,
      search, searchThreshold, searchPlaceholder } = p;
    formBind(host, { name, value, disabled });

    const open = signal(false);
    const activeIndex = signal(-1);
    let fieldEl = null;
    let searchEl = null;
    let stopAuto = null;

    // ---- option tracking: light-DOM children, live through a version signal
    const optsVersion = signal(0);
    const MO = typeof MutationObserver === 'function'
      ? MutationObserver
      : (typeof window !== 'undefined' ? window.MutationObserver : null);
    if (MO) {
      const mo = new MO(() => optsVersion.update((n) => n + 1));
      mo.observe(host, { childList: true, subtree: true, characterData: true });
      onCleanup(() => mo.disconnect());
    }
    // Children may not be parsed yet when we upgrade mid-stream.
    queueMicrotask(() => optsVersion.update((n) => n + 1));

    const optionEls = computed(() => {
      optsVersion();
      return [...host.querySelectorAll('ui-option')];
    });
    const optValue = (o) => {
      const v = o.value;
      return (v === undefined || v === null || v === '') ? (o.getAttribute('value') || '') : String(v);
    };
    const isDisabled = (o) => !!o.disabled || o.hasAttribute('disabled');
    const optLabel = (o) => (o.textContent || '').trim();

    // ---- filtering
    //
    // Scrolling is not a way to find one set among nine hundred, so past a
    // handful of options the panel gets a filter. Matching folds case and
    // accents, the same way the rest of a search box is expected to.
    const query = signal('');
    const fold = (v) => String(v).toLowerCase().normalize('NFKD').replace(/[\u0300-\u036f]/g, '');
    const searching = computed(() => {
      const mode = String(search() || 'auto');
      if (mode === 'never' || mode === 'false') return false;
      if (mode === 'always' || mode === 'true') return true;
      return optionEls().length >= Number(searchThreshold() || 8);
    });
    // Every list the keyboard walks and the panel shows is this one, not
    // optionEls: an arrow key that steps onto a hidden option looks broken.
    const shownEls = computed(() => {
      const opts = optionEls();
      const q = fold(query()).trim();
      if (!q || !searching()) return opts;
      return opts.filter((o) => fold(optLabel(o)).includes(q));
    });
    effect(() => {
      const shown = new Set(shownEls());
      for (const o of optionEls()) o.toggleAttribute('data-ui-filtered', !shown.has(o));
    });
    // A narrowed list has a new first row, and the active option has to be on
    // it — otherwise Enter commits something no longer on screen.
    effect(() => {
      const shown = shownEls();
      if (!open()) return;
      const cur = shown[activeIndex()];
      if (!cur || isDisabled(cur)) activeIndex.set(shown.findIndex((o) => !isDisabled(o)));
    });

    const display = computed(() => {
      const match = optionEls().find((o) => optValue(o) === value());
      return match ? optLabel(match) : value();
    });
    const floating = computed(() => display() !== '' || placeholder() !== '' || open());
    const cls = computed(() =>
      ['root', variant(), floating() && 'floating', open() && 'open',
       disabled() && 'disabled', label() && 'has-label'].filter(Boolean).join(' '));

    // Mirror selection + keyboard-active state onto the options (attributes,
    // so options that upgrade after us still pick the state up).
    effect(() => {
      const opts = optionEls();
      const v = value();
      const isOpen = open();
      const ai = activeIndex();
      const activeEl = isOpen ? shownEls()[ai] : null;
      opts.forEach((o) => {
        o.toggleAttribute('selected', optValue(o) === v);
        o.toggleAttribute('active', o === activeEl);
      });
    });
    const activeId = computed(() => (open() ? shownEls()[activeIndex()]?.id ?? null : null));

    // ---- open/close
    const openPanel = () => {
      if (disabled() || open()) return;
      const opts = shownEls();
      let i = opts.findIndex((o) => optValue(o) === value() && !isDisabled(o));
      if (i < 0) i = opts.findIndex((o) => !isDisabled(o));
      activeIndex.set(i);
      open.set(true);
    };
    // The query does not survive the panel: reopening to a list still narrowed
    // by what was typed last time reads as a select that has lost its options.
    const closePanel = () => {
      open.set(false);
      query.set('');
    };

    const commit = (o) => {
      if (!o || isDisabled(o)) return;
      const v = optValue(o);
      value.set(v);
      host.emit('change', { value: v });
      closePanel();
      fieldEl?.focus();
    };

    // Escape belongs to the panel while it is open, not to whatever encloses
    // it. A dialog listens for the key in the capture phase at the document,
    // so without claiming it a step earlier one press closes the panel and the
    // dialog together.
    effect(() => {
      if (!open()) return;
      return escapeLayer(() => {
        closePanel();
        fieldEl?.focus();
      });
    });

    // Outside pointerdown closes (scrim-less popup).
    effect(() => {
      if (!open()) return;
      const onDoc = (e) => {
        if (e.composedPath().includes(host)) return;
        closePanel();
      };
      document.addEventListener('pointerdown', onDoc);
      return () => document.removeEventListener('pointerdown', onDoc);
    });
    effect(() => {
      if (!open() && stopAuto) { stopAuto(); stopAuto = null; }
    });
    onCleanup(() => stopAuto?.());

    // Option activation: option clicks bubble up from the light DOM.
    host.addEventListener('click', (e) => {
      const o = e.target instanceof Element && e.target.closest('ui-option');
      if (o && open()) commit(o);
    });

    // ---- keyboard
    const move = (delta) => {
      const opts = shownEls();
      if (!opts.length) return;
      let i = activeIndex();
      for (let n = 0; n < opts.length; n++) {
        i = i < 0 ? (delta > 0 ? 0 : opts.length - 1) : (i + delta + opts.length) % opts.length;
        if (!isDisabled(opts[i])) { activeIndex.set(i); return; }
      }
    };
    const typeahead = (ch) => {
      const opts = shownEls();
      const lower = ch.toLowerCase();
      const start = activeIndex() + 1;
      for (let n = 0; n < opts.length; n++) {
        const i = (start + n) % opts.length;
        if (!isDisabled(opts[i]) && optLabel(opts[i]).toLowerCase().startsWith(lower)) {
          activeIndex.set(i);
          return;
        }
      }
    };
    const onKeydown = (e) => {
      if (disabled()) return;
      const opts = shownEls();
      if (!open()) {
        if (e.key === 'Enter' || e.key === ' ' || e.key === 'ArrowDown' || e.key === 'ArrowUp') {
          e.preventDefault();
          openPanel();
        } else if (e.key.length === 1 && e.key !== ' ') {
          typeahead(e.key);
          const o = opts[activeIndex()];
          if (o) { value.set(optValue(o)); host.emit('change', { value: value() }); }
        }
        return;
      }
      switch (e.key) {
        case 'ArrowDown': e.preventDefault(); move(1); break;
        case 'ArrowUp': e.preventDefault(); move(-1); break;
        case 'Home': e.preventDefault(); activeIndex.set(opts.findIndex((o) => !isDisabled(o))); break;
        case 'End': {
          e.preventDefault();
          for (let i = opts.length - 1; i >= 0; i--) if (!isDisabled(opts[i])) { activeIndex.set(i); break; }
          break;
        }
        case 'Enter':
        case ' ': e.preventDefault(); commit(opts[activeIndex()]); break;
        case 'Tab': closePanel(); break;
        default:
          if (e.key.length === 1 && e.key !== ' ') typeahead(e.key);
      }
    };

    const panelRef = (el) => {
      stopAuto?.();
      stopAuto = autoUpdate(el, fieldEl, { placement: 'bottom-start', matchWidth: true, offset: 4 });
    };

    // Typing filters, so the arrows and Enter have to work from the field the
    // typing goes into. Space is deliberately absent: it is a character here,
    // not a way to choose.
    const onSearchKeydown = (e) => {
      // One press, one action. The panel is rendered through presence(), which
      // gives the subtree its own delegation root on top of the shadow root's,
      // so a keydown inside it reaches this handler twice — and the second
      // call arrives after commit() has already closed the panel and cleared
      // the query, which is to say against the unfiltered list. Enter then
      // chose the first option of the full list instead of the one on screen,
      // and an arrow key moved two rows at a time. Stopping propagation
      // settles the arrows; the open() guard is what makes it not matter how
      // many times this runs.
      if (!open()) return;
      switch (e.key) {
        case 'ArrowDown': e.preventDefault(); e.stopPropagation(); move(1); break;
        case 'ArrowUp': e.preventDefault(); e.stopPropagation(); move(-1); break;
        case 'Enter': e.preventDefault(); e.stopPropagation(); commit(shownEls()[activeIndex()]); break;
        case 'Tab': closePanel(); break;
        default:
      }
    };
    // Focus follows the panel into existence; ref runs once the element is
    // there, which setup() is far too early for.
    const searchRef = (el) => {
      searchEl = el;
      if (el) queueMicrotask(() => el.focus());
    };
    const searchView = () => html`
      <div class="search">
        <ui-icon name="search"></ui-icon>
        <input type="text" part="search" autocomplete="off" spellcheck="false"
               role="combobox" aria-expanded="true" aria-controls="listbox"
               aria-autocomplete="list" aria-activedescendant=${activeId}
               aria-label=${() => searchPlaceholder() || 'Search'}
               placeholder=${() => searchPlaceholder() || 'Search'}
               .value=${query}
               @input=${(e) => query.set(e.composedPath()[0].value)}
               @keydown=${onSearchKeydown}
               ref=${searchRef}></div>`;
    const panelView = () => html`
      <div class="panel" part="panel" ref=${panelRef}>
        ${() => (searching() ? searchView() : null)}
        <div class="options" role="listbox" id="listbox" aria-label=${() => label() || null}>
          <slot></slot>
        </div>
        ${() => (shownEls().length ? null : html`<div class="empty">No matches</div>`)}
      </div>`;

    return html`
      <div class=${cls}>
        <div class="field-wrap">
          ${() => (variant() === 'outlined'
            ? html`<fieldset aria-hidden="true"><legend><span>${label}${() => (required() ? ' *' : '')}</span></legend></fieldset>`
            : null)}
          <button type="button" class="field" part="control" role="combobox"
                  aria-haspopup="listbox" aria-controls="listbox"
                  aria-expanded=${() => String(open())}
                  aria-required=${() => (required() ? 'true' : null)}
                  aria-activedescendant=${activeId}
                  aria-label=${() => label() || placeholder() || null}
                  ?disabled=${disabled}
                  @click=${() => (open() ? closePanel() : openPanel())}
                  @keydown=${onKeydown}
                  ref=${(el) => (fieldEl = el)}>
            ${() => (label() ? html`<span class="label" part="label">${label}${() => (required() ? ' *' : '')}</span>` : null)}
            <span class=${() => `value${display() ? '' : ' empty'}`}>${() => display() || placeholder()}</span>
            <ui-icon class="arrow" name="arrow-drop-down"></ui-icon>
          </button>
        </div>
        ${presence(open, panelView, {
          enter: fx.scaleIn,
          exit: fx.scaleOut,
          enterDuration: 'short4',
          exitDuration: 'short2',
          enterEasing: 'emphasizedDecelerate',
          exitEasing: 'emphasizedAccelerate',
          onEntered: () => host.emit('open'),
          onExited: () => host.emit('close'),
        })}
      </div>`;
  },
});

export const tag = 'ui-select';
export const themeVars = t;
