// <ui-alert> — a severity-colored callout for statuses and messages.
//
//   <ui-alert severity="success" title="Saved" dismissible @dismiss=${remove}>
//     Your changes are safe.
//     <ui-button slot="action" variant="text">Undo</ui-button>
//   </ui-alert>
//
// Live-region semantics: the host defaults to role="status" (announced
// politely). If the alert appears dynamically in response to an event and must
// interrupt, set role="alert" on the element yourself — an author-set role is
// never overwritten.
//
// Dismissing: the close button collapses the alert (height + opacity), then
// emits `dismiss`. The PARENT owns the DOM and removes the element (or flips
// the condition rendering it).
//
// @prop  {string}  severity='info'  — info | success | warning | error
// @prop  {string}  variant='tonal'  — tonal | filled | outlined
// @prop  {boolean} dismissible=false — shows a trailing close button
// @prop  {string}  icon=''  — leading icon override; defaults per severity
//                             (info → 'info', success → 'check-circle',
//                              warning → 'warning', error → 'error')
// @prop  {string}  title='' — optional bold first line
// @event dismiss — close button pressed, after the collapse animation
// @slot  (default) — message body
// @slot  action    — trailing action, e.g. a text <ui-button>
// @part  container, icon, title, message, action, close
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, computed } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base, focusRingOn } from './base.js';
import { animate, settled } from '../motion/animate.js';
import { ripple } from '../motion/ripple.js';
import './ui-icon.js';

const SEVERITY_ICONS = {
  info: 'info',
  success: 'check-circle',
  warning: 'warning',
  error: 'error',
};

const t = vars('ui-alert', {
  radius: sys.radius.sm,
  padding: sys.space(4),
  gap: sys.space(3),
  font: sys.type.bodyMd,
  titleFont: sys.type.titleSm,
});

const styles = css`
  :host { display: block; }
  .container {
    display: flex;
    align-items: flex-start;
    gap: ${t.gap};
    padding: ${t.padding};
    border: 1px solid transparent; /* structural: keeps outlined the same size */
    border-radius: ${t.radius};
    font: ${t.font};
    letter-spacing: ${sys.tracking.bodyMd};
  }
  .icon { flex: none; --ui-icon-size: 1.375rem; }
  .content { flex: 1; min-inline-size: 0; }
  .title {
    font: ${t.titleFont};
    letter-spacing: ${sys.tracking.titleSm};
    margin-block-end: ${sys.space(1)};
  }
  .action { flex: none; align-self: center; display: flex; align-items: center; }

  /* tonal — severity container bg, on-container fg */
  .tonal.info    { background: ${sys.color.infoContainer};    color: ${sys.color.onInfoContainer}; }
  .tonal.success { background: ${sys.color.successContainer}; color: ${sys.color.onSuccessContainer}; }
  .tonal.warning { background: ${sys.color.warningContainer}; color: ${sys.color.onWarningContainer}; }
  .tonal.error   { background: ${sys.color.errorContainer};   color: ${sys.color.onErrorContainer}; }

  /* filled — severity bg, on-severity fg */
  .filled.info    { background: ${sys.color.info};    color: ${sys.color.onInfo}; }
  .filled.success { background: ${sys.color.success}; color: ${sys.color.onSuccess}; }
  .filled.warning { background: ${sys.color.warning}; color: ${sys.color.onWarning}; }
  .filled.error   { background: ${sys.color.error};   color: ${sys.color.onError}; }

  /* outlined — transparent bg, severity fg + border */
  .outlined { background: transparent; border-color: currentColor; }
  .outlined.info    { color: ${sys.color.info}; }
  .outlined.success { color: ${sys.color.success}; }
  .outlined.warning { color: ${sys.color.warning}; }
  .outlined.error   { color: ${sys.color.error}; }

  .close {
    position: relative;
    isolation: isolate;
    flex: none;
    align-self: center;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    inline-size: 28px;
    block-size: 28px;
    padding: 0;
    border: none;
    border-radius: ${sys.radius.full};
    background: transparent;
    color: inherit;
    cursor: pointer;
    --ui-icon-size: 18px;
  }
  ${focusRingOn('.close')}
  .layer {
    position: absolute; inset: 0; z-index: -1;
    border-radius: inherit; background: currentColor; opacity: 0;
    transition: opacity ${sys.duration.short2} ${sys.easing.standard};
  }
  .close:hover .layer { opacity: ${sys.state.hover}; }
  .close:focus-visible .layer { opacity: ${sys.state.focus}; }
  .close:active .layer { opacity: ${sys.state.pressed}; }
`;

define('ui-alert', {
  props: { severity: 'info', variant: 'tonal', dismissible: false, icon: '', title: '' },
  styles: [base, styles],
  setup({ severity, variant, dismissible, icon, title }, host) {
    // Polite live region by default; authors set role="alert" for interrupts.
    if (!host.hasAttribute('role')) host.setAttribute('role', 'status');

    const cls = computed(() => `container ${variant()} ${severity()}`);
    const shownIcon = computed(() => icon() || SEVERITY_ICONS[severity()] || SEVERITY_ICONS.info);

    let dismissing = false;
    const onDismiss = () => {
      if (dismissing) return;
      dismissing = true;
      const height = host.scrollHeight || 0;
      // Simulated DOMs (and display:none hosts) measure 0 — skip straight out.
      if (!height) { host.emit('dismiss'); return; }
      host.style.overflow = 'hidden';
      const anim = animate(
        host,
        [{ blockSize: `${height}px`, opacity: 1 }, { blockSize: '0px', opacity: 0 }],
        { duration: 'medium1', easing: 'emphasizedAccelerate' },
      );
      settled(anim).then(() => host.emit('dismiss'));
    };

    return html`
      <div class=${cls} part="container">
        <ui-icon class="icon" part="icon" name=${shownIcon}></ui-icon>
        <div class="content">
          ${() => (title() ? html`<div class="title" part="title">${title}</div>` : null)}
          <div class="message" part="message"><slot></slot></div>
        </div>
        <div class="action" part="action"><slot name="action"></slot></div>
        ${() =>
          dismissible()
            ? html`<button class="close" part="close" type="button" aria-label="Dismiss"
                      @click=${onDismiss} ref=${(el) => ripple(el, { centered: true })}>
                  <span class="layer" aria-hidden="true"></span>
                  <ui-icon name="close"></ui-icon>
                </button>`
            : null}
      </div>`;
  },
});

export const tag = 'ui-alert';
export const themeVars = t;
