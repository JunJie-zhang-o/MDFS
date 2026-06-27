<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ path: string }>()
const emit = defineEmits<{ navigate: [path: string] }>()

const crumbs = computed(() => {
  const parts = props.path.split('/').filter(Boolean)
  return [
    { name: '根目录', path: '/' },
    ...parts.map((name, index) => ({
      name,
      path: `/${parts.slice(0, index + 1).join('/')}`,
    })),
  ]
})
</script>

<template>
  <nav class="breadcrumbs" aria-label="当前位置">
    <template v-for="(crumb, index) in crumbs" :key="crumb.path">
      <span v-if="index" class="separator">/</span>
      <button type="button" @click="emit('navigate', crumb.path)">
        {{ crumb.name }}
      </button>
    </template>
  </nav>
</template>

