<script setup lang="ts">
import { archiveURL, contentURL } from '../api/files'
import { translate } from '../i18n'
import type { FileEntry, Language } from '../types/files'
import { formatDate, formatSize, type FileAction, type SortKey } from '../utils/files'

defineProps<{ entries: FileEntry[]; language: Language; loading: boolean; sortKey: SortKey; sortDirection: 'asc' | 'desc' }>()
const emit = defineEmits<{ navigate: [path: string]; action: [action: FileAction, entry: FileEntry]; sort: [key: SortKey] }>()

function icon(entry: FileEntry): string {
  if (entry.kind === 'directory') return '/icons/folder.svg'
  if (entry.previewKind === 'text') return '/icons/text.svg'
  if (entry.previewKind === 'image') return '/icons/image.svg'
  if (entry.previewKind === 'media') return '/icons/media.svg'
  if (entry.previewKind === 'document') return '/icons/pdf.svg'
  return '/icons/file.svg'
}

function onRowClick(entry: FileEntry, event: MouseEvent) {
  if (entry.kind !== 'directory') return
  const target = event.target as HTMLElement | null
  if (target?.closest('.actions-col') || target?.closest('.action-btn')) return
  emit('navigate', entry.path)
}
</script>

<template>
  <div class="table-card">
    <div class="table-wrap">
      <table class="file-table">
        <thead>
          <tr>
            <th class="name-col" :title="translate(language, 'file')" @click="emit('sort', 'name')">
              <span class="th-content">
                <span>{{ translate(language, 'file') }}</span>
                <span class="sort-icon-wrap" :class="{ active: sortKey === 'name' }">
                  <svg v-if="sortKey === 'name' && sortDirection === 'asc'" class="sort-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="18 15 12 9 6 15" />
                  </svg>
                  <svg v-else-if="sortKey === 'name' && sortDirection === 'desc'" class="sort-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="6 9 12 15 18 9" />
                  </svg>
                  <svg v-else class="sort-icon inactive" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="7 15 12 20 17 15" />
                    <polyline points="7 9 12 4 17 9" />
                  </svg>
                </span>
              </span>
            </th>
            <th class="size-col" :title="translate(language, 'size')" @click="emit('sort', 'size')">
              <span class="th-content">
                <span>{{ translate(language, 'size') }}</span>
                <span class="sort-icon-wrap" :class="{ active: sortKey === 'size' }">
                  <svg v-if="sortKey === 'size' && sortDirection === 'asc'" class="sort-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="18 15 12 9 6 15" />
                  </svg>
                  <svg v-else-if="sortKey === 'size' && sortDirection === 'desc'" class="sort-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="6 9 12 15 18 9" />
                  </svg>
                  <svg v-else class="sort-icon inactive" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="7 15 12 20 17 15" />
                    <polyline points="7 9 12 4 17 9" />
                  </svg>
                </span>
              </span>
            </th>
            <th class="time-col" :title="translate(language, 'modified')" @click="emit('sort', 'modifiedAt')">
              <span class="th-content">
                <span>{{ translate(language, 'modified') }}</span>
                <span class="sort-icon-wrap" :class="{ active: sortKey === 'modifiedAt' }">
                  <svg v-if="sortKey === 'modifiedAt' && sortDirection === 'asc'" class="sort-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="18 15 12 9 6 15" />
                  </svg>
                  <svg v-else-if="sortKey === 'modifiedAt' && sortDirection === 'desc'" class="sort-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="6 9 12 15 18 9" />
                  </svg>
                  <svg v-else class="sort-icon inactive" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="7 15 12 20 17 15" />
                    <polyline points="7 9 12 4 17 9" />
                  </svg>
                </span>
              </span>
            </th>
            <th class="actions-col text-right">{{ translate(language, 'actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td colspan="4" class="state-row">
              <div class="loading-state">
                <span class="spinner" />
                <span>{{ translate(language, 'loading') }}</span>
              </div>
            </td>
          </tr>
          <tr v-else-if="entries.length === 0">
            <td colspan="4" class="state-row">
              <div class="empty-state">
                <!-- 几何描边极简文件夹，像素级对齐参考图 -->
                <svg class="empty-icon" viewBox="0 0 48 48" fill="none" stroke="currentColor">
                  <path d="M6 14C6 11.7909 7.79086 10 10 10H19.1716C20.2324 10 21.2497 10.4214 22 11.1716L24.8284 14H38C40.2091 14 42 15.7909 42 18V36C42 38.2091 40.2091 40 38 40H10C7.79086 40 6 38.2091 6 36V14Z" stroke-width="1.3" stroke-linejoin="round" />
                </svg>
                <p class="empty-text">{{ translate(language, 'emptyDir') }}</p>
              </div>
            </td>
          </tr>
          <tr
            v-for="entry in entries"
            v-else
            :key="entry.path"
            class="file-row"
            :class="{ 'is-dir': entry.kind === 'directory' }"
            @click="onRowClick(entry, $event)"
          >
            <td class="name-col-cell">
              <button
                v-if="entry.kind === 'directory'"
                type="button"
                class="entry-link dir"
                @click.stop="emit('navigate', entry.path)"
              >
                <img :src="icon(entry)" alt="" class="file-icon" />
                <span class="file-name">{{ entry.name }}</span>
              </button>
              <a
                v-else
                class="entry-link file"
                :href="contentURL(entry.path, 'inline')"
                target="_blank"
                @click.stop
              >
                <img :src="icon(entry)" alt="" class="file-icon" />
                <span class="file-name">{{ entry.name }}</span>
              </a>
            </td>
            <td class="size-cell">{{ entry.kind === 'directory' ? '—' : formatSize(entry.size) }}</td>
            <td class="time-cell">{{ formatDate(entry.modifiedAt) }}</td>
            <td class="actions-col" @click.stop>
              <div class="row-actions">
                <a
                  v-if="entry.kind === 'directory' && entry.permissions.read"
                  class="action-btn"
                  :href="archiveURL(entry.path)"
                  :title="translate(language, 'archive')"
                  target="_blank"
                >
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                    <polyline points="7 10 12 15 17 10" />
                    <line x1="12" y1="15" x2="12" y2="3" />
                  </svg>
                </a>
                <a
                  v-if="entry.kind === 'file' && entry.permissions.read"
                  class="action-btn"
                  :href="contentURL(entry.path, 'attachment')"
                  :title="translate(language, 'download')"
                  download
                >
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                    <polyline points="7 10 12 15 17 10" />
                    <line x1="12" y1="15" x2="12" y2="3" />
                  </svg>
                </a>
                <button
                  v-if="entry.kind === 'file' && entry.permissions.read"
                  type="button"
                  class="action-btn"
                  :title="translate(language, 'qr')"
                  @click="emit('action', 'qr', entry)"
                >
                  <svg viewBox="0 0 24 24" fill="currentColor">
                    <path d="M16 17V16H13V13H16V15H18V17H17V19H15V21H13V18H15V17H16ZM21 21H17V19H19V17H21V21ZM3 3H11V11H3V3ZM5 5V9H9V5H5ZM13 3H21V11H13V3ZM15 5V9H19V5H15ZM3 13H11V21H3V13ZM5 15V19H9V15H5ZM18 13H21V15H18V13ZM6 6H8V8H6V6ZM6 16H8V18H6V16ZM16 6H18V8H16V6Z" />
                  </svg>
                </button>
                <a
                  v-if="entry.kind === 'file' && entry.previewKind === 'media'"
                  class="action-btn"
                  :href="contentURL(entry.path, 'inline')"
                  :title="translate(language, 'play')"
                  target="_blank"
                >
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polygon points="5 3 19 12 5 21 5 3" />
                  </svg>
                </a>
                <button
                  v-if="entry.kind === 'file' && entry.previewKind === 'text' && entry.permissions.write"
                  type="button"
                  class="action-btn"
                  :title="translate(language, 'edit')"
                  @click="emit('action', 'edit', entry)"
                >
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                  </svg>
                </button>
                <button
                  v-if="entry.permissions.write"
                  type="button"
                  class="action-btn"
                  :title="translate(language, 'rename')"
                  @click="emit('action', 'rename', entry)"
                >
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M12 20h9" />
                    <path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z" />
                  </svg>
                </button>
                <button
                  v-if="entry.permissions.delete"
                  type="button"
                  class="action-btn danger"
                  :title="translate(language, 'remove')"
                  @click="emit('action', 'delete', entry)"
                >
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="3 6 5 6 21 6" />
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                  </svg>
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

