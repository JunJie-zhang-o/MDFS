<script setup lang="ts">
defineProps<{ title?: string; wide?: boolean }>()
const emit = defineEmits<{ close: [] }>()
</script>

<template>
  <div class="modal-backdrop" role="presentation" @mousedown.self="emit('close')">
    <section class="modal-card" :class="{ wide }" role="dialog" aria-modal="true" :aria-label="title">
      <header v-if="title" class="modal-header">
        <h2>{{ title }}</h2>
        <button type="button" class="modal-x" aria-label="Close" @click="emit('close')">×</button>
      </header>
      <div class="modal-body"><slot /></div>
      <footer v-if="$slots.footer" class="modal-footer"><slot name="footer" /></footer>
    </section>
  </div>
</template>
