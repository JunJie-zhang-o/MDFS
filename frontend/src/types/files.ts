export interface Permissions {
  read: boolean
  write: boolean
  delete: boolean
}

export type EntryKind = 'directory' | 'file'
export type PreviewKind = 'text' | 'image' | 'media' | 'document' | 'browser' | 'none'

export interface FileEntry {
  name: string
  path: string
  kind: EntryKind
  size: number
  modifiedAt: string
  hasChildren: boolean
  previewKind: PreviewKind
  permissions: Permissions
}

export interface DirectoryListing {
  path: string
  virtualRoot: boolean
  permissions: Permissions
  entries: FileEntry[]
}

export interface SessionInfo {
  authenticated: boolean
  user: string
  permissions: Permissions
}

export interface MetaInfo {
  title: string
  version: string
  notice: string
  defaultLanguage: Language
  publicURL: string
  features: { webdav: boolean; imagePreview: boolean; directoryUpload: boolean; readmeFiles?: string[] }
}

export interface SearchResult {
  entries: FileEntry[]
  truncated: boolean
}

export type Language = 'zh-CN' | 'zh-TW' | 'en'
