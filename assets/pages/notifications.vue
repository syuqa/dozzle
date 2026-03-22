<template>
  <PageWithLinks>
    <section>
      <!-- Header -->
      <div class="mb-8">
        <h2 class="text-2xl font-bold">{{ $t("notifications.title") }}</h2>
        <p class="text-base-content/60">{{ $t("notifications.description") }}</p>
      </div>

      <!-- Destinations Section -->
      <div class="mb-8">
        <h3 class="text-base-content/60 mb-4 font-semibold tracking-wide uppercase">
          {{ $t("notifications.destinations") }}
        </h3>
        <div class="flex flex-wrap gap-4">
          <DestinationCard
            v-for="dest in dispatchers"
            :key="dest.id"
            :destination="dest"
            :on-updated="fetchAll"
            :existing-dispatchers="dispatchers"
            class="w-full md:w-72"
          />
          <!-- Add Destination Card -->
          <button
            class="card card-border border-base-content/30 hover:border-base-content/50 w-full cursor-pointer border-dashed transition-colors md:w-72"
            @click="openAddDestination"
          >
            <div class="card-body items-center justify-center gap-1 p-4">
              <mdi:plus class="text-2xl" />
              <span class="text-base-content/60 text-sm">{{ $t("notifications.add-destination") }}</span>
            </div>
          </button>
        </div>
      </div>

      <!-- Alerts Section -->
      <div>
        <div class="mb-4 flex items-center justify-between">
          <h3 class="text-base-content/60 font-semibold tracking-wide uppercase">{{ $t("notifications.alerts") }}</h3>
          <button class="btn btn-primary btn-sm" @click="openCreateAlert">
            <mdi:plus />
            {{ $t("notifications.add") }}
          </button>
        </div>

        <!-- Filter Tabs -->
        <div class="tabs tabs-box mb-6">
          <button class="tab" :class="{ 'tab-active': filter === 'all' }" @click="filter = 'all'">
            {{ $t("notifications.filter.all", { count: unifiedAlerts.length }) }}
          </button>
          <button class="tab" :class="{ 'tab-active': filter === 'enabled' }" @click="filter = 'enabled'">
            {{ $t("notifications.filter.enabled", { count: enabledCount }) }}
          </button>
          <button class="tab" :class="{ 'tab-active': filter === 'paused' }" @click="filter = 'paused'">
            {{ $t("notifications.filter.paused", { count: pausedCount }) }}
          </button>
        </div>

        <!-- Alerts List -->
        <div v-if="!filteredAlerts.length" class="text-base-content/60 py-4">
          {{ $t("notifications.no-alerts") }}
        </div>
        <div v-else class="space-y-4">
          <AlertCard v-for="alert in filteredAlerts" :key="`${alert.type ?? 'log'}:${alert.id}`" :alert="alert" :on-updated="fetchAll" />
        </div>
      </div>
    </section>
  </PageWithLinks>
</template>

<script lang="ts" setup>
import type { NotificationRule, Dispatcher, ScanAlert, UnifiedAlert } from "@/types/notifications";
import AlertForm from "@/components/Notification/AlertForm.vue";
import DestinationForm from "@/components/Notification/DestinationForm.vue";

const showDrawer = useDrawer();
const router = useRouter();

// State
const alerts = ref<NotificationRule[]>([]);
const scanAlerts = ref<ScanAlert[]>([]);
const dispatchers = ref<Dispatcher[]>([]);

async function fetchAlerts() {
  const res = await fetch(withBase("/api/notifications/rules"));
  alerts.value = await res.json();
}

async function fetchDispatchers() {
  const res = await fetch(withBase("/api/notifications/dispatchers"));
  dispatchers.value = await res.json();
}

async function fetchScanAlerts() {
  const res = await fetch(withBase("/api/scans/alerts"));
  if (res.ok) {
    scanAlerts.value = await res.json();
  }
}

async function fetchAll() {
  await Promise.all([fetchAlerts(), fetchDispatchers(), fetchScanAlerts()]);
}

// Handle cloudLinkSuccess hash param
onMounted(async () => {
  await fetchAll();
  const hash = window.location.hash;
  if (hash.startsWith("#cloudLinkSuccess=")) {
    const id = Number(hash.replace("#cloudLinkSuccess=", ""));
    if (!isNaN(id)) {
      const destination = dispatchers.value.find((d) => d.id === id);
      if (destination) {
        showDrawer(
          DestinationForm,
          {
            destination,
            existingDispatchers: dispatchers.value,
            showLinkSuccess: true,
          },
          "md",
        );
      }
    }
    router.replace({ hash: "" });
  }
});

// Local state
const filter = ref<"all" | "enabled" | "paused">("all");

const unifiedAlerts = computed<UnifiedAlert[]>(() => [
  ...alerts.value.map((alert) => ({ ...alert, type: (alert.metricExpression ? "metric" : "log") as const })),
  ...scanAlerts.value.map((alert) => {
    const dispatcher = dispatchers.value.find((item) => item.id === alert.dispatcherId) ?? null;
    return { ...alert, dispatcher, triggeredContainers: 0, type: "scan" as const } as UnifiedAlert;
  }),
]);

const enabledCount = computed(() => unifiedAlerts.value.filter((a) => a.enabled).length);
const pausedCount = computed(() => unifiedAlerts.value.filter((a) => !a.enabled).length);

const filteredAlerts = computed(() => {
  if (filter.value === "enabled") return unifiedAlerts.value.filter((a) => a.enabled);
  if (filter.value === "paused") return unifiedAlerts.value.filter((a) => !a.enabled);
  return unifiedAlerts.value;
});

function openCreateAlert() {
  showDrawer(AlertForm, { onCreated: fetchAll }, "lg");
}

function openAddDestination() {
  showDrawer(
    DestinationForm,
    {
      onCreated: fetchAll,
      existingDispatchers: dispatchers.value,
    },
    "md",
  );
}

</script>
