<template>
  <div class="w-auto bg-white rounded-md p-4 outline-1 outline-surface-container">
    <div class="text-lg font-bold line-clamp-1">{{ subscription.feed_name }}</div>
    <div class="text-base line-clamp-3">{{ subscription.feed_description }}</div>
    <div class="text-base">Last synced: {{ subscription.last_synced }}</div>
  </div>
</template>

<script setup>
defineProps(['subscription'])
</script>
