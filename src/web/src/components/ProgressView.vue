<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { connect, request } from '../api';
import {
  activeTasks, applicationCount, isComplete, lastSevenDays, percentage, targetFor, todayIn,
} from '../progress';
import type { ProgressSnapshot, ProgressTask } from '../progress';

const props = defineProps<{ shared?: boolean }>();
const emit = defineEmits<{ loaded: [snapshot: ProgressSnapshot]; signedOut: [] }>();
const data = ref<ProgressSnapshot | null>(null);
const day = ref(todayIn());
const today = ref(todayIn());
const loading = ref(true);
const saving = ref(false);
const error = ref('');
const password = ref('');
const locked = ref(false);
const title = ref('');
const recurrence = ref<ProgressTask['recurrence']>('daily');
let rolloverTimer: ReturnType<typeof setInterval> | undefined;
const tasks = computed(() => (data.value ? activeTasks(data.value, day.value) : []));
const completed = computed(() => tasks.value.filter((task) => data.value
  && isComplete(data.value, task, day.value)).length);
const count = computed(() => (data.value ? applicationCount(data.value, day.value) : 0));
const goal = computed(() => (data.value ? targetFor(data.value, day.value) : 0));
const applicationPercent = computed(() => percentage(count.value, goal.value));
const taskPercent = computed(() => percentage(completed.value, tasks.value.length));
const week = computed(() => lastSevenDays(day.value));
const average = computed(() => {
  if (!data.value) return null;
  const measured = week.value.filter((date) => targetFor(data.value!, date) > 0);
  if (!measured.length) return null;
  return Math.round(measured.reduce((sum, date) => sum
    + percentage(applicationCount(data.value!, date), targetFor(data.value!, date)), 0)
    / measured.length);
});

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    if (!props.shared) await connect();
    const response = await fetch('/api/progress', { cache: 'no-store' });
    if (response.status === 401) {
      locked.value = true;
      data.value = null;
      emit('signedOut');
      return;
    }
    if (!response.ok) {
      const body = await response.json() as { message?: string };
      throw new Error(body.message ?? 'Could not load progress.');
    }
    const followToday = day.value === today.value;
    data.value = await response.json() as ProgressSnapshot;
    emit('loaded', data.value);
    locked.value = false;
    today.value = todayIn(data.value.timezone);
    if (followToday) day.value = today.value;
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : 'Could not load progress.';
  } finally { loading.value = false; }
}

async function login(): Promise<void> {
  saving.value = true;
  error.value = '';
  try {
    await request('/api/login', { method: 'POST', body: JSON.stringify({ password: password.value }) });
    password.value = '';
    await load();
  } catch (reason) { error.value = reason instanceof Error ? reason.message : 'Could not sign in.'; }
  finally { saving.value = false; }
}

async function mutate(input: Record<string, unknown>): Promise<void> {
  saving.value = true;
  error.value = '';
  try {
    data.value = await request<ProgressSnapshot>('/api/progress', {
      method: 'POST', body: JSON.stringify({ ...input, date: day.value }),
    });
    if (input.action === 'add') title.value = '';
  } catch (reason) { error.value = reason instanceof Error ? reason.message : 'Could not save.'; }
  finally { saving.value = false; }
}

async function logout(): Promise<void> {
  await request('/api/logout', { method: 'POST' });
  data.value = null;
  locked.value = true;
  emit('signedOut');
}

function onFocus(): void {
  const next = todayIn(data.value?.timezone);
  if (day.value === today.value) day.value = next;
  today.value = next;
  if (!locked.value && !saving.value) void load();
}
onMounted(() => {
  void load();
  window.addEventListener('focus', onFocus);
  rolloverTimer = setInterval(() => {
    if (todayIn(data.value?.timezone) !== today.value) onFocus();
  }, 30_000);
});
onUnmounted(() => {
  window.removeEventListener('focus', onFocus);
  clearInterval(rolloverTimer);
});
</script>

<template>
  <main class="progress-page">
    <header class="progress-heading">
      <div>
        <h1>Progress</h1>
      </div>
      <div v-if="!locked" class="progress-actions">
        <label>Date <input v-model="day" type="date" :max="today" /></label>
        <button v-if="shared" class="quiet-button" :disabled="loading" @click="load">Refresh</button>
        <button v-if="shared" class="quiet-button" @click="logout">Sign out</button>
      </div>
    </header>

    <p v-if="error" class="progress-error" role="alert">{{ error }}</p>
    <form v-if="locked" class="progress-login" @submit.prevent="login">
      <h2>Sign in</h2>
      <p>Enter the password to see the latest update.</p>
      <label>Password <input v-model="password" type="password" autocomplete="current-password" required maxlength="256" /></label>
      <button class="primary-button" :disabled="saving">{{ saving ? 'Opening…' : 'View progress' }}</button>
    </form>
    <p v-else-if="loading && !data" role="status">Reading progress…</p>
    <template v-else-if="data">
      <div class="progress-columns">
        <section class="intentions" aria-label="Tasks">
          <div class="section-label"><h2>Tasks</h2><span>{{ day === today ? 'Today' : day }}</span></div>
          <ul class="intention-list">
            <li v-for="task in tasks" :key="task.id">
              <label :class="{ 'intention-complete': isComplete(data, task, day) }">
                <input type="checkbox" :checked="isComplete(data, task, day)" :disabled="shared || saving"
                  @change="mutate({ action: 'complete', id: task.id, completed: ($event.target as HTMLInputElement).checked })" />
                <span>{{ task.title }} <small>{{ task.recurrence === 'once' ? 'One-time' : task.recurrence }}</small></span>
              </label>
              <button v-if="!shared" class="archive-task" :aria-label="`Archive ${task.title}`" :disabled="saving"
                @click="mutate({ action: 'archive', id: task.id })">Archive</button>
            </li>
          </ul>
          <p v-if="!tasks.length" class="progress-empty">No tasks.</p>
          <form v-if="!shared" class="add-intention" @submit.prevent="mutate({ action: 'add', title, recurrence })">
            <label class="sr-only" for="new-task">New task</label>
            <input id="new-task" v-model="title" placeholder="New task" required maxlength="200" />
            <label class="sr-only" for="task-recurrence">Repeat</label>
            <select id="task-recurrence" v-model="recurrence">
              <option value="once">One-time</option><option value="daily">Daily</option>
              <option value="weekly">Weekly</option><option value="monthly">Monthly</option><option value="quarterly">Quarterly</option>
            </select>
            <button class="primary-button" :disabled="saving">Add task</button>
          </form>
        </section>

        <aside class="daily-progress" aria-label="Progress summary">
          <p class="eyebrow">{{ day === today ? "Today's momentum" : 'Daily momentum' }}</p>
          <div class="progress-metric"><span>Applications</span><strong>{{ goal ? `${applicationPercent}%` : '—' }}</strong></div>
          <progress :value="applicationPercent" max="100" aria-label="Application target progress"></progress>
          <p>{{ count }} applied<span v-if="goal"> of {{ goal }} target</span><span v-else> · No target set</span></p>
          <div class="progress-metric"><span>Tasks</span><strong>{{ tasks.length ? `${taskPercent}%` : '—' }}</strong></div>
          <progress :value="taskPercent" max="100" aria-label="Task completion progress"></progress>
          <p>{{ completed }} of {{ tasks.length }} completed</p>
          <div class="weekly-progress">
            <div class="progress-metric"><span>7-day target average</span><strong>{{ average === null ? '—' : `${average}%` }}</strong></div>
            <div class="week-bars">
              <div v-for="date in week" :key="date" :title="`${date}: ${applicationCount(data, date)} applications, target ${targetFor(data, date) || 'not set'}`">
                <span class="week-bars__track"><i :style="{ height: `${percentage(applicationCount(data, date), targetFor(data, date))}%` }"></i></span>
                <span>{{ new Date(`${date}T12:00:00Z`).toLocaleDateString('en', { weekday: 'narrow', timeZone: 'UTC' }) }}</span>
              </div>
            </div>
            <p class="progress-hint">Includes days with a target. Each day is capped at 100%.</p>
          </div>
        </aside>
      </div>
      <footer class="progress-footer">
        <span>Local time · {{ data.timezone }}</span>
        <span v-if="shared">Updated {{ new Date(data.updatedAt).toLocaleString('en', { timeZone: data.timezone, timeZoneName: 'short' }) }}</span>
      </footer>
    </template>
  </main>
</template>
