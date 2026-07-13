<template>
  <div
    class="flex flex-col items-center justify-center p-16 text-center bg-surface-container rounded-lg"
  >
    <PhKey :size="48" weight="duotone" class="text-gray-400 mb-4" />
    <h2 class="text-2xl font-semibold text-gray-300 mb-2">No API key configured</h2>
    <p class="text-gray-400 mb-2">
      Set the
      <code class="bg-surface-container px-2 py-1 rounded text-sm">CLAUDE_API_KEY</code> environment
      variable to enable prompt features.
    </p>
  </div>
</template>

<script setup>
import { PhKey } from '@phosphor-icons/vue'
</script>
