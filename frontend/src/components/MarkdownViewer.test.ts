import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import MarkdownViewer from './MarkdownViewer.vue'

describe('MarkdownViewer.vue', () => {
  it('renders header with file name and markdown icon', () => {
    const wrapper = mount(MarkdownViewer, {
      props: {
        fileName: 'README.md',
        content: '# Hello World',
        currentPath: '/docs',
      },
    })

    expect(wrapper.find('.markdown-card-header').text()).toContain('README.md')
    expect(wrapper.find('.markdown-header-icon').exists()).toBe(true)
    expect(wrapper.find('h1').text()).toBe('Hello World')
  })

  it('renders code syntax highlighting with highlight.js', () => {
    const markdown = '```js\nconst greeting = "hello";\n```'
    const wrapper = mount(MarkdownViewer, {
      props: {
        fileName: 'README.md',
        content: markdown,
        currentPath: '/',
      },
    })

    const code = wrapper.find('code.hljs')
    expect(code.exists()).toBe(true)
    expect(code.html()).toContain('hljs-keyword')
  })

  it('renders GitHub alerts and task list checkboxes', () => {
    const markdown = '> [!NOTE]\n> This is a tip.\n\n- [ ] Task 1\n- [x] Task 2'
    const wrapper = mount(MarkdownViewer, {
      props: {
        fileName: 'README.md',
        content: markdown,
        currentPath: '/',
      },
    })

    expect(wrapper.find('.markdown-alert.markdown-alert-note').exists()).toBe(true)
    expect(wrapper.find('.task-list-item').exists()).toBe(true)
    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect(checkboxes.length).toBe(2)
  })

  it('rewrites relative image and link paths using MDFS contentURL', () => {
    const markdown = '![Diagram](./assets/arch.png)\n\n[Download Doc](./specs/spec.pdf)\n\n[External](https://example.com)'
    const wrapper = mount(MarkdownViewer, {
      props: {
        fileName: 'README.md',
        content: markdown,
        currentPath: '/project',
      },
    })

    const img = wrapper.find('img')
    expect(img.exists()).toBe(true)
    expect(img.attributes('src')).toBe('/api/v1/content?path=%2Fproject%2Fassets%2Farch.png&disposition=inline')

    const links = wrapper.findAll('a')
    const relativeLink = links.find((l) => l.text() === 'Download Doc')
    expect(relativeLink).toBeDefined()
    expect(relativeLink?.attributes('href')).toBe('/api/v1/content?path=%2Fproject%2Fspecs%2Fspec.pdf&disposition=inline')
    expect(relativeLink?.attributes('target')).toBe('_blank')

    const externalLink = links.find((l) => l.text() === 'External')
    expect(externalLink).toBeDefined()
    expect(externalLink?.attributes('href')).toBe('https://example.com')
    expect(externalLink?.attributes('target')).toBe('_blank')
  })

  it('sanitizes malicious XSS scripts via DOMPurify', () => {
    const markdown = 'Normal text <script>alert("xss")</script><img src="x" onerror="alert(1)">'
    const wrapper = mount(MarkdownViewer, {
      props: {
        fileName: 'README.md',
        content: markdown,
        currentPath: '/',
      },
    })

    expect(wrapper.find('script').exists()).toBe(false)
    const img = wrapper.find('img')
    if (img.exists()) {
      expect(img.attributes('onerror')).toBeUndefined()
    }
  })
})
