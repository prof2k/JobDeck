import type {
  FeedbackVerdict,
  JobState,
  ReportDetail,
  ReportListItem,
  StatusSummary,
} from './types';

let sessionToken = '';

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      Accept: 'application/json',
      ...(init.body ? { 'Content-Type': 'application/json' } : {}),
      ...(sessionToken ? { 'X-Jobdeck-Token': sessionToken } : {}),
      ...init.headers,
    },
  });
  if (!response.ok) {
    let message = `Job Deck returned ${response.status}`;
    try {
      const payload = await response.json() as { message?: string };
      message = payload.message ?? message;
    } catch {
      // Retain the status-based message when the response is not JSON.
    }
    throw new Error(message);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  return response.json() as Promise<T>;
}

export async function connect(): Promise<void> {
  await request<{ app: string }>('/api/health');
  const session = await request<{ token: string }>('/api/session');
  sessionToken = session.token;
}

export function fetchSummary(): Promise<StatusSummary> {
  return request<StatusSummary>('/api/summary');
}

export function fetchReports(): Promise<ReportListItem[]> {
  return request<ReportListItem[]>('/api/reports');
}

export function fetchReport(reportId: string): Promise<ReportDetail> {
  return request<ReportDetail>(`/api/reports/${encodeURIComponent(reportId)}`);
}

export function updateState(jobId: number, state: JobState): Promise<{ state: JobState }> {
  return request<{ state: JobState }>(`/api/jobs/${jobId}/state`, {
    method: 'PUT',
    body: JSON.stringify({ state, note: 'Updated from the Job Deck interface.' }),
  });
}

export function saveFeedback({
  jobId,
  reportId,
  verdict,
  reason,
  note,
}: {
  jobId: number;
  reportId: string;
  verdict: FeedbackVerdict;
  reason: string;
  note: string;
}): Promise<void> {
  return request<void>(`/api/jobs/${jobId}/feedback`, {
    method: 'POST',
    body: JSON.stringify({ reportId, verdict, reason, note }),
  });
}
