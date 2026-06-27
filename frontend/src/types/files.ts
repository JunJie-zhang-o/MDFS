export type EntryType = 'directory' | 'file'

export interface FileEntry {
  name: string
  path: string
  type: EntryType
  size: number
  modifiedAt: string
}

export interface DirectoryListing {
  path: string
  entries: FileEntry[]
}

