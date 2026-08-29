// <ui-text-field> — Material text field, filled and outlined, floating label.
//
// @prop  {string}  variant='filled' — filled | outlined
// @prop  {string}  label=''
// @prop  {string}  value=''
// @prop  {string}  type='text'      — any native input type; 'textarea' renders one
// @prop  {string}  placeholder=''
// @prop  {string}  helper=''        — supporting text under the field
// @prop  {string}  error=''         — error message; non-empty switches to error state
// @prop  {boolean} disabled=false
// @prop  {boolean} required=false
// @prop  {boolean} clearable=false  — trailing ✕ while there is content
// @prop  {string}  name=''          — form participation
// @prop  {number}  maxlength=0      — >0 shows a character counter and enforces it
// @prop  {number}  rows=3
// @prop  {string}  autocomplete=''   — absent unless set; 'off' keeps the
//                                      browser's saved-value list out of a
//                                      field whose contents are not a name,
//                                      an address or a password
// @prop  {string}  autocapitalize='' — 'off' for anything case-sensitive
// @prop  {string}  autocorrect=''    — 'off' to stop substitutions
// @prop  {string}  spellcheck=''     — 'false' for code and query syntax
// @prop  {string}  inputmode=''
// @prop  {string}  enterkeyhint=''           — textarea rows
// @event input  — every keystroke;   detail: { value }
// @event change — committed (blur/Enter); detail: { value }
// @event clear  — the clear affordance was used
// @slot  leading  — icon before the input
// @slot  trailing — icon after the input (replaced by ✕ while clearable+content)
// @part  field, input, label, helper
// @vars  see `t` below

import { define, html, css, vars, computed, signal } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { formBind } from '../util/form.js';
import './ui-icon-button.js';

const t = vars('ui-text-field', {
  bg: sys.color.surfaceContainerHighest,
  fg: sys.color.onSurface,
  labelFg: sys.color.onSurfaceVariant,
  accent: sys.color.primary,
  errorFg: sys.color.error,
  outlineColor: sys.color.outline,
  radius: sys.radius.xs,
  font: sys.type.bodyLg,
  height: '56px',

  // The lane reserved at the end of a number field for its stepper. It is a
  // variable because the buttons are the browser's and their width is not the
  // same on every engine — a consumer with a denser field can shorten it.
  stepperWidth: '18px',
});

const styles = css`
  :host { display: block; inline-size: 240px; }
  .root { display: block; }
  .field {
    position: relative;
    display: flex;
    align-items: center;
    gap: ${sys.space(2)};
    min-block-size: calc(${t.height} + var(--ui-density, 0) * 4px);
    padding-inline: ${sys.space(4)};
    border-radius: ${t.radius};
    font: ${t.font};
    color: ${t.fg};
    cursor: text;
    --ui-icon-size: 1.5rem;
  }
  /* Resize lives on .grow, not the textarea, so padding on the text
     does not inset the native grip from the field's corner. */
  .multiline .field {
    padding-inline: 0;
    align-items: stretch;
  }
  .grow {
    flex: 1;
    min-inline-size: 0;
    display: flex;
    align-self: stretch;
  }
  .multiline .grow {
    resize: vertical;
    overflow: hidden;
    min-block-size: calc(${t.height} + var(--ui-density, 0) * 4px);
  }
  .disabled .grow { resize: none; }
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
  .filled.error .field::after { background: ${t.errorFg}; }
  /* Opaque band so scrolled lines do not paint over the floating label. */
  .filled.multiline.floating.has-label .field::before {
    content: '';
    position: absolute;
    inset-inline: 0;
    inset-block-start: 0;
    block-size: 24px;
    background: ${t.bg};
    pointer-events: none;
    z-index: 1;
    border-start-start-radius: inherit;
    border-start-end-radius: inherit;
  }

  /* Outlined: a fieldset draws the border; its legend opens the label notch. */
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
    padding-inline: ${sys.space(1)} calc(${sys.space(1)} - 3px);
    display: inline-block;
    opacity: 0;
    visibility: visible;
  }
  .outlined.floating legend {
    max-inline-size: 100%;
    transition: max-inline-size ${sys.duration.short3} ${sys.easing.standard};
  }
  .outlined.focused fieldset { border-width: 2px; border-color: ${t.accent}; }
  .outlined.error fieldset { border-color: ${t.errorFg}; }
  .root:hover:not(.focused):not(.error) fieldset { border-color: ${sys.color.onSurface}; }
  .filled .filled-hover {
    position: absolute; inset: 0; pointer-events: none;
    background: ${sys.color.onSurface};
    opacity: 0;
    border-radius: inherit;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  .filled:hover:not(.focused):not(.error):not(.disabled) .filled-hover {
    opacity: ${sys.state.hover};
  }
  .filled:hover:not(.focused):not(.error):not(.disabled) .field::after {
    background: ${sys.color.onSurface};
  }

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
  .with-leading .label { inset-inline-start: calc(${sys.space(4)} + 1.5rem + ${sys.space(2)}); }
  /* Multi-line fields anchor the unfloated label near the top line. */
  .multiline:not(.floating) .label { inset-block-start: 24px; z-index: 2; }
  .filled.floating .label { translate: 0 calc(-50% - 16px); scale: 0.75; }
  .filled.multiline.floating .label { inset-block-start: 24px; translate: 0 -85%; scale: 0.75; }
  .outlined.floating .label {
    inset-inline-start: ${sys.space(4)};
    inset-block-start: 0;
    translate: 0 -50%;
    scale: 0.75;
  }
  .focused .label { color: ${t.accent}; }
  .error .label { color: ${t.errorFg}; }

  input, textarea {
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
  textarea {
    display: block;
    flex: 1;
    inline-size: 100%;
    min-block-size: 0;
    resize: none;
    padding: ${sys.space(4)};
  }
  .filled.has-label textarea { padding-top: ${sys.space(7)}; }
  .with-leading.multiline .field { padding-inline-start: ${sys.space(4)}; }
  .with-leading.multiline textarea { padding-inline-start: 0; }
  /* A number field's spin buttons are laid out at the inline end of the
     input's *content* box rather than at the edge of the field. That is the
     opposite of what it looks like, and it is why the obvious fix makes
     things worse: padding the input to reserve a lane for them moves the end
     of the content box inward and takes the buttons with it, so they land
     further onto the label than they started. That is what shipped in 0.2.3
     — a chevron sitting on the "n" of "Min", clipped to a single arrow —
     and it was a regression, not the original complaint.

     They cannot be moved out of the way either: margin-inline-end on the
     pseudo-element is ignored. So they are left where the browser puts them,
     at the end of the field where there is already room, and only the label
     and the legend are shortened, so a long one ellipsises before it reaches
     the buttons rather than sliding underneath them. appearance:none does
     not help here: it removes the field's own chrome and leaves the
     ::-webkit-*-spin-button alone, which is why this needs saying. */
  .numeric .label {
    max-inline-size: calc(100% - ${sys.space(4)} - ${t.stepperWidth});
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .numeric legend { max-inline-size: calc(100% - ${t.stepperWidth}); }
  .numeric input::-webkit-outer-spin-button,
  .numeric input::-webkit-inner-spin-button {
    /* Nudged off the text rather than sitting tight against it, and always
       visible: a stepper that only appears on hover is a control nobody
       knows is there. */
    margin: 0;
    margin-inline-start: ${sys.space(2)};
    opacity: 1;
  }
  /* Firefox draws no buttons at all unless asked. Asking for them makes the
     two engines agree about what a number field looks like. */
  @supports (-moz-appearance: number-input) {
    .numeric input { -moz-appearance: number-input; }
  }

  input::placeholder, textarea::placeholder { color: ${t.labelFg}; opacity: 0; transition: opacity ${sys.duration.short2} linear; }
  .floating input::placeholder, .floating textarea::placeholder { opacity: 1; }

  .below {
    display: flex;
    justify-content: space-between;
    gap: ${sys.space(4)};
    padding-inline: ${sys.space(4)};
    padding-block-start: ${sys.space(1)};
    font: ${sys.type.bodySm};
    letter-spacing: ${sys.tracking.bodySm};
    color: ${t.labelFg};
  }
  .error .below .msg { color: ${t.errorFg}; }
  .disabled { opacity: ${sys.state.disabledContent}; pointer-events: none; }
  ::slotted([slot]) { color: ${t.labelFg}; }
`;

define('ui-text-field', {
  formAssociated: true,
  props: {
    variant: 'filled', label: '', value: '', type: 'text', placeholder: '',
    helper: '', error: '', disabled: false, required: false, clearable: false,
    name: '', maxlength: 0, rows: 3,
    // A field the platform must keep its hands off has to be able to say so.
    // These are absent unless set, so nothing changes for an ordinary field.
    autocomplete: '', autocapitalize: '', autocorrect: '', spellcheck: '',
    inputmode: '', enterkeyhint: '',
  },
  styles: [base, styles],
  setup(p, host) {
    const { variant, label, value, type, placeholder, helper, error, disabled, required, clearable, name, maxlength, rows,
      autocomplete, autocapitalize, autocorrect, spellcheck, inputmode, enterkeyhint } = p;

    // An empty prop means "say nothing", so the browser keeps its default.
    // Writing the attribute as an empty string would not: autocomplete="" is
    // a value, and spellcheck="" reads as spellcheck="true".
    const attr = (sig) => () => (sig() === '' ? null : sig());
    formBind(host, { name, value, disabled });

    const focused = signal(false);
    const hasLeading = signal(false);
    let input;

    const floating = computed(() => focused() || value() !== '' || placeholder() !== '');
    const cls = computed(() =>
      ['root', variant(), floating() && 'floating', focused() && 'focused',
       error() && 'error', disabled() && 'disabled', label() && 'has-label',
       hasLeading() && 'with-leading', type() === 'textarea' && 'multiline',
       type() === 'number' && 'numeric'].filter(Boolean).join(' '));

    let isComposing = false;
    const onCompositionStart = () => { isComposing = true; };
    const onCompositionEnd = (e) => {
      isComposing = false;
      onInput(e);
    };

    const onInput = (e) => {
      if (isComposing) return;
      value.set(e.target.value);
      host.emit('input', { value: value() });
    };
    const commit = () => host.emit('change', { value: value() });
    const clear = () => {
      value.set('');
      host.emit('clear');
      host.emit('input', { value: '' });
      input?.focus();
    };

    const describe = computed(() => error() || helper() || null);
    const counter = computed(() => (maxlength() > 0 ? `${value().length} / ${maxlength()}` : null));

    const fieldBody = html`
      <slot name="leading" ref=${(el) => el.addEventListener('slotchange', () => hasLeading.set(el.assignedElements().length > 0))}></slot>
      ${() => (type() === 'textarea'
        ? html`<span class="grow"><textarea part="input" ref=${(el) => (input = el)}
              .value=${value} rows=${rows}
              placeholder=${() => placeholder() || null}
              maxlength=${() => (maxlength() > 0 ? maxlength() : null)}
              ?required=${required} ?disabled=${disabled}
              aria-invalid=${() => (error() ? 'true' : null)}
              autocomplete=${attr(autocomplete)}
              autocapitalize=${attr(autocapitalize)}
              autocorrect=${attr(autocorrect)}
              spellcheck=${attr(spellcheck)}
              inputmode=${attr(inputmode)}
              enterkeyhint=${attr(enterkeyhint)}
              @compositionstart=${onCompositionStart}
              @compositionend=${onCompositionEnd}
              @input=${onInput} @change=${commit}
              @focus=${() => focused.set(true)} @blur=${() => focused.set(false)}></textarea></span>`
        : html`<input part="input" ref=${(el) => (input = el)}
              type=${type} .value=${value}
              placeholder=${() => placeholder() || null}
              maxlength=${() => (maxlength() > 0 ? maxlength() : null)}
              ?required=${required} ?disabled=${disabled}
              aria-invalid=${() => (error() ? 'true' : null)}
              autocomplete=${attr(autocomplete)}
              autocapitalize=${attr(autocapitalize)}
              autocorrect=${attr(autocorrect)}
              spellcheck=${attr(spellcheck)}
              inputmode=${attr(inputmode)}
              enterkeyhint=${attr(enterkeyhint)}
              @compositionstart=${onCompositionStart}
              @compositionend=${onCompositionEnd}
              @input=${onInput} @change=${commit}
              @focus=${() => focused.set(true)} @blur=${() => focused.set(false)}>`)}
      ${() => (clearable() && value() !== '' && !disabled()
        ? html`<ui-icon-button icon="close" label="Clear" @click=${clear}></ui-icon-button>`
        : html`<slot name="trailing"></slot>`)}`;

    return html`
      <label class=${cls}>
        <span class="field" part="field">
          ${() => (variant() === 'filled' ? html`<span class="filled-hover" aria-hidden="true"></span>` : null)}
          ${() => (variant() === 'outlined'
            ? html`<fieldset aria-hidden="true"><legend><span>${label}${() => (required() ? ' *' : '')}</span></legend></fieldset>`
            : null)}
          ${() => (label() ? html`<span class="label" part="label">${label}${() => (required() ? ' *' : '')}</span>` : null)}
          ${fieldBody}
        </span>
        ${() => (describe() || counter()
          ? html`<span class="below" part="helper">
              <span class="msg" role=${() => (error() ? 'alert' : null)}>${describe}</span>
              ${() => counter() && html`<span class="count">${counter}</span>`}
            </span>`
          : null)}
      </label>`;
  },
});

export const tag = 'ui-text-field';
export const themeVars = t;
