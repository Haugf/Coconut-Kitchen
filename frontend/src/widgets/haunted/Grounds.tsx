/*
  The bottom of the haunted page: the house on its hill, bare trees, a
  few graves and an iron fence, all in silhouette. The windows flicker
  like candles; bats drift across; when lightning strikes, a bolt comes
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

export function Grounds({ flash }: { flash: number }) {
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

      {/* the house, under the train times where the columns end early */}
      <g transform="translate(-120 0)">
      <circle cx="850" cy="110" r="90" fill="url(#h-glow)" className="h-houseglow" />
      <g className="h-house">
        <rect x="780" y="82" width="160" height="72" />
        <polygon points="770,84 860,32 950,84" />
        <rect x="738" y="52" width="52" height="102" />
        <polygon points="732,54 764,0 796,54" />
        <rect x="904" y="40" width="14" height="34" />
        <polygon points="838,124 861,108 884,124" />
      </g>
      <g className="h-windows">
        <path className="h-win" d="M755 92 v-14 a9 9 0 0 1 18 0 v14 Z" />
        <rect className="h-win is-dark" x="755" y="106" width="18" height="22" />
        <rect className="h-win" x="796" y="94" width="20" height="26" />
        <rect className="h-win is-slow" x="826" y="94" width="20" height="26" />
        <rect className="h-win is-dark" x="876" y="94" width="20" height="26" />
        <rect className="h-win is-slow" x="908" y="94" width="20" height="26" />
        <circle className="h-win" cx="860" cy="64" r="8" />
        <path className="h-win is-door" d="M852 154 v-20 a9 9 0 0 1 18 0 v20 Z" />
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
