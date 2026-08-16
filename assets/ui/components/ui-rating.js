// <ui-rating> — a star rating.
//
// A11y choice: the whole rating is ONE control — role="slider" with
// aria-valuenow/-min/-max and aria-valuetext, a single tab stop, and arrow
// keys adjusting the value (Home=0, End=max). The stars themselves are
// pointer affordances only (hover previews, click commits), so there is no
// tab-stop-per-star noise for keyboard and screen-reader users.
//
// Clicking the star matching the current value clears the rating to 0
// (MUI parity).
//
// @prop  {number}  value=0
// @prop  {number}  max=5
// @prop  {boolean} readonly=false — shows the value, no interaction
// @prop  {boolean} disabled=false
// @prop  {string}  label='Rating' — accessible name
// @prop  {string}  size=''  — CSS length for the stars (overrides --ui-icon-size)
// @event change — committed; detail: { value }
// @part  root — the slider container
// @vars  see `t` below (`themeVars.names`); the fill color defaults to primary

import { define, html, css, vars, computed, signal } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base, focusRingOn } from './base.js';
import './ui-icon.js';

const t = vars('ui-rating', {
  activeFg: sys.color.primary,
  emptyFg: sys.color.onSurfaceVariant,
  gap: '2px',
});

const styles = css`
  :host { display: inline-flex; vertical-align: middle; }
  .root {
    display: inline-flex;
    align-items: center;
    gap: ${t.gap};
    border-radius: ${sys.radius.sm};
    color: ${t.emptyFg};
    cursor: pointer;
  }
  ${focusRingOn('.root')}
  .root.readonly, .root.disabled { cursor: default; }
  .root.disabled { pointer-events: none; }
  .star {
    display: inline-flex;
    transition: color ${sys.duration.short2} ${sys.easing.standard},
                scale ${sys.duration.short2} ${sys.easing.standard};
  }
  .star.filled { color: ${t.activeFg}; }
  .root:not(.readonly):not(.disabled) .star:hover { scale: 1.15; }
  .root.disabled .star,
  .root.disabled .star.filled {
    color: color-mix(in srgb, ${sys.color.onSurface} calc(${sys.state.disabledContent} * 100%), transparent);
  }
`;

define('ui-rating', {
  props: { value: 0, max: 5, readonly: false, disabled: false, label: 'Rating', size: '' },
  styles: [base, styles],
  setup({ value, max, readonly, disabled, label, size }, host) {
    const hover = signal(0);
    const shown = computed(() => (hover() > 0 ? hover() : value()));
    const interactive = () => !readonly() && !disabled();

    const set = (next) => {
      if (next === value()) return;
      value.set(next);
      host.emit('change', { value: next });
    };

    const commit = (n) => {
      if (!interactive()) return;
      hover.set(0);
      set(n === value() ? 0 : n);
    };

    const onKeydown = (e) => {
      if (!interactive()) return;
      let next;
      switch (e.key) {
        case 'ArrowRight': case 'ArrowUp': next = Math.min(max(), value() + 1); break;
        case 'ArrowLeft': case 'ArrowDown': next = Math.max(0, value() - 1); break;
        case 'Home': next = 0; break;
        case 'End': next = max(); break;
        default: return;
      }
      e.preventDefault();
      set(next);
    };

    const cls = computed(() =>
      ['root', readonly() && 'readonly', disabled() && 'disabled'].filter(Boolean).join(' '));

    return html`
      <div part="root" class=${cls} role="slider"
           tabindex=${() => (disabled() ? null : '0')}
           aria-label=${label}
           aria-valuemin="0" aria-valuemax=${max} aria-valuenow=${value}
           aria-valuetext=${() => `${value()} of ${max()} stars`}
           aria-readonly=${() => (readonly() ? 'true' : null)}
           aria-disabled=${() => (disabled() ? 'true' : null)}
           style=${() => (size() ? { '--ui-icon-size': size() } : null)}
           @keydown=${onKeydown}
           @pointerleave=${() => hover.set(0)}>
        ${() => {
          const s = shown();
          const m = Math.max(0, max());
          const stars = [];
          for (let n = 1; n <= m; n++) {
            stars.push(html`
              <span class="star ${s >= n ? 'filled' : ''}" aria-hidden="true"
                    @click=${() => commit(n)}
                    @pointerenter=${() => interactive() && hover.set(n)}>
                <ui-icon name=${s >= n ? 'star' : 'star-border'}></ui-icon>
              </span>`);
          }
          return stars;
        }}
      </div>`;
  },
});

export const tag = 'ui-rating';
export const themeVars = t;
