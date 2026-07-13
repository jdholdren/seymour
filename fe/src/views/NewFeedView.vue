<template>
  <div class="flex-1 w-full">
    <div class="py-8">
      <h1 class="text-5xl font-bold">Add a new subscription</h1>
    </div>
    <TextInput
      name="url"
      label="URL"
      placeholder="https://example.com/feeds.xml"
      v-model="url"
      class="max-w-4xl"
    />
    <p class="text-red-600">{{ error?.message }}</p>
    <p class="text-red-600">{{ urlError }}</p>
    <div v-if="fetching">
      <VueSpinner class="my-8" />
    </div>
    <StyledButton
      v-else
      id="submit"
      label="Subscribe"
      :disabled="url.length == 0"
      @click="onSubmit"
      class="mt-2"
    />
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import useApiFetch from '@/use/useApiFetch'
import { useRouter } from 'vue-router'
import { VueSpinner } from 'vue3-spinners'

import StyledButton from '@/components/StyledButton.vue'
import TextInput from '@/components/TextInput.vue'

import { getViewer } from '@/me'

const url = ref('')

const { fetching, call: submit, error, statusCode } = useApiFetch('POST', `/api/subscriptions`)
const router = useRouter()

async function onSubmit() {
  if (!url.value) return
  error.value = undefined

  await submit({ feed_url: url.value })

  if (statusCode.value >= 200 && statusCode.value < 300) {
    await getViewer()

    router.push({ name: 'subscriptions' })
  }
}

const regex = new RegExp(
  /[-a-zA-Z0-9@:%._+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_+.~#?&//=]*)/gi,
)
const urlError = computed(() => {
  if (!url.value || url.value.match(regex)) return undefined
  return 'Invalid URL'
})
</script>
