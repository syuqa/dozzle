<template>
  <PageWithLinks>
    <section>
      <div class="mb-8">
        <h2 class="text-2xl font-bold">{{ $t("card-templates.title") }}</h2>
        <p class="text-base-content/60 max-w-4xl">{{ $t("card-templates.description") }}</p>
      </div>

      <div v-if="error" class="alert alert-error alert-soft mb-4">
        <span>{{ error }}</span>
      </div>

      <div class="text-base-content/60 mb-4 flex items-center justify-end gap-2 text-sm">
        <span v-if="saving">{{ $t("card-templates.saving") }}</span>
        <span v-else-if="loaded">{{ $t("card-templates.saved") }}</span>
      </div>

      <div class="mb-8">
        <div class="mb-4 flex items-center justify-between">
          <h3 class="text-base-content/60 font-semibold tracking-wide uppercase">{{ $t("card-templates.templates-list") }}</h3>
          <button class="btn btn-primary btn-sm" @click="openCreateTemplate">
            <mdi:plus />
            {{ $t("card-templates.add-template") }}
          </button>
        </div>

        <div v-if="!loaded" class="text-base-content/60 py-4">
          {{ $t("settings.loading-card-templates") }}
        </div>
        <div v-else class="grid gap-4 md:grid-cols-2">
          <CardTemplateCard
            v-for="(template, index) in templates"
            :key="template.id"
            :template="template"
            :index="index"
            :total="templates.length"
            :on-updated="refreshTemplates"
          />

          <button
            class="card card-border border-base-content/30 hover:border-base-content/50 w-full cursor-pointer border-dashed transition-colors"
            @click="openCreateTemplate"
          >
            <div class="card-body items-center justify-center gap-1 p-4">
              <mdi:plus class="text-2xl" />
              <span class="text-base-content/60 text-sm">{{ $t("card-templates.add-template") }}</span>
            </div>
          </button>
        </div>
      </div>

      <div>
        <div class="mb-4 flex items-center justify-between">
          <h3 class="text-base-content/60 font-semibold tracking-wide uppercase">{{ $t("card-templates.report-definitions") }}</h3>
          <button class="btn btn-primary btn-sm" @click="openCreateReport">
            <mdi:plus />
            {{ $t("card-templates.add-report") }}
          </button>
        </div>

        <div v-if="!reportsLoaded" class="text-base-content/60 py-4">
          {{ $t("card-templates.loading-reports") }}
        </div>
        <div v-else class="grid gap-4 md:grid-cols-2">
          <ReportDefinitionCard
            v-for="report in reports"
            :key="report.id"
            :report="report"
            :on-updated="refreshReports"
          />

          <button
            class="card card-border border-base-content/30 hover:border-base-content/50 w-full cursor-pointer border-dashed transition-colors"
            @click="openCreateReport"
          >
            <div class="card-body items-center justify-center gap-1 p-4">
              <mdi:plus class="text-2xl" />
              <span class="text-base-content/60 text-sm">{{ $t("card-templates.add-report") }}</span>
            </div>
          </button>
        </div>
      </div>
    </section>
  </PageWithLinks>
</template>

<script lang="ts" setup>
import CardTemplateCard from "@/components/CardTemplates/CardTemplateCard.vue";
import CardTemplateForm from "@/components/CardTemplates/CardTemplateForm.vue";
import ReportDefinitionCard from "@/components/CardTemplates/ReportDefinitionCard.vue";
import ReportDefinitionForm from "@/components/CardTemplates/ReportDefinitionForm.vue";
import { containerCardReports, loadContainerCardReports, useContainerCardReports } from "@/stores/containerCardReports";
import { containerCardTemplates, loadContainerCardTemplates, useContainerCardTemplates } from "@/stores/containerCardTemplates";

const { t } = useI18n();
setTitle(t("card-templates.title"));

const showDrawer = useDrawer();
const { loaded, saving, error } = useContainerCardTemplates();
const { loaded: reportsLoaded } = useContainerCardReports();
const templates = containerCardTemplates;
const reports = containerCardReports;

onMounted(async () => {
  await loadContainerCardTemplates();
  await loadContainerCardReports();
});

async function refreshTemplates() {
  await loadContainerCardTemplates(true);
}

async function refreshReports() {
  await loadContainerCardReports(true);
}

function openCreateTemplate() {
  showDrawer(CardTemplateForm, { onCreated: refreshTemplates }, "lg");
}

function openCreateReport() {
  showDrawer(ReportDefinitionForm, { onCreated: refreshReports }, "lg");
}
</script>
