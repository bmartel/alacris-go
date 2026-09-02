// Alacris UI — public surface (`@alacris/ui` on npm).
//
// Importing this module registers every component and re-exports the theme,
// motion, token, and utility APIs. Tree-shaking-friendly alternative: import
// only the component files and helpers you use — each component module is
// self-contained. Do not bundle a second copy of `@alacris/core`; these modules
// import it by specifier so the app and the design system share one graph.

// Theme
export {
  createTheme, applyTheme, themeCss, activeTheme, loadThemeFonts,
  scheme, schemePreference, setScheme, toggleScheme,
} from './theme/index.js';

// Tokens
export { sys, typeRule } from './tokens/sys.js';
export {
  FONT_PRESETS, FONT_STACKS, DEFAULT_FONT_PRESET, TYPE_ROLES,
  resolveTypography, googleFontsHref,
} from './tokens/typography.js';

// Motion
export { animate, releaseFill, settled, duration, easing, fx, prefersReducedMotion } from './motion/animate.js';
export { presence } from './motion/presence.js';
export { withFlip } from './motion/flip.js';
export { ripple } from './motion/ripple.js';
export { createSwipeTracker, calculateVelocity, rubberBand } from './motion/gesture.js';

// Utilities
export { position, autoUpdate } from './util/position.js';
export { focusTrap, focusables, scrollLock } from './util/focus.js';
export { rovingTabindex } from './util/keys.js';
export { formBind } from './util/form.js';
export { registerIcons, iconPath, iconNames } from './util/icons.js';
export {
  processTable, filterRows, sortRows, groupRows, paginate,
  aggregateRow, toCsv, downloadText, inferColumns, visibleColumns,
} from './util/table.js';
export { base, scrollbarOn, focusRingOn, stateLayerOn } from './components/base.js';

// Components — importing registers the custom element.
import './components/ui-icon.js';
import './components/ui-text.js';

// Actions
import './components/ui-button.js';
import './components/ui-icon-button.js';
import './components/ui-fab.js';
import './components/ui-fab-menu.js';
import './components/ui-button-group.js';
import './components/ui-split-button.js';
import './components/ui-toggle-button.js';
import './components/ui-toggle-group.js';

// Inputs
import './components/ui-checkbox.js';
import './components/ui-radio.js';
import './components/ui-radio-group.js';
import './components/ui-switch.js';
import './components/ui-slider.js';
import './components/ui-rating.js';
import './components/ui-text-field.js';
import './components/ui-select.js';
import './components/ui-option.js';
import './components/ui-autocomplete.js';
import './components/ui-chip.js';
import './components/ui-chip-set.js';
import './components/ui-date-picker.js';
import './components/ui-time-picker.js';
import './components/ui-search.js';

// Data display
import './components/ui-avatar.js';
import './components/ui-badge.js';
import './components/ui-divider.js';
import './components/ui-list.js';
import './components/ui-list-item.js';
import './components/ui-swipe-row.js';
import './components/ui-table.js';
import './components/ui-table-toolbar.js';
import './components/ui-table-footer.js';
import './components/ui-tooltip.js';
import './components/ui-carousel.js';
import './components/ui-carousel-item.js';

// Feedback
import './components/ui-alert.js';
import './components/ui-progress.js';
import './components/ui-spinner.js';
import './components/ui-loading-indicator.js';
import './components/ui-skeleton.js';
import './components/ui-snackbar.js';
import './components/ui-backdrop.js';
import './components/ui-sheet.js';
import './components/ui-side-sheet.js';

export { showSnackbar } from './components/ui-snackbar.js';

// Surfaces & navigation
import './components/ui-card.js';
import './components/ui-dialog.js';
import './components/ui-tabs.js';
import './components/ui-tab.js';
import './components/ui-tab-panel.js';
import './components/ui-menu.js';
import './components/ui-menu-item.js';
import './components/ui-drawer.js';
import './components/ui-app-bar.js';
import './components/ui-bottom-app-bar.js';
import './components/ui-toolbar.js';
import './components/ui-accordion.js';
import './components/ui-accordion-item.js';
import './components/ui-breadcrumbs.js';
import './components/ui-pagination.js';
import './components/ui-stepper.js';
import './components/ui-step.js';
import './components/ui-bottom-nav.js';
import './components/ui-nav-item.js';
import './components/ui-nav-rail.js';

// Layout
import './components/ui-stack.js';
import './components/ui-container.js';
import './components/ui-surface.js';
