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
</script>

<template>
  <div class="table-wrap">
    <table class="file-table">
      <thead>
        <tr>
          <th class="name-col" @click="emit('sort', 'name')">{{ translate(language, 'file') }}<span v-if="sortKey === 'name'">{{ sortDirection === 'asc' ? '⌃' : '⌄' }}</span></th>
          <th class="size-col" @click="emit('sort', 'size')">{{ translate(language, 'size') }}<span v-if="sortKey === 'size'">{{ sortDirection === 'asc' ? '⌃' : '⌄' }}</span></th>
          <th class="time-col" @click="emit('sort', 'modifiedAt')">{{ translate(language, 'modified') }}<span v-if="sortKey === 'modifiedAt'">{{ sortDirection === 'asc' ? '⌃' : '⌄' }}</span></th>
          <th class="actions-col">{{ translate(language, 'actions') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="loading"><td colspan="4" class="state-row"><span class="spinner" />{{ translate(language, 'loading') }}</td></tr>
        <tr v-else-if="entries.length === 0"><td colspan="4" class="state-row">{{ translate(language, 'empty') }}</td></tr>
        <tr v-for="entry in entries" v-else :key="entry.path">
          <td class="file-cell">
            <img :src="icon(entry)" alt="" class="file-icon" />
            <button v-if="entry.kind === 'directory'" type="button" class="name-link" @click="emit('navigate', entry.path)">{{ entry.name }}</button>
            <a v-else class="name-link" :href="contentURL(entry.path, 'inline')" target="_blank">{{ entry.name }}</a>
          </td>
          <td>{{ formatSize(entry.size) }}</td>
          <td>{{ formatDate(entry.modifiedAt) }}</td>
          <td class="row-actions">
            <a v-if="entry.kind === 'directory' && entry.permissions.read" class="action-btn success" :href="archiveURL(entry.path)" :title="translate(language, 'archive')" target="_blank">⇩</a>
            <a v-if="entry.kind === 'file' && entry.permissions.read" class="action-btn success" :href="contentURL(entry.path, 'attachment')" :title="translate(language, 'download')" download>⇩</a>
            <button v-if="entry.kind === 'file' && entry.permissions.read" type="button" class="action-btn success" :title="translate(language, 'qr')" @click="emit('action', 'qr', entry)">▦</button>
            <a v-if="entry.kind === 'file' && entry.previewKind === 'media'" class="action-btn success" :href="contentURL(entry.path, 'inline')" :title="translate(language, 'play')" target="_blank">▶</a>
            <button v-if="entry.permissions.write" type="button" class="action-btn warning" :title="translate(language, 'rename')" @click="emit('action', 'rename', entry)">✎</button>
            <button v-if="entry.kind === 'file' && entry.previewKind === 'text' && entry.permissions.write" type="button" class="action-btn warning" :title="translate(language, 'edit')" @click="emit('action', 'edit', entry)">▤</button>
            <button v-if="entry.permissions.delete" type="button" class="action-btn danger" :title="translate(language, 'remove')" @click="emit('action', 'delete', entry)">♲</button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
