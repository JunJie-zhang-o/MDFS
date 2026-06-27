<script setup lang="ts">
import { downloadURL } from '../api/files'
import type { FileEntry } from '../types/files'

defineProps<{ entries: FileEntry[] }>()
const emit = defineEmits<{ navigate: [path: string] }>()

function formatSize(entry: FileEntry): string {
  if (entry.type === 'directory') return '—'
  if (entry.size < 1024) return `${entry.size} B`
  if (entry.size < 1024 ** 2) return `${(entry.size / 1024).toFixed(1)} KB`
  if (entry.size < 1024 ** 3) return `${(entry.size / 1024 ** 2).toFixed(1)} MB`
  return `${(entry.size / 1024 ** 3).toFixed(1)} GB`
}

function formatDate(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value))
}
</script>

<template>
  <div class="table-wrap">
    <table>
      <thead>
        <tr>
          <th>名称</th>
          <th>大小</th>
          <th>修改时间</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="entries.length === 0">
          <td colspan="3" class="empty">此目录为空</td>
        </tr>
        <tr v-for="entry in entries" :key="entry.path">
          <td>
            <button
              v-if="entry.type === 'directory'"
              type="button"
              class="entry"
              @click="emit('navigate', entry.path)"
            >
              <span aria-hidden="true">📁</span> {{ entry.name }}
            </button>
            <a v-else class="entry" :href="downloadURL(entry.path)">
              <span aria-hidden="true">📄</span> {{ entry.name }}
            </a>
          </td>
          <td>{{ formatSize(entry) }}</td>
          <td>{{ formatDate(entry.modifiedAt) }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

