// <ui-backdrop> — a full-screen scrim behind custom overlays.
//
//   <ui-backdrop open=${open} @close=${() => open(false)}></ui-backdrop>
//
// Fades in/out, locks page scroll while open, and requests closing by
// emitting `close` when the scrim is clicked — the PARENT owns the state and
// flips the signal.
//
// @prop  {boolean} open=false
// @prop  {boolean} invisible=false — transparent scrim (still catches clicks)
// @event close — scrim clicked; detail: { reason: 'scrim' }
// @part  scrim
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, effect, onCleanup } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { presence } from '../motion/presence.js';
import { fx } from '../motion/animate.js';
import { scrollLock } from '../util/focus.js';

const t = vars('ui-backdrop', {
  bg: `color-mix(in srgb, ${sys.color.scrim} 32%, transparent)`,
});

const styles = css`
  :host { display: contents; }
  .scrim {
    position: fixed;
    inset: 0;
    z-index: ${sys.z.modal};
    background: ${t.bg};
  }
  .invisible { background: transparent; }
`;

define('ui-backdrop', {
  props: { open: false, invisible: false },
  styles: [base, styles],
  setup({ open, invisible }, host) {
    let unlock = null;
    effect(() => {
      if (open()) {
        if (!unlock) unlock = scrollLock();
      } else {
        unlock?.();
        unlock = null;
      }
    });
    onCleanup(() => unlock?.());

    const view = () => html`
      <div part="scrim" aria-hidden="true"
           class=${() => `scrim${invisible() ? ' invisible' : ''}`}
           @click=${() => host.emit('close', { reason: 'scrim' })}></div>`;

    return html`${presence(open, view, {
      enter: fx.fadeIn,
      exit: fx.fadeOut,
      exitDuration: 'short3',
    })}`;
  },
});

export const tag = 'ui-backdrop';
export const themeVars = t;
