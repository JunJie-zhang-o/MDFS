<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listFiles } from './api/files'
import Breadcrumbs from './components/Breadcrumbs.vue'
import FileList from './components/FileList.vue'
import type { FileEntry } from './types/files'

const currentPath = ref('/')
const entries = ref<FileEntry[]>([])
const loading = ref(false)
const error = ref('')

async function navigate(path: string) {
  loading.value = true
  error.value = ''
  try {
    const listing = await listFiles(path)
    currentPath.value = listing.path
    entries.value = listing.entries
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : '无法读取目录'
  } finally {
    loading.value = false
  }
}

onMounted(() => navigate('/'))
</script>

<template>
  <main>
    <header>
      <div>
        <p class="eyebrow">HTTP FILE SERVER</p>
        <h1>MDFS</h1>
      </div>
      <span class="status">只读共享</span>
    </header>

    <section class="browser" aria-live="polite">
      <Breadcrumbs :path="currentPath" @navigate="navigate" />
      <p v-if="error" class="error">{{ error }}</p>
      <p v-else-if="loading" class="loading">正在读取目录…</p>
      <FileList v-else :entries="entries" @navigate="navigate" />
    </section>
  </main>
</template>

