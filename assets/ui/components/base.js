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
  :host([hidden]) { display: none !important; }
  @media (prefers-reduced-motion: reduce) {
    *, *::before, *::after {
      transition-duration: 0.01ms;
      animation-duration: 0.01ms;
      animation-iteration-count: 1;
    }
  }
  @media (hover: hover) and (pointer: fine) {
    :host, *, *::before, *::after {
      scrollbar-width: thin;
      scrollbar-color: ${sys.scrollbar.thumb} ${sys.scrollbar.track};
    }
    ::-webkit-scrollbar {
      inline-size: ${sys.scrollbar.size};
      block-size: ${sys.scrollbar.size};
    }
    ::-webkit-scrollbar-track {
      background: ${sys.scrollbar.track};
    }
    ::-webkit-scrollbar-thumb {
      background-color: ${sys.scrollbar.thumb};
      border-radius: ${sys.scrollbar.radius};
      border: 2px solid transparent;
      background-clip: content-box;
    }
    ::-webkit-scrollbar-thumb:hover {
      background-color: ${sys.scrollbar.thumbHover};
    }
    ::-webkit-scrollbar-thumb:active {
      background-color: ${sys.scrollbar.thumbActive};
    }
    ::-webkit-scrollbar-corner {
      background: transparent;
    }
    ::-webkit-scrollbar-button {
      display: none;
      inline-size: 0;
      block-size: 0;
    }
  }
`;

/**
 * Custom scrollbar rules for a specific selector.
 * Interpolate into a component's css template.
 */
export const scrollbarOn = (selector) => `
  @media (hover: hover) and (pointer: fine) {
    ${selector} {
      scrollbar-width: thin;
      scrollbar-color: var(--ui-scrollbar-thumb) var(--ui-scrollbar-track);
    }
    ${selector}::-webkit-scrollbar {
      inline-size: var(--ui-scrollbar-size);
      block-size: var(--ui-scrollbar-size);
    }
    ${selector}::-webkit-scrollbar-track {
      background: var(--ui-scrollbar-track);
    }
    ${selector}::-webkit-scrollbar-thumb {
      background-color: var(--ui-scrollbar-thumb);
      border-radius: var(--ui-scrollbar-radius);
      border: 2px solid transparent;
      background-clip: content-box;
    }
    ${selector}::-webkit-scrollbar-thumb:hover {
      background-color: var(--ui-scrollbar-thumb-hover);
    }
    ${selector}::-webkit-scrollbar-thumb:active {
      background-color: var(--ui-scrollbar-thumb-active);
    }
    ${selector}::-webkit-scrollbar-corner {
      background: transparent;
    }
    ${selector}::-webkit-scrollbar-button {
      display: none;
      inline-size: 0;
      block-size: 0;
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

