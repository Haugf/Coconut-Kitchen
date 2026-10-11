import { useWidgetData, type WidgetData } from './useWidgetData'

/**
 * What every widget component receives. See WIDGETS.md.
 *
 * - settings: the widget's settings, defaults from widget.json merged with
 *   what's in config.json on the Pi.
 * - sources: where to fetch each data source named in widget.json. Use
 *   useSource rather than reading this directly.
 */
export interface WidgetProps<S = Record<string, unknown>> {
  settings: S
  sources: Record<string, string>
}

/** One enabled widget, as the server lists it at /api/widgets. */
export interface WidgetInfo {
  id: string
  name: string
  width: 'fill' | 'fit'
  settings: Record<string, unknown>
  sources: Record<string, string>
}

/**
 * Fetch one of the widget's data sources and refresh it every refreshMs.
 * Keeps the last good answer on screen if a refresh fails.
 */
export function useSource<T>(props: WidgetProps<unknown>, name: string, refreshMs: number): WidgetData<T> {
  return useWidgetData<T>(props.sources[name] ?? '', refreshMs)
}

/** The widgets enabled on this mirror, in screen order. */
export function useEnabledWidgets(): WidgetInfo[] {
  const { data } = useWidgetData<WidgetInfo[]>('/api/widgets', 5 * 60 * 1000)
  return data ?? []
}
