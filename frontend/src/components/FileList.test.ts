import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import FileList from './FileList.vue'
import type { FileEntry } from '../types/files'
import { formatDate, formatSize } from '../utils/files'

const entry = (permissions: FileEntry['permissions']): FileEntry => ({
  name: 'hello.txt', path: '/hello.txt', kind: 'file', size: 1536,
  modifiedAt: '2026-08-24T10:20:30Z', hasChildren: false, previewKind: 'text', permissions,
})

describe('FileList', () => {
  it('formats CHFS-style sizes and stable dates', () => {
    expect(formatSize(96)).toBe('96B')
    expect(formatSize(1536)).toBe('1.50K')
    expect(formatDate('2026-08-24T10:20:30')).toBe('2026-08-24 10:20:30')
  })

  it('only renders actions allowed by permissions', () => {
    const wrapper = mount(FileList, { props: { entries: [entry({ read: true, write: false, delete: false })], language: 'zh-CN', loading: false, sortKey: 'name', sortDirection: 'asc' } })
    expect(wrapper.find('[title="下载"]').exists()).toBe(true)
    expect(wrapper.find('[title="重命名"]').exists()).toBe(false)
    expect(wrapper.find('[title="删除"]').exists()).toBe(false)
  })
})
