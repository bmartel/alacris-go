// <ui-menu> — a menu anchored to a slotted trigger.
//
//   <ui-menu placement="bottom-start" @select=${(e) => ...}>
//     <ui-icon-button slot="anchor" icon="more-vert" label="More"></ui-icon-button>
//     <ui-menu-item value="edit" icon="edit">Edit</ui-menu-item>
//     <ui-menu-item value="delete" icon="delete" danger>Delete</ui-menu-item>
//   </ui-menu>
//
// The "anchor" slot renders inline; clicking it toggles the menu. The panel is
// position:fixed, anchored to the slotted trigger, flips when it would
// overflow, and stays glued through scroll/resize. While open, focus moves to
// the first item and arrows rove vertically; Escape closes and refocuses the
// anchor, Tab and outside pointerdown close. Selecting an item emits `select`
// and closes.
//
// @prop  {boolean} open=false
// @prop  {string}  placement='bottom-start' — side[-alignment] (see util/position.js)
// @event select — an item was chosen; detail: { value }
// @event open   — panel visible (after the enter animation)
// @event close  — panel removed (after the exit animation)
// @slot  anchor    — the trigger element
// @slot  (default) — <ui-menu-item> children
// @part  panel — the floating menu surface
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, effect, onCleanup } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { presence } from '../motion/presence.js';
import { fx } from '../motion/animate.js';
import { autoUpdate } from '../util/position.js';
import { rovingTabindex } from '../util/keys.js';
import './ui-menu-item.js';

const t = vars('ui-menu', {
  bg: sys.color.surfaceContainer,
  radius: sys.radius.xs,
  minWidth: '112px',
  maxWidth: '280px',
  elevation: sys.elevation[2],
});

const styles = css`
  :host { display: inline-block; }
  .anchor { display: contents; }
  .panel {
    position: fixed;
    inset-inline-start: 0;
    inset-block-start: 0;
    z-index: ${sys.z.modal};
    min-inline-size: ${t.minWidth};
    max-inline-size: ${t.maxWidth};
    padding-block: ${sys.space(2)};
    background: ${t.bg};
    border-radius: ${t.radius};
    box-shadow: ${t.elevation};
    overflow-y: auto;
    max-block-size: calc(100vh - 16px);
  }
`;

// transform-origin on the panel's anchor-facing corner, per placement.
const originFor = (placement) => {
  const [side, align = 'start'] = placement.split('-');
  const cross = align === 'end' ? 'right' : align === 'center' ? 'center' : 'left';
  if (side === 'top') return `bottom ${cross}`;
  if (side === 'bottom') return `top ${cross}`;
  const block = align === 'end' ? 'bottom' : align === 'center' ? 'center' : 'top';
  return `${block} ${side === 'left' ? 'right' : 'left'}`;
};

define('ui-menu', {
  props: { open: false, placement: 'bottom-start' },
  styles: [base, styles],
  setup({ open, placement }, host) {
    let anchorSlot = null;
    let stopPosition = null;
    let roving = null;

    const anchorEl = () => anchorSlot?.assignedElements()[0] || host;

    const popupTrigger = (el) => {
      if (!el || el === host) return null;
      return el.shadowRoot?.querySelector('button, a[href], [role="button"]') || el;
    };

    const syncTrigger = () => {
      const trigger = popupTrigger(anchorEl());
      if (!trigger) return;
      trigger.setAttribute('aria-haspopup', 'menu');
      trigger.setAttribute('aria-expanded', open.peek() ? 'true' : 'false');
    };

    const close = () => {
      if (!open.peek()) return;
      open.set(false);
      (popupTrigger(anchorEl()) || anchorEl()).focus?.();
    };

    host.addEventListener('keydown', (e) => {
      if (!open.peek()) return;
      if (e.key === 'Escape') {
        e.stopPropagation();
        close();
      } else if (e.key === 'Tab') {
        close();
      }
    });

    host.addEventListener('ui-menu-select', (e) => {
      e.stopPropagation();
      host.emit('select', { value: e.detail.value });
      close();
    });

    // Keyboard roving + outside-pointerdown dismissal, exactly while open.
    // Items may be projected through a nested slot (ui-split-button), so
    // collect from the flattened default slot and listen on document.
    effect(() => {
      syncTrigger();
      if (!open()) return;
      const menuItems = () => {
        const slot = host.shadowRoot?.querySelector('.panel > slot');
        return (slot?.assignedElements({ flatten: true }) || [])
          .filter((el) => el.localName === 'ui-menu-item');
      };
      roving = rovingTabindex(host, {
        items: menuItems,
        listenOn: document,
        orientation: 'vertical',
        skip: (el) => el.disabled,
      });
      queueMicrotask(() => {
        if (open.peek()) roving?.focus(0);
      });
      const onPointerDown = (e) => {
        if (!e.composedPath().includes(host)) close();
      };
      const onDocKey = (e) => {
        if (e.key === 'Escape') {
          e.stopPropagation();
          close();
        } else if (e.key === 'Tab') {
          close();
        }
      };
      document.addEventListener('pointerdown', onPointerDown);
      document.addEventListener('keydown', onDocKey);
      return () => {
        document.removeEventListener('pointerdown', onPointerDown);
        document.removeEventListener('keydown', onDocKey);
        roving?.destroy();
        roving = null;
        stopPosition?.();
        stopPosition = null;
      };
    });
    onCleanup(() => stopPosition?.());

    const panelRef = (el) => {
      el.style.transformOrigin = originFor(placement());
      stopPosition = autoUpdate(el, anchorEl(), { placement: placement() });
    };

    const view = () => html`
      <div class="panel" part="panel" role="menu" ref=${panelRef}>
        <slot></slot>
      </div>`;

    return html`
      <span class="anchor" @click=${() => open.set(!open())}>
        <slot name="anchor" ref=${(el) => {
          anchorSlot = el;
          el.addEventListener('slotchange', syncTrigger);
          syncTrigger();
        }}></slot>
      </span>
      ${presence(open, view, {
        enter: fx.scaleIn,
        exit: fx.scaleOut,
        enterDuration: 'short4',
        exitDuration: 'short2',
        enterEasing: 'emphasizedDecelerate',
        exitEasing: 'emphasizedAccelerate',
        onEntered: () => host.emit('open'),
        onExited: () => host.emit('close'),
      })}`;
  },
});

export const tag = 'ui-menu';
export const themeVars = t;
