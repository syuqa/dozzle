export interface NotificationRule {
  id: number;
  type?: "log" | "metric" | "state";
  name: string;
  enabled: boolean;
  containerExpression: string;
  logExpression: string;
  metricExpression?: string;
  eventExpression?: string;
  cooldown?: number;
  sampleWindow?: number;
  stateTriggers?: string[];
  holdoffSeconds?: number;
  template?: string;
  templateId?: number;
  triggerCount: number;
  triggeredContainers: number;
  lastTriggeredAt: string | null;
  dispatcher: Dispatcher | null;
}

export interface UnifiedAlertBase {
  id: number;
  name: string;
  enabled: boolean;
  containerExpression: string;
  triggerCount: number;
  lastTriggeredAt: string | null;
  dispatcher: Dispatcher | null;
}

export interface ScanAlert extends UnifiedAlertBase {
  type: "scan";
  dispatcherId: number;
  minSeverity: string;
  packageTypes?: string[];
  scheduleEnabled?: boolean;
  intervalMinutes?: number;
  cooldownMinutes?: number;
  notifyOnManual?: boolean;
  template?: string;
  templateId?: number;
  triggeredContainers: number;
  lastDispatchAt?: string | null;
  lastDispatchError?: string;
}

export type UnifiedAlert = NotificationRule | ScanAlert;

export interface Dispatcher {
  id: number;
  name: string;
  type: string;
  url?: string;
  template?: string;
  templateId?: number;
  headers?: Record<string, string>;
  prefix?: string;
  expiresAt?: string;
  botToken?: string;
  chatId?: string;
  messageThreadId?: string;
  parseMode?: string;
}

export interface NotificationTemplate {
  id: number;
  name: string;
  body: string;
}

export interface NotificationRuleInput {
  name: string;
  enabled: boolean;
  dispatcherId: number;
  logExpression: string;
  containerExpression: string;
  metricExpression?: string;
  eventExpression?: string;
  cooldown?: number;
  sampleWindow?: number;
  template?: string;
  templateId?: number;
}

export interface PreviewResult {
  containerError?: string;
  logError?: string;
  metricError?: string;
  eventError?: string;
  matchedContainers: {
    id: string;
    name: string;
    image: string;
    host: string;
  }[];
  matchedLogs: {
    id: number;
    t: string;
    m: unknown;
    rm: string;
    ts: number;
    l: string;
    s: string;
  }[];
  totalLogs: number;
  messageKeys?: string[];
}

export interface TestWebhookResult {
  success: boolean;
  statusCode?: number;
  error?: string;
}
