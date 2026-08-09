// The example's components.
//
// Everything reactive lives here, in JavaScript, because setup() runs in the
// browser. Go renders the elements and their props; alacris does the rest.
//
// The server owns the todo list. A component never mutates it: it emits an
// event, the Go handler changes the list, and the new list arrives as one prop
// write — which alacris turns into the smallest DOM change that expresses it.

import { define, html, css, signal, computed, each, vars } from 'alacris';

const base = css`
  :host { display: block; font: inherit; color: inherit }
  button {
    font: inherit; padding: .35rem .7rem; border-radius: 8px; cursor: pointer;
    border: 1px solid var(--line, #ddd); background: transparent; color: inherit;
  }
  button:hover { border-color: var(--accent, #ffb300) }
`;

/**
 * A number the user can nudge, kept in the browser.
 *
 * Nothing here touches the server: local state that no one else needs is
 * exactly what a signal is for.
 *
 * @prop {integer} start  the value it opens on
 * @prop {integer} step   how much each press moves it
 * @fires change {value: integer} - the number changed
 * @goname Counter
 */
define('ala-counter', {
  props: { start: 0, step: 1 },
  styles: [base, css`
    .row { display: flex; align-items: center; gap: .75rem }
    output { font-variant-numeric: tabular-nums; font-size: 1.5rem; min-width: 3ch; text-align: center }
  `],
  setup({ start, step }, host) {
    const n = signal(start());
    const move = (by) => {
      n(n() + by);
      host.emit('change', { value: n() });
    };
    return html`
      <div class="row">
        <button @click=${() => move(-step())} aria-label="decrement">−</button>
        <output>${n}</output>
        <button @click=${() => move(step())} aria-label="increment">+</button>
      </div>`;
  },
});

const chip = vars('chip', { bg: '#eee', fg: '#111' });

/**
 * A small label. Themed entirely through custom properties, so the page can
 * restyle it without knowing anything about its shadow root.
 *
 * @prop {string} tone
 * @slot - the label text
 * @cssprop [--chip-bg=#eee] - the background
 * @cssprop [--chip-fg=#111] - the text colour
 * @goname Chip
 */
define('ala-chip', {
  props: { tone: 'neutral' },
  styles: css`
    :host {
      display: inline-block; padding: .1rem .5rem; border-radius: 999px;
      font-size: .78rem; background: ${chip.bg}; color: ${chip.fg};
    }
  `,
  setup: () => html`<slot></slot>`,
});

/**
 * The todo list.
 *
 * The list itself is a prop, so the server is the only thing that decides what
 * is in it. Every interaction leaves as an event and comes back as a new
 * value for `items`.
 *
 * @prop {go:[]todo.Item} items    the list, as rendered by the server
 * @prop {string}         filter   "all", "active" or "done"
 * @prop {boolean}        busy     true while the server is working
 * @goimport todo github.com/bmartel/alacris-go/examples/todo/model
 * @fires add    {text: string}     - the user entered a new todo
 * @fires toggle {id: integer}      - the user flipped one done or undone
 * @fires remove {id: integer}      - the user deleted one
 * @fires filter {filter: string}   - the user picked a different filter
 * @slot empty - shown instead of the list when there is nothing to show
 * @goname TodoList
 */
define('ala-todo-list', {
  props: { items: [], filter: 'all', busy: false },
  styles: [base, css`
    form { display: flex; gap: .5rem; margin-bottom: 1rem }
    input { flex: 1; font: inherit; padding: .4rem .6rem; border-radius: 8px;
            border: 1px solid var(--line, #ddd); background: transparent; color: inherit }
    ul { list-style: none; margin: 0; padding: 0; display: grid; gap: .35rem }
    li { display: flex; align-items: center; gap: .6rem }
    li[data-done] .text { opacity: .5; text-decoration: line-through }
    .text { flex: 1 }
    .filters { display: flex; gap: .4rem; margin-top: 1rem }
    .filters button[aria-pressed="true"] { border-color: var(--accent, #ffb300) }
    :host([busy]) { opacity: .6 }
  `],
  setup({ items, filter, busy }, host) {
    const draft = signal('');

    // A computed, not an effect: this derives a value, it does not cause one.
    const shown = computed(() => {
      const f = filter();
      return items().filter((t) => f === 'all' || (f === 'done') === !!t.done);
    });

    const submit = (e) => {
      e.preventDefault();
      const text = draft().trim();
      if (!text) return;
      host.emit('add', { text });
      draft('');
    };

    return html`
      <form @submit=${submit}>
        <input
          .value=${draft}
          @input=${(e) => draft(e.target.value)}
          placeholder="What needs doing?"
          aria-label="New todo" />
        <button ?disabled=${busy}>Add</button>
      </form>

      <!--
        The list is not inside a conditional. Putting each() in one would
        rebuild the whole ul every time the array changed, because the thunk
        returns a fresh template result each run — which is exactly the work
        each() exists to avoid. Emptiness toggles an attribute instead, so the
        rows are created once and updated in place from then on.
      -->
      <ul ?hidden=${() => shown().length === 0}>
        ${each(
          () => shown(),
          (todo) => html`
            <li ?data-done=${() => todo().done}>
              <input
                type="checkbox"
                .checked=${() => todo().done}
                @change=${() => host.emit('toggle', { id: todo().id })}
                aria-label=${() => todo().text} />
              <span class="text">${() => todo().text}</span>
              <button @click=${() => host.emit('remove', { id: todo().id })}>×</button>
            </li>`,
          (t) => t.id,
        )}
      </ul>
      <p ?hidden=${() => shown().length > 0}><slot name="empty">Nothing here.</slot></p>

      <div class="filters">
        ${['all', 'active', 'done'].map((f) => html`
          <button
            aria-pressed=${() => String(filter() === f)}
            @click=${() => host.emit('filter', { filter: f })}>${f}</button>`)}
      </div>`;
  },
});
