export interface ScanSchedule {
  enabled: boolean;
  intervalMinutes: number;
}

export interface ScanSummary {
  critical: number;
  high: number;
  medium: number;
  low: number;
  unknown: number;
  total: number;
}

export interface ScanVulnerability {
  id: string;
  packageName: string;
  installedVersion?: string;
  fixedVersion?: string;
  severity: string;
  primaryUrl?: string;
}

export interface ScanTargetResult {
  target: string;
  type?: string;
  vulnerabilities: ScanVulnerability[];
}

export interface ContainerScanState {
  container: {
    id: string;
    name: string;
    image: string;
    host: string;
  };
  summary: ScanSummary;
  result?: {
    image: string;
    generatedAt: string;
    summary: ScanSummary;
    results: ScanTargetResult[];
  };
  lastStartedAt?: string;
  lastFinishedAt?: string;
  lastSuccessAt?: string;
  lastError?: string;
  running: boolean;
  schedule: ScanSchedule;
  packageTypes?: string[];
  severities?: string[];
}

export interface ScanDashboardSummary {
  containers: number;
  scannedContainers: number;
  runningScans: number;
  critical: number;
  high: number;
  medium: number;
  low: number;
  unknown: number;
  total: number;
  items: Array<{
    container: {
      id: string;
      name: string;
      image: string;
      host: string;
    };
    summary: ScanSummary;
    lastSuccessAt?: string;
    running: boolean;
    lastError?: string;
    schedule: ScanSchedule;
  }>;
}

export interface ScanAlert {
  id: number;
  name: string;
  enabled: boolean;
  dispatcherId: number;
  containerExpression: string;
  minSeverity: string;
  packageTypes?: string[];
  cooldownMinutes?: number;
  triggerCount: number;
  lastTriggeredAt?: string | null;
}
