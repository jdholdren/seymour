<template>
  <div>
    <div>
      <label :for="name" class="block text-sm/6 font-medium text-gray-600">{{ label }}</label>
      <div class="mt-2">
        <div
          class="flex items-center rounded-md bg-white/5 pl-3 outline-1 -outline-offset-1 outline-gray-600 has-[input:focus-within]:outline-2 has-[input:focus-within]:-outline-offset-2 has-[input:focus-within]:outline-indigo-500"
        >
          <input
            :id="name"
            type="text"
            v-model="url"
            :name="label"
            :placeholder="placeholder"
            class="block min-w-0 grow py-1.5 m-auto text-base text-black placeholder:text-gray-500 focus:outline-none sm:text-sm/6"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
const url = defineModel()
defineProps(['label', 'name', 'placeholder'])
</script>
