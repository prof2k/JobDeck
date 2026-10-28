<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import ProgressView from './components/ProgressView.vue';
import ReportShelf from './components/ReportShelf.vue';
import ReportWorkspace from './components/ReportWorkspace.vue';
import type { ProgressSnapshot } from './progress';
import type { StatusSummary } from './types';

const snapshot = ref<ProgressSnapshot | null>(null);
const authenticated = ref(false);
const route = ref(window.location.hash);
const error = ref('');
const reportsVisible = computed(() => route.value.startsWith('#/reports'));
const report = computed(() => snapshot.value?.latestReport ?? null);
const detailVisible = computed(() => report.value
  && route.value === `#/reports/${encodeURIComponent(report.value.report.id)}`);
const summary = computed<StatusSummary>(() => ({
  reports: report.value ? 1 : 0,
  jobs: report.value?.jobs.length ?? 0,
  ready: report.value?.jobs.filter((job) => job.state === 'ready').length ?? 0,
  opened: report.value?.jobs.filter((job) => job.state === 'opened').length ?? 0,
  applied: report.value?.report.appliedCount ?? 0,
  skipped: report.value?.jobs.filter((job) => job.state === 'skipped').length ?? 0,
}));

function loaded(data: ProgressSnapshot): void {
  snapshot.value = data;
  authenticated.value = true;
  error.value = '';
}

function signedOut(): void {
  authenticated.value = false;
  snapshot.value = null;
}

async function refresh(): Promise<void> {
  if (!authenticated.value || !reportsVisible.value) return;
  try {
    const response = await fetch('/api/progress', { cache: 'no-store' });
    if (response.status === 401) { signedOut(); return; }
    if (!response.ok) throw new Error('Could not refresh the report.');
    loaded(await response.json() as ProgressSnapshot);
  } catch { error.value = 'Could not refresh. Showing the last loaded report.'; }
}

function followRoute(): void {
  route.value = window.location.hash;
  void refresh();
}

function showReports(): void { window.location.hash = '#/reports'; }
function selectReport(id: string): void {
  window.location.hash = `#/reports/${encodeURIComponent(id)}`;
}

async function logout(): Promise<void> {
  const response = await fetch('/api/logout', { method: 'POST' });
  if (response.ok) signedOut();
  else error.value = 'Could not sign out. Try again.';
}

onMounted(() => {
  window.addEventListener('hashchange', followRoute);
  window.addEventListener('focus', refresh);
});
onUnmounted(() => {
  window.removeEventListener('hashchange', followRoute);
  window.removeEventListener('focus', refresh);
});
</script>

<template>
  <div class="app-shell shared-shell">
    <nav class="topbar" aria-label="Job Deck">
      <a class="wordmark" href="#/progress"><span>J</span><strong>Job Deck</strong></a>
      <div v-if="authenticated" class="topbar-links">
        <a href="#/progress" :aria-current="!reportsVisible ? 'page' : undefined">Progress</a>
        <a href="#/reports" :aria-current="reportsVisible ? 'page' : undefined">Reports</a>
      </div>
      <div class="topbar__status">
        <span class="privacy-mark">Shared · View only</span>
        <button v-if="authenticated && reportsVisible" class="quiet-button" @click="logout">Sign out</button>
      </div>
    </nav>
    <p v-if="error" class="connection-banner" role="status">{{ error }}</p>
    <ProgressView v-if="!authenticated || !reportsVisible" shared @loaded="loaded" @signed-out="signedOut" />
    <ReportWorkspace v-else-if="detailVisible && report" :detail="report" :offline="false" :saving="false" read-only @back="showReports" />
    <ReportShelf v-else :reports="report ? [report.report] : []" :summary="summary" :loading="false" shared @select="selectReport" />
  </div>
</template>
