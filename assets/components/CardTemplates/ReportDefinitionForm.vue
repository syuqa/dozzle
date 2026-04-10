<template>
  <div class="space-y-4 p-4">
    <div class="mb-6">
      <h2 class="text-2xl font-bold">
        {{ isEditing ? $t("card-templates.edit-report") : $t("card-templates.create-report") }}
      </h2>
      <p class="text-base-content/60">{{ $t("card-templates.report-form-description") }}</p>
    </div>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("card-templates.name") }}</legend>
      <div class="grid gap-3 lg:grid-cols-[minmax(0,1fr)_220px]">
        <input v-model="draft.name" type="text" class="input input-bordered w-full text-base" />
        <label class="bg-base-200/60 rounded-box flex items-center justify-between gap-3 px-4 py-3">
          <span class="font-medium">{{ $t("card-templates.enabled") }}</span>
          <Toggle v-model="draft.enabled" />
        </label>
      </div>
    </fieldset>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("card-templates.report-id") }}</legend>
      <input v-model="draft.id" type="text" class="input input-bordered w-full" placeholder="platform-modules" :disabled="isEditing" />
    </fieldset>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("card-templates.report-description") }}</legend>
      <input v-model="draft.description" type="text" class="input input-bordered w-full" />
    </fieldset>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("card-templates.report-parser") }}</legend>
      <DropdownMenu v-model="draft.parser" :options="parserOptions" class="w-full" />
    </fieldset>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("card-templates.report-display-mode") }}</legend>
      <DropdownMenu v-model="draft.displayMode" :options="displayOptions" class="w-full" />
    </fieldset>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("card-templates.report-refresh-mode") }}</legend>
      <DropdownMenu v-model="draft.refreshMode" :options="refreshOptions" class="w-full" />
    </fieldset>

    <fieldset v-if="draft.parser !== 'properties_template' && draft.parser !== 'file_listing' && draft.parser !== 'file_content'" class="fieldset">
      <legend class="fieldset-legend text-lg">
        {{ draft.parser === "jira_issues" ? $t("card-templates.report-jql-template") : $t("card-templates.report-command") }}
      </legend>
      <textarea
        v-model="draft.command"
        class="textarea textarea-bordered min-h-40 w-full font-mono text-sm"
        :placeholder="draft.parser === 'jira_issues' ? $t('card-templates.report-jql-placeholder') : ''"
      />
      <p v-if="draft.parser === 'jira_issues'" class="text-base-content/60 mt-2 text-sm">
        {{ $t("card-templates.report-jql-description") }}
      </p>
    </fieldset>

    <template v-if="draft.parser === 'properties_template' || draft.parser === 'file_listing' || draft.parser === 'file_content'">
      <fieldset class="fieldset">
        <legend class="fieldset-legend text-lg">{{ $t("card-templates.report-file-path") }}</legend>
        <input
          v-model="draft.filePath"
          type="text"
          class="input input-bordered w-full font-mono text-sm"
          :placeholder="$t('card-templates.report-file-path-placeholder')"
        />
      </fieldset>

      <fieldset v-if="draft.parser === 'file_listing'" class="fieldset">
        <legend class="fieldset-legend text-lg">{{ $t("card-templates.report-file-pattern") }}</legend>
        <input
          v-model="draft.filePattern"
          type="text"
          class="input input-bordered w-full font-mono text-sm"
          :placeholder="$t('card-templates.report-file-pattern-placeholder')"
        />
      </fieldset>

      <fieldset class="fieldset">
        <legend class="fieldset-legend text-lg">{{ $t("card-templates.report-exclude-patterns") }}</legend>
        <input
          :value="draft.excludePatterns?.join(', ') || ''"
          @input="draft.excludePatterns = splitCsv(($event.target as HTMLInputElement).value)"
          type="text"
          class="input input-bordered w-full"
          :placeholder="$t('card-templates.report-exclude-patterns-placeholder')"
        />
        <p class="text-base-content/60 mt-2 text-sm">{{ $t("card-templates.report-exclude-patterns-description") }}</p>
      </fieldset>

      <template v-if="draft.parser === 'file_listing'">
        <div class="grid gap-3 md:grid-cols-2">
          <label class="bg-base-200/60 rounded-box flex items-center justify-between gap-3 px-4 py-3">
            <span class="font-medium">{{ $t("card-templates.report-show-file-size") }}</span>
            <Toggle v-model="draft.showFileSize" />
          </label>

          <label class="bg-base-200/60 rounded-box flex items-center justify-between gap-3 px-4 py-3">
            <span class="font-medium">{{ $t("card-templates.report-show-checksum") }}</span>
            <Toggle v-model="draft.showChecksum" />
          </label>
        </div>

        <fieldset class="fieldset">
          <legend class="fieldset-legend text-lg">{{ $t("card-templates.report-linked-report-id") }}</legend>
          <input
            v-model="draft.linkedReportId"
            type="text"
            class="input input-bordered w-full"
            :placeholder="$t('card-templates.report-linked-report-id-placeholder')"
          />
        </fieldset>

        <fieldset class="fieldset">
          <legend class="fieldset-legend text-lg">{{ $t("card-templates.report-linked-report-label") }}</legend>
          <input
            v-model="draft.linkedReportLabel"
            type="text"
            class="input input-bordered w-full"
            :placeholder="$t('card-templates.report-linked-report-label-placeholder')"
          />
        </fieldset>
      </template>
    </template>

    <div class="grid gap-3 md:grid-cols-2">
      <fieldset class="fieldset">
        <legend class="fieldset-legend text-lg">{{ $t("card-templates.report-timeout") }}</legend>
        <input v-model.number="draft.timeoutSeconds" type="number" min="1" class="input input-bordered w-full" />
      </fieldset>
      <fieldset v-if="draft.refreshMode !== 'container_update'" class="fieldset">
        <legend class="fieldset-legend text-lg">{{ $t("card-templates.report-cache-ttl") }}</legend>
        <input v-model.number="draft.cacheTtlSeconds" type="number" min="0" class="input input-bordered w-full" />
      </fieldset>
    </div>

    <div v-if="error" class="alert alert-error alert-soft">
      <span>{{ error }}</span>
    </div>

    <div class="flex justify-end gap-2 pt-4">
      <button class="btn" @click="close?.()">{{ $t("button.cancel") }}</button>
      <button class="btn btn-primary" :disabled="!canSave" @click="saveReport">
        <span v-if="isSaving" class="loading loading-spinner loading-sm"></span>
        {{ isEditing ? $t("notifications.template-form.save") : $t("notifications.template-form.create") }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  containerCardReports,
  containerCardReportDisplayOptions,
  containerCardReportRefreshOptions,
  containerCardReportParserOptions,
  saveContainerCardReports,
  type ContainerCardReportDefinition,
} from "@/stores/containerCardReports";

const { close, onCreated, report } = defineProps<{
  close?: () => void;
  onCreated?: () => void;
  report?: ContainerCardReportDefinition;
}>();

const { t } = useI18n();
const isEditing = computed(() => !!report);
const isSaving = ref(false);
const error = ref<string>();
const draft = reactive<ContainerCardReportDefinition>(
  report
    ? { ...report }
    : {
        id: `report-${Math.random().toString(36).slice(2, 10)}`,
        name: t("card-templates.new-report"),
        enabled: true,
        description: "",
        command: `find /opt/app -type f -name '*.jar' -printf '%f\n'`,
        filePath: "",
        filePattern: "*.jar",
        excludePatterns: [],
        linkedReportId: "",
        linkedReportLabel: "",
        showFileSize: false,
        showChecksum: false,
        parser: "jar_modules",
        displayMode: "inline",
        refreshMode: "ttl",
        timeoutSeconds: 10,
        cacheTtlSeconds: 60,
      },
);

const parserOptions = computed(() => containerCardReportParserOptions.map((item) => ({ label: t(item.labelKey), value: item.value })));
const displayOptions = computed(() => containerCardReportDisplayOptions.map((item) => ({ label: t(item.labelKey), value: item.value })));
const refreshOptions = computed(() => containerCardReportRefreshOptions.map((item) => ({ label: t(item.labelKey), value: item.value })));
const canSave = computed(() =>
  !isSaving.value &&
  draft.id.trim() &&
  draft.name.trim() &&
  (
    draft.parser === "properties_template" || draft.parser === "file_content"
      ? String(draft.filePath || "").trim()
      : draft.parser === "file_listing"
        ? String(draft.filePath || "").trim() && String(draft.filePattern || "").trim()
        : draft.command.trim()
  ),
);

function splitCsv(value: string) {
  return value.split(",").map((item) => item.trim()).filter(Boolean);
}

async function saveReport() {
  if (!canSave.value) return;
  isSaving.value = true;
  error.value = undefined;
  try {
    const next = containerCardReports.value.map((item) => ({ ...item }));
    const sanitized = {
      ...draft,
      id: draft.id.trim(),
      name: draft.name.trim(),
      command: draft.parser === "properties_template" || draft.parser === "file_listing" || draft.parser === "file_content" ? "" : draft.command.trim(),
      description: draft.description?.trim() || "",
      filePath: draft.parser === "properties_template" || draft.parser === "file_listing" || draft.parser === "file_content" ? String(draft.filePath || "").trim() : "",
      filePattern: draft.parser === "file_listing" ? String(draft.filePattern || "").trim() : "",
      excludePatterns: draft.parser === "properties_template" || draft.parser === "file_listing" ? splitCsv((draft.excludePatterns ?? []).join(",")) : [],
      linkedReportId: draft.parser === "file_listing" ? String(draft.linkedReportId || "").trim() : "",
      linkedReportLabel: draft.parser === "file_listing" ? String(draft.linkedReportLabel || "").trim() : "",
      showFileSize: draft.parser === "file_listing" ? draft.showFileSize === true : false,
      showChecksum: draft.parser === "file_listing" ? draft.showChecksum === true : false,
      displayMode: draft.displayMode || "inline",
      refreshMode: draft.refreshMode || "ttl",
      cacheTtlSeconds: draft.refreshMode === "ttl" ? draft.cacheTtlSeconds : 0,
    };
    const existingIndex = next.findIndex((item) => item.id === sanitized.id);
    if (existingIndex >= 0) next.splice(existingIndex, 1, sanitized);
    else next.push(sanitized);
    containerCardReports.value = next;
    await saveContainerCardReports();
    onCreated?.();
    close?.();
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    isSaving.value = false;
  }
}
</script>
