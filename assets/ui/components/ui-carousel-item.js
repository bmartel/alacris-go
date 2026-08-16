// <ui-carousel-item> — one slide inside <ui-carousel>.
//
// `selected` is written by the parent carousel; set the carousel's `index`
// instead of this prop. Width comes from `--ui-carousel-item-basis` on the
// parent (so the carousel variants can size slides without fighting `:host`).
// The last item snaps to the end of the track so it can be scrolled fully into
// view.
//
// @prop  {boolean} selected=false — managed by the parent <ui-carousel>
// @slot  (default) — slide content
// @part  surface — the snap item
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, effect } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';

const t = vars('ui-carousel-item', {
  bg: sys.color.surfaceContainerHighest,
  fg: sys.color.onSurface,
  radius: sys.radius.xl,
});

const styles = css`
  :host {
    display: block;
    flex: 0 0 var(--ui-carousel-item-basis, 40%);
    scroll-snap-align: start;
    min-inline-size: 0;
    transition: flex-basis ${sys.duration.medium2} ${sys.easing.standard};
  }
  :host(:last-child) { scroll-snap-align: end; }
  .surface {
    block-size: 100%;
    overflow: hidden;
    background: ${t.bg};
    color: ${t.fg};
    border-radius: ${t.radius};
  }
`;

define('ui-carousel-item', {
  props: { selected: false },
  styles: [base, styles],
  setup({ selected }, host) {
    effect(() => host.toggleAttribute('selected', selected()));
    return html`<div class="surface" part="surface" role="group"
                     aria-roledescription="slide"><slot></slot></div>`;
  },
});

export const tag = 'ui-carousel-item';
export const themeVars = t;
