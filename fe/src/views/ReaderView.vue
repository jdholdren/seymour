<template>
  <div>
    <h1 class="font-bold text-5xl py-8">{{ data?.title }}</h1>
    <p>
      <RouterLink
        class="text-primary-faded text-md py-16"
        :to="{}"
        @click="goToSource"
        :replace="true"
      >
        Read on {{ host }}
      </RouterLink>
    </p>
    <div class="article-content">
      <div v-html="data?.reader_content"></div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import useApiFetch from '@/use/useApiFetch'

const route = useRoute()
const { articleID } = route.params

const { call, data } = useApiFetch('GET', `/api/feed-entries/${articleID}`)

const host = computed(() => {
  if (!data.value) return

  return new URL(data.value?.url).hostname
})

// Navigates the user to the source of the post.
function goToSource() {
  window.open(data.value?.url, '_blank')
}

call()
</script>

<style scoped>
:deep(h2) {
  font-size: 2rem;
}

:deep(p) {
  padding-top: 8px;
  padding-bottom: 8px;
}
</style>
