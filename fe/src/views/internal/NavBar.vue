<template>
  <div class="flex flex-col">
    <div class="p-4 text-4xl font-bold">Seymour</div>
    <ul>
      <RouterLink :to="{ name: 'subscriptions' }">
        <li :class="{ 'bg-primary': $route.name === 'subscriptions' }" class="p-4 py-2">
          Subscriptions
        </li>
      </RouterLink>
      <RouterLink :to="{ name: 'prompt' }">
        <li :class="{ 'bg-primary': $route.name === 'prompt' }" class="p-4 py-2">
          Curation Prompt
        </li>
      </RouterLink>
    </ul>
    <h1 class="p-4 py-2 font-bold">Timeline</h1>
    <ul>
      <RouterLink :to="{ name: 'timeline' }">
        <li
          :class="{ 'bg-primary': $route.name === 'timeline' && !$route.query.feed_id }"
          class="p-4 py-2"
        >
          All Feeds
        </li>
      </RouterLink>
      <RouterLink
        v-for="feed in viewer.subscriptions"
        :key="feed.feed_id"
        :to="{ name: 'timeline', query: { feed_id: feed.feed_id } }"
      >
        <li
          :class="{
            'bg-primary': $route.name === 'timeline' && $route.query.feed_id === feed.feed_id,
          }"
          class="p-4 py-2"
        >
          {{ capFirst(feed.name) }}
        </li>
      </RouterLink>
    </ul>

    <!-- Alpha disclaimer -->
    <div class="mt-auto p-4 border-t border-stone-200">
      <router-link to="/alpha" class="text-xs text-gray-500 hover:text-gray-700 transition-colors">
        Alpha v{{ version }} • Report Issues
      </router-link>
    </div>
  </div>
</template>

<script setup>
import { RouterLink } from 'vue-router'
import { computed } from 'vue'

import { getViewer, viewer } from '@/me'

// Get version info from build-time constants and package.json
const version = computed(() => {
  const gitHash = import.meta.env.VITE_GIT_HASH || 'unknown'
  return `0.1.0-alpha (${gitHash})`
})

// Capitalize the first letter of the string
function capFirst(s) {
  if (!s) return ''
  return String(s).charAt(0).toUpperCase() + String(s).slice(1)
}

getViewer()
</script>
