export async function mapWithConcurrencyLimit<TItem, TResult>(
  items: readonly TItem[],
  limit: number,
  worker: (item: TItem, index: number) => Promise<TResult>,
): Promise<Array<PromiseSettledResult<TResult>>> {
  const concurrency = Math.max(1, limit);
  const results: Array<PromiseSettledResult<TResult>> = new Array(items.length);
  let nextIndex = 0;

  async function runWorker(): Promise<void> {
    while (true) {
      const currentIndex = nextIndex;
      nextIndex += 1;
      if (currentIndex >= items.length) {
        return;
      }

      try {
        const value = await worker(items[currentIndex], currentIndex);
        results[currentIndex] = { status: 'fulfilled', value };
      } catch (error) {
        results[currentIndex] = { status: 'rejected', reason: error };
      }
    }
  }

  const runners = Array.from(
    { length: Math.min(concurrency, items.length) },
    () => runWorker(),
  );
  await Promise.all(runners);
  return results;
}