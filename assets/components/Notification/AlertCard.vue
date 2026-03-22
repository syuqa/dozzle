<template>
  <div class="card bg-base-100 shadow-sm" :class="{ 'opacity-60': !alert.enabled }">
    <div class="card-body gap-4 p-5">
      <!-- Header -->
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-2">
          <h4 class="flex items-center gap-2 text-lg font-semibold">
            <mdi:shield-search v-if="alert.type === 'scan'" class="text-info" />
            <mdi:chart-line v-else-if="alert.type === 'metric'" class="text-info" />
            <mdi:text-box-outline v-else class="text-info" />
            <span>{{ alert.name }}</span> <span class="text-sm font-light">→</span>
            <div class="group/dispatch dropdown dropdown-hover">
              <div
                tabindex="0"
                role="button"
                class="border-base-content/0 hover:border-base-content/20 flex cursor-pointer items-center gap-1 rounded border px-1.5 py-0.5 text-xs font-light transition-colors"
                :class="{ 'text-warning': !alert.dispatcher }"
              >
                <template v-if="alert.dispatcher">
                  <mdi:webhook v-if="alert.dispatcher.type === 'webhook'" />
                  <mdi:cloud v-else />
                  {{ alert.dispatcher.name }}
                </template>
                <template v-else>
                  <mdi:alert-outline />
                  {{ $t("notifications.alert.dispatcher-deleted") }}
                </template>
                <mdi:chevron-down class="text-[0.6rem] opacity-0 transition-opacity group-hover/dispatch:opacity-100" />
              </div>
              <ul tabindex="0" class="dropdown-content menu bg-base-200 rounded-box z-50 w-48 p-2 shadow-lg">
                <li v-for="dest in dispatchers" :key="dest.id">
                  <a
                    class="flex items-center gap-2"
                    :class="{ active: dest.id === alert.dispatcher?.id }"
                    @click="changeDispatcher(dest.id)"
                  >
                    <mdi:webhook v-if="dest.type === 'webhook'" />
                    <mdi:cloud v-else />
                    {{ dest.name }}
                  </a>
                </li>
              </ul>
            </div>
          </h4>
          <span v-if="!alert.enabled" class="badge badge-warning badge-sm">{{ $t("notifications.alert.paused") }}</span>
        </div>
        <input type="checkbox" class="toggle toggle-primary" :checked="alert.enabled" @change="toggleEnabled" />
      </div>

      <!-- Expressions -->
      <div class="text-base-content/80 grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
        <span>{{ $t("notifications.alert.containers") }}</span>
        <code class="bg-base-200 text-base-content rounded px-2 py-0.5 font-mono">{{ alert.containerExpression }}</code>
        <template v-if="alert.type === 'scan'">
          <span>{{ $t("notifications.alert.scan-severity") }}</span>
          <code class="bg-base-200 text-base-content rounded px-2 py-0.5 font-mono">{{ alert.minSeverity }}</code>
          <span>{{ $t("notifications.alert.scan-package-types") }}</span>
          <code class="bg-base-200 text-base-content rounded px-2 py-0.5 font-mono">
            {{ alert.packageTypes?.length ? alert.packageTypes.join(", ") : $t("notifications.alert.scan-all-package-types") }}
          </code>
          <span>{{ $t("notifications.alert.cooldown") }}</span>
          <span>{{ $t("notifications.alert.scan-cooldown-minutes", { count: alert.cooldownMinutes || 60 }) }}</span>
        </template>
        <template v-else-if="alert.type === 'metric'">
          <span>{{ $t("notifications.alert.metric-filter") }}</span>
          <code class="bg-base-200 text-base-content rounded px-2 py-0.5 font-mono">{{ alert.metricExpression }}</code>
          <span>{{ $t("notifications.alert.sample-window") }}</span>
          <span>{{ formatDuration(alert.sampleWindow || 15, locale || undefined) }}</span>
          <span>{{ $t("notifications.alert.cooldown") }}</span>
          <span>{{ formatDuration(alert.cooldown || 300, locale || undefined) }}</span>
        </template>
        <template v-else>
          <span>{{ $t("notifications.alert.log-filter") }}</span>
          <code class="bg-base-200 text-base-content rounded px-2 py-0.5 font-mono">{{ alert.logExpression }}</code>
        </template>
      </div>

      <!-- Footer -->
      <div class="border-base-content/10 text-base-content/80 flex items-center justify-between border-t pt-3 text-xs">
        <div class="flex items-center gap-4">
          <span>
            {{ $t("notifications.alert.containers-count", { count: alert.triggeredContainers }) }}
          </span>
          <span>
            {{ $t("notifications.alert.triggered-count", { count: alert.triggerCount }) }}
          </span>
          <span v-if="alert.lastTriggeredAt">
            {{ $t("notifications.alert.last-triggered", { time: formatTimeAgo(alert.lastTriggeredAt) }) }}
          </span>
        </div>
        <div class="flex items-center gap-1">
          <button class="btn btn-ghost btn-square" @click="editAlert">
            <mdi:pencil-outline />
          </button>
          <button class="btn btn-ghost btn-square" @click="deleteAlert" :disabled="isDeleting">
            <span v-if="isDeleting" class="loading loading-spinner loading-xs"></span>
            <mdi:trash-can-outline v-else />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { Dispatcher, UnifiedAlert } from "@/types/notifications";
import AlertForm from "./AlertForm.vue";

const { alert, onUpdated } = defineProps<{
  alert: UnifiedAlert;
  onUpdated?: () => void;
}>();

const showDrawer = useDrawer();
const isDeleting = ref(false);
const dispatchers = ref<Dispatcher[]>([]);

onMounted(async () => {
  const res = await fetch(withBase("/api/notifications/dispatchers"));
  dispatchers.value = await res.json();
});

async function changeDispatcher(id: number) {
  const url =
    alert.type === "scan" ? withBase(`/api/scans/alerts/${alert.id}`) : withBase(`/api/notifications/rules/${alert.id}`);
  const method = alert.type === "scan" ? "PUT" : "PATCH";
  await fetch(url, {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(alert.type === "scan" ? { ...alert, dispatcherId: id } : { dispatcherId: id }),
  });
  onUpdated?.();
}

function formatTimeAgo(dateStr: string): string {
  const date = new Date(dateStr);
  if (date.getFullYear() === 0) return "-";
  return toRelativeTime(date, undefined);
}

async function toggleEnabled() {
  const url =
    alert.type === "scan" ? withBase(`/api/scans/alerts/${alert.id}`) : withBase(`/api/notifications/rules/${alert.id}`);
  const method = alert.type === "scan" ? "PUT" : "PATCH";
  await fetch(url, {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(alert.type === "scan" ? { ...alert, enabled: !alert.enabled } : { enabled: !alert.enabled }),
  });
  onUpdated?.();
}

function editAlert() {
  showDrawer(AlertForm, { alert, onCreated: onUpdated }, "lg");
}

async function deleteAlert() {
  isDeleting.value = true;
  try {
    const url =
      alert.type === "scan"
        ? withBase(`/api/scans/alerts/${alert.id}`)
        : withBase(`/api/notifications/rules/${alert.id}`);
    await fetch(url, { method: "DELETE" });
    onUpdated?.();
  } finally {
    isDeleting.value = false;
  }
}
</script>
