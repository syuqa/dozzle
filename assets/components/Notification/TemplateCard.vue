<template>
  <div class="card bg-base-100 shadow-sm">
    <div class="card-body gap-4 p-5">
      <div class="flex items-start justify-between gap-3">
        <div>
          <h4 class="text-lg font-semibold">{{ template.name }}</h4>
          <p class="text-base-content/60 text-sm">{{ $t("notifications.template-card.template-id", { id: template.id }) }}</p>
        </div>
        <div class="flex items-center gap-1">
          <button class="btn btn-ghost btn-square" @click="editTemplate">
            <mdi:pencil-outline />
          </button>
          <button class="btn btn-ghost btn-square" :disabled="isDeleting" @click="deleteTemplate">
            <span v-if="isDeleting" class="loading loading-spinner loading-xs"></span>
            <mdi:trash-can-outline v-else />
          </button>
        </div>
      </div>

      <pre class="bg-base-200 max-h-48 overflow-auto rounded p-3 text-xs whitespace-pre-wrap">{{ template.body }}</pre>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { NotificationTemplate } from "@/types/notifications";
import TemplateForm from "./TemplateForm.vue";

const { template, onUpdated } = defineProps<{
  template: NotificationTemplate;
  onUpdated?: () => void;
}>();

const showDrawer = useDrawer();
const isDeleting = ref(false);

function editTemplate() {
  showDrawer(TemplateForm, { template, onCreated: onUpdated }, "md");
}

async function deleteTemplate() {
  isDeleting.value = true;
  try {
    await fetch(withBase(`/api/notifications/templates/${template.id}`), { method: "DELETE" });
    onUpdated?.();
  } finally {
    isDeleting.value = false;
  }
}
</script>
