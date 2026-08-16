// <ui-slider> — a Material slider on native <input type="range">s for
// keyboard and screen-reader behavior.
//
// The active track portion is painted with `--ui-slider-fill` (or start/end
// when `range`) bound from the template into a gradient; the thumb's
// hover/focus halo is a box-shadow state layer.
//
// @prop  {number}  value=0
// @prop  {number}  min=0
// @prop  {number}  max=100
// @prop  {number}  step=1
// @prop  {boolean} range=false — two thumbs; uses valueStart / valueEnd
// @prop  {number}  valueStart=0
// @prop  {number}  valueEnd=100
// @prop  {string}  label=''   — REQUIRED accessible name (aria-label)
// @prop  {boolean} disabled=false
// @prop  {boolean} showValue=false — value bubble above the thumb while
//        focused/dragging (animates in and out)
// @prop  {string}  name=''    — form participation
// @event input  — every drag/keystroke; detail: { value } or { start, end }
// @event change — committed value; detail: { value } or { start, end }
// @part  input — the native <input type="range"> (both, when range)
// @vars  see `t` below (`themeVars.names`)

import { define, html, css, vars, computed, signal, effect } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';
import { formBind } from '../util/form.js';
import { presence } from '../motion/presence.js';
import { fx } from '../motion/animate.js';

const t = vars('ui-slider', {
  trackHeight: '4px',
  thumbSize: '20px',
  track: sys.color.surfaceContainerHighest,
  active: sys.color.primary,
  thumb: sys.color.primary,
  bubbleBg: sys.color.primary,
  bubbleFg: sys.color.onPrimary,
});

const styles = css`
  :host { display: block; inline-size: 240px; }
  .root { position: relative; display: flex; align-items: center; }
  input {
    appearance: none;
    -webkit-appearance: none;
    inline-size: 100%;
    /* density never shrinks the touch target below 44px */
    block-size: max(44px, calc(48px + var(--ui-density, 0) * 4px));
    margin: 0;
    padding: 0;
    background: transparent;
    cursor: pointer;
  }
  /* The thumb halo below replaces the native outline as the focus indicator. */
  input:focus-visible { outline: none; }

  input::-webkit-slider-runnable-track {
    block-size: ${t.trackHeight};
    border-radius: ${sys.radius.full};
    background: linear-gradient(to right,
      ${t.track} 0%,
      ${t.track} var(--ui-slider-start, 0%),
      ${t.active} var(--ui-slider-start, 0%),
      ${t.active} var(--ui-slider-end, var(--ui-slider-fill, 0%)),
      ${t.track} var(--ui-slider-end, var(--ui-slider-fill, 0%)),
      ${t.track} 100%);
  }
  input::-webkit-slider-thumb {
    appearance: none;
    -webkit-appearance: none;
    inline-size: ${t.thumbSize};
    block-size: ${t.thumbSize};
    margin-block-start: calc((${t.trackHeight} - ${t.thumbSize}) / 2);
    border: none;
    border-radius: ${sys.radius.full};
    background: ${t.thumb};
    transition: box-shadow ${sys.duration.short2} ${sys.easing.standard};
  }
  input:hover::-webkit-slider-thumb {
    box-shadow: 0 0 0 10px color-mix(in srgb, ${t.thumb} calc(${sys.state.hover} * 100%), transparent);
  }
  input:focus-visible::-webkit-slider-thumb,
  input:active::-webkit-slider-thumb {
    box-shadow: 0 0 0 10px color-mix(in srgb, ${t.thumb} calc(${sys.state.focus} * 100%), transparent);
  }

  input::-moz-range-track {
    block-size: ${t.trackHeight};
    border-radius: ${sys.radius.full};
    background: ${t.track};
  }
  input::-moz-range-progress {
    block-size: ${t.trackHeight};
    border-radius: ${sys.radius.full};
    background: ${t.active};
  }
  input::-moz-range-thumb {
    inline-size: ${t.thumbSize};
    block-size: ${t.thumbSize};
    border: none;
    border-radius: ${sys.radius.full};
    background: ${t.thumb};
    transition: box-shadow ${sys.duration.short2} ${sys.easing.standard};
  }
  input:hover::-moz-range-thumb {
    box-shadow: 0 0 0 10px color-mix(in srgb, ${t.thumb} calc(${sys.state.hover} * 100%), transparent);
  }
  input:focus-visible::-moz-range-thumb {
    box-shadow: 0 0 0 10px color-mix(in srgb, ${t.thumb} calc(${sys.state.focus} * 100%), transparent);
  }

  input:disabled { cursor: default; pointer-events: none; opacity: ${sys.state.disabledContent}; }

  .dual {
    position: relative;
    inline-size: 100%;
    /* Abs-pos thumbs are out of flow; without this the track collapses to 0. */
    block-size: max(44px, calc(48px + var(--ui-density, 0) * 4px));
  }
  .dual input { position: absolute; inset: 0; pointer-events: none; }
  .dual input::-webkit-slider-thumb { pointer-events: auto; }
  .dual input::-moz-range-thumb { pointer-events: auto; }
  .dual input:last-child::-webkit-slider-runnable-track { background: transparent; }
  .dual input:last-child::-moz-range-track { background: transparent; }
  .dual input:last-child::-moz-range-progress { background: transparent; }

  .bubble {
    position: absolute;
    inset-inline-start: var(--ui-slider-fill, 0%);
    inset-block-start: 0;
    translate: -50% -100%;
    padding: ${sys.space(1)} ${sys.space(2)};
    border-radius: ${sys.radius.full};
    background: ${t.bubbleBg};
    color: ${t.bubbleFg};
    font: ${sys.type.labelMd};
    letter-spacing: ${sys.tracking.labelMd};
    white-space: nowrap;
    pointer-events: none;
  }
`;

define('ui-slider', {
  formAssociated: true,
  props: {
    value: 0, min: 0, max: 100, step: 1,
    range: false, valueStart: 0, valueEnd: 100,
    label: '', disabled: false, showValue: false, name: '',
  },
  styles: [base, styles],
  setup({ value, min, max, step, range, valueStart, valueEnd, label, disabled, showValue, name }, host) {
    const submitted = signal(String(value.peek()));
    effect(() => {
      submitted.set(range() ? `${valueStart()},${valueEnd()}` : String(value()));
    });
    formBind(host, { name, value: submitted, disabled });

    const active = signal(false);
    const activeThumb = signal('end');

    const pct = (n) => {
      const lo = min();
      const span = max() - lo || 1;
      return Math.min(100, Math.max(0, ((n - lo) / span) * 100));
    };
    const fill = computed(() => pct(value()));
    const startPct = computed(() => (range() ? pct(valueStart()) : 0));
    const endPct = computed(() => (range() ? pct(valueEnd()) : fill()));
    const bubblePct = computed(() =>
      range() ? (activeThumb() === 'start' ? startPct() : endPct()) : fill());
    const bubbleText = computed(() =>
      range() ? (activeThumb() === 'start' ? valueStart() : valueEnd()) : value());

    const read = (el) => {
      const n = Number(el.value);
      return Number.isNaN(n) ? min() : n;
    };
    const emit = (type) => {
      host.emit(type, range()
        ? { start: valueStart(), end: valueEnd(), value: `${valueStart()},${valueEnd()}` }
        : { value: value() });
    };
    const onInput = (e) => {
      e.stopPropagation();
      value.set(read(e.target));
      emit('input');
    };
    const onChange = (e) => {
      e.stopPropagation();
      value.set(read(e.target));
      emit('change');
    };
    const onStartInput = (e) => {
      e.stopPropagation();
      valueStart.set(Math.min(read(e.target), valueEnd()));
      activeThumb.set('start');
      emit('input');
    };
    const onEndInput = (e) => {
      e.stopPropagation();
      valueEnd.set(Math.max(read(e.target), valueStart()));
      activeThumb.set('end');
      emit('input');
    };
    const onStartChange = (e) => {
      e.stopPropagation();
      valueStart.set(Math.min(read(e.target), valueEnd()));
      emit('change');
    };
    const onEndChange = (e) => {
      e.stopPropagation();
      valueEnd.set(Math.max(read(e.target), valueStart()));
      emit('change');
    };

    const styleVars = computed(() => ({
      '--ui-slider-fill': fill() + '%',
      '--ui-slider-start': startPct() + '%',
      '--ui-slider-end': endPct() + '%',
    }));

    return html`
      <div class="root" style=${styleVars}>
        ${() => (range()
          ? html`<div class="dual">
              <input part="input" type="range"
                     min=${min} max=${max} step=${step} .value=${valueStart}
                     ?disabled=${disabled}
                     aria-label=${() => (label() ? label() + ' start' : 'Start')}
                     @input=${onStartInput} @change=${onStartChange}
                     @pointerdown=${() => { active.set(true); activeThumb.set('start'); }}
                     @focus=${() => { active.set(true); activeThumb.set('start'); }}
                     @blur=${() => active.set(false)}>
              <input part="input" type="range"
                     min=${min} max=${max} step=${step} .value=${valueEnd}
                     ?disabled=${disabled}
                     aria-label=${() => (label() ? label() + ' end' : 'End')}
                     @input=${onEndInput} @change=${onEndChange}
                     @pointerdown=${() => { active.set(true); activeThumb.set('end'); }}
                     @focus=${() => { active.set(true); activeThumb.set('end'); }}
                     @blur=${() => active.set(false)}>
            </div>`
          : html`<input part="input" type="range"
                     min=${min} max=${max} step=${step} .value=${value}
                     ?disabled=${disabled}
                     aria-label=${() => label() || 'Slider'}
                     @input=${onInput} @change=${onChange}
                     @pointerdown=${() => active.set(true)}
                     @focus=${() => active.set(true)}
                     @blur=${() => active.set(false)}>`)}
        ${presence(() => showValue() && active(), () => html`
            <output class="bubble" style=${() => ({ insetInlineStart: bubblePct() + '%' })}
                    aria-hidden="true">${bubbleText}</output>`, {
          enter: fx.scaleIn,
          exit: fx.scaleOut,
          enterDuration: 'short4',
          exitDuration: 'short2',
        })}
      </div>`;
  },
});

export const tag = 'ui-slider';
export const themeVars = t;
