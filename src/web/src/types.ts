export type JobState = 'ready' | 'opened' | 'applied' | 'skipped' | 'expired' | 'rejected';
export type FeedbackVerdict = 'strong_fit' | 'maybe' | 'wrong_fit' | 'never_show';

export interface StatusSummary {
  reports: number;
  jobs: number;
  ready: number;
  opened: number;
  applied: number;
  skipped: number;
}

export interface ReportListItem {
  groupBy?: 'company' | 'roleType';
  id: string;
  kind: 'calibration' | 'search';
  title: string;
  summary: string;
  createdAt: string;
  lastOpenedAt: string | null;
  jobCount: number;
  readyCount: number;
  appliedCount: number;
}

export interface EvidenceItem {
  requirement: string;
  match: 'Direct' | 'Adjacent' | 'None' | string;
  evidence: string;
  source: string;
}

export interface CompanyPolicy {
  summary: string;
  maximumApplications?: number;
  windowDays?: number;
  sourceUrl: string;
  verifiedAt: string;
}

export interface Evaluation {
  roleType?: string;
  screeningVerdict?: 'pass' | 'maybe';
  requirements: EvidenceItem[];
  gaps: string[];
  risks: string[];
  companyPolicy?: CompanyPolicy;
}

export interface ReportJob {
  id: number;
  canonicalKey: string;
  company: string;
  title: string;
  location: string;
  arrangement: string;
  salary: string;
  applicationUrl: string;
  source: string;
  liveVerifiedAt: string;
  state: JobState;
  rank: number;
  fitClass: 'A' | 'B' | 'C' | 'D';
  score: number | null;
  recommendation: string;
  evaluation: Evaluation;
  feedbackVerdict: FeedbackVerdict | null;
  feedbackReason: string | null;
  feedbackNote: string | null;
}

export interface ReportDetail {
  report: ReportListItem;
  jobs: ReportJob[];
}
