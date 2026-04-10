import { withBase } from "@/stores/config";

export type ContainerCardReportDefinition = {
  id: string;
  name: string;
  enabled: boolean;
  description?: string;
  command: string;
  filePath?: string;
  filePattern?: string;
  excludePatterns?: string[];
  linkedReportId?: string;
  linkedReportLabel?: string;
  showFileSize?: boolean;
  showChecksum?: boolean;
  parser: "text" | "json" | "jar_modules" | "jira_issues" | "properties_template" | "file_listing" | "file_content";
  displayMode?: "inline" | "page";
  refreshMode?: "ttl" | "container_update" | "manual" | "on_open";
  timeoutSeconds: number;
  cacheTtlSeconds: number;
};

export const containerCardReports = ref<ContainerCardReportDefinition[]>([]);
export const cardReportsLoaded = ref(false);
export const cardReportsSaving = ref(false);
export const cardReportsError = ref<string>();

let loadPromise: Promise<void> | null = null;

function cloneReports(source: ContainerCardReportDefinition[]) {
  return source.map((report) => ({ displayMode: "inline", refreshMode: "ttl", ...report }));
}

export async function loadContainerCardReports(force = false) {
  if (loadPromise && !force) return loadPromise;

  loadPromise = (async () => {
    cardReportsError.value = undefined;
    try {
      const response = await fetch(withBase("/api/card-reports"));
      if (!response.ok) throw new Error(`failed to load card reports (${response.status})`);
      containerCardReports.value = cloneReports((await response.json()) as ContainerCardReportDefinition[]);
    } catch (error) {
      cardReportsError.value = error instanceof Error ? error.message : String(error);
    } finally {
      cardReportsLoaded.value = true;
      loadPromise = null;
    }
  })();

  return loadPromise;
}

export async function saveContainerCardReports() {
  cardReportsSaving.value = true;
  cardReportsError.value = undefined;
  try {
    const response = await fetch(withBase("/api/card-reports"), {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(cloneReports(containerCardReports.value)),
    });
    if (!response.ok) throw new Error(await response.text());
    containerCardReports.value = cloneReports((await response.json()) as ContainerCardReportDefinition[]);
  } catch (error) {
    cardReportsError.value = error instanceof Error ? error.message : String(error);
    throw error;
  } finally {
    cardReportsSaving.value = false;
  }
}

export function useContainerCardReports() {
  onMounted(() => {
    void loadContainerCardReports();
  });
  return {
    reports: containerCardReports,
    loaded: cardReportsLoaded,
    saving: cardReportsSaving,
    error: cardReportsError,
    load: loadContainerCardReports,
    save: saveContainerCardReports,
  };
}

export const containerCardReportParserOptions = [
  { labelKey: "card-templates.report-parser-text", value: "text" as const },
  { labelKey: "card-templates.report-parser-json", value: "json" as const },
  { labelKey: "card-templates.report-parser-jar_modules", value: "jar_modules" as const },
  { labelKey: "card-templates.report-parser-jira_issues", value: "jira_issues" as const },
  { labelKey: "card-templates.report-parser-properties_template", value: "properties_template" as const },
  { labelKey: "card-templates.report-parser-file_listing", value: "file_listing" as const },
  { labelKey: "card-templates.report-parser-file_content", value: "file_content" as const },
];

export const containerCardReportDisplayOptions = [
  { labelKey: "card-templates.report-display-inline", value: "inline" as const },
  { labelKey: "card-templates.report-display-page", value: "page" as const },
];

export const containerCardReportRefreshOptions = [
  { labelKey: "card-templates.report-refresh-ttl", value: "ttl" as const },
  { labelKey: "card-templates.report-refresh-container_update", value: "container_update" as const },
  { labelKey: "card-templates.report-refresh-manual", value: "manual" as const },
  { labelKey: "card-templates.report-refresh-on_open", value: "on_open" as const },
];
