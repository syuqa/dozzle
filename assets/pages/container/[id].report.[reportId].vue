<template>
  <Search />
  <section v-if="currentContainer" class="space-y-6 p-4 md:p-6">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div class="min-w-0">
        <h1 class="truncate text-2xl font-bold">{{ reportState.data?.title || reportDefinition?.name || reportId }}</h1>
        <p v-if="reportDefinition?.description" class="text-base-content/60 mt-2 text-sm">
          {{ reportDefinition.description }}
        </p>
        <div v-if="reportState.data?.format" class="mt-2">
          <span class="badge badge-outline">{{ reportState.data.format }}</span>
        </div>
        <div class="text-base-content/60 mt-2 flex flex-wrap gap-x-4 gap-y-1 text-sm">
          <span>{{ currentContainer.name }}</span>
          <span>{{ currentContainer.hostLabel }}</span>
          <span>{{ containerCardStateLabel(currentContainer.state) }}</span>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <router-link class="btn btn-ghost btn-sm" :to="{ name: '/container/[id].details', params: { id: currentContainer.id } }">
          <mdi:arrow-left />
          {{ $t("card-templates.detail-fields") }}
        </router-link>
        <button class="btn btn-ghost btn-sm" :disabled="reportState.loading" @click="loadReport(true)">
          <span v-if="reportState.loading" class="loading loading-spinner loading-xs"></span>
          <mdi:refresh v-else />
          {{ $t("button.refresh") }}
        </button>
      </div>
    </div>

    <article class="rounded-box border-base-content/10 bg-base-100 border p-5 shadow-sm">
      <div v-if="reportState.loading" class="text-base-content/60">
        {{ $t("card-templates.report-loading") }}
      </div>
      <div v-else-if="reportState.error" class="text-error break-all">
        {{ reportState.error }}
      </div>
      <div v-else-if="reportState.data?.type === 'table'" class="overflow-x-auto">
        <table class="table table-zebra w-full">
          <thead>
            <tr>
              <th v-for="column in reportState.data?.columns || []" :key="column.key">{{ column.label }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(row, index) in reportState.data?.rows || []"
              :key="index"
              :class="{ 'bg-primary/10': matchesReportRow(index, row) }"
              :ref="(el) => setSearchMatchRef(reportRowMatchId(index), el)"
            >
              <td
                v-for="(column, columnIndex) in reportState.data?.columns || []"
                :key="column.key"
                class="break-all align-top"
                :class="{ 'align-top': columnIndex === 0 }"
              >
                <router-link
                  v-if="column.kind === 'report_link' && column.reportId"
                  class="btn btn-ghost btn-xs"
                  :to="buildLinkedReportRoute(column, row)"
                >
                  {{ column.buttonLabel || $t("card-templates.open-report") }}
                </router-link>
                <a
                  v-else-if="column.kind === 'external_link' && column.urlKey"
                  class="link link-primary"
                  :href="row.values[column.urlKey] || '#'"
                  target="_blank"
                  rel="noreferrer noopener"
                >
                  {{ row.values[column.key] || "—" }}
                </a>
                <ul v-else-if="asDelimitedList(row.values[column.key]).length > 1" class="space-y-1">
                  <li
                    v-for="item in asDelimitedList(row.values[column.key])"
                    :key="`${column.key}:${item}`"
                    class="bg-base-200/60 rounded-box truncate px-2 py-1 text-xs"
                    :title="item"
                  >
                    {{ item }}
                  </li>
                </ul>
                <span v-else class="block truncate" :title="row.values[column.key] || '—'">{{ row.values[column.key] || "—" }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <pre
        v-else-if="reportState.data?.text"
        ref="reportTextElement"
        :class="{ 'ring-primary ring-2': matchesReportText }"
        class="bg-base-200/60 rounded-box min-w-0 whitespace-pre-wrap break-all p-4 text-left text-sm"
      >{{ reportState.data?.text }}</pre>
      <div v-else class="text-base-content/60">{{ $t("card-templates.report-not-generated") }}</div>
    </article>
  </section>

  <div v-else-if="ready" class="hero bg-base-200 min-h-screen">
    <div class="hero-content text-center">
      <div class="max-w-md">
        <p class="py-6 text-2xl font-bold">{{ $t("error.container-not-found") }}</p>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { containerCardStateLabel } from "@/stores/containerCardTemplates";
import { containerCardReports, useContainerCardReports } from "@/stores/containerCardReports";
import { withBase } from "@/stores/config";
import { useSearchFilter } from "@/composable/search";

type DetailReportResponse = {
  title: string;
  type: "text" | "table";
  format?: string;
  columns?: Array<{
    key: string;
    label: string;
    kind?: string;
    reportId?: string;
    paramKey?: string;
    buttonLabel?: string;
    passRow?: boolean;
    urlKey?: string;
  }>;
  rows?: Array<{
    values: Record<string, string>;
  }>;
  text?: string;
  generatedAt: string;
  durationMs: number;
  report: string;
};

const route = useRoute("/container/[id].report.[reportId]");
const id = toRef(() => route.params.id);
const reportId = toRef(() => route.params.reportId);
const containerStore = useContainerStore();
const currentContainer = containerStore.currentContainer(id);
const { ready } = storeToRefs(containerStore);
const { t } = useI18n();
const { isSearching, debouncedSearchFilter } = useSearchFilter();
useContainerCardReports();

const reportState = reactive<{ loading: boolean; error?: string; data?: DetailReportResponse }>({ loading: false });
const reportDefinition = computed(() => containerCardReports.value.find((item) => item.id === reportId.value));
const searchMatchElements = new Map<string, HTMLElement>();
const reportTextElement = ref<HTMLElement>();

function asDelimitedList(value?: string) {
  if (!value || !value.includes(",")) return [] as string[];
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function normalizedSearchQuery() {
  return debouncedSearchFilter.value.trim().toLowerCase();
}

function includesSearchQuery(value: string) {
  const query = normalizedSearchQuery();
  return query !== "" && value.toLowerCase().includes(query);
}

function reportRowMatchId(index: number) {
  return `report-row:${index}`;
}

function setSearchMatchRef(matchID: string, el: any) {
  if (!el || !(el instanceof HTMLElement)) {
    searchMatchElements.delete(matchID);
    return;
  }
  searchMatchElements.set(matchID, el);
}

function matchesReportRow(index: number, row: { values: Record<string, string> }) {
  return includesSearchQuery(`${index} ${Object.values(row.values).join(" ")}`);
}

const matchesReportText = computed(() => includesSearchQuery(reportState.data?.text || ""));

const firstSearchMatchId = computed(() => {
  if (!isSearching.value || !normalizedSearchQuery()) return "";
  for (const [index, row] of (reportState.data?.rows || []).entries()) {
    if (matchesReportRow(index, row)) {
      return reportRowMatchId(index);
    }
  }
  if (matchesReportText.value) {
    return "report-text";
  }
  return "";
});

function scrollToSearchMatch(matchID: string) {
  if (!matchID || typeof window === "undefined") return;
  const target = matchID === "report-text" ? reportTextElement.value : searchMatchElements.get(matchID);
  if (!target) return;
  const top = window.scrollY + target.getBoundingClientRect().top - 96;
  window.scrollTo({ top: Math.max(0, top), behavior: "smooth" });
}

function buildLinkedReportRoute(
  column: NonNullable<DetailReportResponse["columns"]>[number],
  row: NonNullable<DetailReportResponse["rows"]>[number],
) {
  if (!currentContainer.value || !column.reportId) {
    return { name: "/container/[id].report.[reportId]", params: { id: currentContainer.value?.id || "", reportId: reportId.value } };
  }
  const paramKey = column.paramKey || column.key;
  const query: Record<string, string> = {
    field: paramKey,
    key: row.values[paramKey] || "",
  };
  if (column.passRow) {
    query.row = JSON.stringify(row.values);
  }
  return {
    name: "/container/[id].report.[reportId]",
    params: { id: currentContainer.value.id, reportId: column.reportId },
    query,
  };
}

async function loadReport(force = false) {
  if (!currentContainer.value) return;
  reportState.loading = true;
  reportState.error = undefined;
  try {
    const params = new URLSearchParams();
    if (force) params.set("force", "1");
    if (typeof route.query.field === "string") params.set("field", route.query.field);
    if (typeof route.query.key === "string") params.set("key", route.query.key);
    if (typeof route.query.row === "string") params.set("row", route.query.row);
    const suffix = params.size ? `?${params.toString()}` : "";
    const response = await fetch(withBase(`/api/hosts/${currentContainer.value.host}/containers/${currentContainer.value.id}/detail-reports/${reportId.value}${suffix}`));
    if (response.status === 204) {
      reportState.data = undefined;
      reportState.error = undefined;
      return;
    }
    if (!response.ok) {
      throw new Error(await response.text());
    }
    reportState.data = (await response.json()) as DetailReportResponse;
  } catch (error) {
    reportState.error = error instanceof Error ? error.message : String(error);
  } finally {
    reportState.loading = false;
  }
}

watch(
  () => [currentContainer.value?.id, currentContainer.value?.host, reportId.value, route.query.field, route.query.key, route.query.row, reportDefinition.value?.refreshMode],
  async () => {
    reportState.data = undefined;
    reportState.error = undefined;
    if ((reportDefinition.value?.refreshMode || "ttl") !== "manual") {
      await loadReport();
    }
  },
  { immediate: true },
);

watch(
  () => firstSearchMatchId.value,
  async (matchID) => {
    if (!matchID) return;
    await nextTick();
    scrollToSearchMatch(matchID);
  },
);

watchEffect(() => {
  if (!ready.value) return;
  if (!currentContainer.value) {
    setTitle("Not Found");
    return;
  }
  setTitle(`${reportState.data?.title || reportDefinition.value?.name || reportId.value} · ${currentContainer.value.name}`);
});
</script>

<route lang="yaml">
meta:
  menu: host
</route>
