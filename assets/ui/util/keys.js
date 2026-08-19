// rovingTabindex — arrow-key navigation for composite widgets
// (tabs, menus, radio groups, chip sets), per the ARIA Authoring Practices:
// one Tab stop for the whole widget, arrows move the active item.

/**
 * rovingTabindex(container, {
 *   selector,                  // matches the items (live-queried each time)
 *   items,                     // () => Element[] — alternative to selector
 *                              // (nested/flattened slots, items outside container)
 *   listenOn,                  // event target; default `container`. Pass
 *                              // `document` when items are not descendants
 *                              // (e.g. projected through a nested slot).
 *   orientation = 'horizontal',// 'horizontal' | 'vertical' | 'both'
 *   wrap = true,
 *   skip,                      // (el) => true to pass over (disabled items)
 *   onMove,                    // (el, index) called after focus moves
 * })
 *
 * Returns { focus(el|index), activate(el|index), refresh(), destroy() }.
 * `refresh()` re-applies tabindexes after items change; call it from a thunk
 * or effect when the item list is dynamic. `activate` sets the tab stop
 * without moving focus.
 */
export function rovingTabindex(container, opts = {}) {
  const { selector = '[role]', items: getItems, listenOn, orientation = 'horizontal', wrap = true, skip, onMove } = opts;

  const items = () => (getItems ? [...getItems()] : [...container.querySelectorAll(selector)])
    .filter((el) => !skip?.(el));

  const setActive = (list, el) => {
    for (const it of list) it.tabIndex = it === el ? 0 : -1;
  };

  const focusedIndex = (list, e) => {
    const path = e?.composedPath?.() || [];
    const fromPath = list.findIndex((el) => path.includes(el));
    if (fromPath >= 0) return fromPath;
    const ae = document.activeElement;
    const fromDoc = list.indexOf(ae);
    if (fromDoc >= 0) return fromDoc;
    return list.findIndex((el) => el.contains?.(ae));
  };

  const move = (delta, edge, e) => {
    const list = items();
    if (!list.length) return;
    let i = edge !== undefined ? edge : focusedIndex(list, e);
    if (edge === undefined) {
      if (i < 0) i = 0;
      else {
        i += delta;
        if (wrap) i = (i + list.length) % list.length;
        else i = Math.max(0, Math.min(list.length - 1, i));
      }
    }
    const el = list[i];
    setActive(list, el);
    el.focus();
    onMove?.(el, i);
  };

  const horizontal = orientation !== 'vertical';
  const vertical = orientation !== 'horizontal';

  const onKeydown = (e) => {
    if (e.defaultPrevented) return;
    const list = items();
    if (!list.length) return;
    const path = e.composedPath();
    if (!list.some((el) => path.includes(el)) && !path.includes(container)) return;
    switch (e.key) {
      case 'ArrowRight': if (horizontal) { e.preventDefault(); move(1, undefined, e); } break;
      case 'ArrowLeft': if (horizontal) { e.preventDefault(); move(-1, undefined, e); } break;
      case 'ArrowDown': if (vertical) { e.preventDefault(); move(1, undefined, e); } break;
      case 'ArrowUp': if (vertical) { e.preventDefault(); move(-1, undefined, e); } break;
      case 'Home': e.preventDefault(); move(0, 0, e); break;
      case 'End': e.preventDefault(); move(0, items().length - 1, e); break;
    }
  };

  const refresh = () => {
    const list = items();
    if (!list.length) return;
    const current = list.find((el) => el.tabIndex === 0) || list[0];
    setActive(list, current);
  };

  const target = listenOn || container;
  target.addEventListener('keydown', onKeydown);
  refresh();

  const activate = (which) => {
    const list = items();
    const el = typeof which === 'number' ? list[which] : which;
    if (!el) return null;
    setActive(list, el);
    return el;
  };

  return {
    focus(which) {
      activate(which)?.focus();
    },
    activate,
    refresh,
    destroy() {
      target.removeEventListener('keydown', onKeydown);
    },
  };
}

// escapeLayer — claim Escape for the innermost open layer.
//
// `ui-dialog` listens for Escape in the capture phase at the document, so that
// the key works wherever focus happens to be. That is right for a dialog and
// wrong for anything transient opened inside one: a select's panel, a menu, a
// date picker. Those handle Escape too, but the dialog has already seen it by
// then, so one press closes both — and choosing a format in a dialog looks
// like the dialog is broken rather than like an ordering problem nobody can
// see.
//
// Capture descends window → document → …, so a layer claims the key one step
// earlier than the dialog and stops it there. Nothing below ever runs.

/**
 * escapeLayer(onEscape)
 *
 * Call while a transient layer is open; call the returned function when it
 * closes. Only registers a listener while it is held, so a page with nothing
 * open behaves exactly as before.
 */
export function escapeLayer(onEscape) {
  const onKeydown = (e) => {
    if (e.key !== 'Escape') return;
    e.preventDefault();
    e.stopPropagation();
    onEscape(e);
  };
  window.addEventListener('keydown', onKeydown, true);
  return () => window.removeEventListener('keydown', onKeydown, true);
}
