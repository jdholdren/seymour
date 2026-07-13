<template>
  <div class="flex flex-col gap-6 w-2xl">
    <div class="py-8">
      <h1 class="text-5xl font-bold">Your Subscriptions</h1>
    </div>
    <EmptySubscriptions v-if="data?.subscriptions?.length === 0" />
    <RouterLink v-else to="/subscriptions/new" class="w-fit mb-1">
      <StyledButton label="+ New Subscription" class="" />
    </RouterLink>
    <div v-if="data?.subscriptions?.length > 0">
      <SubscriptionItem
        v-for="subscription in data?.subscriptions"
        :key="subscription.id"
        :subscription="subscription"
      />
    </div>
  </div>
</template>

<script setup>
import { RouterLink } from 'vue-router'

import StyledButton from '@/components/StyledButton.vue'

import useApiFetch from '@/use/useApiFetch'
import SubscriptionItem from './internal/SubscriptionItem.vue'
import EmptySubscriptions from '@/components/EmptySubscriptions.vue'

const { call: fetchSubs, data } = useApiFetch('GET', `/api/subscriptions`)

fetchSubs()
</script>
