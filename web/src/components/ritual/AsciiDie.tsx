import { useEffect, useRef, useState } from 'preact/hooks';

/*
 * A d20 drawn in characters, in real time: an icosahedron is rotated in 3D,
 * every character cell finds the nearest face under it, and the face's angle
 * to the light picks a character from a brightness ramp. Facet edges are drawn
 * one step darker so the solid reads as cut stone, not a blob. The same
 * technique as the classic spinning donut, applied to the product's own icon.
 */

type Vec = [number, number, number];
type Quat = [number, number, number, number];

const PHI = (1 + Math.sqrt(5)) / 2;
const RAMP = ' .,:;-=+*#%@';

const RAW: Vec[] = [];
for (const a of [-1, 1]) {
  for (const b of [-PHI, PHI]) {
    RAW.push([0, a, b], [a, b, 0], [b, 0, a]);
  }
}
const VERTS: Vec[] = RAW.map(([x, y, z]) => {
  const length = Math.hypot(x, y, z);
  return [x / length, y / length, z / length];
});

/** The 20 faces: triples of vertices that are all one edge apart. */
export const FACES: [number, number, number][] = (() => {
  const edge = Math.min(
    ...VERTS.slice(1).map((v) => Math.hypot(v[0] - VERTS[0][0], v[1] - VERTS[0][1], v[2] - VERTS[0][2])),
  );
  const near = (i: number, j: number) =>
    Math.abs(Math.hypot(VERTS[i][0] - VERTS[j][0], VERTS[i][1] - VERTS[j][1], VERTS[i][2] - VERTS[j][2]) - edge) < 1e-6;
  const faces: [number, number, number][] = [];
  for (let i = 0; i < 12; i += 1)
    for (let j = i + 1; j < 12; j += 1)
      for (let k = j + 1; k < 12; k += 1) if (near(i, j) && near(j, k) && near(i, k)) faces.push([i, j, k]);
  return faces;
})();

const centroid = (face: [number, number, number]): Vec => {
  const [a, b, c] = face.map((index) => VERTS[index]);
  const x = (a[0] + b[0] + c[0]) / 3;
  const y = (a[1] + b[1] + c[1]) / 3;
  const z = (a[2] + b[2] + c[2]) / 3;
  const length = Math.hypot(x, y, z);
  return [x / length, y / length, z / length];
};

/** Face 0 shows a 1 and face 19 shows a 20. */
const FACE_NORMALS = FACES.map(centroid);

function rotate(q: Quat, v: Vec): Vec {
  const [w, x, y, z] = q;
  const [vx, vy, vz] = v;
  const tx = 2 * (y * vz - z * vy);
  const ty = 2 * (z * vx - x * vz);
  const tz = 2 * (x * vy - y * vx);
  return [vx + w * tx + (y * tz - z * ty), vy + w * ty + (z * tx - x * tz), vz + w * tz + (x * ty - y * tx)];
}

function multiply(a: Quat, b: Quat): Quat {
  return [
    a[0] * b[0] - a[1] * b[1] - a[2] * b[2] - a[3] * b[3],
    a[0] * b[1] + a[1] * b[0] + a[2] * b[3] - a[3] * b[2],
    a[0] * b[2] - a[1] * b[3] + a[2] * b[0] + a[3] * b[1],
    a[0] * b[3] + a[1] * b[2] - a[2] * b[1] + a[3] * b[0],
  ];
}

function axisAngle(axis: Vec, angle: number): Quat {
  const length = Math.hypot(...axis) || 1;
  const s = Math.sin(angle / 2) / length;
  return [Math.cos(angle / 2), axis[0] * s, axis[1] * s, axis[2] * s];
}

function slerp(a: Quat, b: Quat, t: number): Quat {
  let dot = a[0] * b[0] + a[1] * b[1] + a[2] * b[2] + a[3] * b[3];
  let target = b;
  if (dot < 0) {
    dot = -dot;
    target = [-b[0], -b[1], -b[2], -b[3]];
  }
  if (dot > 0.9995) {
    const mixed = a.map((value, i) => value + (target[i] - value) * t) as Quat;
    const length = Math.hypot(...mixed);
    return mixed.map((value) => value / length) as Quat;
  }
  const theta = Math.acos(dot);
  const sinTheta = Math.sin(theta);
  const wa = Math.sin((1 - t) * theta) / sinTheta;
  const wb = Math.sin(t * theta) / sinTheta;
  return a.map((value, i) => value * wa + target[i] * wb) as Quat;
}

/** The orientation that turns a face straight at the viewer (+z). */
export function facing(faceIndex: number): Quat {
  const n = FACE_NORMALS[faceIndex];
  const axis: Vec = [n[1], -n[0], 0];
  const angle = Math.acos(Math.max(-1, Math.min(1, n[2])));
  return Math.hypot(...axis) < 1e-6 ? [1, 0, 0, 0] : axisAngle(axis, angle);
}

const LIGHT: Vec = (() => {
  const v: Vec = [-0.45, -0.6, 0.66];
  const length = Math.hypot(...v);
  return [v[0] / length, v[1] / length, v[2] / length];
})();

const COLS = 36;
const ROWS = 18;
/** Monospace cells are about twice as tall as wide. */
const CELL_ASPECT = 0.5;

export function render(q: Quat, label: string | null): string {
  const points = VERTS.map((v) => rotate(q, v));
  const grid: string[][] = Array.from({ length: ROWS }, () => Array(COLS).fill(' '));
  const radius = ROWS / 2 - 0.5;
  let front = -1;
  let frontZ = -Infinity;

  FACES.forEach((face, index) => {
    const [a, b, c] = face.map((i) => points[i]);
    const ux = b[0] - a[0], uy = b[1] - a[1], uz = b[2] - a[2];
    const vx = c[0] - a[0], vy = c[1] - a[1], vz = c[2] - a[2];
    let nx = uy * vz - uz * vy, ny = uz * vx - ux * vz, nz = ux * vy - uy * vx;
    const length = Math.hypot(nx, ny, nz);
    nx /= length;
    ny /= length;
    nz /= length;
    const outward = (a[0] + b[0] + c[0]) * nx + (a[1] + b[1] + c[1]) * ny + (a[2] + b[2] + c[2]) * nz;
    if (outward < 0) {
      nx = -nx;
      ny = -ny;
      nz = -nz;
    }
    if (nz <= 0) return;
    if (nz > frontZ) {
      frontZ = nz;
      front = index;
    }

    const light = Math.max(0, nx * LIGHT[0] + ny * LIGHT[1] + nz * LIGHT[2]);
    const shade = Math.min(RAMP.length - 1, 2 + Math.round(light * (RAMP.length - 3)));
    const sx = (p: Vec) => p[0] * radius / CELL_ASPECT + COLS / 2;
    const sy = (p: Vec) => p[1] * radius + ROWS / 2;
    const [x0, y0, x1, y1, x2, y2] = [sx(a), sy(a), sx(b), sy(b), sx(c), sy(c)];
    const area = (x1 - x0) * (y2 - y0) - (x2 - x0) * (y1 - y0);
    if (Math.abs(area) < 1e-6) return;

    const minX = Math.max(0, Math.floor(Math.min(x0, x1, x2)));
    const maxX = Math.min(COLS - 1, Math.ceil(Math.max(x0, x1, x2)));
    const minY = Math.max(0, Math.floor(Math.min(y0, y1, y2)));
    const maxY = Math.min(ROWS - 1, Math.ceil(Math.max(y0, y1, y2)));
    for (let row = minY; row <= maxY; row += 1) {
      for (let col = minX; col <= maxX; col += 1) {
        const px = col + 0.5;
        const py = row + 0.5;
        const w0 = ((x1 - px) * (y2 - py) - (x2 - px) * (y1 - py)) / area;
        const w1 = ((x2 - px) * (y0 - py) - (x0 - px) * (y2 - py)) / area;
        const w2 = 1 - w0 - w1;
        if (w0 < -0.02 || w1 < -0.02 || w2 < -0.02) continue;
        const edge = Math.min(w0, w1, w2) < 0.07;
        grid[row][col] = RAMP[Math.max(1, edge ? shade - 3 : shade)];
      }
    }
  });

  if (label && front >= 0) {
    const n = rotate(q, FACE_NORMALS[front]);
    const col = Math.round(n[0] * radius * 0.55 / CELL_ASPECT + COLS / 2 - label.length / 2);
    const row = Math.round(n[1] * radius * 0.55 + ROWS / 2);
    for (let i = 0; i < label.length; i += 1) {
      if (grid[row]?.[col + i] !== undefined) grid[row][col + i] = label[i];
    }
  }

  return grid.map((line) => line.join('')).join('\n');
}

const reducedMotion = () =>
  typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;

export type DieMode = 'idle' | 'rolling' | 'twenty' | 'one';

/**
 * The die in four moods: `idle` turns slowly (an empty table), `rolling`
 * tumbles fast (the link is being asked for), `twenty` and `one` settle with a
 * short ease onto that face (the table opened, or the opening failed).
 * Decoration only: hidden from assistive technology.
 */
export function AsciiDie({ mode, onSettled }: { mode: DieMode; onSettled?: () => void }) {
  const q = useRef<Quat>(axisAngle([0.3, 1, 0.2], 0.9));
  const [frame, setFrame] = useState(() => render(q.current, null));
  const settled = mode === 'twenty' || mode === 'one';

  useEffect(() => {
    const target = facing(mode === 'one' ? 0 : FACES.length - 1);
    const label = mode === 'one' ? '1' : '20';

    if (reducedMotion()) {
      if (settled) q.current = target;
      setFrame(render(q.current, settled ? label : null));
      if (settled) onSettled?.();
      return;
    }

    let raf = 0;
    let last = performance.now();
    const start = last;
    const from = q.current;
    const axis: Vec = [Math.random() - 0.5, 1, Math.random() - 0.5];
    const speed = mode === 'rolling' ? 7 : 0.6;
    const settleMs = 900;

    const tick = (now: number) => {
      const dt = Math.min(0.05, (now - last) / 1000);
      last = now;
      if (settled) {
        const t = Math.min(1, (now - start) / settleMs);
        const eased = 1 - Math.pow(1 - t, 3);
        // A last half-turn as it lands, fading out with the ease.
        const wobble = axisAngle(axis, (1 - eased) * Math.PI * 1.5);
        q.current = multiply(slerp(from, target, eased), wobble);
        setFrame(render(q.current, t > 0.82 ? label : null));
        if (t < 1) raf = requestAnimationFrame(tick);
        else onSettled?.();
        return;
      }
      q.current = multiply(axisAngle(axis, speed * dt), q.current);
      setFrame(render(q.current, null));
      raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  }, [mode]);

  return (
    <pre class="ftm-die" data-mode={mode} aria-hidden="true">
      {frame}
    </pre>
  );
}
