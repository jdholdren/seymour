<template>
  <button
    @click="onClick"
    :class="{
      'bg-primary': !disabled,
      'bg-primary-faded': disabled,
      'cursor-pointer': !disabled,
      [textColor || 'text-white']: true,
    }"
    class="py-3 px-6 rounded-sm"
  >
    <span class="label">{{ label || 'unlabled' }}</span>
  </button>
</template>

<script setup>
const props = defineProps(['label', 'disabled', 'textColor'])
const emit = defineEmits(['click'])

function onClick(event) {
  if (props.disabled) return
  emit('click', event)
}
</script>
