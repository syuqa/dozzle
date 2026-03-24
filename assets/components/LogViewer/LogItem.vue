<template>
  <div class="relative flex w-full items-start gap-x-2 group-[.compact]:items-stretch">
    <LogActions :logEntry :container />

    <LogStd :std="logEntry.std" class="shrink-0 select-none" v-if="showStd" />

    <div class="flex gap-x-2 gap-y-1 group-[.compact]:gap-y-0 has-[>_*:nth-of-type(2)]:flex-col-reverse md:flex-row!">
      <RandomColorTag class="w-30 shrink-0 select-none md:w-40" :value="host.name" v-if="showHostname" />
      <RandomColorTag
        v-if="showContainerName"
        class="w-30 shrink-0 select-none group-[.compact]:flex-1 md:w-40"
        :value="container.id"
        truncateRight
      >
        {{ container.name }}
      </RandomColorTag>
      <LogDate
        v-if="showTimestamp"
        :date="logEntry.date"
        class="shrink-0 select-none"
        :class="{ 'bg-secondary': route.query.logId === logEntry.id.toString() }"
      />
      <a
        v-if="incidentState?.status === 'matched' && incidentState.match?.url"
        :href="incidentState.match.url"
        target="_blank"
        rel="noreferrer noopener"
        class="badge badge-outline badge-sm self-start"
        :class="incidentBadgeClass(incidentState.match?.status)"
      >
        {{ $t("label.incident") }} #{{ incidentState.match?.iid }}
      </a>
      <span
        v-else-if="config.enableLogIncidentDebug && incidentState?.status"
        class="badge badge-outline badge-sm self-start"
        :class="incidentDebugBadgeClass(incidentState.status)"
      >
        {{ incidentDebugLabel(incidentState.status) }}
      </span>
    </div>
    <slot />
  </div>
</template>
<script lang="ts" setup>
import { LogEntry } from "@/models/LogEntry";
import type { LogIncidentState } from "@/types/logIncidents";

const { logEntry } = defineProps<{
  logEntry: LogEntry<any>;
}>();
const { showHostname, showContainerName } = useLoggingContext();

const { currentContainer } = useContainerStore();
const { hosts } = useHosts();

const container = currentContainer(toRef(() => logEntry.containerID));
const host = computed(() => hosts.value[container.value.host]);
const resolveIncident = useLogIncident();
const incidentState = computed(() => resolveIncident(logEntry) as LogIncidentState | undefined);

const route = useRoute();

function incidentBadgeClass(status?: string) {
  switch ((status || "").toLowerCase()) {
    case "open":
    case "opened":
      return "badge-warning";
    case "resolved":
    case "closed":
      return "badge-success";
    default:
      return "badge-info";
  }
}

function incidentDebugBadgeClass(status: LogIncidentState["status"]) {
  switch (status) {
    case "checking":
      return "badge-neutral";
    case "none":
      return "badge-ghost";
    case "skipped":
      return "badge-ghost";
    default:
      return "badge-info";
  }
}

function incidentDebugLabel(status: LogIncidentState["status"]) {
  switch (status) {
    case "checking":
      return "incident: checking";
    case "none":
      return "incident: none";
    case "skipped":
      return "incident: skipped";
    default:
      return "incident: matched";
  }
}
</script>
