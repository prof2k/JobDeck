<script setup lang="ts">
defineProps<{
  offline: boolean;
  message: string;
}>();
</script>

<template>
  <div
    v-if="offline || message"
    class="connection-banner"
    :class="{ 'connection-banner--offline': offline }"
    role="status"
  >
    <span class="connection-banner__signal" aria-hidden="true"></span>
    <span>
      <strong>{{ offline ? 'Offline snapshot' : 'Notice' }}</strong>
      {{ message }}
    </span>
  </div>
</template>
