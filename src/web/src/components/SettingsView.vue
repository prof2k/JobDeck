<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { connect, request } from '../api';
import { targetFor, todayIn } from '../progress';
import type { ProgressSnapshot } from '../progress';

const target = ref(0);
const ready = ref(false);
const status = ref('');
const error = ref('');
const timezone = ref(Intl.DateTimeFormat().resolvedOptions().timeZone);
let timer: ReturnType<typeof setTimeout> | undefined;
let queue = Promise.resolve();

async function load(): Promise<void> {
  try {
    await connect();
    const data = await request<ProgressSnapshot>('/api/progress');
    timezone.value = data.timezone;
    target.value = targetFor(data, todayIn(timezone.value));
    ready.value = true;
  } catch { error.value = 'Could not load settings.'; }
}

function save(): void {
  clearTimeout(timer);
  if (!Number.isInteger(target.value) || target.value < 0 || target.value > 1000) {
    error.value = 'Enter a target between 0 and 1000.';
    return;
  }
  error.value = '';
  status.value = 'Saving…';
  const value = target.value;
  timer = setTimeout(() => {
    queue = queue.then(async () => {
      try {
        await request('/api/progress', {
          method: 'POST',
          body: JSON.stringify({ action: 'target', target: value, date: todayIn(timezone.value) }),
        });
        if (target.value === value) status.value = 'Saved';
      } catch { error.value = 'Could not save the target.'; status.value = ''; }
    });
  }, 250);
}

onMounted(load);
</script>

<template>
  <main class="progress-page settings-page">
    <header class="progress-heading"><h1>Settings</h1></header>
    <p v-if="error" class="progress-error" role="alert">{{ error }}</p>
    <section class="settings-section" aria-label="Application target">
      <label>Daily application target
        <input v-model.number="target" :disabled="!ready" type="number" min="0" max="1000" @input="save" />
      </label>
      <span class="settings-status" role="status">{{ status }}</span>
      <p>Applies from today. Earlier targets stay in your history.</p>
    </section>
    <section class="settings-section"><h2>Dates</h2><p>Local time · {{ timezone }}</p></section>
  </main>
</template>
