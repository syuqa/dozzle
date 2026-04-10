<template>
  <div class="dropdown dropdown-end" v-if="visibleActions.length">
    <label tabindex="0" class="btn btn-ghost btn-square btn-sm">
      <ion:ellipsis-vertical />
    </label>
    <ul tabindex="0" class="menu dropdown-content rounded-box bg-base-100 border-base-content/20 z-50 w-56 border p-1 shadow-sm">
      <li v-for="action in visibleActions" :key="action">
        <button @click="runAction(action)" :disabled="isDisabled(action)" class="flex items-center gap-2">
          <carbon:play v-if="action === 'start'" />
          <carbon:stop-filled-alt v-else-if="action === 'stop'" />
          <carbon:restart
            v-else-if="action === 'restart'"
            :class="{ 'animate-spin text-secondary': actionStates.restart }"
          />
          <carbon:upgrade
            v-else-if="action === 'update'"
            :class="{ 'animate-pulse text-secondary': actionStates.update }"
          />
          <mdi:code-tags v-else class="text-secondary" />
          {{ label(action) }}
        </button>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { Container } from "@/models/Container";
import {
  resolveContainerCardTextSource,
  type ContainerCardAction,
  type ContainerCardTemplate,
} from "@/stores/containerCardTemplates";

const { container, template } = defineProps<{
  container: Container;
  template: ContainerCardTemplate;
}>();

const { actionStates, start, stop, restart, update, injectLogsButton } = useContainerActions(toRef(() => container));
const { t } = useI18n();

const visibleActions = computed(() =>
  template.actions.filter((action) => {
    if (action === "start") return container.state !== "running";
    if (action === "stop") return container.state === "running";
    if (action === "restart") return container.state === "running";
    return true;
  }),
);

function resolveAlias() {
  const configured = template.injectAliasSource?.trim() || "";
  if (!configured) return "";
  if (configured === "name" || configured === "id" || configured.startsWith("label:")) {
    return resolveContainerCardTextSource(container, configured) || "";
  }
  return configured;
}

function resolveLogsUrl() {
  const alias = resolveAlias().trim();
  if (!alias) return "";
  return new URL(withBase(`/container/ref/${encodeURIComponent(alias)}`), window.location.origin).toString();
}

function isDisabled(action: ContainerCardAction) {
  if (action === "start") return actionStates.start || actionStates.restart;
  if (action === "stop") return actionStates.stop || actionStates.restart;
  if (action === "restart") return actionStates.restart || container.state !== "running";
  if (action === "update") return actionStates.update;
  return actionStates.injectLogsButton || !template.injectIndexPath.trim() || !resolveAlias().trim();
}

function label(action: ContainerCardAction) {
  if (action === "update" && container.isSwarm) {
    return t("toolbar.update-service");
  }
  if (action === "inject-logs-button") {
    return t("card-templates.inject-logs-button");
  }
  return t(`toolbar.${action}`);
}

async function runAction(action: ContainerCardAction) {
  if (action === "start") return start();
  if (action === "stop") return stop();
  if (action === "restart") return restart();
  if (action === "update") return update();
  return injectLogsButton(template.injectIndexPath, resolveAlias(), resolveLogsUrl());
}
</script>
