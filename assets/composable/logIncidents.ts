import type { InjectionKey, Ref } from "vue";
import type { LogEntry, LogMessage } from "@/models/LogEntry";
import type { LogIncidentMatch, LogIncidentState } from "@/types/logIncidents";
import type { Container } from "@/models/Container";

export const logIncidentContext = Symbol("logIncident") as InjectionKey<(entry: LogEntry<LogMessage>) => LogIncidentState | undefined>;

export function provideLogIncidents(messages: Ref<LogEntry<LogMessage>[]>, containersById: Ref<Record<string, Container>>) {
  const states = ref<Record<number, LogIncidentState>>({});
  const cache = new Map<string, LogIncidentMatch | null>();
  const pending = new Set<string>();
  const waiters = new Map<string, number[]>();

  async function fetchMatch(container: Container, entry: LogEntry<LogMessage>) {
    const raw = entry.rawMessage?.trim();
    if (!raw) {
      states.value[entry.id] = { status: "skipped", match: null };
      return;
    }

    const signature = `${container.host}:${container.id}:${raw}`;
    if (cache.has(signature)) {
      const cached = cache.get(signature) ?? null;
      states.value[entry.id] = { status: cached ? "matched" : "none", match: cached };
      return;
    }
    if (pending.has(signature)) {
      waiters.set(signature, [...(waiters.get(signature) ?? []), entry.id]);
      states.value[entry.id] = { status: "checking" };
      return;
    }

    pending.add(signature);
    waiters.set(signature, [...(waiters.get(signature) ?? []), entry.id]);
    states.value[entry.id] = { status: "checking" };
    try {
      const response = await fetch(withBase(`/api/hosts/${container.host}/containers/${container.id}/logs/match-incident`), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ logMsg: raw }),
      });
      if (!response.ok) {
        cache.set(signature, null);
        for (const id of waiters.get(signature) ?? []) {
          states.value[id] = { status: "none", match: null };
        }
        return;
      }

      const match: LogIncidentMatch = await response.json();
      const resolved = match.found ? match : null;
      cache.set(signature, resolved);
      for (const id of waiters.get(signature) ?? []) {
        states.value[id] = { status: resolved ? "matched" : "none", match: resolved };
      }
    } catch {
      cache.set(signature, null);
      for (const id of waiters.get(signature) ?? []) {
        states.value[id] = { status: "none", match: null };
      }
    } finally {
      pending.delete(signature);
      waiters.delete(signature);
    }
  }

  watchEffect(() => {
    if (config.enableLogIncidentMatch !== true) return;

    const nextContainers = containersById.value;
    for (const entry of messages.value) {
      if (states.value[entry.id] !== undefined) continue;
      const container = nextContainers[entry.containerID];
      if (!container) continue;
      void fetchMatch(container, entry);
    }
  });

  provide(logIncidentContext, (entry) => states.value[entry.id]);
}

export const useLogIncident = () =>
  inject(logIncidentContext, () => undefined);
