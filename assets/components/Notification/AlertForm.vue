<template>
  <div class="space-y-4 p-4">
    <div class="mb-6">
      <h2 class="text-2xl font-bold">
        {{ isEditing ? $t("notifications.alert-form.edit-title") : $t("notifications.alert-form.create-title") }}
      </h2>
      <p class="text-base-content/60">{{ $t("notifications.alert-form.description") }}</p>
    </div>

    <!-- Alert Name -->
    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.alert-form.alert-name") }}</legend>
      <input
        ref="alertNameInput"
        v-model="alertName"
        type="text"
        class="input focus:input-primary w-full text-base"
        :class="alertName.trim() ? 'input-primary' : ''"
        required
        :placeholder="$t('notifications.alert-form.alert-name-placeholder')"
      />
    </fieldset>

    <!-- Alert Type Toggle -->
    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.alert-form.alert-type") }}</legend>
      <div class="flex gap-2">
        <button
          class="btn btn-sm"
          :class="alertType === 'log' ? 'btn-primary' : 'btn-outline'"
          @click="alertType = 'log'"
        >
          <mdi:text-box-outline class="mr-1" />
          {{ $t("notifications.alert-form.log-alert") }}
        </button>
        <button
          class="btn btn-sm"
          :class="alertType === 'metric' ? 'btn-primary' : 'btn-outline'"
          @click="alertType = 'metric'"
        >
          <mdi:chart-line class="mr-1" />
          {{ $t("notifications.alert-form.metric-alert") }}
        </button>
        <button
          class="btn btn-sm"
          :class="alertType === 'scan' ? 'btn-primary' : 'btn-outline'"
          @click="alertType = 'scan'"
          v-if="config.enableContainerScan"
        >
          <mdi:shield-search class="mr-1" />
          {{ $t("notifications.alert-form.scan-alert") }}
        </button>
        <button class="btn btn-sm" :class="alertType === 'state' ? 'btn-primary' : 'btn-outline'" @click="alertType = 'state'">
          <mdi:cube-outline class="mr-1" />
          {{ $t("notifications.alert-form.state-alert") }}
        </button>
      </div>
    </fieldset>

    <!-- Container Filter -->
    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.alert-form.container-filter") }}</legend>
      <div
        class="input focus-within:input-primary w-full focus-within:z-50"
        :class="
          containerExpression.trim() && !containerResult?.error
            ? 'input-primary'
            : { 'input-error!': containerResult?.error }
        "
      >
        <div ref="containerEditorRef" class="w-full"></div>
      </div>
      <div v-if="containerResult" class="fieldset-label">
        <span v-if="containerResult.error" class="text-error">{{ containerResult.error }}</span>
        <span v-else-if="containerResult.containers?.length" class="text-success">
          <mdi:check class="inline" />
          {{
            $t("notifications.alert-form.containers-match", {
              count: containerResult.containers.length,
              names: containerResult.containers.map((c) => c.name).join(", "),
            })
          }}
        </span>
        <span v-else class="text-warning">
          <mdi:alert class="inline" />
          {{ $t("notifications.alert-form.no-containers-match") }}
        </span>
      </div>
    </fieldset>

    <!-- Type-specific fields -->
    <LogAlertFields
      v-if="alertType === 'log'"
      ref="fieldsRef"
      :alert="standardAlert"
      :prefill="prefill"
      :container-expression="containerExpression"
      :is-loading="isLoading"
      :validate-preview="validatePreview"
    />
    <MetricAlertFields
      v-else-if="alertType === 'metric'"
      ref="fieldsRef"
      :alert="standardAlert"
      :prefill="prefill"
      :container-expression="containerExpression"
      :is-loading="isLoading"
      :validate-preview="validatePreview"
    />
    <div v-else-if="alertType === 'state'" class="space-y-4">
      <fieldset class="fieldset">
        <legend class="fieldset-legend text-lg">{{ $t("notifications.alert-form.state-triggers") }}</legend>
        <div class="grid gap-3 md:grid-cols-2">
          <label v-for="trigger in availableStateTriggers" :key="trigger.value" class="label cursor-pointer justify-start gap-3">
            <input
              :checked="stateTriggers.includes(trigger.value)"
              type="checkbox"
              class="checkbox checkbox-primary"
              @change="toggleStateTrigger(trigger.value)"
            />
            <span class="label-text">{{ $t(trigger.label) }}</span>
          </label>
        </div>
      </fieldset>

      <fieldset class="fieldset">
        <legend class="fieldset-legend text-lg">{{ $t("notifications.alert-form.state-holdoff-seconds") }}</legend>
        <input v-model.number="stateHoldoffSeconds" type="number" min="0" step="1" class="input w-full" />
        <p class="text-base-content/50 mt-1 text-xs">
          {{ $t("notifications.alert-form.state-holdoff-hint") }}
        </p>
      </fieldset>
    </div>
    <div v-else class="space-y-4">
      <fieldset class="fieldset">
        <legend class="fieldset-legend text-lg">{{ $t("notifications.alert-form.scan-severity") }}</legend>
        <select class="select w-full" v-model="scanSeverity">
          <option value="LOW">LOW</option>
          <option value="MEDIUM">MEDIUM</option>
          <option value="HIGH">HIGH</option>
          <option value="CRITICAL">CRITICAL</option>
        </select>
      </fieldset>

      <fieldset class="fieldset">
        <legend class="fieldset-legend text-lg">{{ $t("notifications.alert-form.scan-package-types") }}</legend>
        <input
          v-model="scanPackageTypes"
          type="text"
          class="input w-full"
          :placeholder="$t('notifications.alert-form.scan-package-types-placeholder')"
        />
      </fieldset>

      <fieldset class="fieldset">
        <label class="label cursor-pointer justify-start gap-3">
          <input v-model="scanScheduleEnabled" type="checkbox" class="toggle toggle-primary" />
          <span class="label-text">{{ $t("notifications.alert-form.scan-schedule-enabled") }}</span>
        </label>
      </fieldset>

      <fieldset class="fieldset" v-if="scanScheduleEnabled">
        <legend class="fieldset-legend text-lg">{{ $t("notifications.alert-form.scan-interval-minutes") }}</legend>
        <input v-model.number="scanIntervalMinutes" type="number" min="5" step="5" class="input w-full" />
        <p class="text-base-content/50 mt-1 text-xs">
          {{ $t("notifications.alert-form.scan-interval-hint", { count: scanIntervalMinutes }) }}
        </p>
      </fieldset>

      <fieldset class="fieldset">
        <legend class="fieldset-legend text-lg">{{ $t("notifications.alert-form.cooldown-label") }}</legend>
        <input v-model.number="scanCooldownMinutes" type="range" min="5" max="1440" step="5" class="range range-primary" />
        <p class="text-base-content/50 mt-1 text-xs">
          {{ $t("notifications.alert-form.scan-cooldown-hint", { count: scanCooldownMinutes }) }}
        </p>
      </fieldset>

      <fieldset class="fieldset">
        <label class="label cursor-pointer justify-start gap-3">
          <input v-model="scanNotifyOnManual" type="checkbox" class="toggle toggle-primary" />
          <span class="label-text">{{ $t("notifications.alert-form.scan-notify-on-manual") }}</span>
        </label>
        <p class="text-base-content/50 mt-1 text-xs">
          {{ $t("notifications.alert-form.scan-notify-on-manual-hint") }}
        </p>
      </fieldset>
    </div>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">
        {{ $t("notifications.alert-form.shared-template") }}
      </legend>
      <select v-model.number="selectedTemplateId" class="select w-full">
        <option :value="0">{{ $t("notifications.alert-form.no-shared-template") }}</option>
        <option v-for="item in templates" :key="item.id" :value="item.id">{{ item.name }}</option>
      </select>
    </fieldset>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">
        {{ $t("notifications.alert-form.template-override") }}
      </legend>
      <textarea
        v-model="alertTemplate"
        class="textarea textarea-bordered min-h-32 w-full font-mono text-sm"
        :placeholder="$t('notifications.alert-form.template-override-placeholder')"
      ></textarea>
      <p class="text-base-content/50 mt-1 text-xs">
        {{ $t("notifications.alert-form.template-override-hint") }}
      </p>
    </fieldset>

    <!-- Destination -->
    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.alert-form.destination") }}</legend>
      <details class="dropdown w-full" ref="destinationDropdown">
        <summary class="btn btn-outline w-full justify-between" :class="{ 'btn-primary': selectedDestination }">
          <span class="flex items-center gap-2">
            <template v-if="selectedDestination">
              <mdi:webhook v-if="selectedDestination.type === 'webhook'" />
              <mdi:telegram v-else-if="selectedDestination.type === 'telegram'" />
              <mdi:cloud v-else />
              {{ selectedDestination.name }}
            </template>
            <span v-else class="text-base-content/60">{{ $t("notifications.alert-form.select-destination") }}</span>
          </span>
          <carbon:caret-down />
        </summary>
        <ul class="dropdown-content menu bg-base-200 rounded-box z-50 mt-1 w-full border p-2 shadow-sm">
          <li v-for="dest in destinations" :key="dest.id">
            <a
              @click="
                dispatcherId = dest.id;
                destinationDropdown?.removeAttribute('open');
              "
              :class="{ active: dispatcherId === dest.id }"
            >
              <mdi:webhook v-if="dest.type === 'webhook'" />
              <mdi:telegram v-else-if="dest.type === 'telegram'" />
              <mdi:cloud v-else />
              {{ dest.name }}
            </a>
          </li>
        </ul>
      </details>
      <div v-if="!destinations.length" class="fieldset-label">
        <span class="text-warning">
          <mdi:alert class="inline" />
          {{ $t("notifications.alert-form.no-destinations") }}
        </span>
      </div>
    </fieldset>

    <!-- Error -->
    <div v-if="saveError" class="alert alert-error">
      <span>{{ saveError }}</span>
    </div>

    <!-- Actions -->
    <div class="flex justify-end gap-2 pt-4">
      <button class="btn" @click="close?.()">{{ $t("notifications.alert-form.cancel") }}</button>
      <button class="btn btn-primary" :disabled="!canSave" @click="save">
        <span v-if="isSaving" class="loading loading-spinner loading-sm"></span>
        {{ isEditing ? $t("notifications.alert-form.save") : $t("notifications.alert-form.create") }}
      </button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import config from "@/stores/config";
import { useAlertForm } from "@/composable/alertForm";
import LogAlertFields from "./LogAlertFields.vue";
import MetricAlertFields from "./MetricAlertFields.vue";
import type { NotificationRule, NotificationTemplate, UnifiedAlert } from "@/types/notifications";

const props = defineProps<{
  close?: () => void;
  onCreated?: () => void;
  alert?: UnifiedAlert;
  templates?: NotificationTemplate[];
  prefill?: { name?: string; containerExpression?: string; logExpression?: string; metricExpression?: string };
}>();

const {
  isEditing,
  alertName,
  containerExpression,
  dispatcherId,
  destinations,
  selectedDestination,
  containerResult,
  isLoading,
  isSaving,
  saveError,
  baseCanSave,
  initContainerEditor,
  saveAlert,
  validatePreview,
} = useAlertForm(props);

// Template refs
const alertNameInput = ref<HTMLInputElement>();
const containerEditorRef = ref<HTMLElement>();
const destinationDropdown = ref<HTMLDetailsElement>();
const fieldsRef = ref<InstanceType<typeof LogAlertFields> | InstanceType<typeof MetricAlertFields>>();
useFocus(alertNameInput, { initialValue: true });

// Alert type
const standardAlert = computed<NotificationRule | undefined>(() =>
  props.alert && props.alert.type !== "scan" ? props.alert : undefined,
);
const alertType = ref<"log" | "metric" | "scan" | "state">(
  props.alert?.type === "scan"
    ? "scan"
    : standardAlert.value?.stateTriggers?.length
      ? "state"
      : standardAlert.value?.metricExpression
        ? "metric"
        : "log",
);
const availableStateTriggers = [
  { value: "image_updated", label: "notifications.alert-form.trigger-image-updated" },
  { value: "stopped", label: "notifications.alert-form.trigger-stopped" },
  { value: "started", label: "notifications.alert-form.trigger-started" },
  { value: "error", label: "notifications.alert-form.trigger-error" },
  { value: "unhealthy", label: "notifications.alert-form.trigger-unhealthy" },
  { value: "restarted", label: "notifications.alert-form.trigger-restarted" },
  { value: "oom_killed", label: "notifications.alert-form.trigger-oom-killed" },
] as const;
const stateTriggers = ref(props.alert?.type === "scan" ? [] : [...(standardAlert.value?.stateTriggers ?? [])]);
const stateHoldoffSeconds = ref(props.alert?.type === "scan" ? 0 : standardAlert.value?.holdoffSeconds ?? 0);
const alertTemplate = ref(props.alert?.template ?? "");
const selectedTemplateId = ref(props.alert?.templateId ?? 0);
const scanSeverity = ref(props.alert?.type === "scan" ? props.alert.minSeverity : "HIGH");
const scanPackageTypes = ref(props.alert?.type === "scan" ? (props.alert.packageTypes ?? []).join(", ") : "");
const scanScheduleEnabled = ref(props.alert?.type === "scan" ? props.alert.scheduleEnabled ?? false : false);
const scanIntervalMinutes = ref(props.alert?.type === "scan" ? props.alert.intervalMinutes || 60 : 60);
const scanCooldownMinutes = ref(props.alert?.type === "scan" ? props.alert.cooldownMinutes || 60 : 60);
const scanNotifyOnManual = ref(props.alert?.type === "scan" ? props.alert.notifyOnManual ?? true : true);

const canSave = computed(() => {
  if (!baseCanSave.value) return false;
  if (alertType.value === "scan") return true;
  if (alertType.value === "state") return stateTriggers.value.length > 0;
  return fieldsRef.value?.canSave ?? false;
});

async function save() {
  if (!canSave.value) return;
  if (alertType.value === "scan") {
    await saveAlert("scan", {
      minSeverity: scanSeverity.value,
      packageTypes: scanPackageTypes.value
        .split(",")
        .map((value) => value.trim())
        .filter(Boolean),
      scheduleEnabled: scanScheduleEnabled.value,
      intervalMinutes: scanIntervalMinutes.value,
      cooldownMinutes: scanCooldownMinutes.value,
      notifyOnManual: scanNotifyOnManual.value,
      templateId: selectedTemplateId.value || undefined,
      template: alertTemplate.value.trim(),
    });
    return;
  }
  if (alertType.value === "state") {
    await saveAlert("state", {
      stateTriggers: [...stateTriggers.value],
      holdoffSeconds: Math.max(0, stateHoldoffSeconds.value || 0),
      templateId: selectedTemplateId.value || undefined,
      template: alertTemplate.value.trim(),
    });
    return;
  }
  if (!fieldsRef.value) return;
  await saveAlert(alertType.value, {
    ...fieldsRef.value.typeFields,
    templateId: selectedTemplateId.value || undefined,
    template: alertTemplate.value.trim(),
  });
}

function toggleStateTrigger(trigger: string) {
  if (stateTriggers.value.includes(trigger)) {
    stateTriggers.value = stateTriggers.value.filter((item) => item !== trigger);
    return;
  }
  stateTriggers.value = [...stateTriggers.value, trigger];
}

// Container editor
let containerEditorView: Awaited<ReturnType<typeof initContainerEditor>> | undefined;

onMounted(async () => {
  if (containerEditorRef.value) {
    containerEditorView = await initContainerEditor(containerEditorRef.value);
  }
});

onScopeDispose(() => {
  containerEditorView?.destroy();
});
</script>
