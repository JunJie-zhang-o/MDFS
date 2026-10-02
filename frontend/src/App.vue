<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import QrcodeVue from 'qrcode.vue'
import {
  APIError, contentURL, createDirectory, createText, deleteFile, getMeta, getSession, listFiles, login, logout,
  readText, renameFile, searchFiles, updateText, uploadFiles, type UploadTask,
} from './api/files'
import AppModal from './components/AppModal.vue'
import Breadcrumbs from './components/Breadcrumbs.vue'
import FileList from './components/FileList.vue'
import { languages, translate, type MessageKey } from './i18n'
import type { DirectoryListing, FileEntry, Language, MetaInfo, SessionInfo } from './types/files'
import type { FileAction, SortKey } from './utils/files'

type Dialog = null | 'login' | 'logout' | 'folder' | 'newText' | 'rename' | 'delete' | 'edit' | 'qr' | 'upload'

const emptyPermissions = { read: false, write: false, delete: false }
const meta = ref<MetaInfo>({ title: 'MDFS', version: 'dev', notice: '', defaultLanguage: 'zh-CN', publicURL: '', features: { webdav: true, imagePreview: true, directoryUpload: true } })
const session = ref<SessionInfo>({ authenticated: false, user: '', permissions: emptyPermissions })
const listing = ref<DirectoryListing>({ path: '/', virtualRoot: false, permissions: emptyPermissions, entries: [] })
const language = ref<Language>((localStorage.getItem('mdfs-language') as Language | null) ?? 'zh-CN')
const searchMode = ref<'local' | 'global'>(localStorage.getItem('mdfs-search-mode') === 'global' ? 'global' : 'local')
const query = ref('')
const globalEntries = ref<FileEntry[] | null>(null)
const searchTruncated = ref(false)
const searchMenuOpen = ref(false)
const languageMenuOpen = ref(false)
const loading = ref(false)
const error = ref('')
const dialog = ref<Dialog>(null)
const selected = ref<FileEntry | null>(null)
const sortKey = ref<SortKey>('name')
const sortDirection = ref<'asc' | 'desc'>('asc')
const dragActive = ref(false)
const username = ref('')
const password = ref('')
const folderName = ref('')
const textTitle = ref('')
const textContent = ref('')
const newName = ref('')
const uploadProgress = ref(0)
const uploadStatus = ref('')
const uploadTask = ref<UploadTask | null>(null)
const copied = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const folderInput = ref<HTMLInputElement | null>(null)

const t = (key: MessageKey) => translate(language.value, key)

const displayedEntries = computed(() => {
  const source = globalEntries.value ?? listing.value.entries
  const filtered = searchMode.value === 'local' && query.value
    ? source.filter((entry) => entry.name.toLocaleLowerCase().includes(query.value.toLocaleLowerCase()))
    : source
  return [...filtered].sort((left, right) => {
    let result = 0
    if (sortKey.value === 'name') {
      if (left.kind !== right.kind) {
        const dirOrder = left.kind === 'directory' ? -1 : 1
        return sortDirection.value === 'asc' ? dirOrder : -dirOrder
      }
      result = left.name.localeCompare(right.name, language.value, { numeric: true, sensitivity: 'base' })
    } else if (sortKey.value === 'size') {
      const leftSize = left.kind === 'directory' ? -1 : left.size
      const rightSize = right.kind === 'directory' ? -1 : right.size
      result = leftSize - rightSize
      if (result === 0) {
        result = left.name.localeCompare(right.name, language.value, { numeric: true, sensitivity: 'base' })
      }
    } else if (sortKey.value === 'modifiedAt') {
      const leftTime = new Date(left.modifiedAt).getTime() || 0
      const rightTime = new Date(right.modifiedAt).getTime() || 0
      result = leftTime - rightTime
      if (result === 0) {
        result = left.name.localeCompare(right.name, language.value, { numeric: true, sensitivity: 'base' })
      }
    }
    return sortDirection.value === 'asc' ? result : -result
  })
})

const qrLink = computed(() => {
  if (!selected.value) return ''
  const base = meta.value.publicURL || window.location.origin
  return new URL(contentURL(selected.value.path, 'attachment'), base.endsWith('/') ? base : `${base}/`).href
})

function hashPath(): string {
  const raw = window.location.hash.slice(1)
  if (!raw) return '/'
  try { return decodeURI(raw.startsWith('/') ? raw : `/${raw}`) } catch { return '/' }
}

function hashFor(value: string): string {
  return `#${value.split('/').map((part, index) => index === 0 ? '' : encodeURIComponent(part)).join('/') || '/'}`
}

function joinPath(parent: string, name: string): string {
  return `${parent === '/' ? '' : parent}/${name}`
}

async function load(path = hashPath()) {
  loading.value = true
  error.value = ''
  globalEntries.value = null
  searchTruncated.value = false
  try {
    listing.value = await listFiles(path)
    session.value = await getSession(listing.value.path)
  } catch (reason) {
    showError(reason)
  } finally {
    loading.value = false
  }
}

function navigate(path: string) {
  query.value = ''
  if (window.location.hash === hashFor(path)) void load(path)
  else window.location.hash = hashFor(path)
}

async function runGlobalSearch() {
  if (searchMode.value !== 'global' || !query.value.trim()) {
    globalEntries.value = null
    return
  }
  loading.value = true
  try {
    const result = await searchFiles(listing.value.path, query.value.trim())
    globalEntries.value = result.entries
    searchTruncated.value = result.truncated
  } catch (reason) {
    showError(reason)
  } finally {
    loading.value = false
  }
}

function setSearchMode(mode: 'local' | 'global') {
  searchMode.value = mode
  localStorage.setItem('mdfs-search-mode', mode)
  searchMenuOpen.value = false
  globalEntries.value = null
  if (mode === 'global' && query.value) void runGlobalSearch()
}

function setLanguage(value: Language) {
  language.value = value
  localStorage.setItem('mdfs-language', value)
  languageMenuOpen.value = false
  document.documentElement.lang = value
}

function changeSort(key: SortKey) {
  if (sortKey.value === key) {
    sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortKey.value = key
    sortDirection.value = key === 'modifiedAt' ? 'desc' : 'asc'
  }
}

function openAction(action: FileAction, entry: FileEntry) {
  selected.value = entry
  if (action === 'rename') { newName.value = ''; dialog.value = 'rename' }
  else if (action === 'delete') dialog.value = 'delete'
  else if (action === 'qr') { copied.value = false; dialog.value = 'qr' }
  else if (action === 'edit') void openEditor(entry)
}

async function submitLogin() {
  try {
    session.value = await login(username.value, password.value)
    password.value = ''
    dialog.value = null
    await load(listing.value.path)
  } catch (reason) { showError(reason) }
}

async function submitLogout() {
  try { await logout(); dialog.value = null; await load(listing.value.path) } catch (reason) { showError(reason) }
}

async function submitFolder() {
  if (!folderName.value.trim()) return
  try { await createDirectory(joinPath(listing.value.path, folderName.value.trim())); dialog.value = null; await load(listing.value.path) } catch (reason) { showError(reason) }
}

async function submitText() {
  if (!textTitle.value.trim()) return
  try { await createText(joinPath(listing.value.path, textTitle.value.trim()), textContent.value); dialog.value = null; await load(listing.value.path) } catch (reason) { showError(reason) }
}

async function submitRename() {
  if (!selected.value || !newName.value.trim()) return
  try { await renameFile(selected.value.path, newName.value.trim()); dialog.value = null; await load(listing.value.path) } catch (reason) { showError(reason) }
}

async function submitDelete() {
  if (!selected.value) return
  try { await deleteFile(selected.value.path); dialog.value = null; await load(listing.value.path) } catch (reason) { showError(reason) }
}

async function openEditor(entry: FileEntry) {
  try { textContent.value = await readText(entry.path); dialog.value = 'edit' } catch (reason) { showError(reason) }
}

async function saveEditor() {
  if (!selected.value) return
  try { await updateText(selected.value.path, textContent.value); dialog.value = null; await load(listing.value.path) } catch (reason) { showError(reason) }
}

async function copyLink() {
  try { await navigator.clipboard.writeText(qrLink.value); copied.value = true } catch { showError(new Error('Clipboard is unavailable')) }
}

async function handleFiles(files: File[]) {
  if (!files.length) return
  uploadProgress.value = 0
  uploadStatus.value = ''
  dialog.value = 'upload'
  const task = uploadFiles(listing.value.path, files, (progress) => { uploadProgress.value = progress })
  uploadTask.value = task
  try {
    await task.promise
    uploadProgress.value = 100
    uploadStatus.value = t('uploadDone')
    await load(listing.value.path)
  } catch (reason) {
    uploadStatus.value = reason instanceof APIError && reason.code === 'upload_canceled' ? t('uploadCanceled') : message(reason)
  } finally { uploadTask.value = null }
}

function onFiles(event: Event) {
  const input = event.target as HTMLInputElement
  void handleFiles(Array.from(input.files ?? []))
  input.value = ''
}

function onDrop(event: DragEvent) {
  dragActive.value = false
  if (event.dataTransfer) void filesFromDrop(event.dataTransfer).then(handleFiles)
}

async function filesFromDrop(data: DataTransfer): Promise<File[]> {
  const entries = Array.from(data.items).map((item) => item.webkitGetAsEntry()).filter((entry): entry is FileSystemEntry => entry !== null)
  if (!entries.length) return Array.from(data.files)
  const files: File[] = []
  for (const entry of entries) files.push(...await readDropEntry(entry, ''))
  return files
}

async function readDropEntry(entry: FileSystemEntry, parent: string): Promise<File[]> {
  if (entry.isFile) {
    const file = await new Promise<File>((resolve, reject) => (entry as FileSystemFileEntry).file(resolve, reject))
    Object.defineProperty(file, 'webkitRelativePath', { value: `${parent}${file.name}`, configurable: true })
    return [file]
  }
  if (!entry.isDirectory) return []
  const reader = (entry as FileSystemDirectoryEntry).createReader()
  const children: FileSystemEntry[] = []
  while (true) {
    const batch = await new Promise<FileSystemEntry[]>((resolve, reject) => reader.readEntries(resolve, reject))
    if (!batch.length) break
    children.push(...batch)
  }
  const nested: File[] = []
  for (const child of children) nested.push(...await readDropEntry(child, `${parent}${entry.name}/`))
  return nested
}

function showError(reason: unknown) { error.value = `${t('error')}: ${message(reason)}` }
function message(reason: unknown) { return reason instanceof Error ? reason.message : String(reason) }

function closeDialog() {
  if (dialog.value === 'upload' && uploadTask.value) uploadTask.value.abort()
  dialog.value = null
}

function onDocumentClick(event: MouseEvent) {
  const target = event.target as HTMLElement | null
  if (!target) return
  if (!target.closest('.language-picker')) languageMenuOpen.value = false
  if (!target.closest('.search-mode-selector')) searchMenuOpen.value = false
}

const onHashChange = () => { void load() }

onMounted(async () => {
  window.addEventListener('hashchange', onHashChange)
  document.addEventListener('click', onDocumentClick)
  try {
    meta.value = await getMeta()
    document.title = meta.value.title
    if (!localStorage.getItem('mdfs-language')) setLanguage(meta.value.defaultLanguage)
  } catch (reason) { showError(reason) }
  if (!window.location.hash) window.location.hash = '#/'
  else await load()
})

onUnmounted(() => {
  window.removeEventListener('hashchange', onHashChange)
  document.removeEventListener('click', onDocumentClick)
})
</script>

<template>
  <main id="maincontainer" class="container" @dragenter.prevent="dragActive = true" @dragover.prevent @dragleave.self="dragActive = false" @drop.prevent="onDrop">
    <div v-if="dragActive" class="drop-overlay">{{ t('dragHere') }}</div>

    <!-- 1. 最上方标题栏 (Top Header Bar) -->
    <header class="top-header">
      <div class="header-left">
        <h1 class="header-title">{{ meta.title || t('fileManagement') }}</h1>
        <p class="header-subtitle">{{ meta.notice || t('defaultNotice') }}</p>
      </div>

      <!-- 右上角：用户登录按钮 -->
      <div class="header-right">
        <button
          type="button"
          class="user-btn"
          @click="dialog = session.authenticated ? 'logout' : 'login'"
        >
          <svg class="user-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
            <circle cx="12" cy="7" r="4" />
          </svg>
          <span>{{ session.authenticated ? session.user : t('login') }}</span>
        </button>
      </div>
    </header>

    <!-- 错误警告提示 -->
    <div v-if="error" class="alert">
      <div class="alert-content">
        <svg class="alert-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10" />
          <line x1="12" y1="8" x2="12" y2="12" />
          <line x1="12" y1="16" x2="12.01" y2="16" />
        </svg>
        <span>{{ error }}</span>
      </div>
      <button type="button" class="alert-close" @click="error = ''">×</button>
    </div>

    <!-- 2. 面包屑导航卡片 -->
    <Breadcrumbs
      :path="listing.path"
      :root-label="t('root')"
      :back-label="t('parentDir')"
      :search-label="globalEntries ? `${t('searchResults')} “${query}”` : ''"
      @navigate="navigate"
    />

    <p v-if="searchTruncated" class="search-warning">{{ t('tooMany') }}</p>

    <!-- 3. 搜索与操作按钮栏 (Toolbar Row) -->
    <section class="toolbar-row">
      <!-- 搜索区 -->
      <div v-if="listing.permissions.read" class="search-wrap">
        <div class="search-input-box">
          <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="11" cy="11" r="8" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <input
            v-model="query"
            type="search"
            :placeholder="t('searchPlaceholder')"
            @keyup.enter="runGlobalSearch"
          />
          <button
            v-if="searchMode === 'global'"
            type="button"
            class="search-mode-submit"
            title="执行搜索"
            @click="runGlobalSearch"
          >
            ⌕
          </button>
          <div class="search-mode-selector">
            <button
              type="button"
              class="search-mode-btn"
              :title="searchMode === 'local' ? t('localSearch') : t('globalSearch')"
              @click.stop="searchMenuOpen = !searchMenuOpen"
            >
              <span>{{ searchMode === 'local' ? '当前' : '全局' }}</span>
              <svg class="chevron-icon-sm" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9" /></svg>
            </button>
            <div v-if="searchMenuOpen" class="dropdown search-dropdown">
              <button type="button" :class="{ active: searchMode === 'local' }" @click="setSearchMode('local')">
                {{ t('localSearch') }}
              </button>
              <button type="button" :class="{ active: searchMode === 'global' }" @click="setSearchMode('global')">
                {{ t('globalSearch') }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 操作按钮组：刷新放置在上传文件左侧 -->
      <div class="action-buttons-group">
        <!-- 刷新列表按钮 (放到上传文件左侧) -->
        <button
          type="button"
          class="btn-secondary"
          :disabled="loading"
          :title="t('refresh')"
          @click="load(listing.path)"
        >
          <svg class="btn-icon" :class="{ 'animate-spin': loading }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8" />
            <path d="M21 3v5h-5" />
            <path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16" />
            <path d="M3 21v-5h5" />
          </svg>
          <span>{{ t('refresh') }}</span>
        </button>

        <!-- 上传文件 (实心蓝底主按钮) -->
        <button
          v-if="listing.permissions.write"
          type="button"
          class="btn-primary"
          @click="fileInput?.click()"
        >
          <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
            <polyline points="17 8 12 3 7 8" />
            <line x1="12" y1="3" x2="12" y2="15" />
          </svg>
          <span>{{ t('uploadFiles') }}</span>
        </button>

        <!-- 上传目录 (若支持目录上传) -->
        <button
          v-if="listing.permissions.write && meta.features.directoryUpload"
          type="button"
          class="btn-secondary"
          @click="folderInput?.click()"
        >
          <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M4 14.899A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 2.5 8.242" />
            <path d="M12 12v9" />
            <path d="m8 17 4 4 4-4" />
          </svg>
          <span>{{ t('uploadFolder') }}</span>
        </button>

        <!-- 新建文件夹 (轮廓白底次级按钮) -->
        <button
          v-if="listing.permissions.write"
          type="button"
          class="btn-secondary"
          @click="folderName = ''; dialog = 'folder'"
        >
          <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />
            <line x1="12" y1="11" x2="12" y2="17" />
            <line x1="9" y1="14" x2="15" y2="14" />
          </svg>
          <span>{{ t('newFolder') }}</span>
        </button>

        <!-- 新建文件 (轮廓白底次级按钮) -->
        <button
          v-if="listing.permissions.write"
          type="button"
          class="btn-secondary"
          @click="textTitle = ''; textContent = ''; dialog = 'newText'"
        >
          <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
            <polyline points="14 2 14 8 20 8" />
            <line x1="12" y1="18" x2="12" y2="12" />
            <line x1="9" y1="15" x2="15" y2="15" />
          </svg>
          <span>{{ t('newFile') }}</span>
        </button>

        <input ref="fileInput" class="hidden-input" type="file" multiple @change="onFiles" />
        <input ref="folderInput" class="hidden-input" type="file" multiple webkitdirectory @change="onFiles" />
      </div>
    </section>

    <!-- 4. 文件列表卡片 (Table Card) -->
    <FileList
      :entries="displayedEntries"
      :language="language"
      :loading="loading"
      :sort-key="sortKey"
      :sort-direction="sortDirection"
      @navigate="navigate"
      @action="openAction"
      @sort="changeSort"
    />

    <!-- 5. 底部信息与右下角语言切换 (Page Footer) -->
    <footer class="page-footer">
      <p>MDFS · v{{ meta.version }}</p>

      <!-- 语言选择 (移动到右下角) -->
      <div class="language-picker">
        <button type="button" class="language-btn" @click.stop="languageMenuOpen = !languageMenuOpen">
          <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10" />
            <line x1="2" y1="12" x2="22" y2="12" />
            <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
          </svg>
          <span>{{ t('languageName') }}</span>
          <svg class="chevron-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="18 15 12 9 6 15" /></svg>
        </button>
        <div v-if="languageMenuOpen" class="dropdown language-dropdown-up">
          <button v-for="item in languages" :key="item" type="button" :class="{ active: language === item }" @click="setLanguage(item)">
            {{ translate(item, 'languageName') }}
          </button>
        </div>
      </div>
    </footer>

    <!-- 模态框区 -->
    <AppModal v-if="dialog === 'login'" :title="t('login')" @close="closeDialog">
      <form class="form-stack" @submit.prevent="submitLogin">
        <label>{{ t('username') }}<input v-model="username" autofocus autocomplete="username" /></label>
        <label>{{ t('password') }}<input v-model="password" type="password" autocomplete="current-password" /></label>
        <button type="submit" class="btn primary block">{{ t('login') }}</button>
      </form>
    </AppModal>

    <AppModal v-if="dialog === 'logout'" :title="t('logout')" @close="closeDialog">
      <p>{{ t('logout') }} {{ session.user }}?</p>
      <template #footer>
        <button type="button" class="btn default" @click="closeDialog">{{ t('cancel') }}</button>
        <button type="button" class="btn danger" @click="submitLogout">{{ t('confirm') }}</button>
      </template>
    </AppModal>

    <AppModal v-if="dialog === 'folder'" :title="t('newFolder')" @close="closeDialog">
      <form class="form-stack" @submit.prevent="submitFolder"><input v-model="folderName" autofocus /></form>
      <template #footer>
        <button type="button" class="btn default" @click="closeDialog">{{ t('cancel') }}</button>
        <button type="button" class="btn primary" @click="submitFolder">{{ t('create') }}</button>
      </template>
    </AppModal>

    <AppModal v-if="dialog === 'newText'" :title="t('newText')" wide @close="closeDialog">
      <div class="form-stack">
        <label>{{ t('title') }}<input v-model="textTitle" autofocus /></label>
        <label>{{ t('content') }}<textarea v-model="textContent" rows="14" /></label>
      </div>
      <template #footer>
        <button type="button" class="btn default" @click="closeDialog">{{ t('cancel') }}</button>
        <button type="button" class="btn primary" @click="submitText">{{ t('create') }}</button>
      </template>
    </AppModal>

    <AppModal v-if="dialog === 'rename' && selected" :title="t('rename')" @close="closeDialog">
      <div class="form-stack">
        <label>{{ t('oldName') }}<input :value="selected.name" readonly /></label>
        <label>{{ t('newName') }}<input v-model="newName" autofocus /></label>
      </div>
      <template #footer>
        <button type="button" class="btn default" @click="closeDialog">{{ t('cancel') }}</button>
        <button type="button" class="btn primary" @click="submitRename">{{ t('rename') }}</button>
      </template>
    </AppModal>

    <AppModal v-if="dialog === 'delete' && selected" :title="t('remove')" @close="closeDialog">
      <p>{{ t('deleteQuestion') }} “{{ selected.name }}”?</p>
      <template #footer>
        <button type="button" class="btn default" @click="closeDialog">{{ t('cancel') }}</button>
        <button type="button" class="btn danger" @click="submitDelete">{{ t('remove') }}</button>
      </template>
    </AppModal>

    <AppModal v-if="dialog === 'edit' && selected" :title="`${t('edit')} · ${selected.name}`" wide @close="closeDialog">
      <textarea v-model="textContent" class="editor" rows="20" />
      <template #footer>
        <button type="button" class="btn default" @click="closeDialog">{{ t('close') }}</button>
        <button type="button" class="btn primary" @click="saveEditor">{{ t('save') }}</button>
      </template>
    </AppModal>

    <AppModal v-if="dialog === 'qr' && selected" @close="closeDialog">
      <div class="qr-content"><QrcodeVue :value="qrLink" :size="268" level="L" /><p class="qr-url">{{ qrLink }}</p></div>
      <template #footer>
        <button type="button" class="btn info" @click="copyLink">{{ copied ? t('copied') : t('copyLink') }}</button>
        <button type="button" class="btn primary" @click="closeDialog">{{ t('close') }}</button>
      </template>
    </AppModal>

    <AppModal v-if="dialog === 'upload'" :title="t('uploadProgress')" @close="closeDialog">
      <div class="progress"><span :style="{ width: `${uploadProgress}%` }">{{ uploadProgress }}%</span></div><p v-if="uploadStatus">{{ uploadStatus }}</p>
      <template #footer><button type="button" class="btn default" @click="closeDialog">{{ uploadTask ? t('cancel') : t('close') }}</button></template>
    </AppModal>
  </main>
</template>

