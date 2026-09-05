import type { Assistant, AssistantDefinition, AssistantRun } from '@core/types/assistant';
import { assistantEnUS } from '@core/i18n/assistantMessages';
export type Translate = (id: string, values?: Record<string, string | number>) => string;
export const isRunning = (run: AssistantRun) => ['QUEUED', 'RUNNING', 'CANCELLING'].includes(run.status);
export function trialFor(assistant: Assistant, runs: AssistantRun[]): AssistantRun | undefined {
  return runs.find((run) => run.kind === 'trial' && run.revision === assistant.revision);
}
export function canPublish(assistant: Assistant, runs: AssistantRun[]): boolean {
  const trial = trialFor(assistant, runs);
  return assistant.readiness === 'ready' && !!assistant.definition && trial?.status === 'COMPLETED'
    && !!trial.output && trial.output.outcome !== 'insufficient_data'
    && assistant.publishedRevision !== assistant.revision && !runs.some(isRunning);
}
export function triggerText(definition: AssistantDefinition | null | undefined, t: Translate): string {
  if (!definition) return t('assistant.notScheduled');
  const trigger = definition.trigger;
  if (trigger.kind === 'interval') return t('assistant.trigger.interval', { minutes: trigger.intervalMinutes ?? 5 });
  if (trigger.kind === 'schedule') return t('assistant.trigger.schedule', {
    days: (trigger.weekdays ?? []).map((day) => t(`assistant.days.${day}`)).join(' / '),
    time: trigger.time ?? '', timezone: trigger.timezone ?? '',
  });
  if (trigger.kind === 'event') {
    const key = `assistant.event.${trigger.eventType}`;
    return assistantEnUS[key] ? t(key) : t('assistant.trigger.event');
  }
  return t('assistant.trigger.manual');
}
export function errorText(error: unknown, t: Translate): string {
  const text = typeof error === 'string' ? error : error instanceof Error ? error.message : '';
  const code = text.match(/ASSISTANT_[A-Z_]+/)?.[0];
  const key = `assistant.error.${code}`;
  return t(assistantEnUS[key] ? key : 'assistant.error.generic');
}
export function dateText(value: string | null | undefined, locale: string): string {
  if (!value) return '—';
  const date = new Date(value);
  return Number.isNaN(date.valueOf()) ? '—' : new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'short' }).format(date);
}
