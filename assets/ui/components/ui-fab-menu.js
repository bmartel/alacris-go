// <ui-fab-menu> — a Material FAB menu: a trigger FAB that expands related
// actions stacked above it.
//
//   <ui-fab-menu>
//     <ui-fab slot="trigger" icon="add"></ui-fab>
//     <ui-fab icon="edit" label="Edit" size="sm"></ui-fab>
//     <ui-fab icon="send" label="Send" size="sm"></ui-fab>
//   </ui-fab-menu>
//
// The PARENT may pass `open`; clicking the trigger toggles it and emits
// `open`/`close`. Related actions are a disclosure of buttons (not a menu
// widget) so slotted `<ui-fab>`s keep their native button semantics.
// Escape closes and returns focus to the trigger.
//
// @prop  {boolean} open=false
// @prop  {string}  label='' — accessible name for the action list
// @event open  — menu visible (after the enter animation)
// @event close — menu removed (after the exit animation)
// @slot  trigger  — the <ui-fab> that toggles the menu
// @slot  (default) — related <ui-fab> actions
// @part  actions, trigger
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, effect, onCleanup } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { presence } from '../motion/presence.js';
import { fx } from '../motion/animate.js';
import './ui-fab.js';

const t = vars('ui-fab-menu', {
  gap: sys.space(4),
});

const styles = css`
  :host {
    display: inline-flex;
    flex-direction: column-reverse;
    align-items: flex-end;
    gap: ${t.gap};
  }
  .actions {
    display: flex;
    flex-direction: column-reverse;
    align-items: flex-end;
    gap: ${t.gap};
  }
  .trigger { display: inline-flex; }
`;

define('ui-fab-menu', {
  props: { open: false, label: '' },
  styles: [base, styles],
  setup({ open, label }, host) {
    const triggerEl = () => host.querySelector('[slot="trigger"]');
    const triggerControl = () => {
      const el = triggerEl();
      return el?.shadowRoot?.querySelector('button, a[href]') || el;
    };
    const syncTrigger = () => {
      const control = triggerControl();
      if (!control) return;
      control.setAttribute('aria-haspopup', 'true');
      control.setAttribute('aria-expanded', open.peek() ? 'true' : 'false');
    };

    host.addEventListener('click', (e) => {
      const trigger = triggerEl();
      if (trigger && e.composedPath().includes(trigger)) {
        open.set(!open.peek());
        return;
      }
      if (open.peek() && host.contains(e.target) && e.target !== host) {
        open.set(false);
      }
    });

    effect(() => {
      open();
      queueMicrotask(syncTrigger);
    });

    effect(() => {
      if (!open()) return;
      const onKey = (e) => {
        if (e.key !== 'Escape') return;
        e.stopPropagation();
        open.set(false);
        triggerControl()?.focus?.();
      };
      document.addEventListener('keydown', onKey, true);
      return () => document.removeEventListener('keydown', onKey, true);
    });
    onCleanup(() => {
      const control = triggerControl();
      control?.removeAttribute('aria-haspopup');
      control?.removeAttribute('aria-expanded');
    });

    const actionsView = () => html`
      <div class="actions" part="actions" role="group"
           aria-label=${() => label() || 'Actions'}>
        <slot></slot>
      </div>`;

    return html`
      <div class="trigger" part="trigger">
        <slot name="trigger"></slot>
      </div>
      ${presence(open, actionsView, {
        enter: fx.scaleIn,
        exit: fx.scaleOut,
        enterDuration: 'short4',
        exitDuration: 'short2',
        onEntered: () => host.emit('open'),
        onExited: () => host.emit('close'),
      })}`;
  },
});

export const tag = 'ui-fab-menu';
export const themeVars = t;
