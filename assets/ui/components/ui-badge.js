// <ui-badge> — a small status badge anchored to the top-end corner of its
// slotted content (an icon button, a nav item, an avatar…).
//
// Hidden while `value` is 0 unless `dot`. Counts above `max` render '99+'
// style. Appearing/disappearing animates with a scale-in/out.
//
// @prop  {number}  value=0   — the count; 0 hides the badge (unless `dot`)
// @prop  {number}  max=99    — counts above this render as 'max+'
// @prop  {boolean} dot=false — a plain 8px dot instead of a count
// @prop  {boolean} show=true — master visibility switch
// @prop  {string}  label=''  — accessible meaning of the badge ('3 unread');
//                              empty marks the badge decorative
// @slot  (default) — the anchored content
// @part  badge — the badge pill/dot
// @vars  --ui-badge-bg, --ui-badge-fg, --ui-badge-size, --ui-badge-dot-size,
//        --ui-badge-font

import { define, html, css, vars, computed } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { presence } from '../motion/presence.js';
import { fx } from '../motion/animate.js';

const t = vars('ui-badge', {
  bg: sys.color.error,
  fg: sys.color.onError,
  size: '16px',
  dotSize: '8px',
  font: sys.type.labelSm,
  tracking: sys.tracking.labelSm,
});

const styles = css`
  :host {
    position: relative;
    display: inline-flex;
    vertical-align: middle;
  }
  .badge {
    position: absolute;
    inset-block-start: 0;
    inset-inline-end: 0;
    transform: translate(50%, -50%);
    z-index: 1;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-block-size: ${t.size};
    min-inline-size: ${t.size};
    padding-inline: ${sys.space(1)};
    border-radius: ${sys.radius.full};
    background: ${t.bg};
    color: ${t.fg};
    font: ${t.font};
    letter-spacing: ${t.tracking};
    line-height: 1;
    pointer-events: none;
    white-space: nowrap;
  }
  .badge.dot {
    min-block-size: ${t.dotSize};
    min-inline-size: ${t.dotSize};
    block-size: ${t.dotSize};
    inline-size: ${t.dotSize};
    padding: 0;
  }
`;

define('ui-badge', {
  props: { value: 0, max: 99, dot: false, show: true, label: '' },
  styles: [base, styles],
  setup({ value, max, dot, show, label }) {
    const visible = computed(() => show() && (dot() || value() > 0));
    const text = computed(() =>
      dot() ? '' : value() > max() ? `${max()}+` : String(value()));

    const view = () => html`
      <span part="badge"
            class=${() => `badge${dot() ? ' dot' : ''}`}
            aria-label=${() => label() || null}
            aria-hidden=${() => (label() ? null : 'true')}>${text}</span>`;

    return html`
      <slot></slot>
      ${presence(visible, view, {
        enter: fx.scaleIn,
        exit: fx.scaleOut,
        enterDuration: 'short4',
        exitDuration: 'short2',
      })}`;
  },
});

export const tag = 'ui-badge';
export const themeVars = t;
