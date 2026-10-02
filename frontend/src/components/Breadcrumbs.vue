<script setup lang="ts">
const props = defineProps<{ path: string; rootLabel: string; backLabel: string; searchLabel?: string }>()
const emit = defineEmits<{ navigate: [path: string] }>()

function crumbs() {
  const parts = props.path.split('/').filter(Boolean)
  let current = ''
  return parts.map((name) => ({ name, path: (current += `/${name}`) }))
}

function parentPath() {
  const parts = props.path.split('/').filter(Boolean)
  parts.pop()
  return `/${parts.join('/')}`
}
</script>

<template>
  <nav class="breadcrumbs-card" aria-label="Breadcrumb">
    <button
      type="button"
      class="back-btn"
      :class="{ disabled: path === '/' }"
      :disabled="path === '/'"
      @click="path !== '/' && emit('navigate', parentPath())"
    >
      <svg class="back-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
        <line x1="19" y1="12" x2="5" y2="12" />
        <polyline points="12 19 5 12 12 5" />
      </svg>
      <span>{{ backLabel }}</span>
    </button>

    <div class="crumb-divider" />

    <div class="crumb-trail">
      <button type="button" class="crumb-root" @click="emit('navigate', '/')">
        <svg class="root-icon" viewBox="0 0 24 24" fill="currentColor">
          <path d="M20 5h-8.586L9.707 3.293A1 1 0 0 0 9 3H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2z" />
        </svg>
        <span>{{ rootLabel }}</span>
      </button>

      <template v-for="crumb in crumbs()" :key="crumb.path">
        <span class="crumb-separator">/</span>
        <button v-if="crumb.path !== path" type="button" class="crumb-item" @click="emit('navigate', crumb.path)">
          {{ crumb.name }}
        </button>
        <span v-else class="crumb-current">{{ crumb.name }}</span>
      </template>

      <span v-if="searchLabel" class="search-crumb">{{ searchLabel }}</span>
    </div>
  </nav>
</template>

