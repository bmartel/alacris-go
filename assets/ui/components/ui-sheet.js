// <ui-sheet> — a Material bottom sheet.
//
//   <ui-sheet open=${open} @close=${() => open(false)}>
//     <span slot="headline">Title</span>
//     Sheet body
//     <ui-button slot="actions" variant="text">Close</ui-button>
//   </ui-sheet>
//
// Modal (default): a scrim plus a panel that slides up from the bottom.
// Focus is trapped and page scroll locked while open. The PARENT owns
// `open`: Escape and scrim clicks emit `close` with a reason.
// Standard: an in-flow panel that expands from zero height — no scrim,
// no trap.
//
// @prop  {boolean} open=false
// @prop  {string}  variant='modal' — modal | standard
// @prop  {boolean} persistent=false — Escape/scrim do not request closing
// @prop  {string}  label=''         — accessible name if no headline slot
// @event close  — detail: { reason: 'esc' | 'scrim' | 'method' }
// @event opened — enter animation finished
// @event closed — exit animation finished, DOM removed
// @slot  (default) — body content
// @slot  headline
// @slot  actions
// @part  scrim, surface, handle, headline, body, actions
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, effect, onCleanup, signal } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { presence } from '../motion/presence.js';
import { animate, fx } from '../motion/animate.js';
import { focusTrap, scrollLock } from '../util/focus.js';

const t = vars('ui-sheet', {
  bg: sys.color.surfaceContainerLow,
  fg: sys.color.onSurface,
  radius: sys.radius.xl,
  width: 'min(640px, 100%)',
  handle: sys.color.outline,
  scrim: `color-mix(in srgb, ${sys.color.scrim} 32%, transparent)`,
});

const styles = css`
  :host { display: contents; }
  .overlay {
    position: fixed;
    inset: 0;
    z-index: ${sys.z.modal};
    display: flex;
    align-items: flex-end;
    justify-content: center;
  }
  .scrim { position: absolute; inset: 0; background: ${t.scrim}; }
  .surface {
    position: relative;
    display: flex;
    flex-direction: column;
    inline-size: ${t.width};
    max-block-size: calc(100vh - 72px);
    background: ${t.bg};
    color: ${t.fg};
    border-start-start-radius: ${t.radius};
    border-start-end-radius: ${t.radius};
    box-shadow: ${sys.elevation[3]};
  }
  .handle {
    flex: none;
    inline-size: 32px;
    block-size: 4px;
    margin: ${sys.space(4)} auto ${sys.space(2)};
    border-radius: ${sys.radius.full};
    background: ${t.handle};
  }
  .headline {
    padding-inline: ${sys.space(6)};
    padding-block-end: ${sys.space(2)};
    font: ${sys.type.titleLg};
    letter-spacing: ${sys.tracking.titleLg};
  }
  .headline:not(.has) { display: none; }
  .body {
    overflow: auto;
    padding: ${sys.space(2)} ${sys.space(6)} ${sys.space(6)};
    font: ${sys.type.bodyMd};
    letter-spacing: ${sys.tracking.bodyMd};
    color: ${sys.color.onSurfaceVariant};
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    gap: ${sys.space(2)};
    padding: ${sys.space(2)} ${sys.space(4)} ${sys.space(4)};
  }
  .actions:not(.has) { display: none; }

  .std {
    display: grid;
    grid-template-rows: 0fr;
    background: ${t.bg};
    color: ${t.fg};
    border-start-start-radius: ${t.radius};
    border-start-end-radius: ${t.radius};
    overflow: hidden;
    transition: grid-template-rows ${sys.duration.medium2} ${sys.easing.emphasized};
  }
  .std.open { grid-template-rows: 1fr; }
  .std-inner { min-block-size: 0; overflow: hidden; }
`;

define('ui-sheet', {
  props: { open: false, variant: 'modal', persistent: false, label: '' },
  styles: [base, styles],
  setup({ open, variant, persistent, label }, host) {
    let releaseTrap = null;
    let unlock = null;
    let surfaceEl = null;
    const hasHeadline = signal(false);

    const requestClose = (reason) => {
      if (reason !== 'method' && persistent()) return;
      host.emit('close', { reason });
    };

    const onDocKeydown = (e) => {
      if (e.key === 'Escape') requestClose('esc');
    };

    effect(() => {
      if (open() && variant() !== 'standard') {
        document.addEventListener('keydown', onDocKeydown, true);
        unlock = scrollLock();
        queueMicrotask(() => {
          if (open() && !releaseTrap) releaseTrap = focusTrap(host);
        });
      } else {
        document.removeEventListener('keydown', onDocKeydown, true);
        releaseTrap?.();
        releaseTrap = null;
        unlock?.();
        unlock = null;
        if (surfaceEl?.isConnected) {
          animate(surfaceEl, fx.sheetOut, { duration: 'short4', easing: 'emphasizedAccelerate' });
        }
      }
    });
    onCleanup(() => {
      document.removeEventListener('keydown', onDocKeydown, true);
      releaseTrap?.();
      unlock?.();
    });

    const hasSlot = (el, set) => {
      const sync = () => set(el.assignedElements().length > 0);
      el.addEventListener('slotchange', sync);
      sync();
    };

    const modalView = () => html`
      <div class="overlay">
        <div class="scrim" part="scrim" aria-hidden="true" @click=${() => requestClose('scrim')}></div>
        <div class="surface" part="surface" role="dialog" aria-modal="true"
             aria-labelledby=${() => (hasHeadline() ? 'headline' : null)}
             aria-label=${() => (hasHeadline() ? null : (label() || 'Sheet'))}
             tabindex="-1"
             ref=${(el) => { surfaceEl = el; animate(el, fx.sheetIn, { duration: 'medium2', easing: 'emphasizedDecelerate' }); }}>
          <div class="handle" part="handle" aria-hidden="true"></div>
          <div class="headline" part="headline" id="headline"
               ref=${(el) => hasSlot(el.querySelector('slot'), (has) => {
                 hasHeadline.set(has);
                 el.classList.toggle('has', has);
               })}>
            <slot name="headline"></slot>
          </div>
          <div class="body" part="body"><slot></slot></div>
          <div class="actions" part="actions"
               ref=${(el) => hasSlot(el.querySelector('slot'), (has) => el.classList.toggle('has', has))}>
            <slot name="actions"></slot>
          </div>
        </div>
      </div>`;

    return html`
      ${() =>
        variant() === 'standard'
          ? html`<div class=${() => `std${open() ? ' open' : ''}`} part="surface" role="region"
                      aria-label=${() => label() || 'Sheet'}>
              <div class="std-inner">
                <div class="handle" part="handle" aria-hidden="true"></div>
                <div class="body" part="body"><slot></slot></div>
              </div>
            </div>`
          : null}
      ${presence(() => open() && variant() !== 'standard', modalView, {
        enter: fx.fadeIn,
        exit: fx.fadeOut,
        exitDuration: 'short4',
        onEntered: () => host.emit('opened'),
        onExited: () => host.emit('closed'),
      })}`;
  },
});

export const tag = 'ui-sheet';
export const themeVars = t;
