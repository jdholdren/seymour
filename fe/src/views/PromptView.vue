<template>
  <div class="max-w-4xl">
    <h1 class="text-5xl font-bold mb-6">Curation Prompt</h1>

    <EmptyPromptKey v-if="viewer?.has_prompt_key === false" />

    <!-- Instructions section (formerly ConsentView) -->
    <template v-else>
      <div class="mb-8 p-6 bg-surface-container rounded-lg">
        <h2 class="text-xl font-semibold mb-4">How AI Filtering Works</h2>
        <p class="mb-4">
          Seymour uses Claude AI to intelligently filter your RSS feeds based on your preferences.
          You provide custom prompts that act as filtering criteria, and Claude decides which posts
          to show or hide. This creates a personalized, curated timeline that matches your
          interests.
        </p>
        <p class="mb-4">
          To curate your feed, simply describe what you'd like to see...or not see. For example:
        </p>
        <div class="space-y-3">
          <div class="bg-surface p-3 rounded border-l-4 border-primary">
            <code class="text-sm"
              >"Show me recipes for healthy meals, but not ones that include garlic."</code
            >
          </div>
        </div>
        <p class="mt-4 text-sm text-gray-600">
          Your feed is in your control! As with any prompt, being incredibly descriptive or
          providing examples will yield more favorable results.
        </p>
      </div>

      <!-- Form section (formerly EditPromptView) -->
      <form @submit.prevent="handleSubmit" class="mb-6">
        <label for="prompt" class="block text-lg font-medium mb-3"> Your Filtering Prompt </label>
        <textarea
          id="prompt"
          v-model="promptText"
          class="w-full h-64 px-4 py-3 border border-gray-300 rounded-lg resize-none focus:ring-2 focus:ring-primary focus:border-primary"
          placeholder="Enter your filtering prompt here... For example: 'Show me posts about technology, programming, and AI, but hide posts about celebrity gossip or sports.'"
        />

        <!-- Character count and validation feedback -->
        <div class="flex justify-between items-start mt-2">
          <div class="flex flex-col">
            <span
              class="text-sm"
              :class="{
                'text-red-600': promptText.length > maxCharacters,
                'text-gray-600': promptText.length <= maxCharacters,
              }"
            >
              {{ promptText.length }} / {{ maxCharacters }} characters
            </span>
            <div v-if="promptError" class="text-sm text-red-600 mt-1">
              {{ promptError.message }}
            </div>
          </div>

          <button
            type="submit"
            :disabled="isSubmitting"
            class="px-6 py-2 bg-primary text-white rounded-lg hover:bg-primary-dark disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {{ isSubmitting ? 'Saving...' : 'Save Prompt' }}
          </button>
        </div>
      </form>

      <!-- Success feedback (formerly SuccessView) -->
      <div v-if="showSuccess" class="p-4 bg-green-100 text-green-800 rounded-lg flex items-center">
        <svg class="w-5 h-5 mr-2 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
          <path
            fill-rule="evenodd"
            d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
            clip-rule="evenodd"
          />
        </svg>
        <span>Success! Your prompt has been saved and will filter future posts.</span>
      </div>

      <!-- Error feedback -->
      <div v-if="submitErr" class="p-4 bg-red-100 text-red-800 rounded-lg flex items-center">
        <svg class="w-5 h-5 mr-2 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
          <path
            fill-rule="evenodd"
            d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
            clip-rule="evenodd"
          />
        </svg>
        <span>{{ submitErr.message || 'Failed to save prompt. Please try again.' }}</span>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { viewer } from '@/me'
import useApiFetch from '@/use/useApiFetch'
import EmptyPromptKey from '@/components/EmptyPromptKey.vue'

const maxCharacters = 5000

const promptText = ref(viewer.value?.prompt || '')
const promptError = ref(null)
const showSuccess = ref(false)

// API hooks
const {
  call: postPrompt,
  fetching: isSubmitting,
  error: submitErr,
  statusCode,
} = useApiFetch('PUT', '/api/prompt')

// Submit handler
async function handleSubmit() {
  await postPrompt({ prompt: promptText.value })

  if (statusCode.value != 200) return
  showSuccess.value = true

  // Auto-hide success message after 5 seconds
  setTimeout(() => {
    showSuccess.value = false
  }, 5000)
}
</script>
