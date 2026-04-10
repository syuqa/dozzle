<template>
  <PageWithLinks>
    <section>
      <div class="has-underline">
        <h2>{{ $t("settings.about") }}</h2>
      </div>

      <div class="flex flex-row gap-2">
        <span v-html="$t('settings.using-version', { version: config.version })"></span>
        <span
          v-if="hasRelease"
          v-html="$t('settings.update-available', { nextVersion: latestRelease?.name, href: latestRelease?.htmlUrl })"
        ></span>
      </div>

      <div class="mt-4">
        {{ $t("settings.help-support") }}

        <ul class="mt-6 flex gap-2">
          <li>
            <a href="https://github.com/amir20/dozzle" target="_blank" rel="noopener noreferrer" class="btn">
              <mdi:github /> amir20/dozzle
            </a>
          </li>
          <li>
            <a
              href="https://buymeacoffee.com/amirraminfar"
              target="_blank"
              rel="noopener noreferrer"
              class="btn btn-secondary"
            >
              <mdi:beer />
              Buy me a beer
            </a>
          </li>
        </ul>
      </div>
    </section>

    <section class="@container flex flex-col">
      <div class="has-underline">
        <h2>{{ $t("settings.display") }}</h2>
      </div>

      <section class="grid-cols-2 gap-4 @3xl:grid">
        <div class="flex flex-col gap-4 text-balance @3xl:pr-8">
          <Toggle v-model="compact"> {{ $t("settings.compact") }} </Toggle>

          <Toggle v-model="smallerScrollbars"> {{ $t("settings.small-scrollbars") }} </Toggle>

          <Toggle v-model="showTimestamp">{{ $t("settings.show-timestamps") }}</Toggle>

          <Toggle v-model="showStd">{{ $t("settings.show-std") }}</Toggle>

          <Toggle v-model="softWrap">{{ $t("settings.soft-wrap") }}</Toggle>

          <LabeledInput>
            <template #label>
              {{ $t("settings.locale") }}
            </template>
            <template #input>
              <DropdownMenu
                v-model="locale"
                :options="[
                  { label: 'Auto', value: '' },
                  ...availableLocales.map((l) => ({ label: l.toLocaleUpperCase(), value: l })),
                ]"
              />
            </template>
          </LabeledInput>

          <LabeledInput>
            <template #label>
              {{ $t("settings.datetime-format") }}
            </template>
            <template #input>
              <div class="flex gap-4">
                <DropdownMenu
                  v-model="dateLocale"
                  :options="[
                    { label: 'Auto', value: 'auto' },
                    { label: 'MM/DD/YYYY', value: 'en-US' },
                    { label: 'DD/MM/YYYY', value: 'en-GB' },
                    { label: 'DD.MM.YYYY', value: 'de-DE' },
                    { label: 'YYYY-MM-DD', value: 'en-CA' },
                  ]"
                />
                <DropdownMenu
                  v-model="hourStyle"
                  :options="[
                    { label: $t('settings.hour.auto'), value: 'auto' },
                    { label: $t('settings.hour.12'), value: '12' },
                    { label: $t('settings.hour.24'), value: '24' },
                  ]"
                />
              </div>
            </template>
          </LabeledInput>

          <LabeledInput>
            <template #label>
              {{ $t("settings.font-size") }}
            </template>
            <template #input>
              <DropdownMenu
                v-model="size"
                :options="[
                  { label: $t('settings.size.small'), value: 'small' },
                  { label: $t('settings.size.medium'), value: 'medium' },
                  { label: $t('settings.size.large'), value: 'large' },
                ]"
              />
            </template>
          </LabeledInput>

          <LabeledInput>
            <template #label>
              {{ $t("settings.color-scheme") }}
            </template>
            <template #input>
              <DropdownMenu
                v-model="lightTheme"
                :options="[
                  { label: $t('settings.theme.auto'), value: 'auto' },
                  { label: $t('settings.theme.dark'), value: 'dark' },
                  { label: $t('settings.theme.light'), value: 'light' },
                ]"
              />
            </template>
          </LabeledInput>
        </div>
        <LogList
          :messages="fakeMessages"
          :last-selected-item="undefined"
          :show-container-name="false"
          class="border-base-content/50 hidden overflow-hidden rounded-lg border shadow-sm @3xl:block"
        />
      </section>
    </section>

    <section class="flex flex-col gap-4">
      <div class="has-underline">
        <h2>{{ $t("settings.options") }}</h2>
      </div>

      <LabeledInput>
        <template #label>
          {{ $t("settings.automatic-redirect") }}
        </template>
        <template #input>
          <DropdownMenu
            v-model="automaticRedirect"
            :options="[
              { label: $t('settings.redirect.instant'), value: 'instant' },
              { label: $t('settings.redirect.delayed'), value: 'delayed' },
              { label: $t('settings.redirect.none'), value: 'none' },
            ]"
          />
        </template>
      </LabeledInput>
      <LabeledInput>
        <template #label>
          {{ $t("settings.group-containers") }}
        </template>
        <template #input>
          <DropdownMenu
            v-model="groupContainers"
            :options="[
              { label: $t('settings.grouping.always'), value: 'always' },
              { label: $t('settings.grouping.at-least-2'), value: 'at-least-2' },
              { label: $t('settings.grouping.never'), value: 'never' },
            ]"
          />
        </template>
      </LabeledInput>

      <Toggle v-model="search">
        {{ $t("settings.search") }} <key-shortcut char="f" class="align-top"></key-shortcut>
      </Toggle>

      <Toggle v-model="showAllContainers">{{ $t("settings.show-stopped-containers") }}</Toggle>
    </section>

    <section class="flex flex-col gap-4">
      <div class="has-underline">
        <h2>{{ $t("card-templates.title") }}</h2>
      </div>

      <p class="text-base-content/70 max-w-4xl text-sm">
        {{ $t("settings.card-templates-description") }}
      </p>

      <div class="grid gap-4 xl:grid-cols-[minmax(0,1fr)_auto]">
        <div class="rounded-box border-base-content/10 bg-base-100 border p-4">
          <div v-if="!cardTemplatesLoaded" class="text-base-content/60 text-sm">
            {{ $t("settings.loading-card-templates") }}
          </div>
          <div v-else-if="cardTemplatesError" class="text-error text-sm">
            {{ cardTemplatesError }}
          </div>
          <ul v-else class="space-y-2">
            <li
              v-for="template in cardTemplates"
              :key="template.id"
              class="bg-base-200/60 rounded-box flex items-center justify-between gap-3 px-3 py-2 text-sm"
            >
              <div class="min-w-0">
                <div class="truncate font-medium">{{ template.name }}</div>
                <div class="text-base-content/60 truncate text-xs">
                  {{ template.filter || $t("card-templates.default-template") }}
                </div>
              </div>
              <span class="badge shrink-0" :class="template.enabled ? 'badge-success badge-outline' : 'badge-ghost'">
                {{ template.enabled ? $t("card-templates.enabled") : $t("card-templates.disabled") }}
              </span>
            </li>
          </ul>
        </div>

        <div class="flex items-start">
          <router-link to="/settings/cards" class="btn btn-primary">
            <mdi:card-text-outline />
            {{ $t("settings.open-card-templates") }}
          </router-link>
        </div>
      </div>
    </section>
  </PageWithLinks>
</template>

<script lang="ts" setup>
import { ComplexLogEntry, SimpleLogEntry, GroupedLogEntry } from "@/models/LogEntry";

import {
  automaticRedirect,
  compact,
  hourStyle,
  dateLocale,
  lightTheme,
  search,
  showAllContainers,
  showStd,
  showTimestamp,
  size,
  smallerScrollbars,
  softWrap,
  locale,
  groupContainers,
} from "@/stores/settings";
import { useContainerCardTemplates } from "@/stores/containerCardTemplates";

import { availableLocales, i18n } from "@/modules/i18n";

const { t } = useI18n();
const { templates: cardTemplates, loaded: cardTemplatesLoaded, error: cardTemplatesError } = useContainerCardTemplates();

setTitle(t("title.settings"));
const { latestRelease, hasRelease } = useAnnouncements();

const now = new Date();
const hoursAgo = (hours: number) => {
  const date = new Date(now);
  date.setHours(date.getHours() - hours);
  return date;
};

const fakeMessages = computedWithControl(
  () => i18n.global.locale.value,
  () => [
    new SimpleLogEntry(t("settings.log.preview"), "123", 1, hoursAgo(16), "info", "stdout", ""),
    new SimpleLogEntry(t("settings.log.warning"), "123", 2, hoursAgo(12), "warn", "stdout", ""),
    new GroupedLogEntry(
      [
        t("settings.log.multi-line-error.start-line"),
        t("settings.log.multi-line-error.middle-line"),
        t("settings.log.multi-line-error.end-line"),
      ],
      "123",
      3,
      hoursAgo(7),
      "error",
      "stderr",
    ),
    new ComplexLogEntry(
      {
        message: t("settings.log.complex"),
        context: {
          key: "value",
          key2: "value2",
        },
      },
      "123",
      6,
      new Date(),
      "info",
      "stdout",
      "",
    ),
    new SimpleLogEntry(t("settings.log.simple"), "123", 7, new Date(), "debug", "stderr", ""),
  ],
);
</script>
<style scoped>
@reference "@/main.css";

.has-underline {
  @apply border-base-content/50 mb-4 border-b py-2;

  h2 {
    @apply text-3xl;
  }
}

:deep(a:not(.menu a):not(.btn)) {
  @apply text-primary underline-offset-4 hover:underline;
}
</style>
