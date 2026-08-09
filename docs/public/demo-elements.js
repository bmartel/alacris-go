// The components the examples on this site render.
//
// They are deliberately small stand-ins: enough to show that the HTML Go
// produced is a real, working custom element, without turning the docs into a
// component library. They import alacris under its real specifier, which the
// site's import map points at the bytes the Go module vendors.

import { define, html, css, signal, computed, each, vars } from 'alacris';

const base = css`
  :host { display: block; font: inherit; color: inherit }
  .card {
    border: 1px solid var(--sl-color-gray-5); border-radius: 10px;
    padding: .75rem .9rem; background: var(--sl-color-black);
  }
  h3 { margin: 0 0 .2rem; font-size: 1rem }
  .muted { color: var(--sl-color-gray-3); font-size: .85rem }
  button {
    font: inherit; padding: .3rem .6rem; border-radius: 7px; cursor: pointer;
    border: 1px solid var(--sl-color-gray-5); background: transparent; color: inherit;
  }
`;

/* ---------------------------------------------------------------- user-card */

define('user-card', {
  props: { name: 'anon', age: 0, tags: [] },
  styles: [base, css`
    .tags { display: flex; gap: .3rem; flex-wrap: wrap; margin-top: .4rem }
    .tag {
      font-size: .72rem; padding: .05rem .4rem; border-radius: 999px;
      background: var(--sl-color-gray-6); border: 1px solid var(--sl-color-gray-5);
    }
  `],
  setup({ name, age, tags }) {
    const stage = computed(() => (age() >= 18 ? 'adult' : 'minor'));
    return html`
      <div class="card">
        <h3><slot name="title">${name}</slot></h3>
        <div class="muted">${age} · ${stage}</div>
        <div class="tags">
          ${() => tags().map((t) => html`<span class="tag">${t}</span>`)}
        </div>
        <slot></slot>
      </div>`;
  },
});

/* ---------------------------------------------------------------- info-card */

define('info-card', {
  props: { tone: 'neutral' },
  styles: [base, css`
    .card { border-left: 3px solid var(--tone, var(--sl-color-gray-5)) }
    :host([tone="warn"]) .card { --tone: #e0a800 }
    :host([tone="ok"])   .card { --tone: #3aa76d }
  `],
  setup: () => html`
    <div class="card">
      <h3><slot name="title"></slot></h3>
      <slot name="body"></slot>
      <slot></slot>
    </div>`,
});

/* ------------------------------------------------------------------- x-demo */

// Shows what actually arrived and what JavaScript type it became. The point of
// the props page is that an attribute string becomes a typed value on the
// other side, and this is the only way to see that happen.
define('x-demo', {
  props: {
    label: '', count: 0, ratio: 0, open: false, shut: false,
    tags: [], origin: {}, lookup: {}, at: '', timeout: 0,
    name: 'the default', kept: '', payload: {},
  },
  styles: [base, css`
    table { border-collapse: collapse; width: 100%; font-size: .82rem }
    td { padding: .18rem .5rem; border-bottom: 1px solid var(--sl-color-gray-6); vertical-align: top }
    td:first-child { color: var(--sl-color-gray-3); white-space: nowrap; width: 1%; font-weight: 600 }
    td:nth-child(2) { color: var(--sl-color-gray-3); white-space: nowrap; width: 1% }
    code { font-size: .95em }
  `],
  setup(props, host) {
    // Only report the props this element was actually given, so each example
    // shows its own point rather than every prop the stand-in declares.
    const present = Object.keys(props).filter(
      (k) => host.getAttribute(k.replace(/[A-Z]/g, (c) => '-' + c.toLowerCase())) !== null,
    );
    const typeOf = (v) => (Array.isArray(v) ? 'array' : v === null ? 'null' : typeof v);

    return html`
      <div class="card">
        <table>
          ${present.map((k) => html`
            <tr>
              <td>${k}</td>
              <td>${() => typeOf(props[k]())}</td>
              <td><code>${() => JSON.stringify(props[k]())}</code></td>
            </tr>`)}
        </table>
        ${present.length === 0 ? html`<div class="muted">No props were set.</div>` : null}
      </div>`;
  },
});

/* ----------------------------------------------------------------- ala-chip */

const chip = vars('chip', { bg: '#ececf2', fg: '#333', radius: '999px' });

define('ala-chip', {
  props: { tone: 'neutral' },
  styles: css`
    :host {
      display: inline-block; padding: .1rem .55rem; font-size: .8rem;
      background: ${chip.bg}; color: ${chip.fg}; border-radius: ${chip.radius};
    }
  `,
  setup: () => html`<slot></slot>`,
});

/* ------------------------------------------------------------ ala-todo-list */

define('ala-todo-list', {
  props: { items: [], filter: 'all', busy: false },
  styles: [base, css`
    ul { list-style: none; margin: 0; padding: 0; display: grid; gap: .3rem }
    li { display: flex; align-items: center; gap: .5rem }
    li[data-done] .text { opacity: .5; text-decoration: line-through }
    .text { flex: 1 }
    :host([busy]) { opacity: .6 }
  `],
  setup({ items, filter }, host) {
    const shown = computed(() => {
      const f = filter();
      return items().filter((t) => f === 'all' || (f === 'done') === !!t.done);
    });
    return html`
      <div class="card">
        <ul ?hidden=${() => shown().length === 0}>
          ${each(
            () => shown(),
            (todo) => html`
              <li ?data-done=${() => todo().done}>
                <input type="checkbox" .checked=${() => todo().done}
                  @change=${() => host.emit('toggle', { id: todo().id })} />
                <span class="text">${() => todo().text}</span>
              </li>`,
            (t) => t.id,
          )}
        </ul>
        <div class="muted" ?hidden=${() => shown().length > 0}><slot name="empty">Nothing to show.</slot></div>
      </div>`;
  },
});

/* -------------------------------------------------------------- ala-counter */

define('ala-counter', {
  props: { start: 0, step: 1 },
  styles: [base, css`
    .row { display: flex; align-items: center; gap: .6rem }
    output { font-variant-numeric: tabular-nums; font-size: 1.3rem; min-width: 3ch; text-align: center }
  `],
  setup({ start, step }, host) {
    const n = signal(start());
    const move = (by) => { n(n() + by); host.emit('change', { value: n() }); };
    return html`
      <div class="row">
        <button @click=${() => move(-step())} aria-label="decrement">−</button>
        <output>${n}</output>
        <button @click=${() => move(step())} aria-label="increment">+</button>
      </div>`;
  },
});
