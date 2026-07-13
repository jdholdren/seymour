import { ref, watch } from 'vue'
import useApiFetch from '@/use/useApiFetch'

const loaded = ref(false)
const viewer = ref(undefined)

// TODO: Set up a timer to fetch the viewer on an interval

watch(viewer, () => {
  loaded.value = !!viewer.value
})

// Fetches the current viewer and sets it on the ref
async function getViewer() {
  const { data, call } = useApiFetch('GET', '/api/viewer')
  await call()
  viewer.value = data.value
}

export { getViewer, loaded, viewer }
