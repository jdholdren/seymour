import { ref } from 'vue'

const globalError = ref(null)
const timeoutID = ref(null)

export function useGlobalError() {
  return globalError
}

export async function setGlobalError(error) {
  if (globalError.value) {
    globalError.value = null

    // Sleep so the other error can go away
    await new Promise((r) => setTimeout(r, 500))
  }

  globalError.value = error

  // If there's already an ongoing error, clear that
  if (timeoutID.value) {
    clearTimeout(timeoutID.value)
  }

  // Clear the error after 5s
  timeoutID.value = setTimeout(() => {
    globalError.value = null
  }, 5000)
}

export function clearGlobalError() {
  globalError.value = null
}
