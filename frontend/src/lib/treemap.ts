// Squarified treemap (Bruls, Huizing & van Wijk): lays out positive
// values as rectangles inside `rect` with aspect ratios close to 1.
// Returns one rect per input value, in input order.
export type Rect = { x: number; y: number; w: number; h: number };

function worst(row: number[], side: number): number {
  const sum = row.reduce((a, b) => a + b, 0);
  const max = Math.max(...row);
  const min = Math.min(...row);
  return Math.max((side * side * max) / (sum * sum), (sum * sum) / (side * side * min));
}

export function squarify(values: number[], rect: Rect): Rect[] {
  const out: Rect[] = values.map(() => ({ x: rect.x, y: rect.y, w: 0, h: 0 }));
  const total = values.reduce((a, b) => a + Math.max(b, 0), 0);
  if (total <= 0 || rect.w <= 0 || rect.h <= 0) return out;

  const order = values.map((_, i) => i).sort((a, b) => values[b] - values[a]);
  const area = rect.w * rect.h;
  const scaled = order.map((i) => (Math.max(values[i], 0) / total) * area);
  const r = { ...rect };

  const layoutRow = (rowIdx: number[]) => {
    const areas = rowIdx.map((k) => scaled[k]);
    const sum = areas.reduce((a, b) => a + b, 0);
    if (r.w >= r.h) {
      const stripW = sum / r.h;
      let y = r.y;
      rowIdx.forEach((k, j) => {
        const h = areas[j] / stripW;
        out[order[k]] = { x: r.x, y, w: stripW, h };
        y += h;
      });
      r.x += stripW;
      r.w -= stripW;
    } else {
      const stripH = sum / r.w;
      let x = r.x;
      rowIdx.forEach((k, j) => {
        const w = areas[j] / stripH;
        out[order[k]] = { x, y: r.y, w, h: stripH };
        x += w;
      });
      r.y += stripH;
      r.h -= stripH;
    }
  };

  let row: number[] = [];
  let i = 0;
  while (i < scaled.length) {
    const side = Math.min(r.w, r.h);
    const candidate = [...row, i];
    if (row.length === 0 || worst(candidate.map((k) => scaled[k]), side) <= worst(row.map((k) => scaled[k]), side)) {
      row = candidate;
      i++;
    } else {
      layoutRow(row);
      row = [];
    }
  }
  if (row.length) layoutRow(row);
  return out;
}
