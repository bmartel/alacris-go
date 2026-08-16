// <ui-list-item> — one row of a <ui-list>.
//
// One line (56px) by default; supplying supporting text (prop or slot) makes
// it two lines (72px). `interactive` adds button semantics, a state layer and
// a ripple; `href` renders the row as a link instead. Activation emits only
// the native (bubbling) click — no custom event.
//
// @prop  {string}  headline=''    — primary text (or use the default slot)
// @prop  {string}  supporting=''  — secondary text (or use the supporting slot)
// @prop  {boolean} interactive=false — state layer + ripple + role="button"
// @prop  {string}  href=''        — renders the row as an <a>
// @prop  {boolean} selected=false — secondaryContainer background
// @prop  {boolean} disabled=false
// @slot  (default)  — headline content when the prop is empty
// @slot  leading    — icon / avatar / checkbox
// @slot  supporting — secondary line when the prop is empty
// @slot  trailing   — trailing meta text or icon
// @part  control — the row element (<div> or <a>)
// @vars  see `t` below

import { define, html, css, vars, computed, effect, signal } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base, focusRingOn } from './base.js';
import { ripple } from '../motion/ripple.js';

const t = vars('ui-list-item', {
  height: '56px',
  twoLineHeight: '72px',
  padInline: sys.space(4),
  gap: sys.space(4),
  headlineFg: sys.color.onSurface,
  headlineFont: sys.type.bodyLg,
  supportingFg: sys.color.onSurfaceVariant,
  supportingFont: sys.type.bodyMd,
  trailingFg: sys.color.onSurfaceVariant,
  trailingFont: sys.type.labelSm,
  selectedBg: sys.color.secondaryContainer,
  selectedFg: sys.color.onSecondaryContainer,
});

const styles = css`
  :host { display: block; }
  .control {
    position: relative;
    isolation: isolate;
    display: flex;
    align-items: center;
    gap: ${t.gap};
    min-block-size: calc(${t.height} + var(--ui-density, 0) * 4px);
    padding-inline: ${t.padInline};
    padding-block: ${sys.space(2)};
    color: ${t.headlineFg};
    text-decoration: none;
    background: transparent;
    border: none;
    inline-size: 100%;
    text-align: start;
  }
  .two-line { min-block-size: calc(${t.twoLineHeight} + var(--ui-density, 0) * 4px); }
  ${focusRingOn('.interactive')}
  .interactive { cursor: pointer; user-select: none; }
  .layer {
    position: absolute; inset: 0; z-index: -1;
    background: currentColor; opacity: 0; pointer-events: none;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  .interactive:hover .layer { opacity: ${sys.state.hover}; }
  .interactive:focus-visible .layer { opacity: ${sys.state.focus}; }
  .interactive:active .layer { opacity: ${sys.state.pressed}; }
  .selected { background: ${t.selectedBg}; color: ${t.selectedFg}; }

  .text {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-inline-size: 0;
    gap: calc(${sys.space(1)} / 2);
  }
  .headline {
    font: ${t.headlineFont};
    letter-spacing: ${sys.tracking.bodyLg};
  }
  .supporting {
    font: ${t.supportingFont};
    letter-spacing: ${sys.tracking.bodyMd};
    color: ${t.supportingFg};
  }
  .control:not(.two-line) .supporting { display: none; }
  .trailing {
    display: inline-flex;
    align-items: center;
    flex: none;
    font: ${t.trailingFont};
    letter-spacing: ${sys.tracking.labelSm};
    color: ${t.trailingFg};
  }
  .disabled {
    pointer-events: none;
    color: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContent} * 100%), transparent);
  }
  .disabled .supporting, .disabled .trailing { color: inherit; }
`;

define('ui-list-item', {
  props: {
    headline: '', supporting: '',
    interactive: false, href: '', selected: false, disabled: false,
  },
  styles: [base, styles],
  setup({ headline, supporting, interactive, href, selected, disabled }, host) {
    host.setAttribute('role', 'listitem');

    const slotSupporting = signal(false);
    const twoLine = computed(() => !!supporting() || slotSupporting());
    const isInteractive = computed(() => !!href() || interactive());

    const cls = computed(() =>
      'control' +
      (isInteractive() ? ' interactive' : '') +
      (selected() ? ' selected' : '') +
      (twoLine() ? ' two-line' : '') +
      (disabled() ? ' disabled' : ''));

    const onSupportingSlot = (el) =>
      el.addEventListener('slotchange', () => slotSupporting.set(el.assignedNodes().length > 0));

    const onKeydown = (e) => {
      if (!href() && interactive() && !disabled() && (e.key === 'Enter' || e.key === ' ')) {
        e.preventDefault();
        e.currentTarget.click();
      }
    };
    const onClick = (e) => {
      if (disabled()) { e.preventDefault(); e.stopPropagation(); }
    };

    const inner = html`
      <span class="layer" aria-hidden="true"></span>
      <slot name="leading"></slot>
      <span class="text">
        <span class="headline">${headline}<slot></slot></span>
        <span class="supporting">${supporting}<slot name="supporting" ref=${onSupportingSlot}></slot></span>
      </span>
      <span class="trailing"><slot name="trailing"></slot></span>`;

    const notInteractive = () => disabled() || !isInteractive();

    return html`${() =>
      href()
        ? html`<a part="control" class=${cls} href=${href}
                  aria-disabled=${() => (disabled() ? 'true' : null)}
                  tabindex=${() => (disabled() ? '-1' : null)}
                  @click=${onClick}
                  ref=${(el) => ripple(el, { disabled: notInteractive })}>${inner}</a>`
        : html`<div part="control" class=${cls}
                  role=${() => (interactive() ? 'button' : null)}
                  tabindex=${() => (interactive() && !disabled() ? '0' : null)}
                  aria-disabled=${() => (interactive() && disabled() ? 'true' : null)}
                  aria-current=${() => (selected() ? 'true' : null)}
                  @click=${onClick} @keydown=${onKeydown}
                  ref=${(el) => ripple(el, { disabled: notInteractive })}>${inner}</div>`}`;
  },
});

export const tag = 'ui-list-item';
export const themeVars = t;
