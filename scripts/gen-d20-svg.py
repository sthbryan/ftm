"""Regenerate cmd/ftm-desktop/AppIcon.icon/Assets/d20.svg (stdout).

Face-first icosahedron projection as a single-colour glyph: each visible facet
is its own polygon inset by GAP, so facet edges are transparent gaps, and the
central face has "20" punched out with fill-rule=evenodd.
"""
import math
from fontTools.ttLib import TTCollection
from fontTools.misc.transform import Transform
from fontTools.pens.boundsPen import BoundsPen
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.pens.transformPen import TransformPen

SIZE = 1024.0
C = SIZE / 2.0
OUT_R = 400.0
GAP = 11.0
ROT = math.radians(99.09)
FONT = "/System/Library/Fonts/Avenir Next.ttc"
FONT_STYLE = "Bold"

phi = (1 + 5 ** 0.5) / 2
V3 = sorted({tuple(round(c, 9) for c in v) for s1 in (1, -1) for s2 in (1, -1)
             for v in ((0, s1, s2 * phi), (s1, s2 * phi, 0), (s2 * phi, 0, s1))})
EDGE = 2.0
FACES = [(i, j, k) for i in range(12) for j in range(i + 1, 12) for k in range(j + 1, 12)
         if abs(math.dist(V3[i], V3[j]) - EDGE) < 1e-6
         and abs(math.dist(V3[j], V3[k]) - EDGE) < 1e-6
         and abs(math.dist(V3[i], V3[k]) - EDGE) < 1e-6]

# Rotate the solid so face 0's normal points at the viewer, then drop z.
nrm = [sum(V3[i][a] for i in FACES[0]) / 3 for a in range(3)]
mag = math.sqrt(sum(c * c for c in nrm))
nrm = [c / mag for c in nrm]
ax, ay = nrm[1], -nrm[0]
K = [[0, 0, ay], [0, 0, -ax], [-ay, ax, 0]]
K2 = [[sum(K[i][m] * K[m][j] for m in range(3)) for j in range(3)] for i in range(3)]
k = (1 - nrm[2]) / (ax * ax + ay * ay)
R = [[(1 if i == j else 0) + K[i][j] + K2[i][j] * k for j in range(3)] for i in range(3)]
P = [[sum(R[i][j] * p[j] for j in range(3)) for i in range(3)] for p in V3]

scale = OUT_R / max(math.hypot(p[0], p[1]) for p in P)
ca, sa = math.cos(ROT), math.sin(ROT)
XY = [(C + (p[0] * scale) * ca - (p[1] * scale) * sa,
       C - ((p[0] * scale) * sa + (p[1] * scale) * ca)) for p in P]

VISIBLE = [f for f in FACES if sum(P[i][2] for i in f) > 0]


def centroid(face):
    return (sum(XY[i][0] for i in face) / 3, sum(XY[i][1] for i in face) / 3)


def inset(face, d):
    cx, cy = centroid(face)
    out = []
    for i in face:
        x, y = XY[i]
        dx, dy = x - cx, y - cy
        m = math.hypot(dx, dy)
        f = 1 - (d * 2.0) / m
        out.append((cx + dx * f, cy + dy * f))
    return out


def poly_d(points):
    return "M " + " L ".join(f"{x:.2f},{y:.2f}" for x, y in points) + " Z"


def incircle_r(face):
    cx, cy = centroid(face)
    pts = [XY[i] for i in face]
    r = float("inf")
    for i in range(3):
        a, b = pts[i], pts[(i + 1) % 3]
        area2 = abs((b[0] - a[0]) * (a[1] - cy) - (a[0] - cx) * (b[1] - a[1]))
        r = min(r, area2 / math.dist(a, b))
    return r


CENTRAL = min(VISIBLE, key=lambda f: math.dist(centroid(f), (C, C)))


def digits_path(text, cx, cy, target_w):
    """Outlines for `text`, already transformed into icon space.

    Returned as raw path data so it can be appended to the facet's own `d`,
    which is what makes fill-rule=evenodd punch the digits out as holes.
    """
    font = None
    for candidate in TTCollection(FONT).fonts:
        if candidate["name"].getDebugName(2) == FONT_STYLE:
            font = candidate
            break
    if font is None:
        raise SystemExit(f"{FONT_STYLE} not found in {FONT}")
    glyphs = font.getGlyphSet()
    cmap = font.getBestCmap()

    names, advance = [], 0.0
    bounds = BoundsPen(glyphs)
    for ch in text:
        name = cmap[ord(ch)]
        names.append((name, advance))
        glyphs[name].draw(TransformPen(bounds, Transform().translate(advance, 0)))
        advance += font["hmtx"][name][0]

    xmin, ymin, xmax, ymax = bounds.bounds
    s = target_w / (xmax - xmin)
    ox = cx - ((xmin + xmax) / 2) * s
    oy = cy + ((ymin + ymax) / 2) * s

    out = []
    for name, off in names:
        pen = SVGPathPen(glyphs)
        t = Transform(s, 0, 0, -s, ox + off * s, oy)
        glyphs[name].draw(TransformPen(pen, t))
        out.append(pen.getCommands())
    return out


ri = incircle_r(CENTRAL)
cx, cy = centroid(CENTRAL)
# The central face points down, so nudge the digits up into its wide part.
digits = digits_path("20", cx, cy - ri * 0.18, ri * 1.5)

lines = ['<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024" '
         'width="1024" height="1024">', '  <g fill="#FFFFFF">']
for face in VISIBLE:
    if face == CENTRAL:
        continue
    lines.append(f'    <path d="{poly_d(inset(face, GAP))}"/>')
central_d = " ".join([poly_d(inset(CENTRAL, GAP))] + digits)
lines.append(f'    <path fill-rule="evenodd" d="{central_d}"/>')
lines.append('  </g>')
lines.append('</svg>')
print("\n".join(lines))
