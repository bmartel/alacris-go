// src/signal.js
var CLEAN = 0;
var CHECK = 1;
var DIRTY = 2;
var EMPTY = [];
var obs = null;
var owner = null;
var runId = 0;
var batching = 0;
var flushing = 0;
var queue = [];
var node = (fn, v, eff, eq) => ({
  fn,
  v,
  eff,
  eq,
  st: fn ? DIRTY : CLEAN,
  srcs: null,
  slots: null,
  // sources, and our index inside each source's obsv
  obsv: null,
  oslot: null,
  // observers, and their index inside our srcs
  lo: null,
  lr: -1,
  // last observer / last run that linked us
  cl: null,
  own: null,
  // cleanups, owned children
  d: 0
});
function link(o, s) {
  if (s.lo === o && s.lr === runId) return;
  s.lo = o;
  s.lr = runId;
  if (!o.srcs) {
    o.srcs = [];
    o.slots = [];
  }
  if (!s.obsv) {
    s.obsv = [];
    s.oslot = [];
  }
  const k = o.srcs.push(s) - 1;
  o.slots.push(s.obsv.push(o) - 1);
  s.oslot.push(k);
}
function unlink(o) {
  const { srcs, slots } = o;
  if (!srcs) return;
  for (let k = 0; k < srcs.length; k++) {
    const s = srcs[k], i = slots[k];
    const last = s.obsv.pop(), ls = s.oslot.pop();
    if (i < s.obsv.length) {
      s.obsv[i] = last;
      s.oslot[i] = ls;
      last.slots[ls] = i;
    }
  }
  srcs.length = slots.length = 0;
}
function notify(n, st) {
  if (n.st >= st) return;
  const prev = n.st;
  n.st = st;
  if (prev !== CLEAN) return;
  if (n.eff) {
    queue.push(n);
  } else {
    const o = n.obsv;
    if (o) for (let i = 0; i < o.length; i++) notify(o[i], CHECK);
  }
}
function cleanup(n) {
  const own = n.own, cl = n.cl;
  if (own) {
    n.own = null;
    for (let i = 0; i < own.length; i++) destroy(own[i]);
  }
  if (cl) {
    n.cl = null;
    for (let i = 0; i < cl.length; i++) cl[i]();
  }
}
function run(n) {
  cleanup(n);
  unlink(n);
  const po = obs, pw = owner;
  obs = owner = n;
  n.st = CLEAN;
  runId++;
  try {
    const v = n.fn(n.v);
    if (n.eff) {
      if (typeof v === "function") (n.cl || (n.cl = [])).push(v);
    } else if (n.eq ? !n.eq(n.v, v) : n.v !== v) {
      n.v = v;
      const o = n.obsv;
      if (o) for (let i = 0; i < o.length; i++) notify(o[i], DIRTY);
    }
  } finally {
    obs = po;
    owner = pw;
  }
}
function update(n) {
  if (n.st === CHECK) {
    const srcs = n.srcs || EMPTY;
    for (let i = 0; i < srcs.length; i++) {
      if (srcs[i].fn) update(srcs[i]);
      if (n.st === DIRTY) break;
    }
    if (n.st === CHECK) n.st = CLEAN;
  }
  if (n.st === DIRTY) run(n);
}
function destroy(n) {
  if (n.d) return;
  n.d = 1;
  cleanup(n);
  unlink(n);
  n.st = CLEAN;
}
function read(n) {
  if (n.fn && !n.d) update(n);
  if (obs) link(obs, n);
  return n.v;
}
function write(s, v) {
  if (s.eq ? s.eq(s.v, v) : s.v === v) return v;
  s.v = v;
  const o = s.obsv;
  if (o) for (let i = 0; i < o.length; i++) notify(o[i], DIRTY);
  if (!batching) flush();
  return v;
}
function flush() {
  if (flushing) return;
  flushing = 1;
  let err, thrown = 0;
  try {
    for (let i = 0; i < queue.length; i++) {
      const n = queue[i];
      if (!n.d && n.st) try {
        update(n);
      } catch (e) {
        if (!thrown) {
          thrown = 1;
          err = e;
        }
      }
    }
  } finally {
    queue.length = 0;
    flushing = 0;
  }
  if (thrown) throw err;
}
function signal(value, eq) {
  const s = node(null, value, 0, eq);
  function sig(v) {
    return arguments.length ? write(s, v) : read(s);
  }
  sig.set = (v) => write(s, v);
  sig.peek = () => s.v;
  sig.update = (f) => write(s, f(s.v));
  sig.node = s;
  return sig;
}
function computed(fn, eq) {
  const c = node(fn, void 0, 0, eq);
  const get = () => read(c);
  get.peek = () => {
    update(c);
    return c.v;
  };
  get.node = c;
  return get;
}
function effect(fn) {
  return dispose.bind(null, effectNode(fn));
}
function effectNode(fn) {
  const n = node(fn, void 0, 1);
  if (owner) (owner.own || (owner.own = [])).push(n);
  run(n);
  return n;
}
function rerun(n, fn) {
  if (n.d) return null;
  n.fn = fn;
  run(n);
  return n;
}
function dispose(n) {
  destroy(n);
}
function batch(fn) {
  batching++;
  try {
    return fn();
  } finally {
    if (!--batching) flush();
  }
}
function tracking() {
  return obs !== null;
}
function untrack(fn) {
  const p = obs;
  obs = null;
  try {
    return fn();
  } finally {
    obs = p;
  }
}
function root(fn) {
  const n = node(null, void 0, 0);
  const po = obs, pw = owner;
  obs = null;
  owner = n;
  try {
    fn();
  } finally {
    obs = po;
    owner = pw;
  }
  return () => cleanup(n);
}
function onCleanup(fn) {
  if (owner) (owner.cl || (owner.cl = [])).push(fn);
  return fn;
}

// src/style.js
var cache = /* @__PURE__ */ new Map();
var roots = /* @__PURE__ */ new Set();
var globals = [];
var Sheet = class {
  constructor(text) {
    this.text = text;
    this._s = void 0;
  }
  toString() {
    return this.text;
  }
  get sheet() {
    if (this._s === void 0) {
      this._s = null;
      try {
        const s = new CSSStyleSheet();
        s.replaceSync(this.text);
        this._s = s;
      } catch {
      }
    }
    return this._s;
  }
  /**
   * Rewrite this stylesheet in place. Every root that adopted it updates at
   * once, with no re-adoption and no new rules — the cheapest way to reskin a
   * whole page.
   */
  replace(text) {
    this.text = text;
    if (this.sheet) this.sheet.replaceSync(text);
    else for (const el of document.querySelectorAll("style[data-alacris]")) {
      if (el.__t === this) el.textContent = text;
    }
    return this;
  }
};
var intern = (text) => {
  let s = cache.get(text);
  if (!s) cache.set(text, s = new Sheet(text));
  return s;
};
function css(strings, ...values) {
  let text = strings[0];
  for (let i = 0; i < values.length; i++) {
    const v = values[i];
    text += (v == null || v === false ? "" : v) + strings[i + 1];
  }
  return intern(text);
}
var asSheet = (v) => typeof v === "string" ? intern(v) : v;
function flatten(v, out) {
  if (v == null || v === false) return out;
  if (Array.isArray(v)) {
    for (let i = 0; i < v.length; i++) flatten(v[i], out);
    return out;
  }
  out.push(asSheet(v));
  return out;
}
function applyStyles(target, styles) {
  const list = flatten(styles, []);
  if (!list.length) return;
  const adopt = [];
  for (let i = 0; i < list.length; i++) {
    const item = list[i];
    const built = item instanceof Sheet ? item.sheet : item;
    if (built && target.adoptedStyleSheets) {
      if (target.adoptedStyleSheets.indexOf(built) < 0 && adopt.indexOf(built) < 0) adopt.push(built);
    } else {
      const el = document.createElement("style");
      el.setAttribute("data-alacris", "");
      el.__t = item;
      el.textContent = String(item);
      (target.head || target).append(el);
    }
  }
  if (adopt.length) target.adoptedStyleSheets = [...target.adoptedStyleSheets, ...adopt];
}
function trackRoot(target) {
  if (roots.has(target)) return;
  roots.add(target);
  if (globals.length) applyStyles(target, globals);
}
function untrackRoot(target) {
  roots.delete(target);
}
function adoptGlobal(...styles) {
  const added = flatten(styles, []);
  for (let i = 0; i < added.length; i++) globals.push(added[i]);
  for (const target of roots) applyStyles(target, added);
  return () => {
    const built = added.map((s) => s instanceof Sheet ? s.sheet : s);
    for (let i = 0; i < added.length; i++) {
      const at2 = globals.indexOf(added[i]);
      if (at2 > -1) globals.splice(at2, 1);
    }
    for (const target of roots) {
      if (target.adoptedStyleSheets)
        target.adoptedStyleSheets = target.adoptedStyleSheets.filter((s) => built.indexOf(s) < 0);
      target.querySelectorAll?.("style[data-alacris]").forEach((el) => {
        if (added.indexOf(el.__t) > -1) el.remove();
      });
    }
  };
}
var kebab = (s) => s.replace(/[A-Z]/g, (c) => "-" + c.toLowerCase());
function vars(prefix, defaults) {
  const t = {};
  const names = [];
  for (const key in defaults) {
    const name = "--" + prefix + "-" + kebab(key);
    names.push(name);
    t[key] = "var(" + name + ", " + defaults[key] + ")";
  }
  Object.defineProperty(t, "names", { value: names });
  Object.defineProperty(t, "prefix", { value: prefix });
  return t;
}

// src/html.js
var live = (n, fn) => n && rerun(n, fn) || effectNode(fn);
var kill = (n) => (n && dispose(n), null);
function bind(c, v) {
  if (typeof v === "function") c.d = live(c.d, () => c.set(v()));
  else {
    c.d = kill(c.d);
    c.set(v);
  }
}
var tt;
var trust = (s) => {
  if (tt === void 0) {
    try {
      tt = globalThis.trustedTypes ? trustedTypes.createPolicy("alacris", { createHTML: (x) => x }) : null;
    } catch {
      tt = null;
    }
  }
  return tt ? tt.createHTML(s) : s;
};
var cache2 = /* @__PURE__ */ new WeakMap();
var NAME = /([^\s"'=<>/]+)=$/;
var QNAME = /([^\s"'=<>/]+)=["']([^"'<>]*)$/;
var doc = typeof document !== "undefined" ? document : null;
var comment = () => doc.createComment("");
var ATTR = 1;
var SOLE = 2;
var DELEGATED = /* @__PURE__ */ new Set([
  "click",
  "dblclick",
  "input",
  "change",
  "submit",
  "keydown",
  "keyup",
  "keypress",
  "pointerdown",
  "pointerup",
  "mousedown",
  "mouseup",
  "touchstart",
  "touchend"
]);
function delegate(root2, type) {
  const wired = root2.$$w || (root2.$$w = /* @__PURE__ */ new Set());
  if (wired.has(type)) return;
  wired.add(type);
  const key = "$$" + type;
  root2.addEventListener(type, (e) => {
    const path = e.composedPath();
    const ri = path.indexOf(root2);
    if (ri < 0) return;
    let lo = 0;
    for (let i = 0; i < ri; i++) if (path[i].nodeType === 11) lo = i + 1;
    for (let i = lo; i < ri; i++) {
      const h = path[i][key];
      if (h) {
        h.call(path[i], e);
        if (e.cancelBubble) return;
      }
    }
  });
}
var Tpl = class {
  constructor(s, v, ns) {
    this.s = s;
    this.v = v;
    this.ns = ns;
    this.k = void 0;
  }
};
var html = (s, ...v) => new Tpl(s, v, 0);
var svg = (s, ...v) => new Tpl(s, v, 1);
var keyed = (k, t) => (t.k = k, t);
function compile(strings, ns) {
  let out = "", tag = "", inTag = 0, q = 0, sup = 0, acc = "", cur = null, off = 0, com = 0;
  const parts = [];
  const n = strings.length;
  for (let i = 0; i < n; i++) {
    const s = strings[i];
    for (let j = 0; j < s.length; j++) {
      const c = s[j];
      if (sup) {
        if (c === q) {
          cur.s.push(acc);
          acc = "";
          cur = null;
          sup = 0;
          q = 0;
        } else acc += c;
      } else if (q) {
        out += c;
        tag += c;
        if (c === q) q = 0;
      } else if (inTag) {
        out += c;
        tag += c;
        if (c === '"' || c === "'") q = c;
        else if (c === ">") inTag = 0;
      } else if (com) {
        out += c;
        if (c === ">" && out.endsWith("-->")) com = 0;
      } else if (c === "<" && s.startsWith("!--", j + 1)) {
        out += "<!--";
        j += 3;
        com = 1;
      } else {
        out += c;
        if (c === "<") {
          inTag = 1;
          tag = "";
        }
      }
    }
    if (i === n - 1) break;
    const id = parts.length;
    if (com) {
      off++;
      continue;
    }
    if (sup) {
      cur.s.push(acc);
      acc = "";
      cur.h++;
    } else if (inTag && q) {
      const m = QNAME.exec(tag);
      if (!m) throw new SyntaxError("alacris: unsupported binding near `" + tag.slice(-24) + "`");
      const cut = m[1].length + 2 + m[2].length;
      out = out.slice(0, -cut);
      tag = tag.slice(0, -cut);
      cur = { t: 1, n: m[1], s: [m[2]], h: 1, o: off, i: 0 };
      parts.push(cur);
      out += "j:" + id + " ";
      tag += "j:" + id + " ";
      sup = 1;
      acc = "";
    } else if (inTag) {
      const m = NAME.exec(tag);
      if (!m) throw new SyntaxError("alacris: unsupported binding near `" + tag.slice(-24) + "`");
      const cut = m[1].length + 1;
      out = out.slice(0, -cut);
      tag = tag.slice(0, -cut);
      parts.push({ t: 1, n: m[1], s: null, h: 1, o: off, i: 0 });
      out += "j:" + id + " ";
      tag += "j:" + id + " ";
    } else {
      parts.push({ t: 0, n: "", s: null, h: 1, o: off, i: 0 });
      out += "<!--j:" + id + "--><!---->";
    }
    off++;
  }
  const el = doc.createElement("template");
  el.innerHTML = trust(ns ? "<svg>" + out + "</svg>" : out);
  if (ns) {
    const root2 = el.content.firstChild;
    el.content.replaceChildren(...root2.childNodes);
  }
  const targets = [];
  const drop = [];
  const w = doc.createTreeWalker(el.content, 129);
  let node2;
  while (node2 = w.nextNode()) {
    if (node2.nodeType === 8) {
      const d = node2.data;
      if (d.charCodeAt(0) === 106 && d.charCodeAt(1) === 58) {
        const id = +d.slice(2);
        const p = node2.parentNode;
        if (p.nodeType === 1 && p.childNodes.length === 2 && p.firstChild === node2) {
          parts[id].t = SOLE;
          targets.push(id, p);
          drop.push(node2, node2.nextSibling);
        } else {
          node2.data = "";
          targets.push(id, node2);
        }
      }
    } else {
      const a = node2.attributes;
      for (let k = a.length - 1; k >= 0; k--) {
        const an = a[k].name;
        if (an.charCodeAt(0) === 106 && an.charCodeAt(1) === 58) {
          targets.push(+an.slice(2), node2);
          node2.removeAttribute(an);
        }
      }
    }
  }
  for (let i = 0; i < drop.length; i++) drop[i].remove();
  for (let i = 0; i < targets.length; i += 2) {
    const pth = [];
    for (let n2 = targets[i + 1]; n2 !== el.content; n2 = n2.parentNode) {
      let k = 0, c = n2.parentNode.firstChild;
      while (c !== n2) {
        k++;
        c = c.nextSibling;
      }
      pth.push(k);
    }
    pth.reverse();
    parts[targets[i]].pth = pth;
  }
  for (let i = 0; i < parts.length; i++)
    if (parts[i].pth === void 0)
      throw new SyntaxError("alacris: a binding inside raw text (<script>/<style>/<textarea>/<title>) or a nested <template> is not supported \u2014 bind .value / .textContent instead");
  for (let i = 0; i < parts.length; i++) if (parts[i].t === ATTR) prepare(parts[i]);
  return { e: el, p: parts };
}
function at(n, pth) {
  for (let i = 0; i < pth.length; i++) {
    let k = pth[i];
    n = n.firstChild;
    while (k--) n = n.nextSibling;
  }
  return n;
}
function tplOf(t) {
  let c = cache2.get(t.s);
  if (!c) cache2.set(t.s, c = compile(t.s, t.ns));
  return c;
}
function classText(v) {
  if (typeof v === "string") return v;
  if (!v) return "";
  let out = "";
  if (Array.isArray(v)) {
    for (let i = 0; i < v.length; i++) {
      const s = classText(v[i]);
      if (s) out += (out && " ") + s;
    }
    return out;
  }
  for (const k in v) if (v[k]) out += (out && " ") + k;
  return out;
}
var setStyleProp = (style, k, x) => k.charCodeAt(0) === 45 ? style.setProperty(k, x == null || x === false ? "" : x) : style[k] = x == null || x === false ? "" : x;
function setStyle(el, v) {
  if (v && typeof v === "object") {
    const style = el.style;
    const prev = el.$$y;
    if (prev) {
      for (let i = 0; i < prev.length; i++) if (!(prev[i] in v)) setStyleProp(style, prev[i], "");
    }
    const keys = [];
    for (const k in v) {
      keys.push(k);
      setStyleProp(style, k, v[k]);
    }
    el.$$y = keys;
    return;
  }
  el.$$y = null;
  v == null || v === false ? el.removeAttribute("style") : el.setAttribute("style", v);
}
function prepare(p) {
  const n = p.n, c = n.charCodeAt(0);
  if (c === 64) {
    p.raw = 1;
    const bits = n.slice(1).split(".");
    const type = bits[0], mods = bits.slice(1);
    if (!mods.length && DELEGATED.has(type)) {
      p.dl = type;
      const key = "$$" + type;
      p.set = (el, v) => {
        el[key] = v;
      };
      return p;
    }
    p.wrap = 1;
    const stop = mods.indexOf("stop") > -1, prevent = mods.indexOf("prevent") > -1;
    const opts = { capture: mods.indexOf("capture") > -1, once: mods.indexOf("once") > -1, passive: mods.indexOf("passive") > -1 };
    p.mk = (el) => {
      const w = { h: null };
      el.addEventListener(type, (e) => {
        if (stop) e.stopPropagation();
        if (prevent) e.preventDefault();
        if (w.h) w.h.call(el, e);
      }, opts);
      return w;
    };
    p.set = (w, v) => {
      w.h = v;
    };
    return p;
  }
  if (n === "class") {
    p.set = (el, v) => {
      const t = classText(v);
      t ? el.className = t : el.removeAttribute("class");
    };
    return p;
  }
  if (n === "style") {
    p.set = setStyle;
    return p;
  }
  if (c === 46) {
    const k2 = n.slice(1);
    p.set = (el, v) => {
      el[k2] = v;
    };
    return p;
  }
  if (c === 63) {
    const k2 = n.slice(1);
    p.set = (el, v) => el.toggleAttribute(k2, !!v);
    return p;
  }
  if (n === "ref") {
    p.raw = 1;
    p.set = (el, v) => {
      typeof v === "function" ? v(el) : v && (v.current = el);
    };
    return p;
  }
  const k = n.replace(/-([a-z])/g, (_, ch) => ch.toUpperCase());
  p.set = (el, v) => {
    if (el.tagName.includes("-") && !n.startsWith("data-") && !n.startsWith("aria-")) {
      el[k] = v;
      return;
    }
    v == null || v === false ? el.removeAttribute(n) : el.setAttribute(n, v === true ? "" : v);
  };
  return p;
}
var Inst = class {
  constructor(tr, c, rt) {
    this.c = c;
    this.rt = rt;
    const f = this.f = c.e.content.cloneNode(true);
    const ps = c.p, ts = this.t = [], ds = this.d = [];
    for (let i = 0; i < ps.length; i++) {
      const p = ps[i];
      const node2 = at(f, p.pth);
      if (p.t === ATTR) {
        if (p.dl) delegate(rt, p.dl);
        ts.push(p.wrap ? p.mk(node2) : node2);
      } else if (p.t === SOLE) {
        ts.push(node2);
      } else {
        ts.push(new Child(node2, node2.nextSibling, rt));
      }
      ds.push(null);
    }
    this.update(tr.v);
  }
  update(vals) {
    const ps = this.c.p, ts = this.t, ds = this.d;
    for (let i = 0; i < ps.length; i++) {
      const p = ps[i];
      const target = ts[i];
      const attr = p.t === ATTR;
      if (p.s !== null) {
        const join = () => {
          let r = p.s[0];
          for (let k = 0; k < p.h; k++) {
            const x = vals[p.o + k];
            const y = typeof x === "function" ? x() : x;
            r += (y == null || y === false ? "" : y) + p.s[k + 1];
          }
          return r;
        };
        let dyn = 0;
        for (let k = 0; k < p.h; k++) if (typeof vals[p.o + k] === "function") dyn = 1;
        if (dyn) ds[i] = live(ds[i], () => p.set(target, join()));
        else {
          ds[i] = kill(ds[i]);
          p.set(target, join());
        }
        continue;
      }
      const v = vals[p.o];
      if (p.raw) {
        p.set(target, v);
        continue;
      }
      if (p.t === SOLE) {
        if (typeof v === "function") {
          ds[i] = live(ds[i], () => {
            const x = v(), cur = ts[i];
            if (cur.set) {
              cur.set(x);
              return;
            }
            if (writeText(cur, x)) return;
            (ts[i] = new Sole(cur, this.rt)).set(x);
          });
        } else {
          ds[i] = kill(ds[i]);
          if (target.set) target.set(v);
          else if (!writeText(target, v)) (ts[i] = new Sole(target, this.rt)).set(v);
        }
        continue;
      }
      if (typeof v === "function") {
        ds[i] = live(ds[i], attr ? () => p.set(target, v()) : () => target.set(v()));
      } else {
        ds[i] = kill(ds[i]);
        attr ? p.set(target, v) : target.set(v);
      }
    }
  }
  destroy() {
    const ts = this.t, ds = this.d;
    for (let i = 0; i < ts.length; i++) {
      if (ds[i]) {
        dispose(ds[i]);
        ds[i] = null;
      }
      const t = ts[i];
      if (t && t.destroy) t.destroy();
    }
  }
};
var writeText = (el, v) => {
  if (v == null || v === false || v === true) {
    el.textContent = "";
    return 1;
  }
  const t = typeof v;
  if (t === "string") {
    el.textContent = v;
    return 1;
  }
  if (t === "number") {
    el.textContent = "" + v;
    return 1;
  }
  return 0;
};
var Sole = class {
  constructor(el, rt) {
    this.el = el;
    this.rt = rt;
    this.c = null;
    this.d = null;
  }
  set(v) {
    if (this.c) return this.c.set(v);
    if (writeText(this.el, v)) return;
    this.el.textContent = "";
    const s = comment(), e = comment();
    this.el.append(s, e);
    this.c = new Child(s, e, this.rt);
    this.c.set(v);
  }
  destroy() {
    this.d = kill(this.d);
    if (this.c) {
      this.c.destroy();
      this.c = null;
    }
    this.el.textContent = "";
  }
};
var Child = class _Child {
  constructor(s, e, rt) {
    this.s = s;
    this.e = e;
    this.rt = rt;
    this.tn = this.in = this.li = this.no = this.ea = null;
    this.d = null;
    this.k = void 0;
  }
  set(v) {
    if (v == null || typeof v === "boolean" || v === "") return this.clear();
    if (v instanceof Tpl) return this.tpl(v);
    if (Array.isArray(v)) return this.list(v);
    if (v.__e) return this.each(v);
    if (v.nodeType) return this.node(v);
    this.text(v);
  }
  each(spec) {
    if (this.ea && this.ea.spec === spec) return;
    this.clear();
    this.ea = new Each(this, spec);
  }
  text(v) {
    if (this.tn) {
      this.tn.data = v;
      return;
    }
    this.clear();
    this.e.before(this.tn = doc.createTextNode(v));
  }
  node(v) {
    if (this.no === v) return;
    this.clear();
    this.e.before(this.no = v);
  }
  tpl(t) {
    const c = tplOf(t);
    if (this.in && this.in.c === c) return this.in.update(t.v);
    this.clear();
    const i = this.in = new Inst(t, c, this.rt);
    this.e.before(i.f);
  }
  list(vals) {
    if (!this.li) {
      this.clear();
      this.li = [];
    }
    if (vals.length && vals[0] instanceof Tpl && vals[0].k !== void 0) return this.klist(vals);
    const items = this.li, n = vals.length;
    for (let i = items.length - 1; i >= n; i--) {
      const c = items[i];
      c.destroy();
      c.s.remove();
      c.e.remove();
    }
    if (items.length > n) items.length = n;
    for (let i = 0; i < n; i++) {
      let c = items[i];
      if (!c) {
        const s = comment(), e = comment();
        this.e.before(s, e);
        c = items[i] = new _Child(s, e, this.rt);
      }
      bind(c, vals[i]);
    }
  }
  klist(vals) {
    const items = this.li, map = /* @__PURE__ */ new Map();
    for (let i = 0; i < items.length; i++) map.set(items[i].k, items[i]);
    const out = new Array(vals.length);
    for (let i = 0; i < vals.length; i++) {
      const c = map.get(vals[i].k);
      if (c) {
        map.delete(vals[i].k);
        out[i] = c;
      } else out[i] = null;
    }
    map.forEach((c) => {
      c.destroy();
      c.s.remove();
      c.e.remove();
    });
    const parent = this.e.parentNode;
    let ref = this.e;
    for (let i = vals.length - 1; i >= 0; i--) {
      let c = out[i];
      if (!c) {
        const s = comment(), e = comment();
        parent.insertBefore(s, ref);
        parent.insertBefore(e, ref);
        c = out[i] = new _Child(s, e, this.rt);
        c.k = vals[i].k;
      } else if (c.e.nextSibling !== ref) {
        let node2 = c.s;
        const stop = c.e.nextSibling;
        while (node2 !== stop) {
          const nx = node2.nextSibling;
          parent.insertBefore(node2, ref);
          node2 = nx;
        }
      }
      bind(c, vals[i]);
      ref = c.s;
    }
    this.li = out;
  }
  clear() {
    if (this.ea) {
      this.ea.destroy();
      this.ea = null;
    }
    if (this.in) {
      this.in.destroy();
      this.in = null;
    }
    if (this.li) {
      for (let i = 0; i < this.li.length; i++) this.li[i].destroy();
      this.li = null;
    }
    let n = this.s.nextSibling;
    while (n && n !== this.e) {
      const x = n.nextSibling;
      n.remove();
      n = x;
    }
    this.tn = this.no = null;
  }
  destroy() {
    this.d = kill(this.d);
    this.clear();
  }
};
function each(source, render2, key) {
  return { __e: 1, s: source, r: render2, k: key };
}
function lis(arr) {
  const len = arr.length, pred = new Array(len), seq = [];
  for (let i = 0; i < len; i++) {
    const v2 = arr[i];
    if (!v2) continue;
    let lo = 0, hi = seq.length;
    while (lo < hi) {
      const mid = lo + hi >> 1;
      if (arr[seq[mid]] < v2) lo = mid + 1;
      else hi = mid;
    }
    if (lo > 0) pred[i] = seq[lo - 1];
    seq[lo] = i;
  }
  let u = seq.length, v = seq[u - 1];
  while (u-- > 0) {
    seq[u] = v;
    v = pred[v];
  }
  return seq;
}
var Each = class {
  constructor(child, spec) {
    this.c = child;
    this.spec = spec;
    this.k = spec.k;
    this.wantIdx = spec.r.length > 1;
    this.items = [];
    this.rows = [];
    this.dead = 0;
    this.off = onCleanup(() => this.destroy());
    this.stop = effect(() => {
      const list = spec.s() || [];
      const n = list.length;
      const snap = new Array(n);
      for (let i = 0; i < n; i++) snap[i] = list[i];
      untrack(() => this.sync(snap));
    });
  }
  keyOf(v, i) {
    return this.k ? this.k(v, i) : v;
  }
  makeRow(value, index) {
    const row = { v: null, x: null, d: null, f: null, l: null, g: null };
    row.d = root(() => {
      row.v = signal(value);
      if (this.wantIdx) row.x = signal(index);
      const out = this.spec.r(row.v, row.x);
      const inst = new Inst(out, tplOf(out), this.c.rt);
      const frag = inst.f;
      const kids = frag.childNodes;
      if (kids.length === 1 && kids[0].nodeType === 1) {
        row.f = row.l = kids[0];
      } else {
        const a = comment(), b = comment();
        frag.insertBefore(a, frag.firstChild);
        frag.append(b);
        row.f = a;
        row.l = b;
      }
      row.g = frag;
    });
    return row;
  }
  detach(row) {
    const p = row.f.parentNode;
    if (!p) return;
    let n = row.f;
    const stop = row.l.nextSibling;
    while (n !== stop) {
      const x = n.nextSibling;
      p.removeChild(n);
      n = x;
    }
  }
  move(row, parent, ref) {
    let n = row.f;
    const stop = row.l.nextSibling;
    for (; ; ) {
      const x = n.nextSibling;
      parent.insertBefore(n, ref);
      if (n === row.l) break;
      n = x;
    }
  }
  sync(next) {
    if (this.dead) return;
    const prev = this.items;
    const old = this.rows;
    const n = next.length;
    const p = old.length;
    if (n === 0) {
      for (let i = 0; i < p; i++) {
        old[i].d();
        this.detach(old[i]);
      }
      this.items = [];
      this.rows = [];
      return;
    }
    const rows = new Array(n);
    const used = p ? new Uint8Array(p) : null;
    const oldIdx = new Array(n);
    let moved = 0;
    let s = 0;
    while (s < p && s < n && this.keyOf(prev[s], s) === this.keyOf(next[s], s)) {
      rows[s] = old[s];
      used[s] = 1;
      oldIdx[s] = s + 1;
      s++;
    }
    let pe = p - 1, ne = n - 1;
    while (pe >= s && ne >= s && this.keyOf(prev[pe], pe) === this.keyOf(next[ne], ne)) {
      rows[ne] = old[pe];
      used[pe] = 1;
      oldIdx[ne] = pe + 1;
      if (ne !== pe) moved = 1;
      pe--;
      ne--;
    }
    if (s <= ne && s <= pe) {
      const seen = /* @__PURE__ */ new Map();
      for (let i = s; i <= pe; i++) if (!used[i]) seen.set(this.keyOf(prev[i], i), i);
      for (let i = s; i <= ne; i++) {
        const k = this.keyOf(next[i], i);
        const j = seen.get(k);
        if (j !== void 0) {
          rows[i] = old[j];
          used[j] = 1;
          seen.delete(k);
          oldIdx[i] = j + 1;
          if (i !== j) moved = 1;
        }
      }
    }
    for (let i = 0; i < p; i++) if (!used[i]) {
      old[i].d();
      this.detach(old[i]);
    }
    for (let i = 0; i < n; i++) {
      const row = rows[i];
      if (!row) rows[i] = this.makeRow(next[i], i);
      else {
        row.v.set(next[i]);
        if (row.x) row.x.set(i);
      }
    }
    const parent = this.c.e.parentNode;
    let ref = this.c.e;
    let batch2 = null;
    const seq = moved ? lis(oldIdx) : null;
    let si = seq ? seq.length - 1 : -1;
    for (let i = n - 1; i >= 0; i--) {
      const row = rows[i];
      if (row.g) {
        (batch2 || (batch2 = doc.createDocumentFragment())).insertBefore(row.g, batch2.firstChild);
        row.g = null;
        continue;
      }
      if (batch2) {
        const f = batch2.firstChild;
        parent.insertBefore(batch2, ref);
        batch2 = null;
        ref = f;
      }
      if (seq) {
        if (si < 0 || i !== seq[si]) this.move(row, parent, ref);
        else si--;
      }
      ref = row.f;
    }
    if (batch2) parent.insertBefore(batch2, ref);
    this.items = next;
    this.rows = rows;
  }
  destroy() {
    if (this.dead) return;
    this.dead = 1;
    this.stop();
    const rows = this.rows;
    for (let i = 0; i < rows.length; i++) {
      rows[i].d();
      this.detach(rows[i]);
    }
    this.rows = [];
    this.items = [];
  }
};
function render(value, container) {
  const s = comment(), e = comment();
  container.append(s, e);
  const c = new Child(s, e, container);
  bind(c, value);
  return () => {
    c.destroy();
    s.remove();
    e.remove();
  };
}

// src/define.js
var kebab2 = (s) => s.replace(/[A-Z]/g, (c) => "-" + c.toLowerCase());
var coerce = (v, d) => {
  if (v === null) return typeof d === "boolean" ? false : d;
  const t = typeof d;
  if (t === "number") return +v;
  if (t === "boolean") return v !== "false";
  if (d !== null && t === "object") {
    try {
      return JSON.parse(v);
    } catch {
      return d;
    }
  }
  return v;
};
function define(name, opts) {
  if (typeof opts === "function") opts = { setup: opts };
  const { props = {}, setup, styles, shadow = "open" } = opts;
  const keys = Object.keys(props);
  const attrs = keys.map(kebab2);
  class AlacrisElement extends HTMLElement {
    static observedAttributes = attrs;
    constructor() {
      super();
      const p = this.props = {};
      for (let i = 0; i < keys.length; i++) p[keys[i]] = signal(props[keys[i]]);
      for (let i = 0; i < keys.length; i++) {
        const k = keys[i];
        if (Object.prototype.hasOwnProperty.call(this, k)) {
          const v = this[k];
          delete this[k];
          this[k] = v;
        }
      }
    }
    attributeChangedCallback(n, _o, v) {
      const k = keys[attrs.indexOf(n)];
      this.props[k].set(coerce(v, props[k]));
    }
    connectedCallback() {
      this._on = 1;
      if (this._up) return;
      this._up = 1;
      let target = this._t;
      if (!target) {
        target = this._t = shadow ? this.attachShadow({ mode: shadow }) : this;
        const styleRoot = this._sr = shadow ? target : this.getRootNode();
        if (styles) applyStyles(styleRoot, styles);
      }
      trackRoot(this._sr);
      this._d = root(() => {
        this._r = render(setup(this.props, this), target);
      });
    }
    disconnectedCallback() {
      this._on = 0;
      queueMicrotask(() => {
        if (this._on || !this._up) return;
        this._r();
        this._d();
        if (this._sr && shadow) untrackRoot(this._sr);
        this._up = 0;
      });
    }
    /** Dispatch a composed, bubbling CustomEvent. */
    emit(type, detail, o) {
      return this.dispatchEvent(new CustomEvent(type, { detail, bubbles: true, composed: true, ...o }));
    }
  }
  for (let i = 0; i < keys.length; i++) {
    const k = keys[i];
    Object.defineProperty(AlacrisElement.prototype, k, {
      get() {
        return this.props[k]();
      },
      set(v) {
        this.props[k].set(v);
      },
      configurable: true,
      enumerable: true
    });
  }
  customElements.define(name, AlacrisElement);
  return AlacrisElement;
}
export {
  adoptGlobal,
  batch,
  computed,
  css,
  define,
  each,
  effect,
  flush,
  html,
  keyed,
  onCleanup,
  render,
  root,
  signal,
  svg,
  tracking,
  untrack,
  vars
};
