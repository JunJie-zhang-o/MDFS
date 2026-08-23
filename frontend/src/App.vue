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
    if (left.kind !== right.kind) return left.kind === 'directory' ? -1 : 1
    let result = 0
    if (sortKey.value === 'name') result = left.name.localeCompare(right.name, language.value, { sensitivity: 'base' })
    else if (sortKey.value === 'size') result = left.size - right.size
    else result = new Date(left.modifiedAt).getTime() - new Date(right.modifiedAt).getTime()
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
  if (sortKey.value === key) sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
  else { sortKey.value = key; sortDirection.value = 'asc' }
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

const onHashChange = () => { void load() }

onMounted(async () => {
  window.addEventListener('hashchange', onHashChange)
  try {
    meta.value = await getMeta()
    document.title = meta.value.title
    if (!localStorage.getItem('mdfs-language')) setLanguage(meta.value.defaultLanguage)
  } catch (reason) { showError(reason) }
  if (!window.location.hash) window.location.hash = '#/'
  else await load()
})

onUnmounted(() => window.removeEventListener('hashchange', onHashChange))
</script>

<template>
  <main id="maincontainer" class="container" @dragenter.prevent="dragActive = true" @dragover.prevent @dragleave.self="dragActive = false" @drop.prevent="onDrop">
    <div v-if="dragActive" class="drop-overlay">{{ t('dragHere') }}</div>
    <section class="toolbar">
      <button v-if="listing.permissions.write" type="button" class="btn default" @click="fileInput?.click()">{{ t('uploadFiles') }}</button>
      <button v-if="listing.permissions.write && meta.features.directoryUpload" type="button" class="btn default" @click="folderInput?.click()">{{ t('uploadFolder') }}</button>
      <button v-if="listing.permissions.write" type="button" class="btn default" @click="textTitle = ''; textContent = ''; dialog = 'newText'">{{ t('newText') }}</button>
      <button v-if="listing.permissions.write" type="button" class="btn default" @click="folderName = ''; dialog = 'folder'">{{ t('newFolder') }}</button>
      <input ref="fileInput" class="hidden-input" type="file" multiple @change="onFiles" />
      <input ref="folderInput" class="hidden-input" type="file" multiple webkitdirectory @change="onFiles" />
      <div v-if="listing.permissions.read" class="search-group">
        <input v-model="query" type="search" :placeholder="`${searchMode === 'local' ? t('localSearch') : t('globalSearch')}...`" @keyup.enter="runGlobalSearch" />
        <button v-if="searchMode === 'global'" type="button" class="btn default search-submit" @click="runGlobalSearch">⌕</button>
        <button type="button" class="btn default search-menu-button" @click="searchMenuOpen = !searchMenuOpen">⌄</button>
        <div v-if="searchMenuOpen" class="dropdown search-dropdown">
          <button type="button" @click="setSearchMode('local')">{{ t('localSearch') }}</button>
          <button type="button" @click="setSearchMode('global')">{{ t('globalSearch') }}</button>
        </div>
      </div>
      <button type="button" class="btn default user-button" @click="dialog = session.authenticated ? 'logout' : 'login'">{{ session.authenticated ? session.user : t('login') }}</button>
    </section>

    <p v-if="meta.notice" class="notice">{{ meta.notice }}</p>
    <div v-if="error" class="alert"><span>{{ error }}</span><button type="button" @click="error = ''">×</button></div>
    <Breadcrumbs :path="listing.path" :root-label="t('root')" :back-label="t('back')" :search-label="globalEntries ? `${t('searchResults')} “${query}”` : ''" @navigate="navigate" />
    <p v-if="searchTruncated" class="search-warning">{{ t('tooMany') }}</p>
    <FileList :entries="displayedEntries" :language="language" :loading="loading" :sort-key="sortKey" :sort-direction="sortDirection" @navigate="navigate" @action="openAction" @sort="changeSort" />

    <footer class="page-footer">
      <p>v{{ meta.version }}</p>
      <div class="language-picker">
        <button type="button" @click="languageMenuOpen = !languageMenuOpen">{{ t('languageName') }}⌃</button>
        <div v-if="languageMenuOpen" class="dropdown language-dropdown">
          <button v-for="item in languages" :key="item" type="button" @click="setLanguage(item)">{{ translate(item, 'languageName') }}</button>
        </div>
      </div>
    </footer>

    <AppModal v-if="dialog === 'login'" :title="t('login')" @close="closeDialog">
      <form class="form-stack" @submit.prevent="submitLogin">
        <label>{{ t('username') }}<input v-model="username" autofocus autocomplete="username" /></label>
        <label>{{ t('password') }}<input v-model="password" type="password" autocomplete="current-password" /></label>
        <button type="submit" class="btn primary block">{{ t('login') }}</button>
      </form>
    </AppModal>

    <AppModal v-if="dialog === 'logout'" :title="t('logout')" @close="closeDialog">
      <p>{{ t('logout') }} {{ session.user }}?</p>
      <template #footer><button type="button" class="btn default" @click="closeDialog">{{ t('cancel') }}</button><button type="button" class="btn danger" @click="submitLogout">{{ t('confirm') }}</button></template>
    </AppModal>

    <AppModal v-if="dialog === 'folder'" :title="t('newFolder')" @close="closeDialog">
      <form class="form-stack" @submit.prevent="submitFolder"><input v-model="folderName" autofocus /></form>
      <template #footer><button type="button" class="btn default" @click="closeDialog">{{ t('cancel') }}</button><button type="button" class="btn primary" @click="submitFolder">{{ t('create') }}</button></template>
    </AppModal>

    <AppModal v-if="dialog === 'newText'" :title="t('newText')" wide @close="closeDialog">
      <div class="form-stack"><label>{{ t('title') }}<input v-model="textTitle" autofocus /></label><label>{{ t('content') }}<textarea v-model="textContent" rows="14" /></label></div>
      <template #footer><button type="button" class="btn default" @click="closeDialog">{{ t('cancel') }}</button><button type="button" class="btn primary" @click="submitText">{{ t('create') }}</button></template>
    </AppModal>

    <AppModal v-if="dialog === 'rename' && selected" :title="t('rename')" @close="closeDialog">
      <div class="form-stack"><label>{{ t('oldName') }}<input :value="selected.name" readonly /></label><label>{{ t('newName') }}<input v-model="newName" autofocus /></label></div>
      <template #footer><button type="button" class="btn default" @click="closeDialog">{{ t('cancel') }}</button><button type="button" class="btn primary" @click="submitRename">{{ t('rename') }}</button></template>
    </AppModal>

    <AppModal v-if="dialog === 'delete' && selected" :title="t('remove')" @close="closeDialog">
      <p>{{ t('deleteQuestion') }} “{{ selected.name }}”?</p>
      <template #footer><button type="button" class="btn default" @click="closeDialog">{{ t('cancel') }}</button><button type="button" class="btn danger" @click="submitDelete">{{ t('remove') }}</button></template>
    </AppModal>

    <AppModal v-if="dialog === 'edit' && selected" :title="`${t('edit')} · ${selected.name}`" wide @close="closeDialog">
      <textarea v-model="textContent" class="editor" rows="20" />
      <template #footer><button type="button" class="btn default" @click="closeDialog">{{ t('close') }}</button><button type="button" class="btn primary" @click="saveEditor">{{ t('save') }}</button></template>
    </AppModal>

    <AppModal v-if="dialog === 'qr' && selected" @close="closeDialog">
      <div class="qr-content"><QrcodeVue :value="qrLink" :size="268" level="L" /><p class="qr-url">{{ qrLink }}</p></div>
      <template #footer><button type="button" class="btn info" @click="copyLink">{{ copied ? t('copied') : t('copyLink') }}</button><button type="button" class="btn primary" @click="closeDialog">{{ t('close') }}</button></template>
    </AppModal>

    <AppModal v-if="dialog === 'upload'" :title="t('uploadProgress')" @close="closeDialog">
      <div class="progress"><span :style="{ width: `${uploadProgress}%` }">{{ uploadProgress }}%</span></div><p v-if="uploadStatus">{{ uploadStatus }}</p>
      <template #footer><button type="button" class="btn default" @click="closeDialog">{{ uploadTask ? t('cancel') : t('close') }}</button></template>
    </AppModal>
  </main>
</template>
