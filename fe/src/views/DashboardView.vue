<template>
  <div class="flex flex-row min-h-screen">
    <NavBar class="w-xs border-r-1 border-stone-200 bg-surface-container" />
    <div class="p-4 flex flex-1 bg-surface px-8">
      <router-view class="max-w-2xl" v-slot="{ Component }">
        <Transition mode="out-in">
          <component :is="Component" />
        </Transition>
      </router-view>
    </div>
    <Transition>
      <GlobalError v-if="globErr" />
    </Transition>
  </div>
</template>

<script setup>
import NavBar from './internal/NavBar.vue'
import GlobalError from '../components/GlobalError.vue'
import { useGlobalError } from '@/use/globalErr'

const globErr = useGlobalError()
</script>
