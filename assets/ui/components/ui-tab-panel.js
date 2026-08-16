// <ui-tab-panel> — the content pane paired with a <ui-tab>.
//
// Slot it into <ui-tabs slot="panels">. The parent sets `active` when its
// `value` matches; do not set `active` yourself. The panel is hidden while
// inactive and fades in when it becomes active.
//
// @prop  {string}  value=''      — REQUIRED; matched against the tabs' value
// @prop  {boolean} active=false  — managed by the parent <ui-tabs>
// @slot  (default) — panel content
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, effect } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { animate, fx } from '../motion/animate.js';

const t = vars('ui-tab-panel', {
  fg: sys.color.onSurface,
  font: sys.type.bodyMd,
  padBlock: sys.space(4),
});

const styles = css`
  :host { display: block; }
  .body {
    color: ${t.fg};
    font: ${t.font};
    padding-block: ${t.padBlock};
  }
`;

define('ui-tab-panel', {
  props: { value: '', active: false },
  styles: [base, styles],
  setup({ value, active }, host) {
    host.setAttribute('role', 'tabpanel');
    host.tabIndex = 0;
    effect(() => {
      if (active()) host.removeAttribute('hidden');
      else host.setAttribute('hidden', '');
    });

    return html`${() =>
      active()
        ? html`<div class="body"
                    ref=${(el) => animate(el, fx.fadeIn, { duration: 'short4', easing: 'standard' })}>
            <slot></slot>
          </div>`
        : null}`;
  },
});

export const tag = 'ui-tab-panel';
export const themeVars = t;
