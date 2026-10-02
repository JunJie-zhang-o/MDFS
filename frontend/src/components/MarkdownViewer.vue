<script setup lang="ts">
import { computed } from 'vue'
import DOMPurify from 'dompurify'
import hljs from 'highlight.js'
import MarkdownIt from 'markdown-it'
import alerts from 'markdown-it-github-alerts'
import taskLists from 'markdown-it-task-lists'
import { contentURL } from '../api/files'

import 'github-markdown-css/github-markdown-light.css'
import 'markdown-it-github-alerts/styles/github-base.css'
import 'markdown-it-github-alerts/styles/github-colors-light.css'
import 'highlight.js/styles/github.css'

type MarkdownItInstance = ReturnType<typeof MarkdownIt>
type RenderRule = NonNullable<MarkdownItInstance['renderer']['rules'][string]>

const props = defineProps<{
  content: string
  fileName: string
  currentPath: string
}>()

function resolveRelativePath(baseDir: string, relativePath: string): string {
  if (/^(https?:|\/\/|data:|blob:|mailto:|#)/i.test(relativePath)) {
    return relativePath
  }
  if (relativePath.startsWith('/')) {
    return relativePath
  }
  const stack = baseDir === '/' ? [] : baseDir.split('/').filter(Boolean)
  const parts = relativePath.split('/')
  for (const part of parts) {
    if (!part || part === '.') continue
    if (part === '..') {
      if (stack.length > 0) stack.pop()
    } else {
      stack.push(part)
    }
  }
  return '/' + stack.join('/')
}

function escapeHtml(str: string): string {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

const md: MarkdownItInstance = new MarkdownIt({
  html: true,
  linkify: true,
  breaks: true,
  highlight(str: string, lang: string): string {
    if (lang && hljs.getLanguage(lang)) {
      try {
        return `<pre><code class="hljs language-${lang}">${hljs.highlight(str, { language: lang, ignoreIllegals: true }).value}</code></pre>`
      } catch {
        /* ignore highlight error */
      }
    }
    return `<pre><code class="hljs">${escapeHtml(str)}</code></pre>`
  },
})

md.use(alerts)
md.use(taskLists, { enabled: true })

// 相对图片路径重写为 MDFS 资源 URL
const defaultImageRender: RenderRule =
  md.renderer.rules.image || ((tokens, idx, options, _env, self) => self.renderToken(tokens, idx, options))

const imageRule: RenderRule = (tokens, idx, options, env, self) => {
  const token = tokens[idx]
  const srcIndex = token.attrIndex('src')
  if (srcIndex >= 0 && token.attrs) {
    const src = String(token.attrs[srcIndex][1])
    if (!/^(https?:|\/\/|data:|blob:)/i.test(src)) {
      const resolved = resolveRelativePath(props.currentPath, src)
      token.attrs[srcIndex][1] = contentURL(resolved, 'inline')
    }
  }
  return defaultImageRender(tokens, idx, options, env, self)
}
md.renderer.rules.image = imageRule

// 相对超链接路径重写为新标签页打开对应 MDFS 资源
const defaultLinkOpen: RenderRule =
  md.renderer.rules.link_open || ((tokens, idx, options, _env, self) => self.renderToken(tokens, idx, options))

const linkOpenRule: RenderRule = (tokens, idx, options, env, self) => {
  const token = tokens[idx]
  const hrefIndex = token.attrIndex('href')
  if (hrefIndex >= 0 && token.attrs) {
    const href = String(token.attrs[hrefIndex][1])
    if (!/^(https?:|\/\/|mailto:|#)/i.test(href)) {
      const resolved = resolveRelativePath(props.currentPath, href)
      token.attrs[hrefIndex][1] = contentURL(resolved, 'inline')
    }
  }
  token.attrSet('target', '_blank')
  token.attrSet('rel', 'noopener noreferrer')
  return defaultLinkOpen(tokens, idx, options, env, self)
}
md.renderer.rules.link_open = linkOpenRule

const renderedHtml = computed(() => {
  if (!props.content) return ''
  const dirty = md.render(props.content)
  return DOMPurify.sanitize(dirty, {
    USE_PROFILES: { html: true, svg: true },
    ADD_TAGS: ['svg', 'path'],
    ADD_ATTR: ['target', 'rel', 'checked', 'disabled', 'aria-hidden'],
  })
})
</script>

<template>
  <section class="markdown-card">
    <header class="markdown-card-header">
      <svg class="markdown-header-icon" viewBox="0 0 16 16" width="16" height="16" fill="currentColor" aria-hidden="true">
        <path d="M0 1.75A.75.75 0 0 1 .75 1h4.253c1.227 0 2.317.59 3 1.501A3.743 3.743 0 0 1 11.006 1h4.244a.75.75 0 0 1 .75.75v10.5a.75.75 0 0 1-.75.75h-4.244a2.25 2.25 0 0 0-1.506.578l-.5.438a.75.75 0 0 1-.994 0l-.5-.438a2.25 2.25 0 0 0-1.506-.578H.75a.75.75 0 0 1-.75-.75Zm1.5.88v9.006h3.503a3.75 3.75 0 0 1 2.25.75V3.535a2.25 2.25 0 0 0-1.5-.535ZM8.75 12.386c.6-.451 1.343-.75 2.25-.75h3.5V2.63h-3.5a2.25 2.25 0 0 0-1.5.535Z" />
      </svg>
      <span class="markdown-header-title">{{ fileName }}</span>
    </header>
    <div class="markdown-card-body">
      <article class="markdown-body" v-html="renderedHtml" />
    </div>
  </section>
</template>

<style scoped>
.markdown-card {
  margin-top: 20px;
  background: var(--color-card-bg, #ffffff);
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: 12px;
  box-shadow: var(--shadow-2xs, 0 1px 2px 0 rgb(0 0 0 / 0.04));
  overflow: hidden;
}

.markdown-card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 20px;
  background: #f8fafc;
  border-bottom: 1px solid var(--color-border, #e2e8f0);
}

.markdown-header-icon {
  color: var(--color-text-muted, #64748b);
  flex-shrink: 0;
}

.markdown-header-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-main, #0f172a);
}

.markdown-card-body {
  padding: 24px 28px;
  background: #ffffff;
}

.markdown-body {
  font-size: 14px;
  line-height: 1.6;
  background-color: transparent !important;
  color: var(--color-text-main, #0f172a);
}

/* 适配移动端内边距 */
@media (max-width: 640px) {
  .markdown-card-body {
    padding: 16px 16px;
  }
}
</style>
