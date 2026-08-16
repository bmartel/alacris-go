// Shared base for every component's `styles` list. `css` interns by text, so
// this parses exactly once for the whole page no matter how many components
// include it.

import { css } from '@alacris/core';
import { sys } from '../tokens/sys.js';

export const base = css`
  *, *::before, *::after { box-sizing: border-box; }
  :host {
    font-family: ${sys.font.plain};
    font-optical-sizing: auto;
    font-synthesis: none;
    -webkit-tap-highlight-color: transparent;
  }
  :host([hidden]) { display: none; }
  @media (prefers-reduced-motion: reduce) {
    *, *::before, *::after {
      transition-duration: 0.01ms;
      animation-duration: 0.01ms;
      animation-iteration-count: 1;
    }
  }
`;

/**
 * The focus ring rule for a selector, e.g. ${focusRingOn('.control')}.
 * Interpolate into a component's css template.
 */
export const focusRingOn = (selector) => `
  ${selector}:focus-visible {
    outline: var(--ui-focus-ring);
    outline-offset: var(--ui-focus-ring-offset);
  }
`;

/**
 * Hover / focus-visible / pressed opacities for a `.layer` descendant.
 * `host` is the interactive element (`.control`, `.header`, …). Pass
 * `focus` when focus lives on a different node (the host custom element).
 */
export const stateLayerOn = (host, { focus = host } = {}) => `
  ${host}:hover .layer { opacity: var(--ui-state-hover); }
  ${focus}:focus-visible .layer { opacity: var(--ui-state-focus); }
  ${host}:active .layer { opacity: var(--ui-state-pressed); }
`;
