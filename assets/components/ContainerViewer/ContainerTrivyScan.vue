<template>
  <div class="space-y-4 p-4">
    <div class="flex items-center gap-2">
      <div class="min-w-0">
        <h2 class="text-lg font-semibold">{{ $t("scan.title") }}</h2>
        <p class="text-base-content/70 truncate text-sm">{{ container.image }}</p>
      </div>
      <button class="btn btn-sm ml-auto" :disabled="loading" @click="runScan()">
        <span class="loading loading-spinner loading-xs" v-if="loading"></span>
        {{ $t("button.refresh") }}
      </button>
    </div>

    <div class="space-y-3" v-if="loading || state?.running">
      <div class="alert alert-info">
        <span>{{ $t("scan.scanning-image") }}</span>
      </div>
      <div class="stable-scroll bg-base-300 rounded-box text-base-content/80 max-h-56 overflow-auto p-3 font-mono text-xs" v-if="scanLog.length">
        <pre class="whitespace-pre-wrap break-words">{{ scanLog.join("\n") }}</pre>
      </div>
    </div>

    <div class="alert alert-error" v-else-if="error">
      <span>{{ error }}</span>
    </div>

    <template v-else-if="state">
      <div class="flex flex-wrap gap-3">
        <div class="text-base-content/70 text-sm self-center" v-if="state.lastSuccessAt">
          {{ $t("scan.last-successful-scan") }}: <RelativeTime :date="new Date(state.lastSuccessAt)" />
        </div>
      </div>

      <div class="flex flex-wrap gap-2" v-if="availablePackageTypes.length > 0">
        <button
          class="btn btn-xs"
          :class="{ 'btn-primary': packageTypeFilter.includes(packageType) }"
          v-for="packageType in availablePackageTypes"
          :key="packageType"
          @click="togglePackageType(packageType)"
        >
          {{ packageType }}
        </button>
      </div>

      <div class="flex flex-wrap gap-2" v-if="availableSeverities.length > 0">
        <button
          class="btn btn-xs"
          :class="{ 'btn-primary': severityFilter.includes(severity) }"
          v-for="severity in availableSeverities"
          :key="severity"
          @click="toggleSeverity(severity)"
        >
          {{ severity }}
        </button>
      </div>

      <div class="alert alert-info" v-if="!result">
        <span>{{ $t("scan.no-saved-scan") }}</span>
      </div>

      <div class="stats stats-vertical md:stats-horizontal bg-base-200 w-full shadow-sm" v-else>
        <div class="stat">
          <div class="stat-title">{{ $t("scan.critical") }}</div>
          <div class="stat-value text-error">{{ filteredSummary.critical }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">{{ $t("scan.high") }}</div>
          <div class="stat-value text-warning">{{ filteredSummary.high }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">{{ $t("scan.medium") }}</div>
          <div class="stat-value text-info">{{ filteredSummary.medium }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">{{ $t("scan.low") }}</div>
          <div class="stat-value">{{ filteredSummary.low }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">{{ $t("scan.total") }}</div>
          <div class="stat-value">{{ filteredSummary.total }}</div>
        </div>
      </div>

      <div class="alert alert-success" v-if="result && filteredSummary.total === 0">
        <span>{{ $t("scan.no-vulnerabilities") }}</span>
      </div>

      <div class="alert alert-warning" v-if="statusError">
        <span>{{ statusError }}</span>
      </div>

      <div class="space-y-4" v-for="item in filteredResults" :key="item.target">
        <div class="flex items-center gap-2">
          <h3 class="font-semibold">{{ item.target }}</h3>
          <div class="badge badge-outline" v-if="item.type">{{ item.type }}</div>
        </div>

        <div class="overflow-x-auto" v-if="item.vulnerabilities.length > 0">
          <table class="table table-sm">
            <thead>
              <tr>
                <th>CVE</th>
                <th>{{ $t("scan.package") }}</th>
                <th>{{ $t("scan.installed") }}</th>
                <th>{{ $t("scan.fixed") }}</th>
                <th>{{ $t("scan.severity") }}</th>
                <th v-if="enableScanStatus">{{ $t("scan.status") }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="vulnerability in item.vulnerabilities" :key="`${item.target}:${vulnerability.id}`">
                <td class="font-mono">
                  <a
                    :href="vulnerability.primaryUrl"
                    class="link link-hover"
                    target="_blank"
                    rel="noreferrer noopener"
                    v-if="vulnerability.primaryUrl"
                  >
                    {{ vulnerability.id }}
                  </a>
                  <span v-else>{{ vulnerability.id }}</span>
                </td>
                <td>{{ vulnerability.packageName }}</td>
                <td class="font-mono">{{ vulnerability.installedVersion || "-" }}</td>
                <td class="font-mono">{{ vulnerability.fixedVersion || "-" }}</td>
                <td>
                  <span
                    class="badge badge-outline"
                    :class="{
                      'badge-error': vulnerability.severity === 'CRITICAL',
                      'badge-warning': vulnerability.severity === 'HIGH',
                      'badge-info': vulnerability.severity === 'MEDIUM',
                    }"
                  >
                    {{ vulnerability.severity }}
                  </span>
                </td>
                <td v-if="enableScanStatus">
                  <a
                    v-if="cveIssueById[vulnerability.id]"
                    class="badge badge-outline"
                    :class="statusBadgeClass(cveIssueById[vulnerability.id].status)"
                    :href="cveIssueById[vulnerability.id].url"
                    target="_blank"
                    rel="noreferrer noopener"
                  >
                    {{ cveIssueById[vulnerability.id].statusLabel || cveIssueById[vulnerability.id].status }}
                  </a>
                  <span class="text-base-content/50" v-else>-</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import type { ContainerScanState, ScanStatusIssue, ScanStatusResponse, ScanSummary, ScanTargetResult } from "@/types/scans";

const { t } = useI18n();
const { container } = defineProps<{ container: Container }>();

const loading = ref(false);
const error = ref("");
const state = ref<ContainerScanState>();
const statusSummary = ref<ScanStatusResponse>();
const statusError = ref("");
const statusLoading = ref(false);
const severityFilter = ref<string[]>([]);
const packageTypeFilter = ref<string[]>([]);
const enableScanStatus = config.enableScanStatus === true;

const result = computed(() => state.value?.result);
const scanLog = computed(() => state.value?.scanLog ?? []);
const availableSeverities = computed(() => state.value?.severities ?? []);
const availablePackageTypes = computed(() => state.value?.packageTypes ?? []);
const filteredResults = computed(() => {
  if (!result.value) return [] as ScanTargetResult[];
  return result.value.results
    .filter((item) => packageTypeFilter.value.length === 0 || packageTypeFilter.value.includes(item.type || ""))
    .map((item) => ({
      ...item,
      vulnerabilities: item.vulnerabilities.filter(
        (v) => severityFilter.value.length === 0 || severityFilter.value.includes(v.severity),
      ),
    }))
    .filter((item) => item.vulnerabilities.length > 0);
});
const filteredSummary = computed<ScanSummary>(() => {
  return filteredResults.value.reduce<ScanSummary>(
    (summary, item) => {
      for (const vulnerability of item.vulnerabilities) {
        summary.total += 1;
        if (vulnerability.severity === "CRITICAL") summary.critical += 1;
        else if (vulnerability.severity === "HIGH") summary.high += 1;
        else if (vulnerability.severity === "MEDIUM") summary.medium += 1;
        else if (vulnerability.severity === "LOW") summary.low += 1;
        else summary.unknown += 1;
      }
      return summary;
    },
    { critical: 0, high: 0, medium: 0, low: 0, unknown: 0, total: 0 },
  );
});
const cveIssues = computed<ScanStatusIssue[]>(() => [
  ...(statusSummary.value?.cve?.open ?? []),
  ...(statusSummary.value?.cve?.resolved ?? []),
  ...(statusSummary.value?.cve?.falsePositive ?? []),
]);
const cveIssueById = computed<Record<string, ScanStatusIssue>>(() =>
  Object.fromEntries(cveIssues.value.map((issue) => extractCveKeys(issue).map((key) => [key, issue])).flat()),
);

async function fetchState() {
  const response = await fetch(withBase(`/api/hosts/${container.host}/containers/${container.id}/scan`));
  if (!response.ok) return;
  state.value = await response.json();
  if (enableScanStatus && state.value.result && statusSummary.value?.image !== state.value.result.image) {
    void fetchStatusSummary(false);
  }
}

let pollHandle: ReturnType<typeof setInterval> | undefined;

function startPolling() {
  stopPolling();
  pollHandle = setInterval(async () => {
    await fetchState();
    if (!state.value?.running) {
      stopPolling();
      loading.value = false;
    }
  }, 1000);
}

function stopPolling() {
  if (pollHandle) {
    clearInterval(pollHandle);
    pollHandle = undefined;
  }
}

async function runScan() {
  loading.value = true;
  error.value = "";
  statusSummary.value = undefined;
  statusError.value = "";

  try {
    await fetchState();
    startPolling();

    const response = await fetch(withBase(`/api/hosts/${container.host}/containers/${container.id}/scan/run?force=1`), {
      method: "POST",
    });

    if (!response.ok) {
      error.value = (await response.text()).trim() || t("error.unable-to-run-trivy-scan");
      stopPolling();
      return;
    }

    state.value = await response.json();
  } catch (e) {
    error.value = e instanceof Error ? e.message : t("error.unable-to-run-trivy-scan");
    stopPolling();
  } finally {
    if (!state.value?.running) {
      loading.value = false;
      stopPolling();
    }
  }
}

async function fetchStatusSummary(includeHistory: boolean) {
  if (!enableScanStatus || !result.value) return;

  statusLoading.value = true;
  statusError.value = "";

  try {
    const response = await fetch(withBase(`/api/hosts/${container.host}/containers/${container.id}/scan/status`));
    if (!response.ok) {
      throw new Error((await response.text()).trim() || t("scan.unable-to-load-status"));
    }
    statusSummary.value = await response.json();
  } catch (e) {
    statusError.value = e instanceof Error ? e.message : t("scan.unable-to-load-status");
  } finally {
    statusLoading.value = false;
  }
}

function extractCveKeys(issue: ScanStatusIssue): string[] {
  const keys = new Set<string>();
  for (const label of issue.labels ?? []) {
    if (label.startsWith("cve:")) {
      keys.add(label.slice(4));
    }
  }
  if (issue.title) {
    const match = issue.title.match(/CVE-\d{4}-\d+/i);
    if (match) keys.add(match[0].toUpperCase());
  }
  return [...keys];
}

function statusBadgeClass(status?: string) {
  switch ((status || "").toLowerCase()) {
    case "open":
    case "opened":
    case "warning":
      return "badge-warning";
    case "resolved":
    case "closed":
      return "badge-success";
    case "false_positive":
      return "badge-neutral";
    default:
      return "badge-ghost";
  }
}

function toggleSeverity(value: string) {
  if (severityFilter.value.includes(value)) {
    severityFilter.value = severityFilter.value.filter((item) => item !== value);
  } else {
    severityFilter.value.push(value);
  }
}

function togglePackageType(value: string) {
  if (packageTypeFilter.value.includes(value)) {
    packageTypeFilter.value = packageTypeFilter.value.filter((item) => item !== value);
  } else {
    packageTypeFilter.value.push(value);
  }
}

onMounted(async () => {
  await fetchState();
  if (!state.value?.result) {
    await runScan();
  } else if (state.value?.running) {
    loading.value = true;
    startPolling();
  }
});

onScopeDispose(() => {
  stopPolling();
});
</script>
