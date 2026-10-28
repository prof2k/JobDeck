import { expect, test } from 'bun:test';
import { activeTasks, isComplete, periodFor, targetFor, lastSevenDays } from '../src/progress';
import type { ProgressSnapshot, ProgressTask } from '../src/progress';

const task: ProgressTask = { id: 'fictional', title: 'Practice', recurrence: 'daily', startsOn: '2026-10-01', archivedOn: null };
const data: ProgressSnapshot = { version: 1, updatedAt: '', timezone: 'UTC', today: '2026-10-28', tasks: [task], completions: [{ taskId: task.id, period: '2026-10-27', completedOn: '2026-10-27' }], targets: [{ effectiveOn: '2026-10-01', target: 20 }, { effectiveOn: '2026-10-28', target: 30 }], days: [] };

test('daily tasks reset without losing yesterday and target changes preserve history', () => {
  expect(isComplete(data, task, '2026-10-27')).toBe(true);
  expect(isComplete(data, task, '2026-10-28')).toBe(false);
  expect(targetFor(data, '2026-10-27')).toBe(20);
  expect(targetFor(data, '2026-10-28')).toBe(30);
  expect(targetFor(data, '2026-09-30')).toBe(0);
});

test('weekly and quarterly periods match the local server across boundaries', () => {
  expect(periodFor({ ...task, recurrence: 'weekly' }, '2026-11-01')).toBe('2026-10-26');
  expect(periodFor({ ...task, recurrence: 'weekly' }, '2026-11-02')).toBe('2026-11-02');
  expect(periodFor({ ...task, recurrence: 'quarterly' }, '2027-01-01')).toBe('2027-Q1');
  expect(lastSevenDays('2027-01-01')[0]).toBe('2026-12-26');
});

test('archival preserves earlier days and later completions do not rewrite earlier views', () => {
  expect(activeTasks({ ...data, tasks: [{ ...task, archivedOn: '2026-10-28' }] }, '2026-10-27')).toHaveLength(1);
  expect(activeTasks({ ...data, tasks: [{ ...task, archivedOn: '2026-10-28' }] }, '2026-10-28')).toHaveLength(0);
  const weekly = { ...task, recurrence: 'weekly' as const };
  expect(isComplete({ ...data, completions: [{ taskId: task.id, period: '2026-10-26', completedOn: '2026-10-28' }] }, weekly, '2026-10-27')).toBe(false);
});
