// <ui-carousel> — a Material carousel: snap-scrolling slides with previous /
// next controls.
//
//   <ui-carousel label="Photos" variant="multi-browse">
//     <ui-carousel-item>One</ui-carousel-item>
//     <ui-carousel-item>Two</ui-carousel-item>
//   </ui-carousel>
//
// multi-browse shows several items; uncontained lets slides overflow the
// frame; hero makes the selected slide dominate. Selection reflects down as
// `selected` on each <ui-carousel-item>. Prev/next (and arrow keys) scroll
// the selected slide into view; dragging the track updates `index`. The last
// slide snaps to the end of the viewport so a hero (or any oversized) last
// item can still become selected.
//
// @prop  {number} index=0 — the selected slide
// @prop  {string} variant='multi-browse' — multi-browse | uncontained | hero
// @prop  {string} label='' — accessible name for the region
// @event change — index moved; detail: { index }
// @slot  (default) — <ui-carousel-item> children
// @part  viewport, track, prev, next
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, effect, computed, signal, onCleanup } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base, focusRingOn } from './base.js';
import { prefersReducedMotion } from '../motion/animate.js';
import './ui-icon-button.js';
import './ui-carousel-item.js';

const t = vars('ui-carousel', {
  gap: sys.space(2),
  height: '200px',
  heroSize: '80%',
  controlFg: sys.color.onSurface,
  itemBasis: '40%',
});

const styles = css`
  :host { display: block; position: relative; }
  .root { position: relative; }
  .viewport {
    overflow-x: auto;
    overflow-y: hidden;
    scroll-snap-type: x mandatory;
    scrollbar-width: none;
    block-size: ${t.height};
    --ui-carousel-item-basis: ${t.itemBasis};
  }
  .viewport::-webkit-scrollbar { display: none; }
  .track {
    display: flex;
    align-items: stretch;
    gap: ${t.gap};
    block-size: 100%;
    padding-inline: ${sys.space(2)};
  }
  .track > slot { display: contents; }
  .uncontained { --ui-carousel-item-basis: 56%; }
  .hero { --ui-carousel-item-basis: ${t.heroSize}; }
  .hero ::slotted(ui-carousel-item) {
    transition: opacity ${sys.duration.medium2} ${sys.easing.standard};
    opacity: 0.75;
  }
  .hero ::slotted(ui-carousel-item[selected]) {
    opacity: 1;
  }
  .prev, .next {
    position: absolute;
    inset-block-start: 50%;
    translate: 0 -50%;
    z-index: 1;
    color: ${t.controlFg};
  }
  .prev { inset-inline-start: ${sys.space(1)}; }
  .next { inset-inline-end: ${sys.space(1)}; }
  ${focusRingOn('.prev')}
  ${focusRingOn('.next')}
`;

define('ui-carousel', {
  props: { index: 0, variant: 'multi-browse', label: '' },
  styles: [base, styles],
  setup({ index, variant, label }, host) {
    const rev = signal(0);
    const scrollPos = signal(0);
    const maxScroll = signal(0);
    const bump = () => rev.update((n) => n + 1);
    const itemsOf = () => [...host.querySelectorAll('ui-carousel-item')];
    let viewport = null;
    let ignoreScroll = false;
    let ignoreTimer = 0;
    let ignoreGen = 0;

    const clamp = (n) => {
      const max = Math.max(0, itemsOf().length - 1);
      return Math.max(0, Math.min(max, n));
    };

    const updateScrollBounds = () => {
      if (!viewport) return;
      scrollPos.set(viewport.scrollLeft);
      maxScroll.set(Math.max(0, viewport.scrollWidth - viewport.clientWidth));
    };

    let isProgrammatic = true;

    const beginIgnore = () => {
      const gen = ++ignoreGen;
      ignoreScroll = true;
      if (ignoreTimer) clearTimeout(ignoreTimer);
      const done = () => {
        if (gen !== ignoreGen) return;
        ignoreScroll = false;
        ignoreTimer = 0;
        updateScrollBounds();
      };
      viewport.addEventListener('scrollend', done, { once: true });
      ignoreTimer = setTimeout(done, 500);
    };

    const targetScrollFor = (i) => {
      const items = itemsOf();
      if (!items.length || !viewport) return 0;
      const idx = clamp(i);
      const max = Math.max(0, viewport.scrollWidth - viewport.clientWidth);
      if (idx === 0) return 0;
      if (idx === items.length - 1 && items.length > 1) return max;

      const track = viewport.querySelector('.track');
      const trackPad = track ? parseFloat(getComputedStyle(track).paddingInlineStart) || 8 : 8;
      const target = items[idx];
      if (!target) return 0;
      const dest = target.offsetLeft - trackPad;
      return Math.max(0, Math.min(max, dest));
    };

    const go = (n) => {
      const items = itemsOf();
      if (!items.length) return;
      const cur = index.peek();
      const max = viewport ? Math.max(0, viewport.scrollWidth - viewport.clientWidth) : 0;
      const hasLayout = viewport && viewport.clientWidth > 0 && max > 0;

      let targetIdx = clamp(n);

      if (hasLayout) {
        const curScroll = viewport.scrollLeft;
        if (n < cur) {
          for (let i = cur - 1; i >= 0; i--) {
            const dest = targetScrollFor(i);
            if (dest < curScroll - 5 || i === 0) {
              targetIdx = i;
              break;
            }
          }
        } else if (n > cur) {
          for (let i = cur + 1; i < items.length; i++) {
            const dest = targetScrollFor(i);
            if (dest > curScroll + 5 || (i === items.length - 1 && curScroll < max - 5)) {
              targetIdx = i;
              break;
            }
          }
        }
        if (targetIdx === cur && Math.abs(targetScrollFor(targetIdx) - curScroll) < 2) return;
      }

      isProgrammatic = true;
      index.set(targetIdx);
      host.emit('change', { index: targetIdx });
    };

    const scrollToCurrent = () => {
      const items = itemsOf();
      const i = clamp(index.peek());
      const target = items[i];
      if (!target || !viewport) return;
      if (typeof viewport.scrollTo !== 'function') return;
      const dest = targetScrollFor(i);
      if (Math.abs(dest - viewport.scrollLeft) < 1) {
        updateScrollBounds();
        return;
      }
      beginIgnore();
      viewport.scrollTo({
        left: dest,
        behavior: prefersReducedMotion() ? 'auto' : 'smooth',
      });
    };

    const sync = () => {
      rev();
      const i = clamp(index());
      itemsOf().forEach((el, n) => { el.selected = n === i; });
      if (isProgrammatic || !viewport) {
        isProgrammatic = false;
        requestAnimationFrame(scrollToCurrent);
      }
    };
    effect(sync);

    const onScroll = () => {
      updateScrollBounds();
    };

    const onScrollEnd = () => {
      updateScrollBounds();
      if (ignoreScroll) return;
      const items = itemsOf();
      if (!items.length || !viewport) return;
      const curScroll = viewport.scrollLeft;
      const max = viewport.scrollWidth - viewport.clientWidth;

      let best = 0;
      if (max > 1 && curScroll >= max - 2) {
        best = items.length - 1;
      } else if (curScroll <= 2) {
        best = 0;
      } else {
        const origin = viewport.getBoundingClientRect().left;
        let bestDist = Infinity;
        items.forEach((el, n) => {
          const d = Math.abs(el.getBoundingClientRect().left - origin);
          if (d < bestDist) { bestDist = d; best = n; }
        });
      }

      if (best !== index.peek()) {
        index.set(best);
        host.emit('change', { index: best });
      }
    };

    const onKey = (e) => {
      if (e.key === 'ArrowRight' || e.key === 'ArrowDown') { e.preventDefault(); go(index() + 1); }
      else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') { e.preventDefault(); go(index() - 1); }
      else if (e.key === 'Home') { e.preventDefault(); go(0); }
      else if (e.key === 'End') { e.preventDefault(); go(itemsOf().length - 1); }
    };

    const viewportRef = (el) => {
      viewport?.removeEventListener('scrollend', onScrollEnd);
      viewport?.removeEventListener('scroll', onScroll);
      viewport = el;
      el.addEventListener('scrollend', onScrollEnd);
      el.addEventListener('scroll', onScroll, { passive: true });
      requestAnimationFrame(() => {
        updateScrollBounds();
        scrollToCurrent();
      });
    };
    onCleanup(() => {
      if (ignoreTimer) clearTimeout(ignoreTimer);
      viewport?.removeEventListener('scrollend', onScrollEnd);
      viewport?.removeEventListener('scroll', onScroll);
    });

    const cls = computed(() => `viewport ${variant()}`);
    const atStart = computed(() => {
      rev();
      const i = index();
      if (i <= 0) return true;
      if (variant() !== 'hero' && viewport && viewport.clientWidth > 0) {
        return scrollPos() <= 1 && i <= 0;
      }
      return i <= 0;
    });
    const atEnd = computed(() => {
      rev();
      const n = itemsOf().length;
      if (n === 0) return true;
      const i = index();
      if (i >= n - 1) return true;
      if (variant() !== 'hero' && viewport && viewport.clientWidth > 0) {
        const max = maxScroll();
        if (max > 1 && scrollPos() >= max - 2) return true;
      }
      return false;
    });

    return html`
      <div class="root" role="region" aria-roledescription="carousel"
           aria-label=${() => label() || null} @keydown=${onKey}>
        <ui-icon-button class="prev" part="prev" variant="tonal"
                        icon="chevron-left" label="Previous"
                        ?disabled=${atStart} @click=${() => go(index() - 1)}></ui-icon-button>
        <div class=${cls} part="viewport" tabindex="0" ref=${viewportRef}>
          <div class="track" part="track">
            <slot @slotchange=${bump}></slot>
          </div>
        </div>
        <ui-icon-button class="next" part="next" variant="tonal"
                        icon="chevron-right" label="Next"
                        ?disabled=${atEnd} @click=${() => go(index() + 1)}></ui-icon-button>
      </div>`;
  },
});

export const tag = 'ui-carousel';
export const themeVars = t;
