// <ui-stack> — a flexbox layout primitive.
//
//   <ui-stack direction="row" gap="4" align="center">…</ui-stack>
//
// @prop  {string}         direction='column' — row | column | row-reverse | column-reverse
// @prop  {number|string}  gap=2   — a spacing step (maps to the --ui-space-N
//                                   token: 1,2,3,4,5,6,7,8,10,12,16,20,24) or
//                                   any raw CSS length ('1.5rem', '12px')
// @prop  {string}         align=''   — align-items value ('' leaves the default)
// @prop  {string}         justify='' — justify-content value
// @prop  {boolean}        wrap=false — flex-wrap: wrap
// @slot  (default)
// @vars  (none — spacing flows from the --ui-space-* system tokens)

import { define, html, css, effect } from '@alacris/core';
import { sys } from '../tokens/sys.js';
import { base } from './base.js';

define('ui-stack', {
  // gap defaults to a string so raw CSS lengths survive attribute coercion;
  // numeric property writes are normalized below.
  props: { direction: 'column', gap: '2', align: '', justify: '', wrap: false },
  styles: [base, css`:host { display: flex; }`],
  setup({ direction, gap, align, justify, wrap }, host) {
    effect(() => { host.style.flexDirection = direction(); });
    effect(() => {
      const g = String(gap() ?? '').trim();
      host.style.gap = /^\d+$/.test(g) ? sys.space(g) : g;
    });
    effect(() => { host.style.alignItems = align() || ''; });
    effect(() => { host.style.justifyContent = justify() || ''; });
    effect(() => { host.style.flexWrap = wrap() ? 'wrap' : ''; });
    return html`<slot></slot>`;
  },
});

export const tag = 'ui-stack';
