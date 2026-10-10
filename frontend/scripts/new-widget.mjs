// Make a new widget from the template:  npm run new-widget -- plant-water
import { cpSync, existsSync, readFileSync, writeFileSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const widgets = join(dirname(fileURLToPath(import.meta.url)), '..', 'src', 'widgets')
const id = process.argv[2]

if (!id || !/^[a-z][a-z0-9-]{1,30}$/.test(id)) {
  console.error('Give the widget an id: lowercase letters, digits and dashes, like  npm run new-widget -- plant-water')
  process.exit(1)
}
const dest = join(widgets, id)
if (existsSync(dest)) {
  console.error(`There's already a widget called "${id}".`)
  process.exit(1)
}

const title = id.split('-').map((w) => w[0].toUpperCase() + w.slice(1)).join(' ')
const pascal = title.replace(/ /g, '')

cpSync(join(widgets, '_template'), dest, { recursive: true })
const edit = (file, fn) => writeFileSync(join(dest, file), fn(readFileSync(join(dest, file), 'utf8')))
edit('widget.json', (s) => s.replace('"id": "example"', `"id": "${id}"`).replace('"name": "Sun"', `"name": "${title}"`))
edit('index.tsx', (s) => s.replaceAll('w-example', `w-${id}`).replace('function Example(', `function ${pascal}(`))
edit('style.css', (s) => s.replaceAll('w-example', `w-${id}`))
edit('README.md', (s) => s.replace('# Sun', `# ${title}`).replaceAll('"id": "example"', `"id": "${id}"`))

console.log(`Made src/widgets/${id}/ from the template (it starts as a sunrise widget).

Next:
  1. widget.json   describe it, its settings, and the data it fetches
  2. sample.json   an example answer for each data source
  3. index.tsx     what it shows
  4. Preview it:   npm run build && python3 ../dev/mock_api.py --with ${id}
                   then open http://127.0.0.1:8099
See WIDGETS.md for the details.`)
