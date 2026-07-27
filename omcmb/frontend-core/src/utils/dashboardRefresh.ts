type DashboardSummaryRefetch = (
  options: { throwOnError: true }
) => Promise<unknown>;

export async function runDashboardSummaryRefresh(
  refetch: DashboardSummaryRefetch,
): Promise<void> {
  await refetch({ throwOnError: true });
}
