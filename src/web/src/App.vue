<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue';
import {
  connect,
  fetchReport,
  fetchReports,
  fetchSummary,
  saveFeedback,
  updateState,
} from './api';
import ConnectionBanner from './components/ConnectionBanner.vue';
import ReportShelf from './components/ReportShelf.vue';
import ReportWorkspace from './components/ReportWorkspace.vue';
import ProgressView from './components/ProgressView.vue';
import SettingsView from './components/SettingsView.vue';
import { readSnapshot, writeSnapshot } from './offlineStore';
import type {
  FeedbackVerdict,
  JobState,
  ReportDetail,
  ReportJob,
  ReportListItem,
  StatusSummary,
} from './types';

const reports = ref<ReportListItem[]>([]);
const summary = ref<StatusSummary | null>(null);
const currentReport = ref<ReportDetail | null>(null);
const loading = ref(true);
const saving = ref(false);
const offline = ref(false);
const message = ref('');
const progressVisible = ref(window.location.hash === '#/progress');
const settingsVisible = ref(window.location.hash === '#/settings');

function reportIdFromHash(): string | null {
  const match = window.location.hash.match(/^#\/reports\/(.+)$/);
  return match ? decodeURIComponent(match[1]) : null;
}

async function loadShelf(): Promise<void> {
  loading.value = true;
  message.value = '';
  try {
    await connect();
    const [reportItems, status] = await Promise.all([fetchReports(), fetchSummary()]);
    reports.value = reportItems;
    summary.value = status;
    offline.value = false;
    await Promise.all([
      writeSnapshot('reports', reportItems),
      writeSnapshot('summary', status),
    ]);
  } catch {
    const [cachedReports, cachedSummary] = await Promise.all([
      readSnapshot<ReportListItem[]>('reports'),
      readSnapshot<StatusSummary>('summary'),
    ]);
    reports.value = cachedReports ?? [];
    summary.value = cachedSummary;
    offline.value = true;
    message.value = reports.value.length > 0
      ? 'Showing the last saved data. Tell your job-search agent to start Job Deck for updates.'
      : 'No cached data is available. Tell your job-search agent to start Job Deck.';
  } finally {
    loading.value = false;
  }
}

async function loadReport(reportId: string): Promise<void> {
  loading.value = true;
  message.value = '';
  try {
    if (!offline.value) await connect();
    const detail = await fetchReport(reportId);
    currentReport.value = detail;
    offline.value = false;
    await writeSnapshot(`report:${reportId}`, detail);
  } catch {
    const cached = await readSnapshot<ReportDetail>(`report:${reportId}`);
    currentReport.value = cached;
    offline.value = true;
    message.value = cached
      ? 'This is a read-only saved report. Ask your job-search agent to start Job Deck to refresh it.'
      : 'This report has not been cached on this browser.';
  } finally {
    loading.value = false;
  }
}

async function followRoute(): Promise<void> {
  progressVisible.value = window.location.hash === '#/progress';
  settingsVisible.value = window.location.hash === '#/settings';
  if (progressVisible.value || settingsVisible.value) { currentReport.value = null; return; }
  const reportId = reportIdFromHash();
  if (reportId) {
    await loadReport(reportId);
    return;
  }
  currentReport.value = null;
  await loadShelf();
}

function selectReport(reportId: string): void {
  window.location.hash = `#/reports/${encodeURIComponent(reportId)}`;
}

function showShelf(): void {
  window.location.hash = '#/';
}

async function handleState(job: ReportJob, state: JobState): Promise<void> {
  if (!currentReport.value || offline.value) return;
  saving.value = true;
  message.value = '';
  try {
    await updateState(job.id, state);
    job.state = state;
    await writeSnapshot(`report:${currentReport.value.report.id}`, currentReport.value);
  } catch (error) {
    message.value = error instanceof Error ? error.message : 'Could not update this job.';
  } finally {
    saving.value = false;
  }
}

async function handleFeedback(payload: {
  job: ReportJob;
  verdict: FeedbackVerdict;
  reason: string;
  note: string;
}): Promise<void> {
  if (!currentReport.value || offline.value) return;
  saving.value = true;
  message.value = '';
  try {
    await saveFeedback({
      jobId: payload.job.id,
      reportId: currentReport.value.report.id,
      verdict: payload.verdict,
      reason: payload.reason,
      note: payload.note,
    });
    payload.job.feedbackVerdict = payload.verdict;
    payload.job.feedbackReason = payload.reason;
    payload.job.feedbackNote = payload.note;
    await writeSnapshot(`report:${currentReport.value.report.id}`, currentReport.value);
    message.value = 'Feedback saved for your job-search agent to review.';
  } catch (error) {
    message.value = error instanceof Error ? error.message : 'Could not save feedback.';
  } finally {
    saving.value = false;
  }
}

function registerServiceWorker(): void {
  if ('serviceWorker' in navigator) {
    navigator.serviceWorker.register('/sw.js').catch(() => {
      // Offline snapshots still work in an open tab when registration is unavailable.
    });
  }
}

onMounted(async () => {
  registerServiceWorker();
  window.addEventListener('hashchange', followRoute);
  await followRoute();
});

onBeforeUnmount(() => {
  window.removeEventListener('hashchange', followRoute);
});
</script>

<template>
  <div class="app-shell">
    <nav class="topbar" aria-label="Job Deck">
      <button class="wordmark" type="button" @click="showShelf">
        <span aria-hidden="true">J</span>
        <strong>Job Deck</strong>
      </button>
      <div class="topbar-links">
        <a href="#/" :aria-current="!progressVisible && !settingsVisible ? 'page' : undefined">Reports</a>
        <a href="#/progress" :aria-current="progressVisible ? 'page' : undefined">Progress</a>
        <a href="#/settings" :aria-current="settingsVisible ? 'page' : undefined">Settings</a>
      </div>
      <div class="topbar__status">
        <span class="privacy-mark">Local only</span>
        <span class="connection-state" :class="{ offline }">
          <i aria-hidden="true"></i>{{ offline ? 'Offline' : 'Connected' }}
        </span>
      </div>
    </nav>

    <ConnectionBanner v-if="!progressVisible && !settingsVisible" :offline="offline" :message="message" />
    <ProgressView v-if="progressVisible" />
    <SettingsView v-else-if="settingsVisible" />

    <div v-else-if="loading && currentReport" class="screen-loader" role="status">
      Reading report…
    </div>
    <ReportWorkspace
      v-else-if="currentReport"
      :detail="currentReport"
      :offline="offline"
      :saving="saving"
      @back="showShelf"
      @state="handleState"
      @feedback="handleFeedback"
    />
    <ReportShelf
      v-else
      :reports="reports"
      :summary="summary"
      :loading="loading"
      @select="selectReport"
    />
  </div>
</template>
