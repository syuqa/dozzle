const { hostname } = config;
let subtitle = $ref("");
const appName = config.appName?.trim() || "Dozzle";
const title = $computed(() => (subtitle ? `${subtitle} - ` : "") + appName + (hostname ? ` @ ${hostname}` : ""));

useTitle($$(title));

export function setTitle(t: string) {
  subtitle = t;
}
