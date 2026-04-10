<template>
  <div class="card bg-base-100 shadow-sm" :class="{ 'opacity-70': !report.enabled }">
    <div class="card-body gap-4 p-5">
      <div class="flex items-start justify-between gap-3">
        <div class="min-w-0">
          <h4 class="truncate text-lg font-semibold">{{ report.name }}</h4>
          <p class="text-base-content/60 mt-1 text-sm">{{ report.description || report.id }}</p>
        </div>
        <div class="flex items-center gap-1">
          <button class="btn btn-ghost btn-square btn-sm" @click="editReport">
            <mdi:pencil-outline />
          </button>
          <button class="btn btn-ghost btn-square btn-sm" @click="deleteReport" :disabled="isDeleting">
            <span v-if="isDeleting" class="loading loading-spinner loading-xs"></span>
            <mdi:trash-can-outline v-else />
          </button>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <span class="badge" :class="report.enabled ? 'badge-success badge-outline' : 'badge-ghost'">
          {{ report.enabled ? $t("card-templates.enabled") : $t("card-templates.disabled") }}
        </span>
        <span class="badge badge-outline">{{ $t(`card-templates.report-parser-${report.parser}`) }}</span>
        <span class="badge badge-outline">{{ $t(`card-templates.report-display-${report.displayMode || 'inline'}`) }}</span>
        <span class="badge badge-outline">{{ $t(`card-templates.report-refresh-${report.refreshMode || 'ttl'}`) }}</span>
        <span class="badge badge-outline">{{ report.timeoutSeconds }}s</span>
        <span v-if="(report.refreshMode || 'ttl') === 'ttl'" class="badge badge-outline">{{ report.cacheTtlSeconds }}s cache</span>
      </div>

      <pre class="bg-base-200/70 rounded-box overflow-hidden text-ellipsis whitespace-pre-wrap break-all p-3 text-xs">{{ reportPreview }}</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import ReportDefinitionForm from "./ReportDefinitionForm.vue";
import { containerCardReports, saveContainerCardReports, type ContainerCardReportDefinition } from "@/stores/containerCardReports";

const { report, onUpdated } = defineProps<{
  report: ContainerCardReportDefinition;
  onUpdated?: () => void;
}>();

const showDrawer = useDrawer();
const isDeleting = ref(false);
const reportPreview = computed(() => {
  if (report.parser === "properties_template") {
    return report.filePath || "—";
  }
  if (report.parser === "file_listing") {
    const base = report.filePath || "—";
    const options = [
      report.filePattern || "",
      report.showFileSize ? "size" : "",
      report.showChecksum ? "sha256" : "",
    ].filter(Boolean);
    return options.length ? `${base}\n${options.join(" · ")}` : base;
  }
  if (report.parser === "file_content") {
    return report.filePath || "—";
  }
  return report.command || "—";
});

function editReport() {
  showDrawer(ReportDefinitionForm, { report, onCreated: onUpdated }, "lg");
}

async function deleteReport() {
  isDeleting.value = true;
  try {
    containerCardReports.value = containerCardReports.value.filter((item) => item.id !== report.id);
    await saveContainerCardReports();
    onUpdated?.();
  } finally {
    isDeleting.value = false;
  }
}
</script>
