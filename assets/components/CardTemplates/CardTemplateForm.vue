<template>
  <div class="space-y-4 p-4">
    <div class="mb-6">
      <h2 class="text-2xl font-bold">
        {{ isEditing ? $t("card-templates.edit-template") : $t("card-templates.create-template") }}
      </h2>
      <p class="text-base-content/60">{{ $t("card-templates.form-description") }}</p>
    </div>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("card-templates.name") }}</legend>
      <div class="grid gap-3 lg:grid-cols-[minmax(0,1fr)_220px]">
        <div class="grid gap-3">
          <input v-model="draft.name" type="text" class="input input-bordered w-full text-base" />
          <label class="form-control gap-2">
            <span class="label-text font-medium">{{ $t("card-templates.icon-url") }}</span>
            <input
              v-model="draft.iconUrl"
              type="text"
              class="input input-bordered w-full"
              :placeholder="$t('card-templates.icon-url-placeholder')"
            />
          </label>
        </div>
        <label class="bg-base-200/60 rounded-box flex items-center justify-between gap-3 px-4 py-3">
          <span class="font-medium">{{ $t("card-templates.enabled") }}</span>
          <Toggle v-model="draft.enabled" />
        </label>
      </div>
    </fieldset>

    <fieldset class="fieldset min-w-0">
      <legend class="fieldset-legend text-lg">{{ $t("card-templates.filter") }}</legend>
      <div class="space-y-3">
        <textarea
          v-model="draft.filter"
          class="textarea textarea-bordered min-h-32 w-full font-mono text-sm"
          :placeholder="$t('card-templates.filter-placeholder')"
        />
        <div class="bg-base-200/60 rounded-box mt-3 px-4 py-3 text-sm leading-6">
          <div class="text-base-content/70 font-medium">{{ $t("card-templates.filter-help-title") }}</div>
          <div class="text-base-content/60 mt-1">{{ $t("card-templates.filter-hint") }}</div>
        </div>
      </div>
    </fieldset>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("card-templates.display-options") }}</legend>
      <div class="grid gap-3 md:grid-cols-5">
        <label class="bg-base-200/60 rounded-box flex items-center justify-between gap-3 px-4 py-3">
          <span class="font-medium">{{ $t("card-templates.show-host") }}</span>
          <Toggle v-model="draft.showHost" />
        </label>
        <label class="bg-base-200/60 rounded-box flex items-center justify-between gap-3 px-4 py-3">
          <span class="font-medium">{{ $t("card-templates.show-state") }}</span>
          <Toggle v-model="draft.showState" />
        </label>
        <label class="bg-base-200/60 rounded-box flex items-center justify-between gap-3 px-4 py-3">
          <span class="font-medium">{{ $t("card-templates.show-created") }}</span>
          <Toggle v-model="draft.showCreated" />
        </label>
        <label class="bg-base-200/60 rounded-box flex items-center justify-between gap-3 px-4 py-3">
          <span class="font-medium">{{ $t("card-templates.show-details") }}</span>
          <Toggle v-model="draft.showDetails" />
        </label>
        <label class="bg-base-200/60 rounded-box flex items-center justify-between gap-3 px-4 py-3">
          <span class="font-medium">{{ $t("card-templates.show-actions") }}</span>
          <Toggle v-model="draft.showActions" />
        </label>
      </div>

      <div class="bg-base-200/40 rounded-box mt-3 space-y-3 px-4 py-4">
        <label class="bg-base-100 rounded-box flex items-center justify-between gap-3 px-4 py-3">
          <span class="font-medium">{{ $t("card-templates.show-vulnerabilities") }}</span>
          <Toggle v-model="draft.showVulnerabilities" />
        </label>

        <div v-if="draft.showVulnerabilities" class="grid gap-3 xl:grid-cols-2">
          <label class="form-control gap-2">
            <span class="label-text font-medium">{{ $t("card-templates.vulnerability-package-types") }}</span>
            <input
              :value="draft.vulnerabilityPackageTypes?.join(', ') || ''"
              @input="draft.vulnerabilityPackageTypes = splitCsv(($event.target as HTMLInputElement).value)"
              type="text"
              class="input input-bordered w-full"
              :placeholder="$t('card-templates.vulnerability-package-types-placeholder')"
            />
          </label>

          <div class="form-control gap-2">
            <span class="label-text font-medium">{{ $t("card-templates.vulnerability-severities") }}</span>
            <div class="flex flex-wrap gap-2">
              <button
                v-for="severity in severityOptions"
                :key="severity"
                type="button"
                class="btn btn-xs"
                :class="{ 'btn-primary': draft.vulnerabilitySeverities?.includes(severity) }"
                @click="toggleVulnerabilitySeverity(severity)"
              >
                {{ severity }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <div v-if="draft.showActions" class="bg-base-200/40 rounded-box mt-3 space-y-3 px-4 py-4">
        <div class="text-sm font-medium">{{ $t("card-templates.actions") }}</div>
        <div class="grid gap-3 md:grid-cols-2">
          <label
            v-for="action in actionOptions"
            :key="action.value"
            class="bg-base-100 rounded-box flex items-center justify-between gap-3 px-4 py-3"
          >
            <span>{{ $t(action.labelKey) }}</span>
            <input
              type="checkbox"
              class="checkbox checkbox-sm"
              :checked="draft.actions.includes(action.value)"
              @change="toggleAction(action.value)"
            />
          </label>
        </div>

        <div v-if="draft.actions.includes('inject-logs-button')" class="grid gap-3 pt-2 xl:grid-cols-2">
          <label class="form-control gap-2">
            <span class="label-text font-medium">{{ $t("card-templates.inject-index-path") }}</span>
            <input v-model="draft.injectIndexPath" type="text" class="input input-bordered w-full" placeholder="/usr/share/nginx/html/index.html" />
          </label>

          <label class="form-control gap-2">
            <span class="label-text font-medium">{{ $t("card-templates.inject-alias-source") }}</span>
            <input v-model="draft.injectAliasSource" type="text" class="input input-bordered w-full" placeholder="udg-backend-1-prod" />
          </label>
        </div>
      </div>
    </fieldset>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("card-templates.extra-fields") }}</legend>
      <div class="flex items-center justify-between gap-3">
        <p class="text-base-content/60 text-sm">{{ $t("card-templates.extra-fields-description") }}</p>
        <button class="btn btn-sm" @click="addField">
          <mdi:plus />
          {{ $t("card-templates.add-field") }}
        </button>
      </div>

      <div v-if="draft.extraFields.length" class="space-y-3">
        <div
          v-for="(field, index) in draft.extraFields"
          :key="field.id"
          class="bg-base-200/60 rounded-box grid gap-3 p-4 xl:grid-cols-[minmax(0,180px)_minmax(0,220px)_minmax(0,1fr)_auto_auto]"
        >
          <input v-model="field.label" type="text" class="input input-bordered w-full" :placeholder="$t('card-templates.field-label')" />
          <DropdownMenu v-model="field.source" :options="fieldOptions" class="w-full" />
          <input
            v-if="String(field.source).startsWith('label:')"
            :value="labelKeyValue(field.source)"
            @input="updateLabelSource(field, ($event.target as HTMLInputElement).value)"
            type="text"
            class="input input-bordered w-full"
            placeholder="com.docker.compose.service"
          />
          <div v-else class="text-base-content/60 flex items-center text-sm">{{ $t("card-templates.field-source-hint") }}</div>
          <div class="flex items-center gap-1 xl:justify-end">
            <button class="btn btn-ghost btn-square btn-sm" @click="moveField(index, -1)" :disabled="index === 0">
              <mdi:arrow-up />
            </button>
            <button class="btn btn-ghost btn-square btn-sm" @click="moveField(index, 1)" :disabled="index === draft.extraFields.length - 1">
              <mdi:arrow-down />
            </button>
          </div>
          <button class="btn btn-ghost btn-sm justify-self-start xl:justify-self-auto" @click="removeField(field.id)">
            <mdi:close />
          </button>
        </div>
      </div>
      <div v-else class="text-base-content/60 text-sm">{{ $t("card-templates.no-extra-fields") }}</div>
    </fieldset>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("card-templates.detail-groups") }}</legend>
      <div class="flex items-center justify-between gap-3">
        <p class="text-base-content/60 text-sm">{{ $t("card-templates.detail-groups-description") }}</p>
        <button class="btn btn-sm" @click="addDetailGroup">
          <mdi:plus />
          {{ $t("card-templates.add-detail-group") }}
        </button>
      </div>

      <div v-if="draft.detailGroups?.length" class="space-y-4">
        <div v-for="(group, groupIndex) in draft.detailGroups" :key="group.id" class="bg-base-200/60 rounded-box space-y-4 p-4">
          <div class="grid gap-3 xl:grid-cols-[minmax(0,1fr)_auto_auto_auto]">
            <input v-model="group.name" type="text" class="input input-bordered w-full" :placeholder="$t('card-templates.group-name')" />
            <button class="btn btn-ghost btn-square btn-sm" @click="moveDetailGroup(groupIndex, -1)" :disabled="groupIndex === 0">
              <mdi:arrow-up />
            </button>
            <button class="btn btn-ghost btn-square btn-sm" @click="moveDetailGroup(groupIndex, 1)" :disabled="groupIndex === draft.detailGroups.length - 1">
              <mdi:arrow-down />
            </button>
            <button class="btn btn-ghost btn-sm" @click="removeDetailGroup(group.id)" :disabled="draft.detailGroups.length === 1">
              <mdi:close />
            </button>
          </div>

          <div class="flex items-center justify-between gap-3">
            <div class="text-base-content/60 text-sm">{{ $t("card-templates.detail-fields-description") }}</div>
            <button class="btn btn-sm" @click="addDetailField(group.id)">
              <mdi:plus />
              {{ $t("card-templates.add-detail-field") }}
            </button>
          </div>

          <div v-if="nonReportFields(group).length" class="space-y-3">
            <div
              v-for="(field, index) in nonReportFields(group)"
              :key="field.id"
              class="bg-base-100 rounded-box grid gap-3 p-4 xl:grid-cols-[minmax(0,170px)_minmax(0,180px)_minmax(0,1fr)_auto_auto]"
            >
              <input v-model="field.label" type="text" class="input input-bordered w-full" :placeholder="$t('card-templates.field-label')" />
              <DropdownMenu v-model="field.kind" :options="detailKindOptions" class="w-full" />
              <div class="min-w-0">
                <template v-if="field.kind === 'link'">
                  <div class="grid gap-2">
                    <input v-model="field.linkUrl" type="text" class="input input-bordered w-full" :placeholder="$t('card-templates.detail-link-url-placeholder')" />
                    <input v-model="field.linkText" type="text" class="input input-bordered w-full" :placeholder="$t('card-templates.detail-link-text-placeholder')" />
                  </div>
                </template>
                <template v-else-if="field.kind === 'template_text'">
                  <div class="grid gap-2">
                    <textarea
                      v-model="field.textValue"
                      class="textarea textarea-bordered min-h-24 w-full"
                      :placeholder="$t('card-templates.detail-template-text-placeholder')"
                    />
                    <div class="text-base-content/60 text-sm">{{ $t("card-templates.detail-template-text-hint") }}</div>
                  </div>
                </template>
                <template v-else-if="field.kind === 'static_text'">
                  <div class="grid gap-2">
                    <textarea
                      v-model="field.textValue"
                      class="textarea textarea-bordered min-h-24 w-full"
                      :placeholder="$t('card-templates.detail-static-text-placeholder')"
                    />
                    <div class="text-base-content/60 text-sm">{{ $t("card-templates.detail-static-text-hint") }}</div>
                  </div>
                </template>
                <template v-else>
                  <div class="grid gap-2">
                    <DropdownMenu v-model="field.source" :options="fieldOptions" class="w-full" />
                    <input
                      v-if="String(field.source).startsWith('label:')"
                      :value="labelKeyValue(field.source)"
                      @input="updateLabelSource(field, ($event.target as HTMLInputElement).value)"
                      type="text"
                      class="input input-bordered w-full"
                      placeholder="com.docker.compose.service"
                    />
                    <div v-else class="text-base-content/60 flex items-center text-sm">{{ $t("card-templates.field-source-hint") }}</div>
                  </div>
                </template>
              </div>
              <div class="flex items-center gap-1 xl:justify-end">
                <button class="btn btn-ghost btn-square btn-sm" @click="moveDetailField(group.id, field.id, -1)" :disabled="index === 0">
                  <mdi:arrow-up />
                </button>
                <button
                  class="btn btn-ghost btn-square btn-sm"
                  @click="moveDetailField(group.id, field.id, 1)"
                  :disabled="index === nonReportFields(group).length - 1"
                >
                  <mdi:arrow-down />
                </button>
              </div>
              <button class="btn btn-ghost btn-sm justify-self-start xl:justify-self-auto" @click="removeDetailField(group.id, field.id)">
                <mdi:close />
              </button>
            </div>
          </div>
          <div v-else class="text-base-content/60 text-sm">{{ $t("card-templates.no-detail-fields") }}</div>
        </div>
      </div>
      <div v-else class="text-base-content/60 text-sm">{{ $t("card-templates.no-detail-groups") }}</div>
    </fieldset>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("card-templates.report-definitions") }}</legend>
      <div class="flex items-center justify-between gap-3">
        <p class="text-base-content/60 text-sm">{{ $t("card-templates.detail-reports-description") }}</p>
        <button class="btn btn-sm" @click="addDetailReportField">
          <mdi:plus />
          {{ $t("card-templates.add-detail-report") }}
        </button>
      </div>

      <div v-if="detailReportFields.length" class="space-y-3">
        <div
          v-for="(field, index) in detailReportFields"
          :key="field.id"
          class="bg-base-200/60 rounded-box grid gap-3 p-4 xl:grid-cols-[minmax(0,170px)_minmax(0,1fr)_auto_auto]"
        >
          <input v-model="field.label" type="text" class="input input-bordered w-full" :placeholder="$t('card-templates.field-label')" />
          <DropdownMenu
            v-model="field.reportId"
            :options="reportOptions"
            class="w-full"
            :placeholder="$t('card-templates.detail-report-placeholder')"
          />
          <div class="flex items-center gap-1 xl:justify-end">
            <button class="btn btn-ghost btn-square btn-sm" @click="moveDetailReportField(field.id, -1)" :disabled="index === 0">
              <mdi:arrow-up />
            </button>
            <button class="btn btn-ghost btn-square btn-sm" @click="moveDetailReportField(field.id, 1)" :disabled="index === detailReportFields.length - 1">
              <mdi:arrow-down />
            </button>
          </div>
          <button class="btn btn-ghost btn-sm justify-self-start xl:justify-self-auto" @click="removeDetailReportField(field.id)">
            <mdi:close />
          </button>
        </div>
      </div>
      <div v-else class="text-base-content/60 text-sm">{{ $t("card-templates.no-detail-reports") }}</div>
    </fieldset>

    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("card-templates.preview") }}</legend>
      <div v-if="previewContainers.length" class="grid min-w-0 gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(280px,1fr)]">
        <div class="grid min-w-0 gap-4 md:grid-cols-1">
          <article
            v-for="container in previewContainers"
            :key="container.id"
            class="rounded-box border-base-content/10 bg-base-100 min-w-0 border p-5 shadow-sm"
          >
            <div class="mb-4 flex items-start justify-between gap-3">
              <div class="flex min-w-0 items-start gap-3">
                <div
                  v-if="draft.iconUrl"
                  class="bg-base-200/70 flex h-11 w-11 shrink-0 items-center justify-center overflow-hidden rounded-xl"
                >
                  <img :src="draft.iconUrl" :alt="draft.name" class="h-full w-full object-cover" />
                </div>
                <div class="min-w-0">
                  <div class="truncate text-lg font-semibold">{{ container.name }}</div>
                  <div class="text-base-content/60 mt-1 flex flex-wrap gap-x-3 gap-y-1 text-sm">
                    <span v-if="draft.showHost">{{ container.hostLabel }}</span>
                    <span v-if="draft.showState">{{ containerCardStateLabel(container.state) }}</span>
                  </div>
                </div>
              </div>
              <div class="flex flex-col items-end gap-2">
                <span class="badge" :class="matchesContainerCardTemplate(container, draft.filter) ? 'badge-success badge-outline' : 'badge-ghost'">
                  {{ matchesContainerCardTemplate(container, draft.filter) ? $t("card-templates.match-yes") : $t("card-templates.match-no") }}
                </span>
                <div class="badge badge-outline shrink-0" v-if="draft.showCreated">
                  <RelativeTime :date="container.created" />
                </div>
              </div>
            </div>

            <div class="grid gap-3" v-if="draft.extraFields.length">
              <div
                v-for="field in draft.extraFields"
                :key="field.id"
                class="bg-base-200/70 rounded-box grid min-w-0 grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)] items-start gap-3 px-3 py-2 text-sm"
              >
                <span class="text-base-content/60 min-w-0 truncate">{{ displayContainerCardFieldLabel(field) }}</span>
                <span
                  class="min-w-0 truncate text-right font-medium"
                  :title="resolveContainerCardField(container, field.source) || '—'"
                >
                  {{ resolveContainerCardField(container, field.source) || "—" }}
                </span>
              </div>
            </div>
          </article>
        </div>

        <div class="grid gap-4">
          <article class="rounded-box border-base-content/10 bg-base-100 min-w-0 border p-5 shadow-sm">
            <div class="space-y-3 text-sm">
              <div class="font-semibold">{{ $t("card-templates.preview-meta") }}</div>
              <div class="text-base-content/70">{{ $t("card-templates.preview-summary") }}</div>
              <div class="bg-base-200/60 rounded-box flex items-center justify-between px-3 py-2">
                <span>{{ $t("card-templates.matched-count") }}</span>
                <span class="font-semibold">{{ matchedPreviewCount }}</span>
              </div>
              <div class="bg-base-200/60 rounded-box flex items-center justify-between px-3 py-2">
                <span>{{ $t("card-templates.visible-count") }}</span>
                <span class="font-semibold">{{ previewContainers.length }}</span>
              </div>
              <div class="bg-base-200/60 rounded-box space-y-2 px-3 py-3">
                <div class="text-base-content/60">{{ $t("card-templates.preview-filter") }}</div>
                <code class="block min-w-0 whitespace-pre-wrap break-all">{{ draft.filter || $t("card-templates.default-template") }}</code>
              </div>
            </div>
          </article>

          <article class="rounded-box border-base-content/10 bg-base-100 min-w-0 border p-5 shadow-sm">
            <div class="space-y-3 text-sm">
              <div class="font-semibold">{{ $t("card-templates.layout-summary") }}</div>
              <div class="text-base-content/70">{{ $t("card-templates.layout-summary-description") }}</div>
              <div class="flex flex-wrap gap-2">
                <span class="badge badge-outline" :class="{ 'opacity-50': !draft.showHost }">{{ $t("card-templates.show-host") }}</span>
                <span class="badge badge-outline" :class="{ 'opacity-50': !draft.showState }">{{ $t("card-templates.show-state") }}</span>
                <span class="badge badge-outline" :class="{ 'opacity-50': !draft.showCreated }">{{ $t("card-templates.show-created") }}</span>
                <span class="badge badge-outline" :class="{ 'opacity-50': !draft.showDetails }">{{ $t("card-templates.show-details") }}</span>
                <span class="badge badge-outline" :class="{ 'opacity-50': !draft.showActions }">{{ $t("card-templates.show-actions") }}</span>
                <span class="badge badge-outline" :class="{ 'opacity-50': !draft.showVulnerabilities }">{{ $t("card-templates.show-vulnerabilities") }}</span>
              </div>
              <div class="bg-base-200/60 rounded-box space-y-2 px-3 py-3">
                <div class="text-base-content/60">{{ $t("card-templates.preview-extra-fields") }}</div>
                <ul v-if="draft.extraFields.length" class="space-y-1">
                  <li v-for="field in draft.extraFields.slice(0, 5)" :key="field.id" class="flex items-center justify-between gap-2">
                    <span class="truncate">{{ displayContainerCardFieldLabel(field) }}</span>
                    <code class="text-xs">{{ field.source }}</code>
                  </li>
                </ul>
                <div v-else class="text-base-content/60">{{ $t("card-templates.no-extra-fields") }}</div>
              </div>
              <div class="bg-base-200/60 rounded-box space-y-2 px-3 py-3">
                <div class="text-base-content/60">{{ $t("card-templates.preview-detail-fields") }}</div>
                <ul v-if="draft.detailGroups?.length" class="space-y-2">
                  <li v-for="group in draft.detailGroups.slice(0, 4)" :key="group.id" class="space-y-1">
                    <div class="font-medium">{{ group.name }}</div>
                    <div class="text-base-content/60 text-xs">{{ $t("card-templates.detail-fields-count-label", { count: nonReportFields(group).length }) }}</div>
                  </li>
                </ul>
                <div v-else class="text-base-content/60">{{ $t("card-templates.no-detail-groups") }}</div>
              </div>
              <div class="bg-base-200/60 rounded-box space-y-2 px-3 py-3">
                <div class="text-base-content/60">{{ $t("card-templates.report-definitions") }}</div>
                <ul v-if="detailReportFields.length" class="space-y-1">
                  <li v-for="field in detailReportFields.slice(0, 5)" :key="field.id" class="flex items-center justify-between gap-2">
                    <span class="truncate">{{ field.label || $t("card-templates.open-report") }}</span>
                    <code class="text-xs">{{ field.reportId || "—" }}</code>
                  </li>
                </ul>
                <div v-else class="text-base-content/60">{{ $t("card-templates.no-detail-reports") }}</div>
              </div>
              <div v-if="draft.showVulnerabilities" class="bg-base-200/60 rounded-box space-y-2 px-3 py-3">
                <div class="text-base-content/60">{{ $t("card-templates.vulnerability-report-title") }}</div>
                <div class="flex flex-wrap gap-2">
                  <span v-if="draft.vulnerabilityPackageTypes?.length" class="badge badge-outline">{{ draft.vulnerabilityPackageTypes.join(", ") }}</span>
                  <span v-else class="text-base-content/60">{{ $t("card-templates.vulnerability-all-package-types") }}</span>
                </div>
                <div class="flex flex-wrap gap-2">
                  <span v-if="draft.vulnerabilitySeverities?.length" class="badge badge-outline">{{ draft.vulnerabilitySeverities.join(", ") }}</span>
                  <span v-else class="text-base-content/60">{{ $t("card-templates.vulnerability-all-severities") }}</span>
                </div>
              </div>
              <div v-if="draft.showActions" class="bg-base-200/60 rounded-box space-y-2 px-3 py-3">
                <div class="text-base-content/60">{{ $t("card-templates.actions") }}</div>
                <div class="flex flex-wrap gap-2">
                  <span v-for="action in draft.actions" :key="action" class="badge badge-outline">
                    {{ action === "inject-logs-button" ? $t("card-templates.inject-logs-button") : $t(`toolbar.${action}`) }}
                  </span>
                </div>
                <div v-if="draft.actions.includes('inject-logs-button')" class="space-y-2 pt-2 text-xs">
                  <div class="flex items-center justify-between gap-2">
                    <span class="text-base-content/60">{{ $t("card-templates.inject-index-path") }}</span>
                    <code class="min-w-0 break-all text-right">{{ draft.injectIndexPath || "—" }}</code>
                  </div>
                  <div class="flex items-center justify-between gap-2">
                    <span class="text-base-content/60">{{ $t("card-templates.inject-alias-source") }}</span>
                    <code class="min-w-0 break-all text-right">{{ draft.injectAliasSource || "—" }}</code>
                  </div>
                </div>
              </div>
            </div>
          </article>
        </div>
      </div>
    </fieldset>

    <div v-if="error" class="alert alert-error alert-soft">
      <span>{{ error }}</span>
    </div>

    <div class="flex justify-end gap-2 pt-4">
      <button class="btn" @click="close?.()">{{ $t("button.cancel") }}</button>
      <button class="btn btn-primary" :disabled="!canSave" @click="saveTemplate">
        <span v-if="isSaving" class="loading loading-spinner loading-sm"></span>
        {{ isEditing ? $t("notifications.template-form.save") : $t("notifications.template-form.create") }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Container } from "@/models/Container";
import {
  containerCardActionOptions,
  containerCardFieldOptions,
  containerCardStateLabel,
  containerCardTemplates,
  createContainerCardDetailGroup,
  createContainerCardField,
  createDetailTextField,
  createContainerCardTemplate,
  detailFieldKindOptions,
  displayContainerCardFieldLabel,
  matchesContainerCardTemplate,
  resolveContainerCardField,
  saveContainerCardTemplates,
  type ContainerCardField,
  type ContainerCardTemplate,
  vulnerabilitySeverityOptions,
} from "@/stores/containerCardTemplates";
import { containerCardReports, useContainerCardReports } from "@/stores/containerCardReports";

const { close, onCreated, template } = defineProps<{
  close?: () => void;
  onCreated?: () => void;
  template?: ContainerCardTemplate;
}>();

const { t } = useI18n();
const { containers } = storeToRefs(useContainerStore()) as unknown as { containers: Ref<Container[]> };
useContainerCardReports();
const isEditing = computed(() => !!template);
const isSaving = ref(false);
const error = ref<string>();
const draft = reactive<ContainerCardTemplate>(
  template
    ? {
        ...template,
        showActions: template.showActions ?? false,
        showDetails: template.showDetails ?? true,
        showVulnerabilities: template.showVulnerabilities ?? false,
        actions: [...(template.actions ?? [])],
        injectIndexPath: template.injectIndexPath || "/usr/share/nginx/html/index.html",
        injectAliasSource: template.injectAliasSource || "",
        vulnerabilityPackageTypes: [...(template.vulnerabilityPackageTypes ?? [])],
        vulnerabilitySeverities: [...(template.vulnerabilitySeverities ?? [])],
        extraFields: template.extraFields.map((field) => ({ ...field })),
        detailGroups: (template.detailGroups ?? []).map((group) => ({
          ...group,
          fields: (group.fields ?? []).map((field) => ({ ...field })),
        })),
        detailFields: [],
      }
    : createContainerCardTemplate(),
);

if (!draft.detailGroups?.length) {
  draft.detailGroups = [createContainerCardDetailGroup()];
}

const detailReportFields = computed(() =>
  (draft.detailGroups ?? []).flatMap((group) => group.fields.filter((field) => field.kind === "report")),
);

const fieldOptions = computed(() => [
  ...containerCardFieldOptions().map((item) => ({ label: item.label, value: item.value })),
  { label: t("card-templates.field-option-label"), value: "label:" as const },
]);
const detailKindOptions = computed(() => detailFieldKindOptions.map((item) => ({ label: t(item.labelKey), value: item.value })));
const reportOptions = computed(() =>
  containerCardReports.value.filter((report) => report.enabled).map((report) => ({ label: report.name, value: report.id })),
);
const actionOptions = containerCardActionOptions;
const severityOptions = vulnerabilitySeverityOptions;

const runningPreviewContainers = computed(() =>
  containers.value.filter((container) => container.state === "running").slice().sort((a, b) => a.name.localeCompare(b.name)),
);
const matchedPreviewContainers = computed(() =>
  runningPreviewContainers.value.filter((container) => matchesContainerCardTemplate(container, draft.filter)),
);
const previewContainers = computed(() => matchedPreviewContainers.value.slice(0, 1));
const matchedPreviewCount = computed(() => matchedPreviewContainers.value.length);
const canSave = computed(() => !isSaving.value && draft.name.trim().length > 0);

function addField() {
  draft.extraFields = [...draft.extraFields, createContainerCardField()];
}

function removeField(id: string) {
  draft.extraFields = draft.extraFields.filter((field) => field.id !== id);
}

function addDetailGroup() {
  draft.detailGroups = [...(draft.detailGroups ?? []), createContainerCardDetailGroup()];
}

function removeDetailGroup(id: string) {
  if ((draft.detailGroups?.length ?? 0) <= 1) return;
  draft.detailGroups = (draft.detailGroups ?? []).filter((group) => group.id !== id);
}

function moveDetailGroup(index: number, delta: number) {
  const nextIndex = index + delta;
  const groups = [...(draft.detailGroups ?? [])];
  if (nextIndex < 0 || nextIndex >= groups.length) return;
  const [group] = groups.splice(index, 1);
  groups.splice(nextIndex, 0, group);
  draft.detailGroups = groups;
}

function addDetailField(groupID: string) {
  draft.detailGroups = (draft.detailGroups ?? []).map((group) =>
    group.id === groupID ? { ...group, fields: [...group.fields, createDetailTextField()] } : group,
  );
}

function nonReportFields(group: { fields: ContainerCardField[] }) {
  return group.fields.filter((field) => field.kind !== "report");
}

function setDetailReportFields(fields: ContainerCardField[]) {
  const groups = [...(draft.detailGroups ?? [])];
  if (!groups.length) {
    draft.detailGroups = [createContainerCardDetailGroup()];
    return setDetailReportFields(fields);
  }

  const nextGroups = groups.map((group, index) => {
    const preservedFields = group.fields.filter((field) => field.kind !== "report");
    if (index === 0) {
      return { ...group, fields: [...preservedFields, ...fields] };
    }
    return { ...group, fields: preservedFields };
  });

  draft.detailGroups = nextGroups;
}

function addDetailReportField() {
  setDetailReportFields([
    ...detailReportFields.value,
    {
      ...createDetailTextField(),
      kind: "report",
      source: "image",
      reportId: "",
      label: "",
    },
  ]);
}

function removeDetailReportField(fieldID: string) {
  setDetailReportFields(detailReportFields.value.filter((field) => field.id !== fieldID));
}

function removeDetailField(groupID: string, fieldID: string) {
  draft.detailGroups = (draft.detailGroups ?? []).map((group) =>
    group.id === groupID ? { ...group, fields: group.fields.filter((field) => field.id !== fieldID) } : group,
  );
}

function moveField(index: number, delta: number) {
  const nextIndex = index + delta;
  if (nextIndex < 0 || nextIndex >= draft.extraFields.length) return;
  const copy = [...draft.extraFields];
  const [field] = copy.splice(index, 1);
  copy.splice(nextIndex, 0, field);
  draft.extraFields = copy;
}

function moveDetailField(groupID: string, fieldID: string, delta: number) {
  draft.detailGroups = (draft.detailGroups ?? []).map((group) => {
    if (group.id !== groupID) return group;
    const fields = [...group.fields];
    const visibleFields = fields.filter((field) => field.kind !== "report");
    const index = visibleFields.findIndex((field) => field.id === fieldID);
    const nextIndex = index + delta;
    if (index < 0 || nextIndex < 0 || nextIndex >= visibleFields.length) return group;
    const [field] = visibleFields.splice(index, 1);
    visibleFields.splice(nextIndex, 0, field);
    const reportFields = fields.filter((field) => field.kind === "report");
    return { ...group, fields: [...visibleFields, ...reportFields] };
  });
}

function moveDetailReportField(fieldID: string, delta: number) {
  const reports = [...detailReportFields.value];
  const index = reports.findIndex((field) => field.id === fieldID);
  const nextIndex = index + delta;
  if (index < 0 || nextIndex < 0 || nextIndex >= reports.length) return;
  const [field] = reports.splice(index, 1);
  reports.splice(nextIndex, 0, field);
  setDetailReportFields(reports);
}

function labelKeyValue(source: ContainerCardField["source"]) {
  return String(source).startsWith("label:") ? String(source).slice("label:".length) : "";
}

function updateLabelSource(field: ContainerCardField, value: string) {
  field.source = `label:${value}` as ContainerCardField["source"];
}

function toggleAction(action: ContainerCardTemplate["actions"][number]) {
  if (draft.actions.includes(action)) {
    draft.actions = draft.actions.filter((item) => item !== action);
  } else {
    draft.actions = [...draft.actions, action];
  }
}

function splitCsv(value: string) {
  return value.split(",").map((item) => item.trim()).filter(Boolean);
}

function toggleVulnerabilitySeverity(value: string) {
  const current = draft.vulnerabilitySeverities ?? [];
  if (current.includes(value)) {
    draft.vulnerabilitySeverities = current.filter((item) => item !== value);
  } else {
    draft.vulnerabilitySeverities = [...current, value];
  }
}

async function saveTemplate() {
  if (!canSave.value) return;
  isSaving.value = true;
  error.value = undefined;

  try {
    const next = containerCardTemplates.value.map((item) => ({
      ...item,
      extraFields: item.extraFields.map((field) => ({ ...field })),
      detailGroups: (item.detailGroups ?? []).map((group) => ({ ...group, fields: group.fields.map((field) => ({ ...field })) })),
    }));

    const sanitized = {
      ...draft,
      name: draft.name.trim(),
      filter: draft.filter.trim(),
      showDetails: draft.showDetails,
      extraFields: draft.extraFields.map((field) => ({
        ...field,
        kind: "text",
        label: field.label.trim() || displayContainerCardFieldLabel(field),
      })),
      detailGroups: (draft.detailGroups ?? []).map((group) => ({
        ...group,
        name: group.name.trim() || t("card-templates.default-detail-group"),
        fields: group.fields.map((field) => ({
          ...field,
          label: field.label.trim() || (field.kind === "report" ? t("card-templates.open-report") : displayContainerCardFieldLabel(field)),
          kind: field.kind || "text",
          source: field.kind === "text" ? field.source : "image",
          textValue: field.kind === "template_text" || field.kind === "static_text" ? String(field.textValue || "").trim() : "",
          linkUrl: field.kind === "link" ? String(field.linkUrl || "").trim() : "",
          linkText: field.kind === "link" ? String(field.linkText || "").trim() : "",
          reportId: field.kind === "report" ? String(field.reportId || "").trim() : "",
        })),
      })),
      detailFields: [],
      actions: draft.showActions ? draft.actions : [],
      injectIndexPath: draft.showActions ? draft.injectIndexPath.trim() : "",
      injectAliasSource: draft.showActions ? draft.injectAliasSource.trim() : "",
      showVulnerabilities: draft.showVulnerabilities,
      vulnerabilityPackageTypes: draft.showVulnerabilities ? splitCsv((draft.vulnerabilityPackageTypes ?? []).join(",")) : [],
      vulnerabilitySeverities: draft.showVulnerabilities ? [...(draft.vulnerabilitySeverities ?? [])] : [],
    };

    const existingIndex = next.findIndex((item) => item.id === sanitized.id);
    if (existingIndex >= 0) next.splice(existingIndex, 1, sanitized);
    else next.push(sanitized);

    containerCardTemplates.value = next;
    await saveContainerCardTemplates();
    onCreated?.();
    close?.();
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    isSaving.value = false;
  }
}
</script>
