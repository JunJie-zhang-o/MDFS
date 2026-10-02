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

  it('renders dash for directory size and formats file size', () => {
    const dirEntry: FileEntry = {
      name: 'folder', path: '/folder', kind: 'directory', size: 96,
      modifiedAt: '2026-08-24T10:20:30Z', hasChildren: true, previewKind: 'none',
      permissions: { read: true, write: true, delete: true },
    }
    const wrapper = mount(FileList, { props: { entries: [dirEntry, entry({ read: true, write: true, delete: true })], language: 'zh-CN', loading: false, sortKey: 'name', sortDirection: 'asc' } })
    const sizeCells = wrapper.findAll('.size-cell')
    expect(sizeCells[0].text()).toBe('—')
    expect(sizeCells[1].text()).toBe('1.50K')
  })

  it('emits sort when clicking name, size, and modifiedAt headers', async () => {
    const wrapper = mount(FileList, { props: { entries: [entry({ read: true, write: false, delete: false })], language: 'zh-CN', loading: false, sortKey: 'name', sortDirection: 'asc' } })
    await wrapper.find('.name-col').trigger('click')
    expect(wrapper.emitted('sort')?.[0]).toEqual(['name'])

    await wrapper.find('.size-col').trigger('click')
    expect(wrapper.emitted('sort')?.[1]).toEqual(['size'])

    await wrapper.find('.time-col').trigger('click')
    expect(wrapper.emitted('sort')?.[2]).toEqual(['modifiedAt'])
  })

  it('navigates when clicking directory link or row', async () => {
    const dirEntry: FileEntry = {
      name: 'folder', path: '/folder', kind: 'directory', size: 96,
      modifiedAt: '2026-08-24T10:20:30Z', hasChildren: true, previewKind: 'none',
      permissions: { read: true, write: true, delete: true },
    }
    const wrapper = mount(FileList, { props: { entries: [dirEntry], language: 'zh-CN', loading: false, sortKey: 'name', sortDirection: 'asc' } })
    await wrapper.find('.entry-link.dir').trigger('click')
    expect(wrapper.emitted('navigate')?.[0]).toEqual(['/folder'])

    await wrapper.find('.file-row.is-dir').trigger('click')
    expect(wrapper.emitted('navigate')?.[1]).toEqual(['/folder'])
  })
})
