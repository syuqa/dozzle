<template>
  <div class="space-y-4">
    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.destination-form.name") }}</legend>
      <input
        ref="nameInput"
        v-model="name"
        type="text"
        class="input focus:input-primary w-full text-base"
        required
        :class="{ 'input-primary': name.trim().length > 0 }"
        :placeholder="$t('notifications.destination-form.name-placeholder')"
      />
    </fieldset>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.destination-form.telegram-bot-token") }}</legend>
      <input v-model="botToken" type="password" class="input input-bordered w-full" />
    </fieldset>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.destination-form.telegram-chat-id") }}</legend>
      <input v-model="chatId" type="text" class="input input-bordered w-full" />
    </fieldset>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.destination-form.telegram-thread-id") }}</legend>
      <input v-model="messageThreadId" type="text" class="input input-bordered w-full" />
    </fieldset>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.destination-form.telegram-parse-mode") }}</legend>
      <select v-model="parseMode" class="select select-bordered w-full">
        <option value="HTML">HTML</option>
        <option value="MarkdownV2">MarkdownV2</option>
        <option value="None">{{ $t("notifications.destination-form.telegram-no-parse-mode") }}</option>
      </select>
    </fieldset>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.destination-form.telegram-proxy-type") }}</legend>
      <select v-model="proxyType" class="select select-bordered w-full">
        <option value="">{{ $t("notifications.destination-form.telegram-proxy-none") }}</option>
        <option value="socks5">SOCKS5</option>
      </select>
      <p class="text-base-content/50 mt-1 text-xs">
        {{ $t("notifications.destination-form.telegram-proxy-type-hint") }}
      </p>
    </fieldset>

    <fieldset v-if="proxyType" class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.destination-form.telegram-proxy-address") }}</legend>
      <input v-model="proxyAddress" type="text" class="input input-bordered w-full" placeholder="host:port" />
    </fieldset>

    <fieldset v-if="proxyType === 'socks5'" class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.destination-form.telegram-proxy-username") }}</legend>
      <input v-model="proxyUsername" type="text" class="input input-bordered w-full" />
    </fieldset>

    <fieldset v-if="proxyType === 'socks5'" class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.destination-form.telegram-proxy-password") }}</legend>
      <input v-model="proxyPassword" type="password" class="input input-bordered w-full" />
    </fieldset>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.destination-form.shared-template") }}</legend>
      <select v-model.number="templateId" class="select select-bordered w-full">
        <option :value="0">{{ $t("notifications.destination-form.no-shared-template") }}</option>
        <option v-for="item in templates" :key="item.id" :value="item.id">{{ item.name }}</option>
      </select>
    </fieldset>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.destination-form.template") }}</legend>
      <textarea
        v-model="template"
        class="textarea textarea-bordered min-h-40 w-full font-mono text-sm"
        :placeholder="$t('notifications.destination-form.telegram-template-placeholder')"
      ></textarea>
      <p class="text-base-content/50 mt-1 text-xs">
        {{ $t("notifications.destination-form.telegram-template-hint") }}
      </p>
    </fieldset>

    <div v-if="error" class="alert alert-error">
      <span>{{ error }}</span>
    </div>

    <div v-if="testResult" class="alert" :class="testResult.success ? 'alert-success' : 'alert-error'">
      <span v-if="testResult.success">{{ $t("notifications.destination-form.test-success") }}</span>
      <span v-else>{{ testResult.error }}</span>
    </div>

    <div class="flex items-center gap-2 pt-4">
      <button class="btn" @click="testDestination" :disabled="!canTest || isTesting">
        <span v-if="isTesting" class="loading loading-spinner loading-sm"></span>
        {{ $t("notifications.destination-form.test") }}
      </button>
      <div class="flex-1"></div>
      <button class="btn" @click="close?.()">{{ $t("notifications.destination-form.cancel") }}</button>
      <button class="btn btn-primary" :disabled="!canSave" @click="saveDestination">
        <span v-if="isSaving" class="loading loading-spinner loading-sm"></span>
        {{ isEditing ? $t("notifications.destination-form.save") : $t("notifications.destination-form.add") }}
      </button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { Dispatcher, NotificationTemplate, TestWebhookResult } from "@/types/notifications";

const { close, onCreated, destination, isEditing, templates = [] } = defineProps<{
  close?: () => void;
  onCreated?: () => void;
  destination?: Dispatcher;
  isEditing: boolean;
  templates?: NotificationTemplate[];
}>();

const nameInput = ref<HTMLInputElement>();
const name = ref(destination?.name ?? "");
const botToken = ref(destination?.botToken ?? "");
const chatId = ref(destination?.chatId ?? "");
const messageThreadId = ref(destination?.messageThreadId ?? "");
const parseMode = ref(destination?.parseMode ?? "HTML");
const proxyType = ref(destination?.proxyType ?? "");
const proxyAddress = ref(destination?.proxyAddress ?? "");
const proxyUsername = ref(destination?.proxyUsername ?? "");
const proxyPassword = ref(destination?.proxyPassword ?? "");
const template = ref(destination?.template ?? "");
const templateId = ref(destination?.templateId ?? 0);
const isTesting = ref(false);
const isSaving = ref(false);
const error = ref<string | null>(null);
const testResult = ref<TestWebhookResult | null>(null);

useFocus(nameInput, { initialValue: true });

const canTest = computed(() => botToken.value.trim().length > 0 && chatId.value.trim().length > 0);
const canSave = computed(() => !isSaving.value && name.value.trim() && canTest.value);

async function testDestination() {
  if (!canTest.value) return;
  isTesting.value = true;
  testResult.value = null;
  try {
    const res = await fetch(withBase("/api/notifications/test-webhook"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        type: "telegram",
        botToken: botToken.value.trim(),
        chatId: chatId.value.trim(),
        messageThreadId: messageThreadId.value.trim() || undefined,
        parseMode: parseMode.value,
        proxyType: proxyType.value || undefined,
        proxyAddress: proxyAddress.value.trim() || undefined,
        proxyUsername: proxyUsername.value.trim() || undefined,
        proxyPassword: proxyPassword.value.trim() || undefined,
        templateId: templateId.value || undefined,
        template: template.value.trim() || undefined,
      }),
    });
    testResult.value = await res.json();
  } catch (e) {
    testResult.value = { success: false, error: e instanceof Error ? e.message : "Test failed" };
  } finally {
    isTesting.value = false;
  }
}

async function saveDestination() {
  if (!canSave.value) return;
  isSaving.value = true;
  error.value = null;
  try {
    const input = {
      name: name.value.trim(),
      type: "telegram",
      botToken: botToken.value.trim(),
      chatId: chatId.value.trim(),
      messageThreadId: messageThreadId.value.trim() || undefined,
      parseMode: parseMode.value,
      proxyType: proxyType.value || undefined,
      proxyAddress: proxyAddress.value.trim() || undefined,
      proxyUsername: proxyUsername.value.trim() || undefined,
      proxyPassword: proxyPassword.value.trim() || undefined,
      templateId: templateId.value || undefined,
      template: template.value.trim() || undefined,
    };
    const url = isEditing
      ? withBase(`/api/notifications/dispatchers/${destination!.id}`)
      : withBase("/api/notifications/dispatchers");
    const res = await fetch(url, {
      method: isEditing ? "PUT" : "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(input),
    });
    if (!res.ok) {
      const data = await res.json();
      throw new Error(data.error || "Failed to save destination");
    }
    onCreated?.();
    close?.();
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Failed to save destination";
  } finally {
    isSaving.value = false;
  }
}
</script>
