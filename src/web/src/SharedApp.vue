<script setup lang="ts">
import ProgressView from './components/ProgressView.vue';
</script>

<template>
  <div class="app-shell">
    <nav class="topbar" aria-label="Job Deck"><div class="wordmark"><span>J</span><strong>Job Deck</strong></div><span class="privacy-mark">Shared · View only</span></nav>
    <ProgressView shared />
  </div>
</template>
