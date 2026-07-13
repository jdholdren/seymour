<template>
  <div
    class="w-full p-5 border border-stone-300 rounded-lg cursor-pointer hover:border-primary bg-white"
  >
    <h2 class="text-neutral-500">
      {{ props.entry.feed_name }} &#183; {{ formatDate(props.entry.publish_date) }}
    </h2>
    <h1 class="text-neutral-800 font-bold">{{ props.entry.title }}</h1>
    <p class="text-neutral-500 line-clamp-2">{{ props.entry.description }}</p>
  </div>
</template>

<script setup>
import { defineProps } from 'vue'

const props = defineProps({
  entry: {
    type: Object,
    required: true,
  },
})

function formatDate(publishDate) {
  // Handle cases where server returns 0, null, undefined, or empty string
  if (!publishDate || publishDate === 0) {
    return 'No date'
  }

  const date = new Date(publishDate)

  // Check if the date is valid
  if (isNaN(date.getTime())) {
    return 'Invalid date'
  }

  return date.toLocaleDateString()
}
</script>
