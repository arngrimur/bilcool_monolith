export const BOOKING_COLORS = [
  '#3b82f6',
  '#22c55e',
  '#a855f7',
  '#f43f5e',
  '#f97316',
  '#14b8a6',
  '#6366f1',
  '#f59e0b',
]

export const DEFAULT_BOOKING_COLOR = BOOKING_COLORS[0]

const otherUserColorMap = new Map<string, string>()

export function getColorForUserRef(userRef: string, excludeColor: string): string {
  if (otherUserColorMap.has(userRef)) {
    return otherUserColorMap.get(userRef)!
  }
  const available = BOOKING_COLORS.filter((c) => c !== excludeColor)
  const pool = available.length > 0 ? available : BOOKING_COLORS
  const usedColors = new Set(otherUserColorMap.values())
  const unused = pool.filter((c) => !usedColors.has(c))
  const color = unused.length > 0 ? unused[0] : pool[otherUserColorMap.size % pool.length]
  otherUserColorMap.set(userRef, color)
  return color
}
