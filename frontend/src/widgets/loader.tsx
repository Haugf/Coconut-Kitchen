import { Component, type ComponentType, type ReactNode } from 'react'
import type { WidgetInfo, WidgetProps } from '../lib/widgets'

/**
 * Finds every widget folder (one with an index.tsx) automatically, so
 * adding a widget never means editing this file. Folders starting with
 * "_" (the template) are left out.
 */
const modules = import.meta.glob<{ default: ComponentType<WidgetProps> }>(['./*/index.tsx', '!./_*/**'], {
  eager: true,
})

const components: Record<string, ComponentType<WidgetProps>> = {}
for (const [path, mod] of Object.entries(modules)) {
  const id = path.split('/')[1]
  components[id] = mod.default
}

/** Grid columns for the bottom band: Today, then one per widget. */
export function widgetColumns(widgets: WidgetInfo[]): string {
  return ['minmax(0, 1.2fr)', ...widgets.map((w) => (w.width === 'fit' ? 'auto' : 'minmax(0, 1fr)'))].join(' ')
}

/** Renders one enabled widget. A widget that crashes disappears quietly. */
export function WidgetSlot({ widget }: { widget: WidgetInfo }) {
  const Widget = components[widget.id]
  if (!Widget) return null
  return (
    <Guard id={widget.id}>
      <Widget settings={widget.settings} sources={widget.sources} />
    </Guard>
  )
}

class Guard extends Component<{ id: string; children: ReactNode }, { failed: boolean }> {
  state = { failed: false }

  static getDerivedStateFromError() {
    return { failed: true }
  }

  componentDidCatch(error: unknown) {
    console.error(`widget ${this.props.id} crashed and is hidden:`, error)
  }

  render() {
    return this.state.failed ? null : this.props.children
  }
}
