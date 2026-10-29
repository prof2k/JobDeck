<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { FeedbackVerdict, JobState, ReportJob } from '../types';

const props = defineProps<{
  job: ReportJob;
  reportId: string;
  calibration: boolean;
  offline: boolean;
  readOnly?: boolean;
}>();

const emit = defineEmits<{
  state: [job: ReportJob, state: JobState];
  feedback: [payload: {
    job: ReportJob;
    verdict: FeedbackVerdict;
    reason: string;
    note: string;
  }];
}>();

const feedbackReason = ref('');
const feedbackNote = ref('');
const feedbackVerdict = ref<FeedbackVerdict | ''>('');

watch(
  () => props.job.id,
  () => {
    feedbackReason.value = props.job.feedbackReason ?? '';
    feedbackNote.value = props.job.feedbackNote ?? '';
    feedbackVerdict.value = props.job.feedbackVerdict ?? '';
  },
  { immediate: true },
);

const scoreLabel = computed(() => props.job.score ?? '—');

function formatDate(value: string): string {
  return new Intl.DateTimeFormat('en', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value));
}

function openApplication(): void {
  if (props.offline || props.readOnly) return;
  if (props.job.state === 'ready') {
    emit('state', props.job, 'opened');
  }
}

function submitFeedback(): void {
  if (!feedbackVerdict.value || props.offline || props.readOnly) return;
  emit('feedback', {
    job: props.job,
    verdict: feedbackVerdict.value,
    reason: feedbackReason.value,
    note: feedbackNote.value,
  });
}
</script>

<template>
  <article class="evidence-rail">
    <div class="evidence-rail__scroll">
      <header class="job-heading">
        <div>
          <p class="eyebrow">{{ job.company }}</p>
          <h2>{{ job.title }}</h2>
        </div>
        <div class="score-stamp" :class="`score-stamp--${job.fitClass.toLowerCase()}`">
          <span>{{ scoreLabel }}</span>
          <small v-if="job.evaluation.screeningVerdict" :class="`screening-verdict--${job.evaluation.screeningVerdict}`">{{ job.evaluation.screeningVerdict === 'maybe' ? 'Maybe' : 'Pass' }}</small>
          <small v-else>Class {{ job.fitClass }}</small>
        </div>
      </header>

      <div class="badge-row">
        <span class="badge" :class="`badge--state-${job.state}`">{{ job.state }}</span>
        <span class="badge badge--quiet">{{ job.arrangement }}</span>
        <span class="badge badge--quiet">{{ job.salary }}</span>
      </div>

      <dl class="job-facts">
        <div>
          <dt>Location</dt>
          <dd>{{ job.location }}</dd>
        </div>
        <div>
          <dt>Verified</dt>
          <dd>{{ formatDate(job.liveVerifiedAt) }}</dd>
        </div>
        <div>
          <dt>Official source</dt>
          <dd>{{ job.source }}</dd>
        </div>
        <div>
          <dt>Canonical key</dt>
          <dd class="mono">{{ job.canonicalKey }}</dd>
        </div>
      </dl>

      <section class="recommendation">
        <p class="eyebrow">Recommendation</p>
        <p>{{ job.recommendation }}</p>
      </section>

      <section
        v-if="calibration && !readOnly"
        class="feedback-panel"
        :aria-labelledby="`feedback-title-${job.id}`"
      >
        <div class="feedback-panel__heading">
          <div>
            <p class="eyebrow">Your feedback</p>
            <h3 :id="`feedback-title-${job.id}`">Is this the kind of job you want?</h3>
          </div>
          <span>Optional</span>
        </div>
        <p class="feedback-intro">
          Rate the jobs that give you a clear reaction. Skip any role you do not want to judge.
        </p>
        <div class="feedback-options" role="group" aria-label="Job fit feedback">
          <button
            v-for="option in [
              ['strong_fit', 'Yes — strong fit'],
              ['maybe', 'Maybe'],
              ['wrong_fit', 'No — wrong fit'],
              ['never_show', 'Never show similar'],
            ]"
            :key="option[0]"
            type="button"
            :class="{ active: feedbackVerdict === option[0] }"
            :aria-pressed="feedbackVerdict === option[0]"
            :disabled="offline"
            @click="feedbackVerdict = option[0] as FeedbackVerdict"
          >
            {{ option[1] }}
          </button>
        </div>
        <div class="feedback-fields">
          <label>
            <span>What influenced your choice? <small>Optional</small></span>
            <input
              v-model="feedbackReason"
              type="text"
              placeholder="Title, level, salary, location, scope…"
              :disabled="offline"
            >
          </label>
          <label>
            <span>Anything the next search should know? <small>Optional</small></span>
            <textarea
              v-model="feedbackNote"
              rows="3"
              placeholder="Add context in your own words"
              :disabled="offline"
            ></textarea>
          </label>
        </div>
        <button
          class="button button--secondary"
          type="button"
          :disabled="!feedbackVerdict || offline"
          @click="submitFeedback"
        >
          Save job feedback
        </button>
      </section>

      <section class="evidence-section">
        <div class="section-heading">
          <h3>Requirement evidence</h3>
          <span>{{ job.evaluation.requirements.length }} reviewed</span>
        </div>
        <div v-if="job.evaluation.requirements.length" class="evidence-list">
          <article
            v-for="requirement in job.evaluation.requirements"
            :key="requirement.requirement"
            class="evidence-item"
          >
            <div>
              <strong>{{ requirement.requirement }}</strong>
              <span class="match" :class="`match--${requirement.match.toLowerCase()}`">
                {{ requirement.match }}
              </span>
            </div>
            <p>{{ requirement.evidence }}</p>
            <small>Source: {{ requirement.source }}</small>
          </article>
        </div>
        <p v-else class="muted-copy">No requirement-level evidence is stored.</p>
      </section>

      <div class="gap-grid">
        <section>
          <h3>Gaps</h3>
          <ul v-if="job.evaluation.gaps.length">
            <li v-for="gap in job.evaluation.gaps" :key="gap">{{ gap }}</li>
          </ul>
          <p v-else class="muted-copy">No material onboarding gap recorded.</p>
        </section>
        <section>
          <h3>Risks</h3>
          <ul v-if="job.evaluation.risks.length">
            <li v-for="risk in job.evaluation.risks" :key="risk">{{ risk }}</li>
          </ul>
          <p v-else class="muted-copy">No additional risk recorded.</p>
        </section>
      </div>

      <section v-if="job.evaluation.companyPolicy" class="policy-note">
        <p class="eyebrow">Verified company policy</p>
        <p>{{ job.evaluation.companyPolicy.summary }}</p>
        <a
          :href="job.evaluation.companyPolicy.sourceUrl"
          target="_blank"
          rel="noopener noreferrer"
        >
          Source · verified {{ formatDate(job.evaluation.companyPolicy.verifiedAt) }}
        </a>
      </section>

    </div>

    <footer class="evidence-actions">
      <span>{{ readOnly ? 'Read-only report' : offline ? 'Read-only cached snapshot' : 'Progress saves locally' }}</span>
      <div>
        <button
          v-if="!readOnly && job.state !== 'applied'"
          class="button button--quiet"
          type="button"
          :disabled="offline"
          @click="emit('state', job, 'applied')"
        >
          Mark applied
        </button>
        <button
          v-else-if="!readOnly"
          class="button button--quiet"
          type="button"
          :disabled="offline"
          @click="emit('state', job, 'opened')"
        >
          Undo applied
        </button>
        <a
          class="button button--primary"
          :class="{ disabled: offline }"
          :href="offline ? undefined : job.applicationUrl"
          target="_blank"
          rel="noopener noreferrer"
          :aria-disabled="offline"
          @click="openApplication"
        >
          Open official application ↗
        </a>
      </div>
    </footer>
  </article>
</template>
