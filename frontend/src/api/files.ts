import type { DirectoryListing } from '../types/files'

export async function listFiles(path: string): Promise<DirectoryListing> {
  const response = await fetch(`/api/files?path=${encodeURIComponent(path)}`)
  if (!response.ok) {
    const body = (await response.json().catch(() => null)) as { error?: string } | null
    throw new Error(body?.error ?? `请求失败（${response.status}）`)
  }
  return response.json() as Promise<DirectoryListing>
}

export function downloadURL(path: string): string {
  return `/api/download?path=${encodeURIComponent(path)}`
}

