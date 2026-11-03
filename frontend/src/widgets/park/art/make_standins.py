"""Generate the stand-in SVG art for the Central Park theme.

Sizes are double the on-screen size, matching what Ally is asked to deliver.
"""
import math
import pathlib
import random

OUT = pathlib.Path(__file__).resolve().parent
OUT.mkdir(exist_ok=True)

LEAF = "M0 -26 C16 -14 18 8 0 26 C-18 8 -16 -14 0 -26Z"
FALL = ["#C9772F", "#D9A23A", "#A8482B", "#E3B94E", "#B85A28", "#8F3F27"]


def save(name, w, h, body):
    (OUT / name).write_text(
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{w}" height="{h}" '
        f'viewBox="0 0 {w} {h}">{body}</svg>\n'
    )


def leaf(x, y, scale, rot, fill):
    return (
        f'<g transform="translate({x:.1f} {y:.1f}) rotate({rot:.1f}) scale({scale:.2f})">'
        f'<path d="{LEAF}" fill="{fill}"/>'
        '<line x1="0" y1="-20" x2="0" y2="24" stroke="#5B3A23" stroke-width="1.2"/></g>'
    )


def trunk():
    bark = "#3E2A22"
    limbs = [
        ("M1300 2160 C1290 1640 1270 1320 1300 900", 170),
        ("M1290 1320 C1100 1140 900 1000 720 720", 74),
        ("M1310 1260 C1500 1100 1700 980 1900 680", 74),
        ("M1300 960 C1220 720 1140 480 1080 120", 56),
        ("M1300 940 C1400 720 1480 460 1520 80", 56),
        ("M1000 1040 C860 1020 700 1040 560 1100", 28),
        ("M1600 1060 C1760 1080 1900 1060 2040 1100", 28),
        ("M900 880 C820 720 740 560 700 380", 24),
        ("M1720 900 C1800 740 1860 560 1880 360", 24),
    ]
    body = "".join(
        f'<path d="{d}" fill="none" stroke="{bark}" stroke-width="{w}" stroke-linecap="round"/>'
        for d, w in limbs
    )
    save("trunk.svg", 2600, 2160, body)


def clump(n):
    rng = random.Random(100 + n)
    w, h = 640, 560
    parts = []
    for _ in range(46):
        a = rng.uniform(0, math.tau)
        r = math.sqrt(rng.random())
        x = w / 2 + r * 250 * math.cos(a)
        y = h / 2 + r * 205 * math.sin(a)
        parts.append(leaf(x, y, rng.uniform(1.1, 1.8), rng.uniform(0, 360), rng.choice(FALL)))
    save(f"clump-{n:02d}.svg", w, h, "".join(parts))


def leaves():
    for n, fill in enumerate(FALL, 1):
        save(f"leaf-{n}.svg", 200, 280, leaf(100, 140, 4.6, 0, fill))


def bridge():
    deck = "#F3EFE4"
    body = (
        f'<path d="M20 330 Q700 60 1380 330" fill="none" stroke="{deck}" stroke-width="16"/>'
        f'<path d="M20 390 Q700 160 1380 390" fill="none" stroke="{deck}" stroke-width="16"/>'
    )
    for i in range(1, 16):
        x = 20 + i * 85
        t = (x - 20) / 1360
        top = 330 - 4 * 135 * t * (1 - t)
        bot = 390 - 4 * 115 * t * (1 - t)
        body += f'<line x1="{x}" y1="{top:.0f}" x2="{x}" y2="{bot:.0f}" stroke="{deck}" stroke-width="6"/>'
    save("bridge.svg", 1400, 480, body)


def skyline():
    blocks = [
        (120, 300, 220, 300), (420, 120, 90, 480), (444, 40, 44, 90), (510, 300, 70, 300),
        (580, 120, 90, 480), (604, 40, 44, 90), (740, 260, 180, 340), (1000, 320, 260, 280),
    ]
    body = "".join(
        f'<rect x="{x}" y="{y}" width="{w}" height="{h}" fill="#B7CBCB"/>' for x, y, w, h in blocks
    )
    save("skyline.svg", 3840, 600, body)


def treeline():
    rng = random.Random(7)
    colors = ["#C9772F", "#D9A23A", "#A8482B", "#5F7A4E"]
    body = ""
    for x in range(-40, 3960, 240):
        body += (
            f'<ellipse cx="{x + rng.randint(-40, 40)}" cy="{rng.randint(250, 290)}" '
            f'rx="{rng.randint(190, 280)}" ry="{rng.randint(110, 160)}" '
            f'fill="{rng.choice(colors)}"/>'
        )
    save("treeline.svg", 3840, 440, body)


def clouds():
    shapes = [
        [(500, 220, 420, 110), (640, 150, 240, 110), (320, 220, 200, 80)],
        [(500, 230, 450, 100), (360, 170, 210, 90), (700, 200, 170, 80)],
        [(500, 220, 400, 90), (560, 160, 200, 80)],
    ]
    for n, parts in enumerate(shapes, 1):
        body = "".join(
            f'<ellipse cx="{x}" cy="{y}" rx="{a}" ry="{b}" fill="#EEF0EE"/>' for x, y, a, b in parts
        )
        save(f"cloud-{n}.svg", 1000, 360, body)


def orbs():
    save("sun.svg", 360, 360, '<circle cx="180" cy="180" r="170" fill="#F7D98A"/>')


def sticker():
    save(
        "sticker.svg", 480, 480,
        '<circle cx="240" cy="240" r="230" fill="#A8322A"/>'
        '<circle cx="240" cy="240" r="200" fill="none" stroke="#FFFFFF" stroke-width="6"/>'
        '<text x="240" y="225" text-anchor="middle" font-family="Georgia, serif" font-size="110" fill="#FFFFFF">I</text>'
        '<text x="240" y="325" text-anchor="middle" font-family="Georgia, serif" font-size="96" fill="#FFFFFF">voted</text>',
    )


def table():
    save(
        "table.svg", 1800, 480,
        '<rect x="40" y="180" width="1720" height="44" fill="#F3EFE4"/>'
        '<rect x="40" y="224" width="1720" height="30" fill="#8A5A34"/>'
        '<rect x="130" y="254" width="34" height="226" fill="#5B3A23"/>'
        '<rect x="1636" y="254" width="34" height="226" fill="#5B3A23"/>'
        '<ellipse cx="900" cy="176" rx="220" ry="24" fill="#E8E2D2"/>'
        '<path d="M740 170 C770 30 1030 30 1060 170Z" fill="#A8482B"/>'
        '<ellipse cx="300" cy="172" rx="120" ry="20" fill="#D9A23A"/>'
        '<ellipse cx="1500" cy="172" rx="120" ry="20" fill="#D9A23A"/>',
    )


def turkey():
    fan = "".join(
        f'<ellipse cx="300" cy="170" rx="46" ry="150" fill="{FALL[i % 6]}" '
        f'transform="rotate({-70 + i * 28} 300 340)"/>'
        for i in range(6)
    )
    save(
        "turkey.svg", 600, 600,
        fan
        + '<ellipse cx="300" cy="380" rx="130" ry="140" fill="#6B3F2A"/>'
        '<circle cx="300" cy="230" r="66" fill="#8A5A34"/>'
        '<circle cx="278" cy="220" r="9" fill="#141413"/><circle cx="322" cy="220" r="9" fill="#141413"/>'
        '<path d="M286 244 L314 244 L300 268Z" fill="#D9A23A"/>'
        '<rect x="240" y="510" width="16" height="70" fill="#D9A23A"/>'
        '<rect x="344" y="510" width="16" height="70" fill="#D9A23A"/>',
    )


trunk()
for i in range(1, 11):
    clump(i)
leaves()
bridge()
skyline()
treeline()
clouds()
orbs()
sticker()
table()
turkey()
print(len(list(OUT.glob("*.svg"))), "assets")
