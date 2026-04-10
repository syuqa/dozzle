import { Container } from "@/models/Container";
import { i18n } from "@/modules/i18n";
import { withBase } from "@/stores/config";

export type ContainerCardFieldSource =
  | "image"
  | "command"
  | "group"
  | "namespace"
  | "health"
  | "id"
  | "host"
  | "state"
  | "created"
  | `label:${string}`;

export type ContainerCardField = {
  id: string;
  label: string;
  kind?: "text" | "template_text" | "static_text" | "link" | "report";
  source: ContainerCardFieldSource;
  textValue?: string;
  linkUrl?: string;
  linkText?: string;
  reportId?: string;
};

export type ContainerCardDetailGroup = {
  id: string;
  name: string;
  fields: ContainerCardField[];
};

export type ContainerCardAction = "start" | "stop" | "restart" | "update" | "inject-logs-button";

export type ContainerCardTemplate = {
  id: string;
  name: string;
  iconUrl?: string;
  enabled: boolean;
  filter: string;
  showHost: boolean;
  showState: boolean;
  showCreated: boolean;
  showDetails: boolean;
  showActions: boolean;
  showVulnerabilities?: boolean;
  actions: ContainerCardAction[];
  injectIndexPath: string;
  injectAliasSource: string;
  vulnerabilityPackageTypes?: string[];
  vulnerabilitySeverities?: string[];
  extraFields: ContainerCardField[];
  detailGroups?: ContainerCardDetailGroup[];
  detailFields: ContainerCardField[];
};

function createId(prefix: string) {
  return `${prefix}-${Math.random().toString(36).slice(2, 10)}`;
}

function t(key: string) {
  return i18n.global.t(key);
}

export function createDefaultContainerCardTemplate(): ContainerCardTemplate {
  return {
    id: "default",
    name: t("card-templates.default-name"),
    iconUrl: "",
    enabled: true,
    filter: "",
    showHost: true,
    showState: true,
    showCreated: true,
    showDetails: true,
    showActions: false,
    showVulnerabilities: false,
    actions: ["restart", "stop", "start", "update"],
    injectIndexPath: "/usr/share/nginx/html/index.html",
    injectAliasSource: "",
    vulnerabilityPackageTypes: [],
    vulnerabilitySeverities: [],
    extraFields: [
      { id: createId("field"), label: t("card-templates.field-option-image"), source: "image" },
      { id: createId("field"), label: t("card-templates.field-option-namespace"), source: "namespace" },
    ],
    detailGroups: [
      {
        id: createId("group"),
        name: t("card-templates.default-detail-group"),
        fields: [
          { id: createId("detail"), label: t("card-templates.field-option-image"), source: "image" },
          { id: createId("detail"), label: t("card-templates.field-option-command"), source: "command" },
          { id: createId("detail"), label: t("card-templates.field-option-host"), source: "host" },
          { id: createId("detail"), label: t("card-templates.field-option-state"), source: "state" },
          { id: createId("detail"), label: t("card-templates.field-option-created"), source: "created" },
        ],
      },
    ],
    detailFields: [],
  };
}

const defaultTemplates = [createDefaultContainerCardTemplate()];

export const containerCardTemplates = ref<ContainerCardTemplate[]>([...defaultTemplates]);
export const cardTemplatesLoaded = ref(false);
export const cardTemplatesSaving = ref(false);
export const cardTemplatesError = ref<string>();
export const cardTemplatesSyncedSnapshot = ref("");

let loadPromise: Promise<void> | null = null;
let saveRevision = 0;
let loadedOnce = false;

function cloneTemplates(source: ContainerCardTemplate[]) {
  return source.map((template) => ({
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
    detailGroups: (
      template.detailGroups?.length
        ? template.detailGroups
        : [
            {
              id: createId("group"),
              name: t("card-templates.default-detail-group"),
              fields: template.detailFields ?? [],
            },
          ]
    ).map((group) => ({
      ...group,
      fields: (group.fields ?? []).map((field) => ({ kind: "text", ...field })),
    })),
    detailFields: (template.detailFields ?? []).map((field) => ({ kind: "text", ...field })),
  }));
}

export function serializeContainerCardTemplates(source = containerCardTemplates.value) {
  return JSON.stringify(source);
}

export async function loadContainerCardTemplates(force = false) {
  if (loadPromise && !force) {
    return loadPromise;
  }

  loadPromise = (async () => {
    cardTemplatesError.value = undefined;

    try {
      const response = await fetch(withBase("/api/card-templates"));
      if (!response.ok) {
        throw new Error(`failed to load card templates (${response.status})`);
      }
      const templates = (await response.json()) as ContainerCardTemplate[];
      containerCardTemplates.value = templates.length ? cloneTemplates(templates) : cloneTemplates(defaultTemplates);
      cardTemplatesSyncedSnapshot.value = serializeContainerCardTemplates(containerCardTemplates.value);
      loadedOnce = true;
    } catch (error) {
      cardTemplatesError.value = error instanceof Error ? error.message : String(error);
      if (!loadedOnce) {
        containerCardTemplates.value = cloneTemplates(defaultTemplates);
        cardTemplatesSyncedSnapshot.value = serializeContainerCardTemplates(containerCardTemplates.value);
      }
    } finally {
      cardTemplatesLoaded.value = true;
      loadPromise = null;
    }
  })();

  return loadPromise;
}

export async function saveContainerCardTemplates() {
  const templates = cloneTemplates(containerCardTemplates.value);
  const currentRevision = ++saveRevision;

  cardTemplatesSaving.value = true;
  cardTemplatesError.value = undefined;

  try {
    const response = await fetch(withBase("/api/card-templates"), {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(templates),
    });
    if (!response.ok) {
      throw new Error(await response.text());
    }

    const saved = (await response.json()) as ContainerCardTemplate[];
    if (currentRevision === saveRevision) {
      containerCardTemplates.value = saved.length ? cloneTemplates(saved) : cloneTemplates(defaultTemplates);
      cardTemplatesSyncedSnapshot.value = serializeContainerCardTemplates(containerCardTemplates.value);
    }
  } catch (error) {
    cardTemplatesError.value = error instanceof Error ? error.message : String(error);
    throw error;
  } finally {
    if (currentRevision === saveRevision) {
      cardTemplatesSaving.value = false;
    }
  }
}

export function useContainerCardTemplates() {
  onMounted(() => {
    void loadContainerCardTemplates();
  });

  return {
    templates: containerCardTemplates,
    loaded: cardTemplatesLoaded,
    saving: cardTemplatesSaving,
    error: cardTemplatesError,
    load: loadContainerCardTemplates,
    save: saveContainerCardTemplates,
  };
}

function stripQuotes(value: string) {
  if (
    (value.startsWith(`"`) && value.endsWith(`"`)) ||
    (value.startsWith(`'`) && value.endsWith(`'`))
  ) {
    return value.slice(1, -1);
  }
  return value;
}

function stringValue(container: Container, field: string): string {
  switch (field) {
    case "name":
      return container.name;
    case "image":
      return container.image;
    case "command":
      return container.command;
    case "host":
      return container.hostLabel ?? container.host;
    case "state":
      return container.state;
    case "group":
      return container.group ?? "";
    case "namespace":
      return container.namespace ?? "";
    case "health":
      return container.health ?? "";
    case "id":
      return container.id;
    default:
      if (field.startsWith("label:")) {
        return container.labels[field.slice("label:".length)] ?? "";
      }
      return "";
  }
}

function evaluateCondition(container: Container, rawCondition: string): boolean {
  const condition = rawCondition.trim();
  const match = condition.match(/^(name|image|command|host|state|group|namespace|health|id|label:[^\s]+)\s*(==|contains)\s*(.+)$/i);
  if (!match) return false;

  const [, field, operator, rawValue] = match;
  const left = stringValue(container, field);
  const right = stripQuotes(rawValue.trim());

  if (operator === "==") {
    return left === right;
  }
  return left.toLowerCase().includes(right.toLowerCase());
}

export function matchesContainerCardTemplate(container: Container, filter: string): boolean {
  if (!filter.trim()) return true;
  return filter
    .split("&&")
    .map((part) => part.trim())
    .filter(Boolean)
    .every((condition) => evaluateCondition(container, condition));
}

export function resolveContainerCardTemplate(container: Container): ContainerCardTemplate {
  return (
    containerCardTemplates.value.find((template) => template.enabled && matchesContainerCardTemplate(container, template.filter)) ??
    containerCardTemplates.value[0] ??
    createDefaultContainerCardTemplate()
  );
}

export function resolveContainerCardField(container: Container, source: ContainerCardFieldSource): string {
  if (source === "created") {
    return container.created.toLocaleString();
  }
  if (source === "state") {
    return containerCardStateLabel(container.state);
  }
  return stringValue(container, source);
}

export function resolveContainerTemplateString(container: Container, template: string): string {
  return String(template || "").replace(/\{([^}]+)\}/g, (_, rawKey: string) => {
    const key = rawKey.trim();
    if (!key) return "";
    return resolveContainerCardTextSource(container, key);
  });
}

export function resolveContainerCardTextSource(container: Container, source: string): string {
  const normalized = String(source || "").trim();
  if (!normalized) return "";
  if (normalized === "name") return container.name;
  if (normalized === "created") return container.created.toLocaleString();
  return stringValue(container, normalized);
}

export function containerCardStateLabel(state: string) {
  const key = `card-templates.state-${state}`;
  const translated = t(key);
  return translated === key ? state : translated;
}

export function containerCardFieldLabel(source: ContainerCardFieldSource) {
  switch (source) {
    case "image":
      return t("card-templates.field-option-image");
    case "command":
      return t("card-templates.field-option-command");
    case "group":
      return t("card-templates.field-option-group");
    case "namespace":
      return t("card-templates.field-option-namespace");
    case "health":
      return t("card-templates.field-option-health");
    case "id":
      return t("card-templates.field-option-id");
    case "host":
      return t("card-templates.field-option-host");
    case "state":
      return t("card-templates.field-option-state");
    case "created":
      return t("card-templates.field-option-created");
    default:
      if (source.startsWith("label:")) {
        return t("card-templates.field-option-label");
      }
      return source;
  }
}

export function displayContainerCardFieldLabel(field: ContainerCardField) {
  const builtInSource = field.source.replace(/^label:.+$/, "label:") as ContainerCardFieldSource;
  const translated = containerCardFieldLabel(builtInSource);
  if (field.label === translated) return translated;

  const fallbackLabels = new Map<ContainerCardFieldSource, string>([
    ["image", "Image"],
    ["command", "Command"],
    ["group", "Group"],
    ["namespace", "Namespace"],
    ["health", "Health"],
    ["id", "Container ID"],
    ["host", "Host"],
    ["state", "State"],
    ["created", "Created"],
    ["label:", "Container Label"],
  ]);

  if (field.label === fallbackLabels.get(builtInSource)) {
    return translated;
  }

  return field.label;
}

export function containerCardFieldOptions() {
  return [
    { label: t("card-templates.field-option-image"), value: "image" as const },
    { label: t("card-templates.field-option-command"), value: "command" as const },
    { label: t("card-templates.field-option-group"), value: "group" as const },
    { label: t("card-templates.field-option-namespace"), value: "namespace" as const },
    { label: t("card-templates.field-option-health"), value: "health" as const },
    { label: t("card-templates.field-option-id"), value: "id" as const },
    { label: t("card-templates.field-option-host"), value: "host" as const },
    { label: t("card-templates.field-option-state"), value: "state" as const },
    { label: t("card-templates.field-option-created"), value: "created" as const },
  ];
}

export function createContainerCardField(): ContainerCardField {
  return {
    id: createId("field"),
    label: t("card-templates.field-option-image"),
    kind: "text",
    source: "image",
  };
}

export function createContainerCardTemplate(): ContainerCardTemplate {
  return {
    id: createId("template"),
    name: t("card-templates.new-template"),
    iconUrl: "",
    enabled: true,
    filter: "",
    showHost: true,
    showState: true,
    showCreated: true,
    showDetails: true,
    showActions: false,
    showVulnerabilities: false,
    actions: ["restart", "stop", "start", "update"],
    injectIndexPath: "/usr/share/nginx/html/index.html",
    injectAliasSource: "",
    vulnerabilityPackageTypes: [],
    vulnerabilitySeverities: [],
    extraFields: [createContainerCardField()],
    detailGroups: [createContainerCardDetailGroup()],
    detailFields: [],
  };
}

export function createDetailTextField(): ContainerCardField {
  return createContainerCardField();
}

export function createContainerCardDetailGroup(): ContainerCardDetailGroup {
  return {
    id: createId("group"),
    name: t("card-templates.default-detail-group"),
    fields: [
      { id: createId("detail"), label: t("card-templates.field-option-image"), source: "image" },
      { id: createId("detail"), label: t("card-templates.field-option-command"), source: "command" },
      { id: createId("detail"), label: t("card-templates.field-option-host"), source: "host" },
    ],
  };
}

export const detailFieldKindOptions = [
  { labelKey: "card-templates.kind-container-field", value: "text" as const },
  { labelKey: "card-templates.kind-template-text", value: "template_text" as const },
  { labelKey: "card-templates.kind-static-text", value: "static_text" as const },
  { labelKey: "card-templates.kind-link", value: "link" as const },
  { labelKey: "card-templates.kind-report", value: "report" as const },
];

export const vulnerabilitySeverityOptions = ["CRITICAL", "HIGH", "MEDIUM", "LOW", "UNKNOWN"];

export const containerCardActionOptions: { labelKey: string; value: ContainerCardAction }[] = [
  { labelKey: "toolbar.start", value: "start" },
  { labelKey: "toolbar.stop", value: "stop" },
  { labelKey: "toolbar.restart", value: "restart" },
  { labelKey: "toolbar.update", value: "update" },
  { labelKey: "card-templates.inject-logs-button", value: "inject-logs-button" },
];
