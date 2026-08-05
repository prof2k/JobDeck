<script setup lang="ts">
import type { ReportListItem, StatusSummary } from '../types';

defineProps<{
  reports: ReportListItem[];
  summary: StatusSummary | null;
  loading: boolean;
}>();

const emit = defineEmits<{
  select: [reportId: string];
}>();

function formatDate(value: string | null): string {
  if (!value) return 'Not opened yet';
  return new Intl.DateTimeFormat('en', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value));
}

function progress(report: ReportListItem): number {
  if (report.jobCount === 0) return 0;
  return Math.round((report.appliedCount / report.jobCount) * 100);
}
</script>

<template>
  <main class="report-shelf">
    <header class="shelf-heading">
      <div>
        <p class="eyebrow">Local search ledger</p>
        <h1>Reports</h1>
        <p class="shelf-heading__copy">
          Calibrated job batches, evidence, and application progress in one private place.
        </p>
      </div>
      <dl v-if="summary" class="ledger-summary" aria-label="Job Deck summary">
        <div>
          <dt>Reports</dt>
          <dd>{{ summary.reports }}</dd>
        </div>
        <div>
          <dt>Ready</dt>
          <dd>{{ summary.ready + summary.opened }}</dd>
        </div>
        <div>
          <dt>Applied</dt>
          <dd>{{ summary.applied }}</dd>
        </div>
      </dl>
    </header>

    <div v-if="loading" class="empty-state" role="status">
      Reading the local ledger…
    </div>
    <div v-else-if="reports.length === 0" class="empty-state">
      <p class="eyebrow">No reports yet</p>
      <h2>The desk is ready.</h2>
      <p>
        Ask your job-search agent to create the first 15-role calibration report.
      </p>
    </div>
    <ol v-else class="report-list">
      <li v-for="report in reports" :key="report.id">
        <button class="report-entry" type="button" @click="emit('select', report.id)">
          <span class="report-entry__date">
            {{ formatDate(report.createdAt) }}
          </span>
          <span class="report-entry__body">
            <span class="report-entry__topline">
              <strong>{{ report.title }}</strong>
              <span class="badge" :class="`badge--${report.kind}`">{{ report.kind }}</span>
            </span>
            <span class="report-entry__summary">{{ report.summary }}</span>
            <span class="report-entry__meta">
              {{ report.jobCount }} jobs · {{ report.readyCount }} ready ·
              last opened {{ formatDate(report.lastOpenedAt) }}
            </span>
          </span>
          <span class="report-entry__progress" aria-hidden="true">
            <span>{{ progress(report) }}%</span>
            <i><b :style="{ width: `${progress(report)}%` }"></b></i>
          </span>
          <span class="report-entry__arrow" aria-hidden="true">→</span>
        </button>
      </li>
    </ol>
  </main>
</template>
