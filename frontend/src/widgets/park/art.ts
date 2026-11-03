// The drawings. Stand-ins are SVG. Drop a same-named file from Ally into
// art/ (trunk.png next to trunk.svg, say) and delete the stand-in.
const files = import.meta.glob('./art/*.{svg,png,webp}', { eager: true, query: '?url', import: 'default' }) as Record<string, string>

const find = (name: string) => {
  const hit = Object.keys(files).find((path) => path.startsWith(`./art/${name}.`))
  if (!hit) throw new Error(`missing park art: ${name}`)
  return files[hit]
}
const many = (count: number, name: (n: number) => string) => Array.from({ length: count }, (_, i) => find(name(i + 1)))

export const art = {
  trunk: find('trunk'),
  clumps: many(10, (n) => `clump-${String(n).padStart(2, '0')}`),
  leaves: many(6, (n) => `leaf-${n}`),
  bridge: find('bridge'),
  skyline: find('skyline'),
  treeline: find('treeline'),
  clouds: many(3, (n) => `cloud-${n}`),
  sun: find('sun'),
  moon: find('moon'),
  sticker: find('sticker'),
  table: find('table'),
  turkey: find('turkey'),
}
