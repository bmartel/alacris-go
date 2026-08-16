// <ui-text> — typography, the type scale as an element.
//
//   <ui-text variant="headline-md" as="h2">Section</ui-text>
//
// @prop  {string} variant='body-md' — any type role: display|headline|title|body|label × lg|md|sm
// @prop  {string} color=''          — a color role name ('primary', 'onSurfaceVariant', …)
// @slot  (default)
// @vars  (uses the --ui-type-* system tokens directly)
//
// Semantics: <ui-text> is styling only. Keep real headings in your markup
// (`<h2><ui-text variant="headline-md">…</ui-text></h2>`) or set an ARIA role
// on the host when the document outline needs one.

import { define, html, css, effect } from '@alacris/core';
import { base } from './base.js';

const kebab = (s) => s.replace(/[A-Z]/g, (c) => '-' + c.toLowerCase());

define('ui-text', {
  props: { variant: 'body-md', color: '' },
  styles: [base, css`:host { display: block; margin: 0; }`],
  setup({ variant, color }, host) {
    effect(() => {
      const v = kebab(variant());
      host.style.font = `var(--ui-type-${v})`;
      host.style.letterSpacing = `var(--ui-type-${v}-tracking)`;
    });
    effect(() => {
      if (color()) host.style.color = `var(--ui-color-${kebab(color())})`;
      else host.style.removeProperty('color');
    });
    return html`<slot></slot>`;
  },
});

export const tag = 'ui-text';
