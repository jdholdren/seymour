<template>
  <div class="flex items-center justify-between px-4 py-3">
    <!-- Results text on left -->
    <div>
      <p class="text-sm text-gray-700">
        Showing
        <span class="font-medium">{{ startItem }}</span>
        to
        <span class="font-medium">{{ endItem }}</span>
        of
        <span class="font-medium">{{ totalItems }}</span>
        results
      </p>
    </div>
    <!-- Pagination buttons on right -->
    <div>
      <nav
        class="relative z-0 inline-flex rounded-md shadow-sm -space-x-px"
        aria-label="Pagination"
      >
        <!-- Page numbers -->
        <button
          v-for="page in visiblePages"
          :key="page"
          @click="goToPage(page)"
          :class="[
            'relative inline-flex items-center px-4 py-2 border text-sm font-medium',
            page === currentPage
              ? 'z-10 bg-primary border-primary text-white'
              : 'bg-white border-gray-300 text-gray-500 hover:bg-gray-50',
            page === visiblePages[0] ? 'rounded-l-md' : '',
            page === visiblePages[visiblePages.length - 1] ? 'rounded-r-md' : '',
          ]"
        >
          {{ page }}
        </button>
      </nav>
    </div>
  </div>
</template>

<script setup>
import { computed, defineProps, defineEmits } from 'vue'

const props = defineProps({
  currentPage: {
    type: Number,
    required: true,
  },
  totalItems: {
    type: Number,
    required: true,
  },
  itemsPerPage: {
    type: Number,
    default: 20,
  },
  maxVisiblePages: {
    type: Number,
    default: 5,
  },
})

const emit = defineEmits(['page-changed'])

const totalPages = computed(() => Math.ceil(props.totalItems / props.itemsPerPage))

const startItem = computed(() => {
  if (props.totalItems === 0) return 0
  return (props.currentPage - 1) * props.itemsPerPage + 1
})

const endItem = computed(() => {
  const end = props.currentPage * props.itemsPerPage
  return Math.min(end, props.totalItems)
})

const visiblePages = computed(() => {
  const pages = []
  const total = totalPages.value
  const current = props.currentPage
  const maxVisible = props.maxVisiblePages

  if (total <= maxVisible) {
    // Show all pages if we have fewer than maxVisible pages
    for (let i = 1; i <= total; i++) {
      pages.push(i)
    }
  } else {
    // Calculate which pages to show around current page
    let start = Math.max(1, current - Math.floor(maxVisible / 2))
    let end = Math.min(total, start + maxVisible - 1)

    // Adjust start if we're near the end
    if (end - start + 1 < maxVisible) {
      start = Math.max(1, end - maxVisible + 1)
    }

    for (let i = start; i <= end; i++) {
      pages.push(i)
    }
  }

  return pages
})

function goToPage(page) {
  if (page >= 1 && page <= totalPages.value && page !== props.currentPage) {
    emit('page-changed', page)
  }
}
</script>
