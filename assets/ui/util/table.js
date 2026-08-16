// Table data pipeline — pure functions used by <ui-table>.
//
// Order: normalize columns → filter → sort → group → paginate → aggregate.
// Each step is independently testable and reusable; the component only
// wires signals into this and renders the result.

export const titleCase = (k) => String(k)
  .replace(/[_-]+/g, ' ')
  .replace(/([a-z])([A-Z])/g, '$1 $2')
  .replace(/\b\w/g, (c) => c.toUpperCase());

export const normalizeCol = (c) => {
  if (typeof c === 'string') return { key: c, label: titleCase(c) };
  return { label: c.label || titleCase(c.key), ...c };
};

export const inferColumns = (rows) => {
  const first = (rows || []).find((r) => r && typeof r === 'object');
  if (!first) return [];
  return Object.keys(first)
    .filter((k) => k !== 'id')
    .map((key) => ({
      key,
      label: titleCase(key),
      numeric: typeof first[key] === 'number',
      sortable: true,
    }));
};

export const visibleColumns = (columns, rows, hidden) => {
  const raw = columns || [];
  const list = (raw.length ? raw.map(normalizeCol) : inferColumns(rows))
    .filter((c) => !c.hidden);
  if (!hidden || !hidden.length) return list;
  const skip = new Set(hidden);
  const shown = list.filter((c) => !skip.has(c.key));
  return shown.length ? shown : list.slice(0, 1);
};

export const compare = (a, b) => {
  const aU = a == null || a === '';
  const bU = b == null || b === '';
  if (aU && bU) return 0;
  if (aU) return 1;
  if (bU) return -1;
  if (typeof a === 'number' && typeof b === 'number') return a - b;
  const an = +a;
  const bn = +b;
  if (a !== '' && b !== '' && !Number.isNaN(an) && !Number.isNaN(bn)) return an - bn;
  return String(a).localeCompare(String(b), undefined, { numeric: true, sensitivity: 'base' });
};

export const sortValue = (row, col) => {
  if (!col) return '';
  if (typeof col.sortValue === 'function') return col.sortValue(row);
  return row == null ? '' : row[col.key];
};

export const filterRows = (rows, cols, query) => {
  if (!query) return rows || [];
  const needle = String(query).toLowerCase();
  return (rows || []).filter((row) => cols.some((c) => {
    const v = sortValue(row, c);
    return v != null && String(v).toLowerCase().includes(needle);
  }));
};

export const sortRows = (rows, cols, sortBy, sortDir) => {
  if (!sortBy) return rows || [];
  const col = (cols || []).find((c) => c.key === sortBy);
  const dir = sortDir === 'desc' ? -1 : 1;
  return (rows || []).slice().sort((a, b) => dir * compare(sortValue(a, col), sortValue(b, col)));
};

export const paginate = (rows, page, pageSize) => {
  const size = Math.max(0, Math.floor(pageSize) || 0);
  if (!size) return rows || [];
  const p = Math.max(1, Math.floor(page) || 1);
  const start = (p - 1) * size;
  return (rows || []).slice(start, start + size);
};

export const makeIdOf = (getRowId) => {
  const anon = new WeakMap();
  let n = 0;
  return (row, i) => {
    if (typeof getRowId === 'function') return getRowId(row, i);
    if (row != null && row.id != null) return row.id;
    if (row && typeof row === 'object') {
      let id = anon.get(row);
      if (id == null) { id = `row-${++n}`; anon.set(row, id); }
      return id;
    }
    return i;
  };
};

const asNumber = (v) => {
  if (typeof v === 'number' && !Number.isNaN(v)) return v;
  const n = +v;
  return Number.isNaN(n) ? null : n;
};

export const aggregateValues = (rows, col, fn) => {
  const kind = fn || col.aggregate;
  if (!kind) return null;
  const nums = [];
  for (const row of rows || []) {
    const n = asNumber(sortValue(row, col));
    if (n != null) nums.push(n);
  }
  if (kind === 'count') return (rows || []).length;
  if (!nums.length) return null;
  if (kind === 'sum') return nums.reduce((a, b) => a + b, 0);
  if (kind === 'avg') return nums.reduce((a, b) => a + b, 0) / nums.length;
  if (kind === 'min') return Math.min(...nums);
  if (kind === 'max') return Math.max(...nums);
  return null;
};

export const aggregateRow = (rows, cols) => {
  const out = {};
  for (const col of cols || []) {
    if (!col.aggregate) continue;
    out[col.key] = aggregateValues(rows, col, col.aggregate);
  }
  return out;
};

export const formatAgg = (v) => {
  if (v == null) return '';
  if (typeof v !== 'number') return String(v);
  if (Number.isInteger(v)) return v.toLocaleString();
  return v.toLocaleString(undefined, { maximumFractionDigits: 2 });
};

/** Flatten rows into group headers + member rows. `expanded` is a Set of group keys; omit to expand all. */
export const groupRows = (rows, groupBy, { expanded, idOf, cols } = {}) => {
  const list = rows || [];
  const id = idOf || ((row, i) => (row && row.id != null ? row.id : i));
  if (!groupBy) {
    return list.map((row, i) => ({ type: 'row', id: id(row, i), row }));
  }
  const buckets = new Map();
  for (const row of list) {
    const raw = row == null ? '' : row[groupBy];
    const key = raw == null ? '' : String(raw);
    let bucket = buckets.get(key);
    if (!bucket) { bucket = []; buckets.set(key, bucket); }
    bucket.push(row);
  }
  const out = [];
  for (const [key, members] of buckets) {
    const open = !expanded || expanded.has(key);
    out.push({
      type: 'group',
      id: `group:${key}`,
      key,
      label: key,
      count: members.length,
      rows: members,
      expanded: open,
      aggregates: aggregateRow(members, cols),
    });
    if (open) {
      for (let i = 0; i < members.length; i++) {
        out.push({ type: 'row', id: id(members[i], i), row: members[i], group: key, depth: 1 });
      }
    }
  }
  return out;
};

export const toCsv = (rows, cols) => {
  const esc = (v) => {
    const s = v == null ? '' : String(v);
    return /[",\n\r]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
  };
  const header = (cols || []).map((c) => esc(c.label)).join(',');
  const body = (rows || []).map((row) =>
    (cols || []).map((c) => esc(sortValue(row, c))).join(',')).join('\n');
  return body ? `${header}\n${body}` : header;
};

export const downloadText = (filename, text, type = 'text/csv;charset=utf-8') => {
  const blob = new Blob([text], { type });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.rel = 'noopener';
  document.body.append(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
};

/**
 * Run the full client-side pipeline. Server modes skip filter/sort/page.
 */
export const processTable = ({
  rows = [],
  columns = [],
  hiddenColumns = [],
  filter = '',
  sortBy = '',
  sortDir = 'asc',
  sortMode = 'client',
  groupBy = '',
  expanded,
  page = 1,
  pageSize = 0,
  paginationMode = 'client',
  rowCount = 0,
  idOf,
} = {}) => {
  const cols = visibleColumns(columns, rows, hiddenColumns);
  let list = rows || [];
  if (sortMode !== 'server') {
    list = filterRows(list, cols, filter);
    list = sortRows(list, cols, sortBy, sortDir);
  }
  const filteredCount = (Math.floor(rowCount) || 0) > 0 ? Math.floor(rowCount) : list.length;
  const items = groupRows(list, groupBy, { expanded, idOf, cols });
  const visible = paginationMode === 'server' || !pageSize
    ? items
    : paginate(items, page, pageSize);
  const totals = aggregateRow(list, cols);
  return { cols, list, filteredCount, items, visible, totals };
};
