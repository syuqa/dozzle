<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-row">
      <div v-if="Object.keys(hosts).length > 1" class="flex-1">
        <div role="tablist" class="tabs-boxed tabs block" v-if="Object.keys(hosts).length < 4">
          <input
            type="radio"
            name="host"
            role="tab"
            class="tab rounded-sm!"
            aria-label="Show All"
            v-model="selectedHost"
            :value="null"
          />
          <input
            type="radio"
            name="host"
            role="tab"
            class="tab rounded-sm!"
            :aria-label="host.name"
            v-for="host in hosts"
            :value="host.id"
            :key="host.id"
            v-model="selectedHost"
          />
        </div>

        <DropdownMenu
          class="btn-sm"
          v-model="selectedHost"
          :options="[
            { label: 'Show All', value: null },
            ...Object.values(hosts).map((host) => ({ label: host.name, value: host.id })),
          ]"
          v-else
        />
      </div>
      <div class="flex flex-1 items-center justify-end gap-2">
        <div v-show="containers.length > pageSizes[0]">
          {{ $t("label.per-page") }}

          <DropdownMenu
            class="dropdown-left btn-xs md:btn-sm"
            v-model="perPage"
            :options="pageSizes.map((i) => ({ label: i.toLocaleString(), value: i }))"
          />
        </div>
        <div class="join">
          <button
            class="btn join-item btn-xs md:btn-sm"
            :class="layoutMode === 'table' ? 'btn-active' : 'btn-ghost'"
            @click="layoutMode = 'table'"
            :title="$t('label.table-view')"
          >
            <mdi:table-large />
          </button>
          <button
            class="btn join-item btn-xs md:btn-sm"
            :class="layoutMode === 'cards' ? 'btn-active' : 'btn-ghost'"
            @click="layoutMode = 'cards'"
            :title="$t('label.card-view')"
          >
            <mdi:view-grid-outline />
          </button>
        </div>
        <div class="join max-md:hidden">
          <button
            class="btn join-item btn-xs md:btn-sm"
            :class="statMode === 'chart' ? 'btn-active' : 'btn-ghost'"
            @click="statMode = 'chart'"
          >
            <mdi:chart-bar />
          </button>
          <button
            class="btn join-item btn-xs md:btn-sm"
            :class="statMode === 'progress' ? 'btn-active' : 'btn-ghost'"
            @click="statMode = 'progress'"
          >
            <mdi:poll class="scale-x-[-1] rotate-90" />
          </button>
          <button
            class="btn join-item btn-xs md:btn-sm"
            :class="statMode === 'scan' ? 'btn-active' : 'btn-ghost'"
            @click="statMode = 'scan'"
            v-if="config.enableContainerScan"
            :title="$t('label.vulnerabilities')"
          >
            <mdi:shield-search />
          </button>
        </div>
      </div>
    </div>
    <div v-if="layoutMode === 'table'" class="rounded-box border-base-content/10 overflow-x-auto border">
      <table class="table-md md:table-lg table-zebra table">
        <thead>
          <tr :data-direction="direction > 0 ? 'asc' : 'desc'">
            <th
              v-for="(value, key) in fields"
              :key="key"
              @click.prevent="sort(key)"
              :class="[value.customClass, { 'selected-sort': key === sortField }]"
              v-show="isVisible(key)"
            >
              <a class="inline-flex cursor-pointer gap-2 text-sm uppercase">
                <span>{{ columnLabel(key) }}</span>
                <span class="h-4" data-icon>
                  <mdi:arrow-up />
                </span>
              </a>
            </th>
          </tr>
        </thead>
        <tbody class="bg-base-300/30">
          <tr
            v-for="container in paginated"
            :key="container.id"
            class="hover:bg-base-100/80!"
          >
            <td v-if="isVisible('name')" class="max-w-80 truncate">
              <router-link :to="{ name: '/container/[id]', params: { id: container.id } }" :title="container.name">
                {{ container.name }}
              </router-link>
            </td>
            <td v-if="isVisible('host')">{{ container.hostLabel }}</td>
            <td v-if="isVisible('state')">{{ container.state }}</td>
            <td v-if="isVisible('created')">
              <RelativeTime :date="container.created" />
            </td>
            <td v-if="isVisible('cpu')">
              <template v-if="statMode === 'scan'">
                <div class="flex flex-wrap items-center gap-2" v-if="scanItem(container)">
                  <span class="badge badge-error badge-outline" v-if="scanItem(container)?.summary.critical">
                    C {{ scanItem(container)?.summary.critical }}
                  </span>
                  <span class="badge badge-warning badge-outline" v-if="scanItem(container)?.summary.high">
                    H {{ scanItem(container)?.summary.high }}
                  </span>
                  <span class="badge badge-info badge-outline" v-if="scanItem(container)?.summary.medium">
                    M {{ scanItem(container)?.summary.medium }}
                  </span>
                  <span class="badge badge-ghost">{{ scanItem(container)?.summary.total || 0 }}</span>
                </div>
                <div class="text-base-content/70 text-sm" v-if="scanItem(container)?.running">
                  {{ $t("label.scanning") }}
                </div>
                <div class="text-error truncate text-sm" v-else-if="scanItem(container)?.lastError">
                  {{ scanItem(container)?.lastError }}
                </div>
                <div class="text-base-content/70 text-sm" v-else-if="!scanItem(container)">
                  {{ $t("label.scan-not-scanned") }}
                </div>
              </template>
              <ContainerStatCell v-else :container="container" type="cpu" :host="hosts[container.host]" :mode="statMode" />
            </td>
            <td v-if="isVisible('mem')">
              <template v-if="statMode === 'scan'">
                <div class="flex items-center justify-between gap-3">
                  <div class="text-base-content/70 text-sm">
                    <span v-if="scanItem(container)?.lastSuccessAt">
                      <RelativeTime :date="scanLastSuccessDate(container)!" />
                    </span>
                    <span v-else>{{ $t("label.scan-manual") }}</span>
                  </div>
                  <button
                    class="btn btn-ghost btn-xs md:btn-sm"
                    @click.stop="openScanReport(container)"
                    :title="$t('label.scan-report')"
                  >
                    <mdi:file-document-outline />
                  </button>
                </div>
              </template>
              <ContainerStatCell v-else :container="container" type="mem" :host="hosts[container.host]" :mode="statMode" />
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-else class="space-y-4">
      <div
        v-for="group in paginatedCardGroups"
        :key="group.templateId"
        class="grid gap-4 md:grid-cols-2 xl:grid-cols-3"
      >
        <article
          v-for="container in group.containers"
          :key="container.id"
          class="rounded-box border-base-content/10 bg-base-100 border p-5 shadow-sm transition-all duration-200 hover:-translate-y-0.5 hover:border-primary/20 hover:shadow-md"
        >
          <div class="mb-4 flex items-start justify-between gap-3">
            <div class="flex min-w-0 items-start gap-3">
              <div
                v-if="cardTemplate(container).iconUrl"
                class="bg-base-200/70 flex h-11 w-11 shrink-0 items-center justify-center overflow-hidden rounded-xl"
              >
                <img :src="cardTemplate(container).iconUrl" :alt="cardTemplate(container).name" class="h-full w-full object-cover" />
              </div>
              <div class="min-w-0">
                <router-link
                  :to="{ name: '/container/[id]', params: { id: container.id } }"
                  :title="container.name"
                  class="block truncate text-lg font-semibold"
                >
                  {{ container.name }}
                </router-link>
                <div class="text-base-content/60 mt-1 flex flex-wrap gap-x-3 gap-y-1 text-sm">
                  <span v-if="cardTemplate(container).showHost">{{ container.hostLabel }}</span>
                  <span v-if="cardTemplate(container).showState">{{ containerCardStateLabel(container.state) }}</span>
                </div>
              </div>
            </div>
            <div class="badge badge-outline shrink-0" v-if="cardTemplate(container).showCreated">
              <RelativeTime :date="container.created" />
            </div>
          </div>

          <div class="mb-4 grid gap-2" v-if="cardTemplate(container).extraFields.length">
            <div
              v-for="field in cardTemplate(container).extraFields"
              :key="field.id"
              class="bg-base-200/60 rounded-box grid min-w-0 grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)] items-start gap-3 px-3 py-2 text-sm"
            >
              <span class="text-base-content/60 min-w-0 truncate">{{ displayContainerCardFieldLabel(field) }}</span>
              <span
                class="min-w-0 truncate text-right font-medium"
                :title="resolveContainerCardField(container, field.source) || '—'"
              >
                {{ resolveContainerCardField(container, field.source) || "—" }}
              </span>
            </div>
          </div>

          <div v-if="statMode === 'scan'" class="space-y-3">
            <div class="bg-base-200/50 rounded-box flex justify-center px-3 py-3" v-if="scanItem(container)">
              <div class="flex flex-wrap items-center justify-center gap-2">
                <span
                  class="border-error/25 bg-error/10 text-error inline-flex min-w-12 items-center justify-center rounded-full border px-3 py-1 text-xs font-semibold"
                  v-if="scanItem(container)?.summary.critical"
                >
                  C {{ scanItem(container)?.summary.critical }}
                </span>
                <span
                  class="border-warning/25 bg-warning/10 text-warning inline-flex min-w-12 items-center justify-center rounded-full border px-3 py-1 text-xs font-semibold"
                  v-if="scanItem(container)?.summary.high"
                >
                  H {{ scanItem(container)?.summary.high }}
                </span>
                <span
                  class="border-info/25 bg-info/10 text-info inline-flex min-w-12 items-center justify-center rounded-full border px-3 py-1 text-xs font-semibold"
                  v-if="scanItem(container)?.summary.medium"
                >
                  M {{ scanItem(container)?.summary.medium }}
                </span>
                <span
                  class="border-base-content/10 bg-base-100 text-base-content/80 inline-flex min-w-12 items-center justify-center rounded-full border px-3 py-1 text-xs font-semibold"
                >
                  {{ scanItem(container)?.summary.total || 0 }}
                </span>
              </div>
            </div>
            <div class="text-base-content/70 text-sm" v-if="scanItem(container)?.running">
              {{ $t("label.scanning") }}
            </div>
            <div class="text-error text-sm" v-else-if="scanItem(container)?.lastError">
              {{ scanItem(container)?.lastError }}
            </div>
            <div class="text-base-content/70 text-sm" v-else-if="!scanItem(container)">
              {{ $t("label.scan-not-scanned") }}
            </div>

            <div class="bg-base-200/70 rounded-box flex items-center justify-between gap-3 px-3 py-2 text-sm">
              <div class="text-base-content/70">
                <span v-if="scanItem(container)?.lastSuccessAt">
                  <RelativeTime :date="scanLastSuccessDate(container)!" />
                </span>
                <span v-else>{{ $t("label.scan-manual") }}</span>
              </div>
              <button
                class="btn btn-ghost btn-xs md:btn-sm"
                @click.stop="openScanReport(container)"
                :title="$t('label.scan-report')"
              >
                <mdi:file-document-outline />
              </button>
            </div>
          </div>

          <div v-else class="grid gap-3 md:grid-cols-2">
            <div class="bg-base-200/70 rounded-box p-3">
              <div class="text-base-content/60 mb-2 text-xs uppercase">{{ $t("label.avg-cpu") }}</div>
              <ContainerStatCell :container="container" type="cpu" :host="hosts[container.host]" :mode="statMode" />
            </div>
            <div class="bg-base-200/70 rounded-box p-3">
              <div class="text-base-content/60 mb-2 text-xs uppercase">{{ $t("label.avg-mem") }}</div>
              <ContainerStatCell :container="container" type="mem" :host="hosts[container.host]" :mode="statMode" />
            </div>
          </div>

          <div
            v-if="cardTemplate(container).showDetails || (config.enableActions && cardTemplate(container).showActions)"
            class="border-base-content/10 mt-4 flex items-center justify-between border-t pt-4"
          >
            <router-link
              v-if="cardTemplate(container).showDetails"
              :to="{ name: '/container/[id].details', params: { id: container.id } }"
              class="btn btn-ghost btn-sm gap-2"
            >
              <mdi:file-document-outline />
              {{ $t("card-templates.open-details") }}
            </router-link>
            <div v-else></div>

            <ContainerCardActions
              v-if="config.enableActions && cardTemplate(container).showActions"
              :container="container"
              :template="cardTemplate(container)"
            />
          </div>
        </article>
      </div>
    </div>
    <div class="p-4 text-center">
      <nav class="join" v-if="isPaginated && totalPages <= 15">
        <input
          class="btn btn-square join-item"
          type="radio"
          v-model="currentPage"
          :aria-label="`${i}`"
          :value="i"
          v-for="i in totalPages"
        />
      </nav>
      <DropdownMenu
        v-else-if="isPaginated"
        class="btn-sm"
        v-model="currentPage"
        :options="Array.from({ length: totalPages }, (_, i) => ({ label: `${i + 1}`, value: i + 1 }))"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import ContainerCardActions from "@/components/CardTemplates/ContainerCardActions.vue";
import ContainerTrivyScan from "@/components/ContainerViewer/ContainerTrivyScan.vue";
import { Container } from "@/models/Container";
import {
  containerCardStateLabel,
  displayContainerCardFieldLabel,
  resolveContainerCardField,
  resolveContainerCardTemplate,
  useContainerCardTemplates,
} from "@/stores/containerCardTemplates";
import config, { withBase } from "@/stores/config";
import type { ScanDashboardSummary } from "@/types/scans";
import { toRefs } from "@vueuse/core";

const { t } = useI18n();
const showDrawer = useDrawer();
const { hosts } = useHosts();
useContainerCardTemplates();
const selectedHost = ref(null);

const fields: Record<
  string,
  {
    label: string;
    sortFunc: (a: Container, b: Container) => number;
    mobileVisible: boolean;
    customClass?: string;
  }
> = {
  name: {
    label: "label.container-name",
    sortFunc: (a: Container, b: Container) => a.name.localeCompare(b.name) * direction.value,
    mobileVisible: true,
  },
  host: {
    label: "label.host",
    sortFunc: (a: Container, b: Container) => a.hostLabel.localeCompare(b.hostLabel) * direction.value,
    mobileVisible: false,
    customClass: "w-1",
  },
  state: {
    label: "label.status",
    sortFunc: (a: Container, b: Container) => a.state.localeCompare(b.state) * direction.value,
    mobileVisible: false,
    customClass: "w-1",
  },
  created: {
    label: "label.created",
    sortFunc: (a: Container, b: Container) => (a.created.getTime() - b.created.getTime()) * direction.value,
    mobileVisible: true,
    customClass: "w-1",
  },
  cpu: {
    label: "label.avg-cpu",
    sortFunc: (a: Container, b: Container) => (a.movingAverage.cpu - b.movingAverage.cpu) * direction.value,
    mobileVisible: false,
    customClass: "min-w-48",
  },
  mem: {
    label: "label.avg-mem",
    sortFunc: (a: Container, b: Container) =>
      (a.movingAverage.memoryUsage - b.movingAverage.memoryUsage) * direction.value,
    mobileVisible: false,
    customClass: "min-w-48",
  },
};

const { containers } = defineProps<{
  containers: Container[];
}>();
type keys = keyof typeof fields;

type StatMode = "chart" | "progress" | "scan";
type LayoutMode = "table" | "cards";

const statMode = useStorage<StatMode>("DOZZLE_TABLE_STAT_MODE", "chart");
const layoutMode = useStorage<LayoutMode>("DOZZLE_CONTAINER_LAYOUT_MODE", "table");
const perPage = useStorage("DOZZLE_TABLE_PAGE_SIZE", 15);
const pageSizes = [15, 30, 50, 100];
const scanSummary = ref<ScanDashboardSummary>();
const scanItems = computed(
  () =>
    new Map(
      (scanSummary.value?.items ?? []).map((item) => [`${item.container.host}:${item.container.id}`, item] as const),
    ),
);
let scanTimer: number | undefined;

const storage = useStorage<{ column: keys; direction: 1 | -1 }>("DOZZLE_TABLE_CONTAINERS_SORT", {
  column: "created" as keys,
  direction: -1 as 1 | -1,
});
const { column: sortField, direction } = toRefs(storage.value);
const counter = useInterval(10000);
const filteredContainers = computed(() =>
  containers.filter((c) => selectedHost.value === null || c.host === selectedHost.value),
);
const sortedContainers = computedWithControl(
  () => [filteredContainers.value.length, sortField.value, direction.value, counter.value, statMode.value, scanSummary.value],
  () => filteredContainers.value.slice().sort((a, b) => sortContainers(a, b, sortField.value)),
);

const totalPages = computed(() => Math.ceil(sortedContainers.value.length / perPage.value));
const isPaginated = computed(() => totalPages.value > 1);
const currentPage = ref(1);
watch(perPage, () => (currentPage.value = 1));
const paginated = computed(() => {
  const start = (currentPage.value - 1) * perPage.value;
  const end = start + perPage.value;

  return sortedContainers.value.slice(start, end);
});
const paginatedCardGroups = computed(() => {
  const groups: Array<{ templateId: string; containers: Container[] }> = [];
  let currentGroup: { templateId: string; containers: Container[] } | null = null;

  for (const container of paginated.value) {
    const templateId = cardTemplate(container).id;
    if (!currentGroup || currentGroup.templateId !== templateId) {
      currentGroup = { templateId, containers: [] };
      groups.push(currentGroup);
    }
    currentGroup.containers.push(container);
  }

  return groups;
});

function sort(field: keys) {
  if (sortField.value === field) {
    direction.value *= -1;
  } else {
    sortField.value = field;
    direction.value = 1;
  }
}
function isVisible(field: keys) {
  return fields[field].mobileVisible || !isMobile.value;
}

function columnLabel(field: keys) {
  if (statMode.value === "scan") {
    if (field === "cpu") return t("label.vulnerabilities");
    if (field === "mem") return t("label.last-scan");
  }
  return t(fields[field].label);
}

function scanItem(container: Container) {
  return scanItems.value.get(`${container.host}:${container.id}`);
}

function scanSortValue(container: Container) {
  const item = scanItem(container);
  return {
    critical: item?.summary.critical ?? -1,
    high: item?.summary.high ?? -1,
    total: item?.summary.total ?? -1,
  };
}

function scanLastSuccessValue(container: Container) {
  const item = scanItem(container);
  if (!item?.lastSuccessAt) return 0;
  return new Date(item.lastSuccessAt).getTime();
}

function scanLastSuccessDate(container: Container) {
  const item = scanItem(container);
  if (!item?.lastSuccessAt) return undefined;
  return new Date(item.lastSuccessAt);
}

function sortContainers(a: Container, b: Container, field: keys) {
  if (statMode.value === "scan" && field === "cpu") {
    const left = scanSortValue(a);
    const right = scanSortValue(b);
    if (left.critical !== right.critical) return (left.critical - right.critical) * direction.value;
    if (left.high !== right.high) return (left.high - right.high) * direction.value;
    if (left.total !== right.total) return (left.total - right.total) * direction.value;
    return a.name.localeCompare(b.name) * direction.value;
  }
  if (statMode.value === "scan" && field === "mem") {
    return (scanLastSuccessValue(a) - scanLastSuccessValue(b)) * direction.value;
  }
  return fields[field].sortFunc(a, b);
}

function openScanReport(container: Container) {
  showDrawer(ContainerTrivyScan, { container }, "lg");
}

function cardTemplate(container: Container) {
  return resolveContainerCardTemplate(container);
}

async function fetchScanSummary() {
  if (!config.enableContainerScan) return;
  const response = await fetch(withBase("/api/scans/summary"));
  if (!response.ok) return;
  scanSummary.value = await response.json();
}

onMounted(async () => {
  if (!config.enableContainerScan) return;
  await fetchScanSummary();
  scanTimer = window.setInterval(fetchScanSummary, 60000);
});

onBeforeUnmount(() => {
  if (scanTimer) {
    window.clearInterval(scanTimer);
  }
});
</script>

<style scoped>
@reference "@/main.css";

[data-icon] {
  display: none;
  transition: transform 0.2s ease-in-out;
  [data-direction="desc"] & {
    transform: rotate(180deg);
  }
}

th {
  @apply border-base-200 border-b-2;
  &.selected-sort {
    font-weight: bold;
    @apply border-primary;
    [data-icon] {
      display: inline-block;
    }
  }
}

tbody td {
  white-space: nowrap;
}

a {
  @apply hover:text-primary;
}
</style>
