// Checks every widget folder against the contract in WIDGETS.md.
// Runs before each build; a failing widget stops the build with a reason.
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const widgets = join(dirname(fileURLToPath(import.meta.url)), '..', 'src', 'widgets')
const ID = /^[a-z][a-z0-9-]{1,30}$/
const problems = []
let count = 0

for (const folder of readdirSync(widgets)) {
  const dir = join(widgets, folder)
  if (!statSync(dir).isDirectory() || !existsSync(join(dir, 'widget.json'))) continue
  count++
  const say = (msg) => problems.push(`${folder}: ${msg}`)

  let m
  try {
    m = JSON.parse(readFileSync(join(dir, 'widget.json'), 'utf8'))
  } catch (e) {
    say(`widget.json isn't valid JSON (${e.message})`)
    continue
  }
  const template = folder.startsWith('_')
  if (!ID.test(m.id ?? '')) say('"id" must be lowercase letters, digits and dashes')
  if (!template && m.id !== folder) say(`"id" is "${m.id}" but the folder is "${folder}"; they must match`)
  if (!m.name) say('"name" is required')
  if (!m.description) say('"description" is required: one sentence on what it shows')
  if (m.width && !['fill', 'fit'].includes(m.width)) say('"width" must be "fill" or "fit"')

  const settings = Object.keys(m.settings ?? {})
  const secrets = m.secrets ?? []
  for (const [name, s] of Object.entries(m.settings ?? {})) {
    if (!s || typeof s !== 'object' || !('default' in s)) say(`setting "${name}" needs a "default"`)
  }

  let sample = null
  if (!existsSync(join(dir, 'sample.json'))) say('sample.json is missing: an example answer for each data source')
  else {
    try {
      sample = JSON.parse(readFileSync(join(dir, 'sample.json'), 'utf8'))
    } catch (e) {
      say(`sample.json isn't valid JSON (${e.message})`)
    }
  }

  for (const [name, src] of Object.entries(m.data ?? {})) {
    if (!ID.test(name)) say(`data source "${name}": names are lowercase letters, digits and dashes`)
    const url = src?.url ?? ''
    if (!url.startsWith('/api/')) {
      if (!url.startsWith('https://')) say(`data source "${name}": url must start with https://`)
      const host = url.replace('https://', '').split(/[/?#]/)[0]
      if (!host || /[{}@]/.test(host)) say(`data source "${name}": write the host out in full, without {placeholders}`)
      if (src.every && !/^\d+(s|m|h)$/.test(src.every)) say(`data source "${name}": "every" looks like "30s", "10m" or "1h"`)
    }
    const text = url + JSON.stringify(src?.headers ?? {})
    for (const [, kind, key] of text.matchAll(/\{(settings|secrets)\.([A-Za-z0-9_]+)\}/g)) {
      if (kind === 'settings' && !settings.includes(key)) say(`data source "${name}" uses {settings.${key}}, which isn't in "settings"`)
      if (kind === 'secrets' && !secrets.includes(key)) say(`data source "${name}" uses {secrets.${key}}, which isn't in "secrets"`)
    }
    if (sample && !(name in sample)) say(`sample.json needs an example for "${name}"`)
  }

  const index = join(dir, 'index.tsx')
  if (!existsSync(index)) say('index.tsx is missing')
  else if (!/export default function/.test(readFileSync(index, 'utf8'))) say('index.tsx must `export default function YourWidget(props: WidgetProps)`')
  if (existsSync(join(dir, 'style.css'))) {
    const css = readFileSync(join(dir, 'style.css'), 'utf8')
    if (/#[0-9a-fA-F]{3,8}\b|rgb\(|hsl\(/.test(css.replace(/\/\*[\s\S]*?\*\//g, ''))) {
      say('style.css uses its own colors; use the page tokens (var(--ink), var(--ink-soft), var(--hairline), var(--clay) ...) so day and night both work')
    }
  }
}

if (problems.length) {
  console.error(`Widget check found ${problems.length} problem(s):\n  ` + problems.join('\n  '))
  process.exit(1)
}
console.log(`Widget check: ${count} widgets look good.`)
