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
  <nav class="breadcrumbs" aria-label="Breadcrumb">
    <template v-if="path !== '/'">
      <button type="button" @click="emit('navigate', parentPath())">{{ backLabel }}</button><span class="crumb-separator">|</span>
    </template>
    <button type="button" @click="emit('navigate', '/')">{{ rootLabel }}</button>
    <template v-for="crumb in crumbs()" :key="crumb.path">
      <span class="crumb-separator">/</span>
      <button v-if="crumb.path !== path" type="button" @click="emit('navigate', crumb.path)">{{ crumb.name }}</button>
      <span v-else>{{ crumb.name }}</span>
    </template>
    <span v-if="searchLabel" class="search-crumb">{{ searchLabel }}</span>
  </nav>
</template>
