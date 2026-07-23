export interface WaitForExpectedParameterValuesOptions {
  expected: ReadonlyMap<string, string>;
  read: (signal: AbortSignal) => Promise<ReadonlyMap<string, string>>;
  intervalMs?: number;
  timeoutMs?: number;
  signal?: AbortSignal;
}

export interface SubmittedReadbackState {
  taskId?: string;
  syncedForTaskId?: string;
  submittedDraftRevision?: number;
  currentDraftRevision: number;
}

export function canApplySubmittedReadback({
  taskId,
  syncedForTaskId,
  submittedDraftRevision,
  currentDraftRevision,
}: SubmittedReadbackState): boolean {
  return Boolean(
    taskId
    && syncedForTaskId !== taskId
    && submittedDraftRevision !== undefined
    && submittedDraftRevision === currentDraftRevision,
  );
}

export class ParameterReadbackTimeoutError extends Error {
  constructor() {
    super('parameter readback timed out before submitted values were observed');
    this.name = 'ParameterReadbackTimeoutError';
  }
}

function createAbortError(): Error {
  if (typeof DOMException !== 'undefined') {
    return new DOMException('The operation was aborted.', 'AbortError');
  }
  const error = new Error('The operation was aborted.');
  error.name = 'AbortError';
  return error;
}

function throwIfAborted(signal?: AbortSignal): void {
  if (signal?.aborted) {
    throw createAbortError();
  }
}

async function delayWithSignal(delayMs: number, signal?: AbortSignal): Promise<void> {
  throwIfAborted(signal);
  if (!signal) {
    await new Promise<void>((resolve) => {
      globalThis.setTimeout(resolve, delayMs);
    });
    return;
  }

  await new Promise<void>((resolve, reject) => {
    const timerId = globalThis.setTimeout(() => {
      signal.removeEventListener('abort', onAbort);
      resolve();
    }, delayMs);
    const onAbort = () => {
      globalThis.clearTimeout(timerId);
      signal.removeEventListener('abort', onAbort);
      reject(createAbortError());
    };
    signal.addEventListener('abort', onAbort, { once: true });
    if (signal.aborted) {
      onAbort();
    }
  });
}

async function readWithDeadline(
  read: (signal: AbortSignal) => Promise<ReadonlyMap<string, string>>,
  remainingMs: number,
  signal?: AbortSignal,
): Promise<ReadonlyMap<string, string>> {
  throwIfAborted(signal);
  const readController = new AbortController();

  return new Promise<ReadonlyMap<string, string>>((resolve, reject) => {
    let settled = false;
    const finish = (
      callback: (value: ReadonlyMap<string, string> | Error) => void,
      value: ReadonlyMap<string, string> | Error,
    ) => {
      if (settled) return;
      settled = true;
      globalThis.clearTimeout(timeoutId);
      signal?.removeEventListener('abort', onAbort);
      callback(value);
    };
    const onAbort = () => {
      readController.abort();
      finish((error) => reject(error), createAbortError());
    };
    const timeoutId = globalThis.setTimeout(() => {
      readController.abort();
      finish((error) => reject(error), new ParameterReadbackTimeoutError());
    }, Math.max(0, remainingMs));

    signal?.addEventListener('abort', onAbort, { once: true });
    if (signal?.aborted) {
      onAbort();
      return;
    }
    void Promise.resolve()
      .then(() => read(readController.signal))
      .then(
        (value) => finish((result) => resolve(result as ReadonlyMap<string, string>), value),
        (error: unknown) => finish(
          (reason) => reject(reason),
          error instanceof Error ? error : new Error(String(error)),
        ),
      );
  });
}

function parameterValuesMatch(
  expected: ReadonlyMap<string, string>,
  actual: ReadonlyMap<string, string>,
): boolean {
  for (const [path, expectedValue] of expected) {
    if (actual.get(path) !== expectedValue) {
      return false;
    }
  }
  return true;
}

export async function waitForExpectedParameterValues({
  expected,
  read,
  intervalMs = 500,
  timeoutMs = 30_000,
  signal,
}: WaitForExpectedParameterValuesOptions): Promise<ReadonlyMap<string, string>> {
  const deadlineAt = Date.now() + timeoutMs;
  while (true) {
    throwIfAborted(signal);
    const remainingBeforeRead = deadlineAt - Date.now();
    if (remainingBeforeRead <= 0) {
      throw new ParameterReadbackTimeoutError();
    }
    const actual = await readWithDeadline(read, remainingBeforeRead, signal);
    throwIfAborted(signal);
    if (parameterValuesMatch(expected, actual)) {
      return actual;
    }
    const remainingBeforeDelay = deadlineAt - Date.now();
    if (remainingBeforeDelay <= 0) {
      throw new ParameterReadbackTimeoutError();
    }
    await delayWithSignal(Math.min(intervalMs, remainingBeforeDelay), signal);
  }
}
