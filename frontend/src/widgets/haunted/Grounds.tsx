import { OnTheGrounds, type Visitor } from './Visitors'

/*
  The bottom of the haunted page: a small church on its hill, bare trees,
  a few graves and an iron fence, all in silhouette. The windows flicker
  like candles; bats drift across, and now and then a visitor (Visitors.tsx); when lightning strikes, a bolt comes
  down behind them while the room flashes.
*/

const FENCE = Array.from({ length: 21 }, (_, i) => 452 + i * 14)

function Tree({ x, y, s, flip }: { x: number; y: number; s: number; flip?: boolean }) {
  return (
    <g transform={`translate(${x} ${y}) scale(${flip ? -s : s} ${s})`} className="h-tree">
      <path d="M-4 0 L-2 -70 L2 -70 L5 0 Z" />
      <path className="h-branch" d="M0 -48 L-26 -76 L-42 -74 M0 -56 L24 -86 L40 -84 M0 -66 L-8 -98 L-20 -110 M0 -66 L8 -102 L16 -118 M-26 -76 L-30 -92 M24 -86 L30 -100 M-8 -98 L4 -112" />
    </g>
  )
}

function Grave({ x, y, w, h, cross }: { x: number; y: number; w: number; h: number; cross?: boolean }) {
  if (cross) {
    return (
      <g className="h-stone">
        <rect x={x - 2.5} y={y - h} width={5} height={h} />
        <rect x={x - w / 2} y={y - h * 0.72} width={w} height={5} />
      </g>
    )
  }
  return <path className="h-stone" d={`M${x - w / 2} ${y} L${x - w / 2} ${y - h + w / 2} A${w / 2} ${w / 2} 0 0 1 ${x + w / 2} ${y - h + w / 2} L${x + w / 2} ${y} Z`} />
}

function Bat({ x, y, delay }: { x: number; y: number; delay: number }) {
  return (
    <g className="h-bat" style={{ animationDelay: `${delay}s` }}>
      <path transform={`translate(${x} ${y})`} d="M0 0 q4 -6 8 -2 q2 -4 4 0 q2 -4 4 0 q4 -4 8 2 q-6 -2 -12 3 q-6 -5 -12 -3 z" />
    </g>
  )
}

export function Grounds({ flash, visitor }: { flash: number; visitor: Visitor | null }) {
  // Each strike lands somewhere different.
  const bx = 240 + ((flash * 337) % 700)
  return (
    <svg className="h-grounds" viewBox="0 0 1200 200" preserveAspectRatio="xMidYMax meet" aria-hidden="true">
      <defs>
        <radialGradient id="h-glow">
          <stop offset="0" stopColor="#F2B44B" stopOpacity="0.5" />
          <stop offset="1" stopColor="#F2B44B" stopOpacity="0" />
        </radialGradient>
      </defs>
      {flash > 0 && (
        <path
          key={`bolt${flash}`}
          className="h-bolt"
          d={`M${bx} -10 L${bx - 18} 36 L${bx - 4} 38 L${bx - 30} 92 L${bx - 12} 94 L${bx - 40} 150`}
        />
      )}
      <Bat x={300} y={40} delay={0} />
      <Bat x={980} y={20} delay={-14} />
      <Bat x={640} y={60} delay={-27} />

      <path className="h-hill" d="M0 200 L0 168 C150 150 300 175 450 162 C600 150 700 158 800 150 C950 140 1050 160 1200 152 L1200 200 Z" />

      {/* a small church, under the train times where the columns end early */}
      <g transform="translate(-120 0)">
        <circle cx="850" cy="112" r="90" fill="url(#h-glow)" className="h-houseglow" />
        <g className="h-house">
          {/* the nave and its roof */}
          <rect x="800" y="94" width="140" height="60" />
          <polygon points="792,96 870,56 948,96" />
          {/* the bell tower, its spire and cross */}
          <rect x="752" y="72" width="48" height="82" />
          <polygon points="746,74 776,20 806,74" />
          <rect x="774" y="4" width="4" height="20" />
          <rect x="768" y="9" width="16" height="4" />
        </g>
        <g className="h-windows">
          <path className="h-win is-dark" d="M768 100 v-12 a8 8 0 0 1 16 0 v12 Z" />
          <path className="h-win is-door" d="M766 154 v-18 a10 10 0 0 1 20 0 v18 Z" />
          <path className="h-win" d="M820 138 v-22 a7 7 0 0 1 14 0 v22 Z" />
          <path className="h-win is-slow" d="M853 138 v-22 a7 7 0 0 1 14 0 v22 Z" />
          <path className="h-win" d="M886 138 v-22 a7 7 0 0 1 14 0 v22 Z" />
          <path className="h-win is-slow" d="M916 138 v-22 a7 7 0 0 1 14 0 v22 Z" />
          <circle className="h-win" cx="870" cy="80" r="7" />
        </g>
      </g>

      <Tree x={118} y={168} s={1.15} />
      <Tree x={1112} y={156} s={1.3} flip />
      <Tree x={1010} y={150} s={0.7} />

      <g transform="translate(-60 0)">
      <Grave x={262} y={170} w={16} h={26} />
      <Grave x={300} y={168} w={12} h={30} cross />
      <Grave x={338} y={166} w={18} h={22} />
      <Grave x={392} y={164} w={14} h={28} />
      <Grave x={418} y={163} w={10} h={24} cross />
      </g>

      <OnTheGrounds visitor={visitor} />

      <g className="h-fence" transform="translate(-122 0)">
        <rect x="452" y="140" width="282" height="2.5" />
        <rect x="452" y="152" width="282" height="2.5" />
        {FENCE.map((x) => (
          <path key={x} d={`M${x - 1} 160 L${x - 1} 132 L${x - 3} 132 L${x} 126 L${x + 3} 132 L${x + 1} 132 L${x + 1} 160 Z`} />
        ))}
      </g>
    </svg>
  )
}
