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

    <div class="alert alert-info" v-if="loading">
      <span>{{ $t("scan.scanning-image") }}</span>
    </div>

    <div class="alert alert-error" v-else-if="error">
      <span>{{ error }}</span>
    </div>

    <template v-else-if="state">
      <div class="flex flex-wrap gap-3">
        <label class="label cursor-pointer gap-2">
          <span class="label-text">{{ $t("scan.scheduled") }}</span>
          <input type="checkbox" class="toggle toggle-sm" v-model="scheduleEnabled" @change="saveSchedule()" />
        </label>
        <label class="label gap-2">
          <span class="label-text">{{ $t("scan.interval-minutes") }}</span>
          <input
            type="number"
            min="5"
            class="input input-sm w-24"
            v-model.number="intervalMinutes"
            @change="saveSchedule()"
          />
        </label>
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
          <div class="stat-value text-error">{{ result.summary.critical }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">{{ $t("scan.high") }}</div>
          <div class="stat-value text-warning">{{ result.summary.high }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">{{ $t("scan.medium") }}</div>
          <div class="stat-value text-info">{{ result.summary.medium }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">{{ $t("scan.low") }}</div>
          <div class="stat-value">{{ result.summary.low }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">{{ $t("scan.total") }}</div>
          <div class="stat-value">{{ result.summary.total }}</div>
        </div>
      </div>

      <div class="alert alert-success" v-if="result && result.summary.total === 0">
        <span>{{ $t("scan.no-vulnerabilities") }}</span>
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
import type { ContainerScanState, ScanTargetResult } from "@/types/scans";

const { t } = useI18n();
const { container } = defineProps<{ container: Container }>();

const loading = ref(false);
const error = ref("");
const state = ref<ContainerScanState>();
const severityFilter = ref<string[]>([]);
const packageTypeFilter = ref<string[]>([]);
const scheduleEnabled = ref(false);
const intervalMinutes = ref(60);

const result = computed(() => state.value?.result);
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

async function fetchState() {
  const response = await fetch(withBase(`/api/hosts/${container.host}/containers/${container.id}/scan`));
  if (!response.ok) return;
  state.value = await response.json();
  scheduleEnabled.value = state.value.schedule.enabled;
  intervalMinutes.value = state.value.schedule.intervalMinutes || 60;
}

async function runScan() {
  loading.value = true;
  error.value = "";

  try {
    const response = await fetch(withBase(`/api/hosts/${container.host}/containers/${container.id}/scan/run?force=1`), {
      method: "POST",
    });

    if (!response.ok) {
      error.value = (await response.text()).trim() || t("error.unable-to-run-trivy-scan");
      return;
    }

    state.value = await response.json();
  } catch (e) {
    error.value = e instanceof Error ? e.message : t("error.unable-to-run-trivy-scan");
  } finally {
    loading.value = false;
  }
}

async function saveSchedule() {
  const response = await fetch(withBase(`/api/hosts/${container.host}/containers/${container.id}/scan/schedule`), {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ enabled: scheduleEnabled.value, intervalMinutes: intervalMinutes.value }),
  });
  if (response.ok) {
    state.value = await response.json();
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
  }
});
</script>
