<template>
  <div class="flex flex-col gap-6 w-2xl">
    <div class="py-8">
      <h1 class="text-5xl font-bold">{{ feed.name }}</h1>
      <h2 class="text-xl py-4">{{ truncatedDescription }}</h2>
    </div>

    <EmptyFeed v-if="data && data.items?.length === 0 && route.query.feed_id" />
    <EmptySubscriptions v-else-if="data && data.items?.length === 0" />

    <template v-else>
      <!-- Top pagination -->
      <PaginationControls
        v-if="data?.pagination && data.pagination.total > data.pagination.limit"
        :current-page="currentPage"
        :total-items="data.pagination.total"
        :items-per-page="data.pagination.limit"
        @page-changed="handlePageChange"
      />

      <RouterLink
        v-for="entry in data?.items"
        :key="entry.id"
        :to="`/article/${entry.entry_id}`"
        class="w-full place-self-center mb-1"
      >
        <TimelineItem :entry="entry" />
      </RouterLink>

      <!-- Bottom pagination -->
      <PaginationControls
        v-if="
          data?.pagination &&
          data.pagination.total > data.pagination.limit &&
          data?.items?.length >= 5
        "
        :current-page="currentPage"
        :total-items="data.pagination.total"
        :items-per-page="data.pagination.limit"
        @page-changed="handlePageChange"
      />
    </template>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import useApiFetch from '@/use/useApiFetch'
import { viewer } from '@/me'

import TimelineItem from './internal/TimelineItem.vue'
import PaginationControls from './internal/PaginationControls.vue'
import EmptySubscriptions from '@/components/EmptySubscriptions.vue'
import EmptyFeed from '@/components/EmptyFeed.vue'
import { computed } from 'vue'

const data = ref(null)

const route = useRoute()
const router = useRouter()

const currentPage = computed(() => {
  return parseInt(route.query.page) || 1
})

watch(route, async (_, r) => {
  getFeedEntries(r.query.feed_id, r.query.page)
})
const feed = computed(() => {
  const feedID = route.query.feed_id
  if (!feedID)
    return {
      name: 'All Feeds',
      description: 'All of your subscriptions combined into a single feed',
    }

  return viewer.value.subscriptions[feedID]
})

const truncatedDescription = computed(() => {
  const description = feed.value?.description || ''
  const maxLength = 200 // Adjust as needed

  if (description.length <= maxLength) {
    return description
  }

  return description.substring(0, maxLength).trim() + '...'
})

async function getFeedEntries(feedID, page = 1) {
  const offset = (page - 1) * 20 // Assuming 20 items per page
  const queryParams = new URLSearchParams({
    ...(feedID && { feed_id: feedID }),
    limit: '20',
    offset: offset.toString(),
  })

  const { call, data: resp } = useApiFetch('GET', `/api/timeline?${queryParams}`)
  await call()

  data.value = resp.value
}

function handlePageChange(page) {
  const query = { ...route.query }
  if (page === 1) {
    delete query.page
  } else {
    query.page = page.toString()
  }

  router.push({
    name: route.name,
    query,
  })
}

getFeedEntries(route.query.feed_id, route.query.page)
</script>
