/** Formatadores de exibição da viewer. */

/** 3725 -> "1h 2m 5s" */
export function formatUptime(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return "—"
  const total = Math.floor(seconds)
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const secs = total % 60
  if (hours > 0) return `${hours}h ${minutes}m ${secs}s`
  if (minutes > 0) return `${minutes}m ${secs}s`
  return `${secs}s`
}

/** 1536 -> "1,5 kB" */
export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes)) return "—"
  if (bytes < 1024) return `${bytes} B`
  const units = ["kB", "MB", "GB"]
  let value = bytes / 1024
  let unitIndex = 0
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024
    unitIndex += 1
  }
  return `${value.toFixed(1).replace(".", ",")} ${units[unitIndex]}`
}

/** 0 -> "indexado antes do metadado existir" (mtime = 0 no ADR-057). */
export function formatUnixDate(unix: number): string {
  if (!Number.isFinite(unix) || unix <= 0) return "desconhecido"
  return new Date(unix * 1000).toLocaleString("pt-BR")
}

/** 0.7342 -> "73,4%" */
export function formatPercent(value: number): string {
  if (!Number.isFinite(value)) return "—"
  return `${(value * 100).toFixed(1).replace(".", ",")}%`
}

/** 0.8234 -> "0,8234" */
export function formatScore(value: number, digits = 4): string {
  if (!Number.isFinite(value)) return "—"
  return value.toFixed(digits).replace(".", ",")
}
