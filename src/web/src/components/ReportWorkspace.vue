<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import EvidenceRail from './EvidenceRail.vue';
import type {
  FeedbackVerdict,
  JobState,
  ReportDetail,
  ReportJob,
} from '../types';

const props = defineProps<{
  detail: ReportDetail;
  offline: boolean;
  saving: boolean;
}>();

const emit = defineEmits<{
  back: [];
  state: [job: ReportJob, state: JobState];
  feedback: [payload: {
    job: ReportJob;
    verdict: FeedbackVerdict;
    reason: string;
    note: string;
  }];
}>();

const selectedJobId = ref<number>(props.detail.jobs[0]?.id ?? 0);
const query = ref('');
const stateFilter = ref('all');

watch(
  () => props.detail.report.id,
  () => {
    selectedJobId.value = props.detail.jobs[0]?.id ?? 0;
    query.value = '';
    stateFilter.value = 'all';
  },
);

const filteredJobs = computed(() => {
  const normalizedQuery = query.value.trim().toLowerCase();
  return props.detail.jobs.filter((job) => {
    const matchesState = stateFilter.value === 'all' || job.state === stateFilter.value;
    const matchesQuery = normalizedQuery === ''
      || `${job.company} ${job.title} ${job.location}`.toLowerCase().includes(normalizedQuery);
    return matchesState && matchesQuery;
  });
});

const selectedJob = computed(() => props.detail.jobs.find(
  (job) => job.id === selectedJobId.value,
) ?? filteredJobs.value[0] ?? null);

function selectJob(job: ReportJob): void {
  selectedJobId.value = job.id;
}
</script>

<template>
  <main class="report-workspace">
    <header class="workspace-heading">
      <button class="back-button" type="button" @click="emit('back')">← Reports</button>
      <div>
        <div class="workspace-heading__title">
          <div>
            <p class="eyebrow">{{ detail.report.kind }} batch</p>
            <h1>{{ detail.report.title }}</h1>
          </div>
          <span v-if="saving" class="saving-note" role="status">Saving…</span>
        </div>
        <p>{{ detail.report.summary }}</p>
      </div>
    </header>

    <div class="workspace-grid">
      <section class="job-ledger" aria-label="Ranked jobs">
        <div class="ledger-controls">
          <label>
            <span class="sr-only">Filter roles</span>
            <input v-model="query" type="search" placeholder="Filter roles">
          </label>
          <label>
            <span class="sr-only">Filter by state</span>
            <select v-model="stateFilter">
              <option value="all">All states</option>
              <option value="ready">Ready</option>
              <option value="opened">Opened</option>
              <option value="applied">Applied</option>
              <option value="skipped">Skipped</option>
            </select>
          </label>
        </div>

        <p class="ledger-count">{{ filteredJobs.length }} of {{ detail.jobs.length }} roles</p>
        <ol v-if="filteredJobs.length" class="job-list">
          <li v-for="job in filteredJobs" :key="job.id">
            <button
              class="job-row"
              :class="{ selected: selectedJob?.id === job.id }"
              type="button"
              :aria-pressed="selectedJob?.id === job.id"
              @click="selectJob(job)"
            >
              <span class="job-row__rank mono">{{ String(job.rank).padStart(2, '0') }}</span>
              <span class="job-row__identity">
                <strong>{{ job.company }}</strong>
                <small>{{ job.title }}</small>
                <span>{{ job.location }}</span>
              </span>
              <span class="job-row__score mono">
                {{ job.score ?? '—' }}
                <small>Class {{ job.fitClass }}</small>
              </span>
              <span class="badge" :class="`badge--state-${job.state}`">{{ job.state }}</span>
              <span class="job-row__arrow" aria-hidden="true">→</span>
            </button>
          </li>
        </ol>
        <div v-else class="empty-state empty-state--compact">
          No roles match this filter.
        </div>
      </section>

      <EvidenceRail
        v-if="selectedJob"
        :job="selectedJob"
        :report-id="detail.report.id"
        :calibration="detail.report.kind === 'calibration'"
        :offline="offline"
        @state="(job, state) => emit('state', job, state)"
        @feedback="(payload) => emit('feedback', payload)"
      />
    </div>
  </main>
</template>
