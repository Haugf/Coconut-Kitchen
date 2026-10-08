import type { ComponentType } from 'react'

/**
 * How much vertical room a widget wants in the portrait column.
 * hero: the one large element at the top of a scene
 * block: a normal section
 * compact: a short strip
 */
export type WidgetSize = 'hero' | 'block' | 'compact'

export interface WidgetDef {
  /** Unique id, referenced by scenes. */
  id: string
  component: ComponentType
  size: WidgetSize
}
