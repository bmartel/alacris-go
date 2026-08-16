// <ui-app-bar> — the Material top app bar.
//
//   <ui-app-bar variant="small" scroll-elevate>
//     <ui-icon-button slot="navigation" icon="menu" label="Menu"></ui-icon-button>
//     Page title
//     <ui-icon-button slot="actions" icon="search" label="Search"></ui-icon-button>
//   </ui-app-bar>
//
// The host is position:sticky at the top. `elevated` forces the on-scroll
// container tint + shadow; `scrollElevate` listens to window scroll and
// applies it automatically once the page is scrolled (listener removed when
// the prop turns off or the element leaves the document).
//
// @prop  {string}  variant='small' — small | center | medium | large
// @prop  {boolean} elevated=false  — force the scrolled container/elevation
// @prop  {boolean} scrollElevate=false — auto-elevate once window is scrolled
// @slot  navigation — leading icon button
// @slot  (default)  — title text
// @slot  actions    — trailing icon buttons
// @part  bar — the header surface
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, signal, effect, computed } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';

const t = vars('ui-app-bar', {
  bg: sys.color.surface,
  scrolledBg: sys.color.surfaceContainer,
  fg: sys.color.onSurface,
  heightSmall: '64px',
  heightMedium: '112px',
  heightLarge: '152px',
  font: sys.type.titleLg,
  tracking: sys.tracking.titleLg,
});

const styles = css`
  :host {
    display: block;
    position: sticky;
    inset-block-start: 0;
    z-index: ${sys.z.appBar};
  }
  .bar {
    display: grid;
    grid-template-columns: auto 1fr auto;
    align-items: center;
    column-gap: ${sys.space(1)};
    padding-inline: ${sys.space(1)};
    block-size: ${t.heightSmall};
    background: ${t.bg};
    color: ${t.fg};
    box-shadow: none;
    transition: background-color ${sys.duration.short4} ${sys.easing.standard},
                box-shadow ${sys.duration.short4} ${sys.easing.standard};
  }
  .raised { background: ${t.scrolledBg}; box-shadow: ${sys.elevation[2]}; }
  .title {
    font: ${t.font};
    letter-spacing: ${t.tracking};
    padding-inline: ${sys.space(1)};
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .center .title { text-align: center; }
  .nav, .actions { display: inline-flex; align-items: center; gap: ${sys.space(1)}; }
  .medium, .large {
    grid-template-rows: ${t.heightSmall} 1fr;
    grid-template-areas: 'nav . actions' 'title title title';
    align-items: start;
  }
  .medium { block-size: ${t.heightMedium}; }
  .large { block-size: ${t.heightLarge}; }
  .medium .nav, .large .nav { grid-area: nav; align-self: center; }
  .medium .actions, .large .actions { grid-area: actions; align-self: center; }
  .medium .title, .large .title {
    grid-area: title;
    align-self: end;
    padding-block-end: ${sys.space(4)};
    padding-inline: ${sys.space(3)};
  }
  .medium .title { font: ${sys.type.headlineSm}; letter-spacing: ${sys.tracking.headlineSm}; }
  .large .title { font: ${sys.type.headlineMd}; letter-spacing: ${sys.tracking.headlineMd}; }
`;

define('ui-app-bar', {
  props: { variant: 'small', elevated: false, scrollElevate: false },
  styles: [base, styles],
  setup({ variant, elevated, scrollElevate }, host) {
    const scrolled = signal(false);

    effect(() => {
      if (!scrollElevate()) {
        scrolled.set(false);
        return;
      }
      const onScroll = () => scrolled.set((window.scrollY || 0) > 0);
      onScroll();
      window.addEventListener('scroll', onScroll, { passive: true });
      return () => window.removeEventListener('scroll', onScroll);
    });

    const cls = computed(
      () => `bar ${variant()}${elevated() || scrolled() ? ' raised' : ''}`
    );

    return html`
      <header class=${cls} part="bar">
        <div class="nav"><slot name="navigation"></slot></div>
        <div class="title"><slot></slot></div>
        <div class="actions"><slot name="actions"></slot></div>
      </header>`;
  },
});

export const tag = 'ui-app-bar';
export const themeVars = t;
