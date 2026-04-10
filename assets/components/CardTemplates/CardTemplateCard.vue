<template>
  <div class="card bg-base-100 shadow-sm" :class="{ 'opacity-70': !template.enabled }">
    <div class="card-body gap-4 p-5">
      <div class="flex items-start justify-between gap-3">
        <div class="flex min-w-0 items-start gap-3">
          <div
            v-if="template.iconUrl"
            class="bg-base-200/70 flex h-11 w-11 shrink-0 items-center justify-center overflow-hidden rounded-xl"
          >
            <img :src="template.iconUrl" :alt="template.name" class="h-full w-full object-cover" />
          </div>
          <div class="min-w-0">
            <h4 class="truncate text-lg font-semibold">{{ template.name }}</h4>
            <p class="text-base-content/60 mt-1 text-sm">
              {{ template.filter || $t("card-templates.default-template") }}
            </p>
          </div>
        </div>
        <div class="flex items-center gap-1">
          <button class="btn btn-ghost btn-square btn-sm" @click="move(-1)" :disabled="index === 0">
            <mdi:arrow-up />
          </button>
          <button class="btn btn-ghost btn-square btn-sm" @click="move(1)" :disabled="index === total - 1">
            <mdi:arrow-down />
          </button>
          <button class="btn btn-ghost btn-square btn-sm" @click="editTemplate">
            <mdi:pencil-outline />
          </button>
          <button class="btn btn-ghost btn-square btn-sm" @click="deleteTemplate" :disabled="isDeleting || total === 1">
            <span v-if="isDeleting" class="loading loading-spinner loading-xs"></span>
            <mdi:trash-can-outline v-else />
          </button>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <span class="badge" :class="template.enabled ? 'badge-success badge-outline' : 'badge-ghost'">
          {{ template.enabled ? $t("card-templates.enabled") : $t("card-templates.disabled") }}
        </span>
        <span class="badge badge-outline">
          {{ $t("card-templates.fields-count-label", { count: template.extraFields.length }) }}
        </span>
        <span class="badge badge-outline">
          {{ $t("card-templates.groups-count-label", { count: detailGroupsCount }) }}
        </span>
        <span class="badge badge-outline">
          {{ $t("card-templates.detail-fields-count-label", { count: detailFieldsCount }) }}
        </span>
        <span class="badge badge-outline" v-if="template.showHost">{{ $t("card-templates.show-host") }}</span>
        <span class="badge badge-outline" v-if="template.showState">{{ $t("card-templates.show-state") }}</span>
        <span class="badge badge-outline" v-if="template.showCreated">{{ $t("card-templates.show-created") }}</span>
        <span class="badge badge-outline" v-if="template.showDetails">{{ $t("card-templates.show-details") }}</span>
        <span class="badge badge-outline" v-if="template.showActions">{{ $t("card-templates.show-actions") }}</span>
      </div>

      <div v-if="template.extraFields.length" class="grid gap-2">
        <div
          v-for="field in template.extraFields.slice(0, 4)"
          :key="field.id"
          class="bg-base-200/70 rounded-box flex items-center justify-between gap-3 px-3 py-2 text-sm"
        >
          <span class="text-base-content/60 truncate">{{ displayContainerCardFieldLabel(field) }}</span>
          <code class="text-xs">{{ field.source }}</code>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import CardTemplateForm from "./CardTemplateForm.vue";
import {
  containerCardTemplates,
  displayContainerCardFieldLabel,
  saveContainerCardTemplates,
  type ContainerCardTemplate,
} from "@/stores/containerCardTemplates";

const { template, index, total, onUpdated } = defineProps<{
  template: ContainerCardTemplate;
  index: number;
  total: number;
  onUpdated?: () => void;
}>();

const showDrawer = useDrawer();
const isDeleting = ref(false);
const detailGroupsCount = computed(() => template.detailGroups?.length || 0);
const detailFieldsCount = computed(() =>
  template.detailGroups?.length
    ? template.detailGroups.reduce((count, group) => count + (group.fields?.length || 0), 0)
    : template.detailFields?.length || 0,
);

function editTemplate() {
  showDrawer(CardTemplateForm, { template, onCreated: onUpdated }, "lg");
}

async function move(delta: number) {
  const nextIndex = index + delta;
  if (nextIndex < 0 || nextIndex >= total) return;

  const copy = [...containerCardTemplates.value];
  const [item] = copy.splice(index, 1);
  copy.splice(nextIndex, 0, item);
  containerCardTemplates.value = copy;
  await saveContainerCardTemplates();
  onUpdated?.();
}

async function deleteTemplate() {
  if (total === 1) return;
  isDeleting.value = true;
  try {
    containerCardTemplates.value = containerCardTemplates.value.filter((item) => item.id !== template.id);
    await saveContainerCardTemplates();
    onUpdated?.();
  } finally {
    isDeleting.value = false;
  }
}
</script>
