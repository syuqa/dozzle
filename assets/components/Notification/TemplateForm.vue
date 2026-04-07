<template>
  <div class="space-y-4 p-4">
    <div class="mb-6">
      <h2 class="text-2xl font-bold">
        {{ isEditing ? $t("notifications.template-form.edit-title") : $t("notifications.template-form.create-title") }}
      </h2>
      <p class="text-base-content/60">{{ $t("notifications.template-form.description") }}</p>
    </div>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.template-form.name") }}</legend>
      <input
        ref="nameInput"
        v-model="name"
        type="text"
        class="input focus:input-primary w-full text-base"
        :class="{ 'input-primary': name.trim().length > 0 }"
        :placeholder="$t('notifications.template-form.name-placeholder')"
      />
    </fieldset>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.template-form.body") }}</legend>
      <textarea
        v-model="body"
        class="textarea textarea-bordered min-h-56 w-full font-mono text-sm"
        :placeholder="$t('notifications.template-form.body-placeholder')"
      ></textarea>
      <p class="text-base-content/50 mt-1 text-xs">{{ $t("notifications.template-form.body-hint") }}</p>
    </fieldset>

    <div v-if="error" class="alert alert-error">
      <span>{{ error }}</span>
    </div>

    <div class="flex justify-end gap-2 pt-4">
      <button class="btn" @click="close?.()">{{ $t("notifications.template-form.cancel") }}</button>
      <button class="btn btn-primary" :disabled="!canSave" @click="saveTemplate">
        <span v-if="isSaving" class="loading loading-spinner loading-sm"></span>
        {{ isEditing ? $t("notifications.template-form.save") : $t("notifications.template-form.create") }}
      </button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { NotificationTemplate } from "@/types/notifications";

const { close, onCreated, template } = defineProps<{
  close?: () => void;
  onCreated?: () => void;
  template?: NotificationTemplate;
}>();

const isEditing = computed(() => !!template);
const nameInput = ref<HTMLInputElement>();
const name = ref(template?.name ?? "");
const body = ref(template?.body ?? "");
const isSaving = ref(false);
const error = ref<string | null>(null);

useFocus(nameInput, { initialValue: true });

const canSave = computed(() => !isSaving.value && name.value.trim().length > 0 && body.value.trim().length > 0);

async function saveTemplate() {
  if (!canSave.value) return;
  isSaving.value = true;
  error.value = null;
  try {
    const res = await fetch(
      isEditing.value ? withBase(`/api/notifications/templates/${template!.id}`) : withBase("/api/notifications/templates"),
      {
        method: isEditing.value ? "PUT" : "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: name.value.trim(),
          body: body.value,
        }),
      },
    );
    if (!res.ok) {
      const data = await res.json();
      throw new Error(data.error || "Failed to save template");
    }
    onCreated?.();
    close?.();
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Failed to save template";
  } finally {
    isSaving.value = false;
  }
}
</script>
