import http from '../http';
import type { Assistant, AssistantCatalog, AssistantDefinition, AssistantRun } from '../../types/assistant';
const root = '/agent/assistants';
const at = (id: string) => `${root}/${encodeURIComponent(id)}`;
export const assistantApi = {
  async list(signal?: AbortSignal) { return (await http.get<Assistant[]>(root, { signal })).data; },
  async catalog(signal?: AbortSignal) { return (await http.get<AssistantCatalog>(`${root}/catalog`, { signal })).data; },
  async get(id: string, signal?: AbortSignal) { return (await http.get<Assistant>(at(id), { signal })).data; },
  async create(id: string, locale: string, timezone: string) { return (await http.post<Assistant>(root, { id, locale, timezone })).data; },
  async plan(id: string, revision: number, message: string) { return (await http.post<Assistant>(`${at(id)}/messages`, { revision, message }, { timeout: 150000 })).data; },
  async save(id: string, revision: number, definition: AssistantDefinition) { return (await http.put<Assistant>(`${at(id)}/draft`, { revision, definition })).data; },
  async publish(id: string, revision: number) { return (await http.post<Assistant>(`${at(id)}/publish`, { revision })).data; },
  async state(id: string, state: 'active' | 'paused') { return (await http.patch<Assistant>(`${at(id)}/state`, { state })).data; },
  async runs(id: string, signal?: AbortSignal) { return (await http.get<AssistantRun[]>(`${at(id)}/runs`, { signal })).data; },
  async start(id: string, revision: number, kind: 'trial' | 'manual', requestId: string) { return (await http.post<AssistantRun>(`${at(id)}/runs`, { revision, kind, requestId })).data; },
  async cancel(id: string) { return (await http.post<AssistantRun>(`${root}/runs/${encodeURIComponent(id)}/cancel`)).data; },
  async read(id: string) { await http.post(`${root}/runs/${encodeURIComponent(id)}/read`); },
};
