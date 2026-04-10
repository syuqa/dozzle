<template>
  <Search />
  <section v-if="currentContainer" class="space-y-6 p-4 md:p-6">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div class="flex min-w-0 items-start gap-4">
        <div
          v-if="currentTemplate?.iconUrl"
          class="bg-base-200/70 flex h-14 w-14 shrink-0 items-center justify-center overflow-hidden rounded-2xl"
        >
          <img :src="currentTemplate.iconUrl" :alt="currentTemplate.name" class="h-full w-full object-cover" />
        </div>
        <div class="min-w-0">
          <h1 class="truncate text-2xl font-bold">{{ currentContainer.name }}</h1>
          <div class="text-base-content/60 mt-2 flex flex-wrap gap-x-4 gap-y-1 text-sm">
            <span>{{ currentContainer.hostLabel }}</span>
            <span>{{ containerCardStateLabel(currentContainer.state) }}</span>
            <span v-if="currentContainer.namespace">{{ currentContainer.namespace }}</span>
          </div>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <router-link class="btn btn-ghost btn-sm" :to="{ name: '/container/[id]', params: { id: currentContainer.id } }">
          <mdi:file-document-outline />
          {{ $t("card-templates.open-logs") }}
        </router-link>
      </div>
    </div>

    <div class="flex flex-wrap items-center justify-end gap-2">
      <button class="btn btn-ghost btn-sm" @click="expandAllSections">
        <mdi:unfold-more-horizontal />
        {{ $t("card-templates.expand-all-sections") }}
      </button>
      <button class="btn btn-ghost btn-sm" @click="collapseAllSections">
        <mdi:unfold-less-horizontal />
        {{ $t("card-templates.collapse-all-sections") }}
      </button>
    </div>

    <article
      v-if="sectionNavItems.length"
      ref="sectionNavElement"
      class="bg-base-100/95 rounded-box border-base-content/10 sticky top-3 z-10 border p-3 shadow-sm backdrop-blur"
    >
      <div class="mb-2 text-sm font-medium">{{ $t("card-templates.section-navigation") }}</div>
      <div class="flex flex-wrap gap-2">
        <button
          v-for="item in sectionNavItems"
          :key="item.id"
          class="btn btn-xs md:btn-sm"
          :class="activeSectionId === item.id ? 'btn-primary' : 'btn-ghost'"
          @click="scrollToSection(item.id)"
        >
          {{ item.label }}
        </button>
      </div>
    </article>

    <article
      v-for="group in infoGroups"
      :key="group.id"
      :ref="(el) => setSectionRef(sectionDomId(groupSectionId(group.id)), el)"
      :id="sectionDomId(groupSectionId(group.id))"
      :data-section-id="groupSectionId(group.id)"
      class="rounded-box border-base-content/10 bg-base-100 scroll-mt-36 border p-5 shadow-sm"
    >
      <div class="mb-4 flex items-center justify-between gap-3">
        <h2 class="truncate text-lg font-semibold">{{ group.name }}</h2>
        <button class="btn btn-ghost btn-sm" @click="toggleGroup(group.id)">
          <mdi:chevron-down v-if="isGroupExpanded(group.id)" />
          <mdi:chevron-right v-else />
          {{ isGroupExpanded(group.id) ? $t("card-templates.collapse-section") : $t("card-templates.expand-section") }}
        </button>
      </div>
      <div v-if="isGroupExpanded(group.id)" class="grid gap-3">
        <div
          v-for="field in group.fields"
          :key="field.id"
          class="bg-base-200/60 rounded-box grid min-w-0 grid-cols-[minmax(0,1fr)_minmax(0,1.6fr)] items-start gap-3 px-4 py-3 text-sm"
          :class="{ 'ring-primary ring-2': matchesDetailField(group.id, field) }"
          :ref="(el) => setSearchMatchRef(detailFieldMatchId(group.id, field.id), el)"
        >
          <span class="text-base-content/60 min-w-0 truncate">{{ displayContainerCardFieldLabel(field) }}</span>
          <div class="min-w-0 text-right font-medium">
            <a
              v-if="field.kind === 'link'"
              class="link link-primary block truncate"
              :href="resolveDetailLink(field)"
              :title="resolveDetailLink(field)"
              target="_blank"
              rel="noopener noreferrer"
            >
              {{ resolveDetailLinkText(field) || "—" }}
            </a>
            <span
              v-else-if="field.kind === 'template_text'"
              class="min-w-0 truncate text-right font-medium"
              :title="resolveDetailTemplateText(field) || '—'"
            >
              {{ resolveDetailTemplateText(field) || "—" }}
            </span>
            <span
              v-else-if="field.kind === 'static_text'"
              class="min-w-0 truncate text-right font-medium"
              :title="field.textValue || '—'"
            >
              {{ field.textValue || "—" }}
            </span>
            <span
              v-else
              class="min-w-0 truncate text-right font-medium"
              :title="resolveContainerCardField(currentContainer, field.source) || '—'"
            >
              {{ resolveContainerCardField(currentContainer, field.source) || "—" }}
            </span>
          </div>
        </div>
      </div>
    </article>

    <article
      v-for="field in pageReportFields"
      :key="`page-report-${field.id}`"
      :id="sectionDomId(pageReportSectionId(field.id))"
      :data-section-id="pageReportSectionId(field.id)"
      class="rounded-box border-base-content/10 bg-base-100 scroll-mt-36 border p-5 shadow-sm"
      :class="{ 'ring-primary ring-2': matchesPageReportCard(field) }"
      :ref="(el) => { setSectionRef(sectionDomId(pageReportSectionId(field.id)), el); setSearchMatchRef(pageReportMatchId(field.id), el); }"
    >
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="min-w-0">
          <h2 class="truncate text-lg font-semibold">{{ displayContainerCardFieldLabel(field) }}</h2>
          <p v-if="field.reportId" class="text-base-content/60 mt-1 text-sm">
            <code>{{ field.reportId }}</code>
          </p>
          <div v-if="reportDefinitions.get(field.reportId || '')?.parser === 'file_content'" class="mt-2">
            <span class="badge badge-outline">text</span>
          </div>
        </div>
        <router-link
          class="btn btn-ghost btn-sm"
          :to="{ name: '/container/[id].report.[reportId]', params: { id: currentContainer.id, reportId: field.reportId! } }"
        >
          <mdi:open-in-new />
          {{ $t("card-templates.open-report") }}
        </router-link>
      </div>
    </article>

    <article
      v-for="field in inlineReportFields"
      :key="`report-${field.id}`"
      :ref="(el) => { setSectionRef(sectionDomId(reportSectionId(field.id)), el); setSearchMatchRef(inlineReportMatchId(field.id), el); }"
      :id="sectionDomId(reportSectionId(field.id))"
      :data-section-id="reportSectionId(field.id)"
      class="rounded-box border-base-content/10 bg-base-100 scroll-mt-36 border p-5 shadow-sm"
      :class="{ 'ring-primary ring-2': matchesInlineReportCard(field) }"
    >
      <div class="mb-4 flex flex-wrap items-start justify-between gap-3">
        <div class="min-w-0">
          <h2 class="truncate text-lg font-semibold">
            {{ reportDefinitions.get(field.reportId || "")?.name || displayContainerCardFieldLabel(field) }}
          </h2>
          <p v-if="reportDefinitions.get(field.reportId || '')?.description" class="text-base-content/60 mt-1 text-sm">
            {{ reportDefinitions.get(field.reportId || "")?.description }}
          </p>
          <p v-else-if="field.reportId" class="text-base-content/60 mt-1 text-sm">
            <code>{{ field.reportId }}</code>
          </p>
          <div v-if="reportState(field).data?.format" class="mt-2">
            <span class="badge badge-outline">{{ reportState(field).data?.format }}</span>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <button class="btn btn-ghost btn-sm" @click="toggleReport(field.id)">
            <mdi:chevron-down v-if="isReportExpanded(field.id)" />
            <mdi:chevron-right v-else />
            {{ isReportExpanded(field.id) ? $t("card-templates.collapse-section") : $t("card-templates.expand-section") }}
          </button>
          <button class="btn btn-ghost btn-sm" :disabled="reportState(field).loading" @click="loadReport(field, true)">
            <span v-if="reportState(field).loading" class="loading loading-spinner loading-xs"></span>
            <mdi:refresh v-else />
            {{ $t("button.refresh") }}
          </button>
          <div v-if="reportState(field).data?.generatedAt" class="text-base-content/60 text-sm">
            <RelativeTime :date="new Date(reportState(field).data!.generatedAt)" />
          </div>
        </div>
      </div>

      <template v-if="isReportExpanded(field.id)">
      <div v-if="reportState(field).loading" class="text-base-content/60">
        {{ $t("card-templates.report-loading") }}
      </div>
      <div v-else-if="reportState(field).error" class="text-error break-all">
        {{ reportState(field).error }}
      </div>
      <div v-else-if="reportState(field).data?.type === 'table'" class="overflow-x-auto">
        <table class="table table-zebra w-full">
          <thead>
            <tr>
              <th v-for="column in reportState(field).data?.columns || []" :key="column.key">{{ column.label }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(row, index) in reportState(field).data?.rows || []"
              :key="`${field.id}-${index}`"
              :class="{ 'bg-primary/10': matchesInlineReportRow(field.id, index, row) }"
              :ref="(el) => setSearchMatchRef(inlineReportRowMatchId(field.id, index), el)"
            >
              <td
                v-for="(column, columnIndex) in reportState(field).data?.columns || []"
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
        v-else-if="reportState(field).data?.text"
        class="bg-base-200/60 rounded-box min-w-0 whitespace-pre-wrap break-all p-4 text-left text-sm"
      >{{ reportState(field).data?.text }}</pre>
      <div v-else class="text-base-content/60">{{ $t("card-templates.report-not-generated") }}</div>
      </template>
    </article>

    <article
      v-if="vulnerabilitiesEnabled"
      :ref="(el) => setSectionRef(sectionDomId(vulnerabilitySectionId), el)"
      :id="sectionDomId(vulnerabilitySectionId)"
      :data-section-id="vulnerabilitySectionId"
      class="rounded-box border-base-content/10 bg-base-100 scroll-mt-36 border p-5 shadow-sm"
    >
      <div class="mb-4 flex flex-wrap items-start justify-between gap-3">
        <div class="min-w-0">
          <h2 class="truncate text-lg font-semibold">{{ $t("card-templates.vulnerability-report-title") }}</h2>
          <p class="text-base-content/60 mt-1 text-sm">
            {{
              currentTemplate?.vulnerabilityPackageTypes?.length
                ? currentTemplate.vulnerabilityPackageTypes.join(", ")
                : $t("card-templates.vulnerability-all-package-types")
            }}
            ·
            {{
              currentTemplate?.vulnerabilitySeverities?.length
                ? currentTemplate.vulnerabilitySeverities.join(", ")
                : $t("card-templates.vulnerability-all-severities")
            }}
          </p>
        </div>
        <div class="flex items-center gap-2">
          <button class="btn btn-ghost btn-sm" @click="toggleVulnerabilitySection">
            <mdi:chevron-down v-if="vulnerabilityExpanded" />
            <mdi:chevron-right v-else />
            {{ vulnerabilityExpanded ? $t("card-templates.collapse-section") : $t("card-templates.expand-section") }}
          </button>
          <button class="btn btn-ghost btn-sm" :disabled="vulnerabilityLoading" @click="loadVulnerabilities">
            <span v-if="vulnerabilityLoading" class="loading loading-spinner loading-xs"></span>
            <mdi:refresh v-else />
            {{ $t("button.refresh") }}
          </button>
        </div>
      </div>

      <template v-if="vulnerabilityExpanded">
      <div v-if="vulnerabilityLoading" class="text-base-content/60">{{ $t("card-templates.vulnerability-loading") }}</div>
      <div v-else-if="vulnerabilityError" class="text-error break-all">{{ vulnerabilityError }}</div>
      <div v-else-if="!vulnerabilityState?.result" class="text-base-content/60">{{ $t("card-templates.vulnerability-no-scan") }}</div>
      <template v-else>
        <div v-if="vulnerabilityStatusError" class="alert alert-warning mb-4">
          <span>{{ vulnerabilityStatusError }}</span>
        </div>
        <div class="stats stats-vertical md:stats-horizontal bg-base-200 mb-4 w-full shadow-sm">
          <div class="stat">
            <div class="stat-title">{{ $t("scan.critical") }}</div>
            <div class="stat-value text-error">{{ vulnerabilityFilteredSummary.critical }}</div>
          </div>
          <div class="stat">
            <div class="stat-title">{{ $t("scan.high") }}</div>
            <div class="stat-value text-warning">{{ vulnerabilityFilteredSummary.high }}</div>
          </div>
          <div class="stat">
            <div class="stat-title">{{ $t("scan.medium") }}</div>
            <div class="stat-value text-info">{{ vulnerabilityFilteredSummary.medium }}</div>
          </div>
          <div class="stat">
            <div class="stat-title">{{ $t("scan.low") }}</div>
            <div class="stat-value">{{ vulnerabilityFilteredSummary.low }}</div>
          </div>
          <div class="stat">
            <div class="stat-title">{{ $t("scan.total") }}</div>
            <div class="stat-value">{{ vulnerabilityFilteredSummary.total }}</div>
          </div>
        </div>

        <div v-if="vulnerabilityFilteredSummary.total === 0" class="text-base-content/60">
          {{ $t("scan.no-vulnerabilities") }}
        </div>

        <div v-for="item in vulnerabilityFilteredResults" :key="`vuln-${item.target}`" class="mb-4 space-y-3">
          <div class="flex items-center gap-2">
            <h3 class="font-semibold">{{ item.target }}</h3>
            <div class="badge badge-outline" v-if="item.type">{{ item.type }}</div>
          </div>
          <div class="overflow-x-auto">
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
                <tr
                  v-for="vulnerability in item.vulnerabilities"
                  :key="`${item.target}:${vulnerability.id}`"
                  :class="{ 'bg-primary/10': matchesVulnerabilityRow(item.target, vulnerability.id, vulnerability) }"
                  :ref="(el) => setSearchMatchRef(vulnerabilityMatchId(item.target, vulnerability.id), el)"
                >
                  <td class="font-mono">
                    <a v-if="vulnerability.primaryUrl" :href="vulnerability.primaryUrl" class="link link-hover" target="_blank" rel="noreferrer noopener">
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
      </template>
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
import { type Container } from "@/models/Container";
import {
  containerCardStateLabel,
  displayContainerCardFieldLabel,
  resolveContainerCardField,
  resolveContainerTemplateString,
  resolveContainerCardTemplate,
  useContainerCardTemplates,
  type ContainerCardDetailGroup,
  type ContainerCardField,
} from "@/stores/containerCardTemplates";
import { containerCardReports, useContainerCardReports } from "@/stores/containerCardReports";
import config, { withBase } from "@/stores/config";
import type { ContainerScanState, ScanStatusIssue, ScanStatusResponse, ScanSummary, ScanTargetResult } from "@/types/scans";
import { useSearchFilter } from "@/composable/search";

const route = useRoute("/container/[id].details");
const id = toRef(() => route.params.id);
const containerStore = useContainerStore();
const currentContainer = containerStore.currentContainer(id);
const { ready } = storeToRefs(containerStore);
const { t } = useI18n();
const { isSearching, debouncedSearchFilter } = useSearchFilter();
const { loaded: cardTemplatesLoaded } = useContainerCardTemplates();
const { loaded: cardReportsLoaded } = useContainerCardReports();

const detailGroups = computed<ContainerCardDetailGroup[]>(() => {
  if (!currentContainer.value || !cardTemplatesLoaded.value) return [];
  const template = resolveContainerCardTemplate(currentContainer.value as Container);
  if (template.detailGroups?.length) {
    return template.detailGroups;
  }
  if (template.detailFields?.length) {
    return [{ id: "general", name: t("card-templates.default-detail-group"), fields: template.detailFields }];
  }
  return [{ id: "general", name: t("card-templates.default-detail-group"), fields: template.extraFields }];
});
const currentTemplate = computed(() =>
  currentContainer.value && cardTemplatesLoaded.value ? resolveContainerCardTemplate(currentContainer.value as Container) : undefined,
);

const infoGroups = computed(() =>
  detailGroups.value
    .map((group) => ({
      ...group,
      fields: group.fields.filter((field) => field.kind !== "report"),
    }))
    .filter((group) => group.fields.length > 0),
);
const reportDefinitions = computed(() => new Map(containerCardReports.value.map((report) => [report.id, report])));
const reportFields = computed(() =>
  detailGroups.value.flatMap((group) =>
    group.fields.filter(
      (field) => field.kind === "report" && field.reportId && reportDefinitions.value.get(field.reportId)?.enabled !== false,
    ),
  ),
);
const vulnerabilitiesEnabled = computed(() => currentTemplate.value?.showVulnerabilities === true);

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

const reportStates = reactive<Record<string, { loading: boolean; error?: string; data?: DetailReportResponse }>>({});
const containerEnv = ref<Record<string, string>>({});
const vulnerabilityState = ref<ContainerScanState>();
const vulnerabilityError = ref("");
const vulnerabilityLoading = ref(false);
const vulnerabilityStatusSummary = ref<ScanStatusResponse>();
const vulnerabilityStatusError = ref("");
const expandedGroups = ref<Record<string, boolean>>({});
const expandedReports = ref<Record<string, boolean>>({});
const vulnerabilityExpanded = ref(true);
const activeSectionId = ref("");
const sectionNavElement = ref<HTMLElement>();
const sectionElements = new Map<string, HTMLElement>();
const searchMatchElements = new Map<string, HTMLElement>();
let sectionScrollFrame = 0;
let sectionSelectionLockUntil = 0;
const enableScanStatus = config.enableScanStatus === true;

function resolveDetailLink(field: ContainerCardField) {
  if (!currentContainer.value) return "#";
  return resolveContainerTemplateString(currentContainer.value as Container, field.linkUrl || "");
}

function resolveDetailLinkText(field: ContainerCardField) {
  if (!currentContainer.value) return "";
  return resolveContainerTemplateString(currentContainer.value as Container, field.linkText || field.linkUrl || "");
}

function resolveDetailTemplateText(field: ContainerCardField) {
  return containerEnv.value[String(field.textValue || "").trim()] || "";
}

async function loadContainerEnv() {
  if (!currentContainer.value) {
    containerEnv.value = {};
    return;
  }
  try {
    const response = await fetch(
      withBase(`/api/hosts/${encodeURIComponent(currentContainer.value.host)}/containers/${encodeURIComponent(currentContainer.value.id)}/env`),
    );
    if (!response.ok) {
      throw new Error(await response.text());
    }
    containerEnv.value = (await response.json()) as Record<string, string>;
  } catch {
    containerEnv.value = {};
  }
}

function reportState(field: ContainerCardField) {
  return reportStates[field.id] || { loading: false };
}

const inlineReportFields = computed(() =>
  reportFields.value.filter((field) => (field.reportId ? (reportDefinitions.value.get(field.reportId)?.displayMode || "inline") === "inline" : false)),
);
const pageReportFields = computed(() =>
  reportFields.value.filter((field) => (field.reportId ? reportDefinitions.value.get(field.reportId)?.displayMode === "page" : false)),
);
const autoInlineReportFields = computed(() =>
  inlineReportFields.value.filter((field) => {
    const mode = field.reportId ? reportDefinitions.value.get(field.reportId)?.refreshMode || "ttl" : "ttl";
    return mode !== "manual";
  }),
);

function buildLinkedReportRoute(
  column: NonNullable<DetailReportResponse["columns"]>[number],
  row: NonNullable<DetailReportResponse["rows"]>[number],
) {
  if (!currentContainer.value || !column.reportId) {
    return { name: "/container/[id].details", params: { id: currentContainer.value?.id || "" } };
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

function asDelimitedList(value?: string) {
  if (!value || !value.includes(",")) return [] as string[];
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function isGroupExpanded(groupID: string) {
  return expandedGroups.value[groupID] !== false;
}

function toggleGroup(groupID: string) {
  expandedGroups.value = { ...expandedGroups.value, [groupID]: !isGroupExpanded(groupID) };
}

function isReportExpanded(reportID: string) {
  return expandedReports.value[reportID] !== false;
}

function toggleReport(reportID: string) {
  expandedReports.value = { ...expandedReports.value, [reportID]: !isReportExpanded(reportID) };
}

function toggleVulnerabilitySection() {
  vulnerabilityExpanded.value = !vulnerabilityExpanded.value;
}

function groupSectionId(groupID: string) {
  return `group:${groupID}`;
}

function reportSectionId(fieldID: string) {
  return `report:${fieldID}`;
}

function pageReportSectionId(fieldID: string) {
  return `page-report:${fieldID}`;
}

const vulnerabilitySectionId = "vulnerabilities";

function sectionDomId(sectionID: string) {
  return `details-section-${sectionID.replace(/[^a-zA-Z0-9_-]/g, "-")}`;
}

const sectionNavItems = computed(() => [
  ...infoGroups.value.map((group) => ({ id: groupSectionId(group.id), label: group.name })),
  ...pageReportFields.value.map((field) => ({ id: pageReportSectionId(field.id), label: displayContainerCardFieldLabel(field) })),
  ...inlineReportFields.value.map((field) => ({ id: reportSectionId(field.id), label: displayContainerCardFieldLabel(field) })),
  ...(vulnerabilitiesEnabled.value ? [{ id: vulnerabilitySectionId, label: t("card-templates.vulnerability-report-title") }] : []),
]);

function setSectionRef(sectionID: string, el: any) {
  if (!el || !(el instanceof HTMLElement)) {
    sectionElements.delete(sectionID);
    return;
  }
  sectionElements.set(sectionID, el);
}

function setSearchMatchRef(matchID: string, el: any) {
  if (!el || !(el instanceof HTMLElement)) {
    searchMatchElements.delete(matchID);
    return;
  }
  searchMatchElements.set(matchID, el);
}

function sectionScrollOffset() {
  const navHeight = sectionNavElement.value?.offsetHeight || 0;
  return navHeight + 28;
}

function scrollToSection(sectionID: string) {
  activeSectionId.value = sectionID;
  sectionSelectionLockUntil = Date.now() + 900;
  const target = sectionElements.get(sectionDomId(sectionID));
  if (!target || typeof window === "undefined") return;

  const top = window.scrollY + target.getBoundingClientRect().top - sectionScrollOffset();
  window.scrollTo({ top: Math.max(0, top), behavior: "smooth" });
}

function normalizedSearchQuery() {
  return debouncedSearchFilter.value.trim().toLowerCase();
}

function includesSearchQuery(value: string) {
  const query = normalizedSearchQuery();
  return query !== "" && value.toLowerCase().includes(query);
}

function detailFieldMatchId(groupID: string, fieldID: string) {
  return `detail-field:${groupID}:${fieldID}`;
}

function pageReportMatchId(fieldID: string) {
  return `page-report:${fieldID}`;
}

function inlineReportMatchId(fieldID: string) {
  return `inline-report:${fieldID}`;
}

function inlineReportRowMatchId(fieldID: string, index: number) {
  return `inline-report-row:${fieldID}:${index}`;
}

function vulnerabilityMatchId(target: string, vulnerabilityID: string) {
  return `vulnerability:${target}:${vulnerabilityID}`;
}

function detailFieldSearchText(field: ContainerCardField) {
  if (!currentContainer.value) return displayContainerCardFieldLabel(field);
  if (field.kind === "link") {
    return `${displayContainerCardFieldLabel(field)} ${resolveDetailLinkText(field)} ${resolveDetailLink(field)}`;
  }
  if (field.kind === "template_text") {
    return `${displayContainerCardFieldLabel(field)} ${resolveDetailTemplateText(field)}`;
  }
  if (field.kind === "static_text") {
    return `${displayContainerCardFieldLabel(field)} ${field.textValue || ""}`;
  }
  return `${displayContainerCardFieldLabel(field)} ${resolveContainerCardField(currentContainer.value, field.source) || ""}`;
}

function matchesDetailField(groupID: string, field: ContainerCardField) {
  return includesSearchQuery(`${groupID} ${detailFieldSearchText(field)}`);
}

function matchesPageReportCard(field: ContainerCardField) {
  return includesSearchQuery(`${displayContainerCardFieldLabel(field)} ${field.reportId || ""}`);
}

function matchesInlineReportCard(field: ContainerCardField) {
  const state = reportState(field);
  return includesSearchQuery(
    `${displayContainerCardFieldLabel(field)} ${field.reportId || ""} ${state.data?.title || ""} ${state.data?.text || ""}`,
  );
}

function rowSearchText(row: { values: Record<string, string> }) {
  return Object.values(row.values).join(" ");
}

function matchesInlineReportRow(fieldID: string, index: number, row: { values: Record<string, string> }) {
  return includesSearchQuery(`${fieldID} ${index} ${rowSearchText(row)}`);
}

function matchesVulnerabilityRow(target: string, vulnerabilityID: string, vulnerability: ScanTargetResult["vulnerabilities"][number]) {
  return includesSearchQuery(
    `${target} ${vulnerabilityID} ${vulnerability.packageName} ${vulnerability.installedVersion} ${vulnerability.fixedVersion} ${vulnerability.severity}`,
  );
}

const firstSearchMatchId = computed(() => {
  if (!isSearching.value || !normalizedSearchQuery()) return "";

  for (const group of infoGroups.value) {
    for (const field of group.fields) {
      if (matchesDetailField(group.id, field)) {
        return detailFieldMatchId(group.id, field.id);
      }
    }
  }
  for (const field of pageReportFields.value) {
    if (matchesPageReportCard(field)) {
      return pageReportMatchId(field.id);
    }
  }
  for (const field of inlineReportFields.value) {
    if (matchesInlineReportCard(field)) {
      return inlineReportMatchId(field.id);
    }
    for (const [index, row] of (reportState(field).data?.rows || []).entries()) {
      if (matchesInlineReportRow(field.id, index, row)) {
        return inlineReportRowMatchId(field.id, index);
      }
    }
  }
  for (const item of vulnerabilityFilteredResults.value) {
    for (const vulnerability of item.vulnerabilities) {
      if (matchesVulnerabilityRow(item.target, vulnerability.id, vulnerability)) {
        return vulnerabilityMatchId(item.target, vulnerability.id);
      }
    }
  }
  return "";
});

function scrollToSearchMatch(matchID: string) {
  if (!matchID || typeof window === "undefined") return;
  const target = searchMatchElements.get(matchID);
  if (!target) return;
  const top = window.scrollY + target.getBoundingClientRect().top - sectionScrollOffset();
  window.scrollTo({ top: Math.max(0, top), behavior: "smooth" });
}

function updateActiveSectionFromScroll() {
  if (typeof window === "undefined" || !sectionNavItems.value.length) return;
  if (Date.now() < sectionSelectionLockUntil) return;

  const offset = sectionScrollOffset();
  let candidateId = sectionNavItems.value[0]?.id || "";
  let bestPastDistance = Number.POSITIVE_INFINITY;
  let bestFutureDistance = Number.POSITIVE_INFINITY;

  for (const item of sectionNavItems.value) {
    const element = sectionElements.get(sectionDomId(item.id));
    if (!element) continue;

    const delta = element.getBoundingClientRect().top - offset;
    if (delta <= 0 && Math.abs(delta) < bestPastDistance) {
      bestPastDistance = Math.abs(delta);
      candidateId = item.id;
      continue;
    }

    if (bestPastDistance === Number.POSITIVE_INFINITY && delta < bestFutureDistance) {
      bestFutureDistance = delta;
      candidateId = item.id;
    }
  }

  activeSectionId.value = candidateId;
}

function handleWindowScroll() {
  if (typeof window === "undefined") return;
  if (sectionScrollFrame) {
    window.cancelAnimationFrame(sectionScrollFrame);
  }
  sectionScrollFrame = window.requestAnimationFrame(() => {
    sectionScrollFrame = 0;
    updateActiveSectionFromScroll();
  });
}

function rebuildSectionTracking() {
  if (typeof window === "undefined") return;
  window.removeEventListener("scroll", handleWindowScroll);
  if (!sectionNavItems.value.length) return;
  window.addEventListener("scroll", handleWindowScroll, { passive: true });
  updateActiveSectionFromScroll();
}

function expandAllSections() {
  const groups: Record<string, boolean> = {};
  const reports: Record<string, boolean> = {};
  for (const group of infoGroups.value) {
    groups[group.id] = true;
  }
  for (const field of reportFields.value) {
    reports[field.id] = true;
  }
  expandedGroups.value = groups;
  expandedReports.value = reports;
  vulnerabilityExpanded.value = true;
}

function collapseAllSections() {
  const groups: Record<string, boolean> = {};
  const reports: Record<string, boolean> = {};
  for (const group of infoGroups.value) {
    groups[group.id] = false;
  }
  for (const field of reportFields.value) {
    reports[field.id] = false;
  }
  expandedGroups.value = groups;
  expandedReports.value = reports;
  vulnerabilityExpanded.value = false;
}

const vulnerabilityFilteredResults = computed(() => {
  const result = vulnerabilityState.value?.result;
  if (!result || !currentTemplate.value) return [] as ScanTargetResult[];

  const packageTypes = currentTemplate.value.vulnerabilityPackageTypes ?? [];
  const severities = currentTemplate.value.vulnerabilitySeverities ?? [];

  return result.results
    .filter((item) => packageTypes.length === 0 || packageTypes.includes(item.type || ""))
    .map((item) => ({
      ...item,
      vulnerabilities: item.vulnerabilities.filter((vulnerability) => severities.length === 0 || severities.includes(vulnerability.severity)),
    }))
    .filter((item) => item.vulnerabilities.length > 0);
});

const vulnerabilityFilteredSummary = computed<ScanSummary>(() =>
  vulnerabilityFilteredResults.value.reduce<ScanSummary>(
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
  ),
);

const cveIssues = computed<ScanStatusIssue[]>(() => [
  ...(vulnerabilityStatusSummary.value?.cve?.open ?? []),
  ...(vulnerabilityStatusSummary.value?.cve?.resolved ?? []),
  ...(vulnerabilityStatusSummary.value?.cve?.falsePositive ?? []),
]);

const cveIssueById = computed<Record<string, ScanStatusIssue>>(() =>
  Object.fromEntries(cveIssues.value.map((issue) => extractCveKeys(issue).map((key) => [key, issue])).flat()),
);

async function loadReport(field: ContainerCardField, force = false) {
  if (!currentContainer.value || !field.reportId) return;
  reportStates[field.id] = { loading: true };
  try {
    const suffix = force ? "?force=1" : "";
    const response = await fetch(withBase(`/api/hosts/${currentContainer.value.host}/containers/${currentContainer.value.id}/detail-reports/${field.reportId}${suffix}`));
    if (response.status === 204) {
      reportStates[field.id] = { loading: false, data: undefined, error: undefined };
      return;
    }
    if (!response.ok) {
      throw new Error(await response.text());
    }
    reportStates[field.id] = { loading: false, data: (await response.json()) as DetailReportResponse };
  } catch (error) {
    reportStates[field.id] = { loading: false, error: error instanceof Error ? error.message : String(error) };
  }
}

async function loadVulnerabilities() {
  if (!currentContainer.value || !vulnerabilitiesEnabled.value) return;
  vulnerabilityLoading.value = true;
  vulnerabilityError.value = "";
  try {
    const response = await fetch(withBase(`/api/hosts/${currentContainer.value.host}/containers/${currentContainer.value.id}/scan`));
    if (!response.ok) {
      throw new Error(await response.text());
    }
    vulnerabilityState.value = (await response.json()) as ContainerScanState;
    if (enableScanStatus && vulnerabilityState.value?.result) {
      await fetchVulnerabilityStatusSummary();
    }
  } catch (error) {
    vulnerabilityError.value = error instanceof Error ? error.message : String(error);
  } finally {
    vulnerabilityLoading.value = false;
  }
}

async function fetchVulnerabilityStatusSummary() {
  if (!enableScanStatus || !currentContainer.value || !vulnerabilityState.value?.result) return;

  vulnerabilityStatusError.value = "";
  try {
    const response = await fetch(withBase(`/api/hosts/${currentContainer.value.host}/containers/${currentContainer.value.id}/scan/status`));
    if (!response.ok) {
      throw new Error((await response.text()).trim() || t("scan.unable-to-load-status"));
    }
    vulnerabilityStatusSummary.value = (await response.json()) as ScanStatusResponse;
  } catch (error) {
    vulnerabilityStatusError.value = error instanceof Error ? error.message : t("scan.unable-to-load-status");
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

watch(
  () => [currentContainer.value?.id, currentContainer.value?.host],
  async () => {
    if (!cardTemplatesLoaded.value) {
      containerEnv.value = {};
      return;
    }
    await loadContainerEnv();
  },
  { immediate: true },
);

watch(
  () => [
    cardTemplatesLoaded.value,
    cardReportsLoaded.value,
    currentContainer.value?.id,
    currentContainer.value?.host,
    inlineReportFields.value.map((field) => `${field.id}:${field.kind}:${field.reportId || ""}`).join("|"),
  ],
  async () => {
    if (!cardTemplatesLoaded.value || !cardReportsLoaded.value || !currentContainer.value) {
      return;
    }
    Object.keys(reportStates).forEach((key) => {
      delete reportStates[key];
    });
    await Promise.all(autoInlineReportFields.value.map((field) => loadReport(field)));
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

watch(
  () => infoGroups.value.map((group) => group.id).join("|"),
  () => {
    const next: Record<string, boolean> = {};
    for (const group of infoGroups.value) {
      next[group.id] = expandedGroups.value[group.id] ?? true;
    }
    expandedGroups.value = next;
  },
  { immediate: true },
);

watch(
  () => reportFields.value.map((field) => field.id).join("|"),
  () => {
    const next: Record<string, boolean> = {};
    for (const field of reportFields.value) {
      next[field.id] = expandedReports.value[field.id] ?? true;
    }
    expandedReports.value = next;
  },
  { immediate: true },
);

watch(
  () => sectionNavItems.value.map((item) => item.id).join("|"),
  async () => {
    await nextTick();
    rebuildSectionTracking();
  },
  { immediate: true },
);

watch(
  () => [
    cardTemplatesLoaded.value,
    currentContainer.value?.id,
    currentContainer.value?.host,
    vulnerabilitiesEnabled.value,
    (currentTemplate.value?.vulnerabilityPackageTypes ?? []).join("|"),
    (currentTemplate.value?.vulnerabilitySeverities ?? []).join("|"),
  ],
  async () => {
    if (!cardTemplatesLoaded.value) {
      vulnerabilityState.value = undefined;
      vulnerabilityError.value = "";
      vulnerabilityStatusSummary.value = undefined;
      vulnerabilityStatusError.value = "";
      return;
    }
    vulnerabilityState.value = undefined;
    vulnerabilityError.value = "";
    vulnerabilityStatusSummary.value = undefined;
    vulnerabilityStatusError.value = "";
    if (vulnerabilitiesEnabled.value) {
      await loadVulnerabilities();
    }
  },
  { immediate: true },
);

watchEffect(() => {
  if (ready.value) {
    if (currentContainer.value) {
      setTitle(`${currentContainer.value.name} ${t("card-templates.details-page-title-suffix")}`);
    } else {
      setTitle("Not Found");
    }
  }
});

onBeforeUnmount(() => {
  if (typeof window !== "undefined") {
    window.removeEventListener("scroll", handleWindowScroll);
    if (sectionScrollFrame) {
      window.cancelAnimationFrame(sectionScrollFrame);
    }
  }
});
</script>

<route lang="yaml">
meta:
  menu: host
</route>
