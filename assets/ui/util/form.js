// formBind — form participation for shadow-DOM controls.
//
// Components that declare `formAssociated: true` get `host.internals` from
// Alacris (the platform's ElementInternals), and formBind drives it: the value
// is reported with `setFormValue`, checkbox semantics submit nothing while
// unchecked, and a form reset restores the value the control had when it was
// bound (assigned to `host.onFormReset` unless the component set its own).
//
// Where ElementInternals is unavailable (older engines, simulated DOMs in
// tests), it falls back to mirroring the value into a hidden light-DOM
// `<input>` on the host — the form sees an ordinary field either way.

import { effect, onCleanup, untrack } from '@alacris/core';

/**
 * Call inside `setup`, passing the prop signals themselves:
 *
 *   formBind(host, { name, value, disabled });          // text-like
 *   formBind(host, { name, value, checked, disabled }); // checkable
 */
export function formBind(host, { name, value, checked, disabled }) {
  const submitted = () => {
    const on = checked ? !!checked() : true;
    const v = value ? value() : 'on';
    return !on ? null : v == null || v === '' ? (checked ? 'on' : String(v ?? '')) : String(v);
  };

  if (host.internals?.setFormValue) {
    // Native form association. `name`/`disabled` are the platform's job here —
    // the browser reads the `name` attribute and the :disabled tree state.
    const initial = untrack(() => ({ value: value?.(), checked: checked?.() }));
    effect(() => {
      host.internals.setFormValue(disabled && disabled() ? null : submitted());
    });
    if (!host.onFormReset) {
      host.onFormReset = () => {
        value?.set(initial.value);
        checked?.set(initial.checked);
      };
    }
    return;
  }

  // Fallback: a hidden light-DOM input mirrors the value into the form.
  let input = null;

  effect(() => {
    const n = name ? name() : '';
    const v = submitted();
    const submit = !!n && !(disabled ? disabled() : false) && v !== null;
    if (submit) {
      if (!input) {
        input = document.createElement('input');
        input.type = 'hidden';
        host.append(input);
      }
      input.name = n;
      input.value = v;
    } else if (input) {
      input.remove();
      input = null;
    }
  });

  onCleanup(() => {
    input?.remove();
    input = null;
  });
}
