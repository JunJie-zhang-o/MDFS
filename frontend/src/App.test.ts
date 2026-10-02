import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import App from './App.vue'

// Mock the fetch APIs used in App.vue
vi.stubGlobal('fetch', vi.fn((url: string) => {
  if (url.includes('/api/v1/meta')) {
    return Promise.resolve({
      ok: true,
      status: 200,
      json: () => Promise.resolve({
        title: 'MDFS',
        version: '1.0.0',
        notice: '测试公告',
        defaultLanguage: 'zh-CN',
        publicURL: '',
        features: { webdav: true, imagePreview: true, directoryUpload: true },
      }),
    })
  }
  if (url.includes('/api/v1/session')) {
    return Promise.resolve({
      ok: true,
      status: 200,
      json: () => Promise.resolve({
        authenticated: true,
        user: 'admin',
        permissions: { read: true, write: true, delete: true },
      }),
    })
  }
  if (url.includes('/api/v1/files')) {
    return Promise.resolve({
      ok: true,
      status: 200,
      json: () => Promise.resolve({
        path: '/',
        virtualRoot: false,
        permissions: { read: true, write: true, delete: true },
        entries: [
          {
            name: 'test-folder',
            path: '/test-folder',
            kind: 'directory',
            size: 96,
            modifiedAt: '2026-09-25T11:00:00Z',
            hasChildren: false,
            previewKind: 'none',
            permissions: { read: true, write: true, delete: true },
          },
          {
            name: 'welcome.txt',
            path: '/welcome.txt',
            kind: 'file',
            size: 85,
            modifiedAt: '2026-09-25T10:00:00Z',
            hasChildren: false,
            previewKind: 'text',
            permissions: { read: true, write: true, delete: true },
          },
        ],
      }),
    })
  }
  if (url.includes('/api/v1/content')) {
    return Promise.resolve({
      ok: true,
      status: 200,
      text: () => Promise.resolve('# MDFS Readme\nWelcome to MDFS!'),
    })
  }
  return Promise.resolve({
    ok: true,
    status: 200,
    json: () => Promise.resolve({}),
  })
}))

describe('App.vue', () => {
  it('mounts cleanly and renders top header, breadcrumbs, search, and table', async () => {
    const wrapper = mount(App)
    await new Promise((resolve) => setTimeout(resolve, 50))

    // Top Header: Title & subtitle
    expect(wrapper.find('.header-title').exists()).toBe(true)
    expect(wrapper.find('.header-subtitle').exists()).toBe(true)

    // Top Right: User login button
    expect(wrapper.find('.user-btn').exists()).toBe(true)

    // Breadcrumbs card
    expect(wrapper.find('.breadcrumbs-card').exists()).toBe(true)
    expect(wrapper.find('.back-btn').exists()).toBe(true)

    // Toolbar row: Refresh, Upload, New Folder, New File
    expect(wrapper.find('.toolbar-row').exists()).toBe(true)
    expect(wrapper.find('.action-buttons-group').exists()).toBe(true)

    // Table card
    expect(wrapper.find('.table-card').exists()).toBe(true)
    expect(wrapper.find('.file-table').exists()).toBe(true)

    // Footer with language picker
    expect(wrapper.find('.page-footer').exists()).toBe(true)
    expect(wrapper.find('.language-picker').exists()).toBe(true)
  })

  it('sorts entries when clicking name, size, and modifiedAt headers', async () => {
    const wrapper = mount(App)
    await new Promise((resolve) => setTimeout(resolve, 50))

    const getNames = () => wrapper.findAll('.file-name').map((el) => el.text())

    // Initial: name asc (test-folder, welcome.txt)
    expect(getNames()).toEqual(['test-folder', 'welcome.txt'])

    // Click name -> name desc (welcome.txt, test-folder)
    await wrapper.find('.name-col').trigger('click')
    expect(getNames()).toEqual(['welcome.txt', 'test-folder'])

    // Click size -> size asc (dirs size -1 first: test-folder, welcome.txt)
    await wrapper.find('.size-col').trigger('click')
    expect(getNames()).toEqual(['test-folder', 'welcome.txt'])

    // Click size -> size desc (welcome.txt 85B first, test-folder second)
    await wrapper.find('.size-col').trigger('click')
    expect(getNames()).toEqual(['welcome.txt', 'test-folder'])

    // Click time -> modifiedAt desc (test-folder 11:00 first, welcome.txt 10:00 second)
    await wrapper.find('.time-col').trigger('click')
    expect(getNames()).toEqual(['test-folder', 'welcome.txt'])

    // Click time -> modifiedAt asc (welcome.txt 10:00 first, test-folder 11:00 second)
    await wrapper.find('.time-col').trigger('click')
    expect(getNames()).toEqual(['welcome.txt', 'test-folder'])
  })

  it('renders markdown card when directory contains README.md', async () => {
    // Override fetch to include README.md in listing
    const originalFetch = globalThis.fetch
    vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/api/v1/files')) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: () => Promise.resolve({
            path: '/with-readme',
            virtualRoot: false,
            permissions: { read: true, write: true, delete: true },
            entries: [
              {
                name: 'README.md',
                path: '/with-readme/README.md',
                kind: 'file',
                size: 120,
                modifiedAt: '2026-09-25T12:00:00Z',
                hasChildren: false,
                previewKind: 'text',
                permissions: { read: true, write: true, delete: true },
              },
            ],
          }),
        }) as any
      }
      if (url.includes('/api/v1/content')) {
        return Promise.resolve({
          ok: true,
          status: 200,
          text: () => Promise.resolve('# MDFS Readme\nWelcome to MDFS!'),
        }) as any
      }
      return originalFetch(input)
    }))

    const wrapper = mount(App)
    await new Promise((resolve) => setTimeout(resolve, 80))

    expect(wrapper.find('.markdown-card').exists()).toBe(true)
    expect(wrapper.find('.markdown-header-title').text()).toBe('README.md')
    expect(wrapper.find('.markdown-body').text()).toContain('MDFS Readme')
    expect(wrapper.find('.markdown-body').text()).toContain('Welcome to MDFS!')

    vi.stubGlobal('fetch', originalFetch)
  })
})

