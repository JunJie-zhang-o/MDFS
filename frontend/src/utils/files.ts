export type FileAction = 'qr' | 'rename' | 'edit' | 'delete'
export type SortKey = 'name' | 'size' | 'modifiedAt'

export function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes}B`
  if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(2).replace(/\.00$/, '')}K`
  if (bytes < 1024 ** 3) return `${(bytes / 1024 ** 2).toFixed(2).replace(/\.00$/, '')}M`
  return `${(bytes / 1024 ** 3).toFixed(2).replace(/\.00$/, '')}G`
}

export function formatDate(value: string): string {
  const date = new Date(value)
  const pad = (number: number) => String(number).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}
