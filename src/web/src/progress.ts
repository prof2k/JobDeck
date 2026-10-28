export interface ProgressTask {
  id: string;
  title: string;
  recurrence: 'once' | 'daily' | 'weekly' | 'monthly' | 'quarterly';
  startsOn: string;
  archivedOn: string | null;
}

export interface ProgressSnapshot {
  version: number;
  updatedAt: string;
  timezone: string;
  today: string;
  tasks: ProgressTask[];
  completions: { taskId: string; period: string; completedOn: string }[];
  targets: { effectiveOn: string; target: number }[];
  days: { date: string; applications: number }[];
}

export interface SharingStatus {
  url: string;
  configured: boolean;
  lastPublishedAt: string;
  error: string;
}

export function todayIn(timezone = 'UTC'): string {
  return new Intl.DateTimeFormat('en-CA', {
    timeZone: timezone, year: 'numeric', month: '2-digit', day: '2-digit',
  }).format(new Date());
}

export function periodFor(task: ProgressTask, day: string): string {
  const date = new Date(`${day}T12:00:00Z`);
  if (task.recurrence === 'once') return 'once';
  if (task.recurrence === 'monthly') return day.slice(0, 7);
  if (task.recurrence === 'quarterly') {
    return `${date.getUTCFullYear()}-Q${Math.floor(date.getUTCMonth() / 3) + 1}`;
  }
  if (task.recurrence === 'weekly') {
    date.setUTCDate(date.getUTCDate() - ((date.getUTCDay() + 6) % 7));
    return date.toISOString().slice(0, 10);
  }
  return day;
}

export function activeTasks(data: ProgressSnapshot, day: string): ProgressTask[] {
  return data.tasks.filter((task) => task.startsOn <= day
    && (!task.archivedOn || task.archivedOn > day));
}

export function isComplete(data: ProgressSnapshot, task: ProgressTask, day: string): boolean {
  return data.completions.some((entry) => entry.taskId === task.id
    && entry.period === periodFor(task, day) && entry.completedOn <= day);
}

export function targetFor(data: ProgressSnapshot, day: string): number {
  return data.targets.filter((entry) => entry.effectiveOn <= day)
    .sort((first, second) => second.effectiveOn.localeCompare(first.effectiveOn))[0]?.target ?? 0;
}

export function applicationCount(data: ProgressSnapshot, day: string): number {
  return data.days.find((entry) => entry.date === day)?.applications ?? 0;
}

export function percentage(value: number, target: number): number {
  return target > 0 ? Math.min(100, Math.round((value / target) * 100)) : 0;
}

export function lastSevenDays(day: string): string[] {
  return Array.from({ length: 7 }, (_, index) => {
    const date = new Date(`${day}T12:00:00Z`);
    date.setUTCDate(date.getUTCDate() - 6 + index);
    return date.toISOString().slice(0, 10);
  });
}
