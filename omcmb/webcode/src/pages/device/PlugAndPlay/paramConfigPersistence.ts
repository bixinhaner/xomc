import type {
  PlugAndPlayPolicy,
  SavePlugAndPlayPolicyRequest,
} from '@core/services/api/provisionApi';

export function buildParamConfigListPolicyUpdate(
  policy: PlugAndPlayPolicy,
  paramConfigList: readonly unknown[],
): SavePlugAndPlayPolicyRequest {
  const { id, createdAt, updatedAt, ...persisted } = policy;
  void id;
  void createdAt;
  void updatedAt;
  return {
    ...persisted,
    config: {
      ...policy.config,
      paramConfigList: [...paramConfigList],
    },
  };
}
