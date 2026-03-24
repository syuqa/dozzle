export interface LogIncidentMatch {
  found: boolean;
  iid?: number;
  url?: string;
  title?: string;
  severity?: string;
  status?: string;
  score?: number;
  incident_key?: string;
  reason?: string;
}

export interface LogIncidentState {
  status: "checking" | "matched" | "none" | "skipped";
  match?: LogIncidentMatch | null;
}
