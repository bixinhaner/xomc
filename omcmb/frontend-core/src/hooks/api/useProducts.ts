import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { productApi } from '../../services/api/productApi';
import { productService } from '../../mock/services/productService';
import { createApiSwitch } from '../../services/apiSwitch';
import type {
  CreateProductInput,
  UpdateProductInput,
  ProductListFilter,
} from '../../types/product';

const api = createApiSwitch(productService, productApi);

const PRODUCTS_KEY = ['products'] as const;

export function useProductList(filter?: ProductListFilter) {
  return useQuery({
    queryKey: [...PRODUCTS_KEY, 'list', filter ?? {}],
    queryFn: () => api.list(filter),
  });
}

export function useProductDetail(id: string | undefined) {
  return useQuery({
    queryKey: [...PRODUCTS_KEY, 'detail', id],
    queryFn: () => api.get(id as string),
    enabled: Boolean(id),
  });
}

export function useCreateProduct() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateProductInput) => api.create(input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: PRODUCTS_KEY });
    },
  });
}

export function useUpdateProduct() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UpdateProductInput }) => api.update(id, input),
    onSuccess: (_data, vars) => {
      void qc.invalidateQueries({ queryKey: [...PRODUCTS_KEY, 'detail', vars.id] });
      void qc.invalidateQueries({ queryKey: [...PRODUCTS_KEY, 'list'] });
    },
  });
}

export function useDeleteProduct() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.delete(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: PRODUCTS_KEY });
    },
  });
}

export function useResetDiscovered() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.resetDiscovered(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['param-models'] });
      void qc.invalidateQueries({ queryKey: PRODUCTS_KEY });
    },
  });
}

export function useCreatePattern() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ productId, productClass }: { productId: string; productClass: string }) =>
      api.createPattern(productId, productClass),
    onSuccess: (_data, vars) => {
      void qc.invalidateQueries({ queryKey: [...PRODUCTS_KEY, 'detail', vars.productId] });
      void qc.invalidateQueries({ queryKey: [...PRODUCTS_KEY, 'match-order'] });
    },
  });
}

export function useUpdatePattern() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      productId,
      patternId,
      productClass,
    }: {
      productId: string;
      patternId: string;
      productClass: string;
    }) => api.updatePattern(productId, patternId, productClass),
    onSuccess: (_data, vars) => {
      void qc.invalidateQueries({ queryKey: [...PRODUCTS_KEY, 'detail', vars.productId] });
      void qc.invalidateQueries({ queryKey: [...PRODUCTS_KEY, 'match-order'] });
    },
  });
}

export function useDeletePattern() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ productId, patternId }: { productId: string; patternId: string }) =>
      api.deletePattern(productId, patternId),
    onSuccess: (_data, vars) => {
      void qc.invalidateQueries({ queryKey: [...PRODUCTS_KEY, 'detail', vars.productId] });
      void qc.invalidateQueries({ queryKey: [...PRODUCTS_KEY, 'match-order'] });
    },
  });
}

export function useMovePattern() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      productId,
      patternId,
      direction,
    }: {
      productId: string;
      patternId: string;
      direction: 'up' | 'down';
    }) => api.movePattern(productId, patternId, direction),
    onSuccess: (_data, vars) => {
      void qc.invalidateQueries({ queryKey: [...PRODUCTS_KEY, 'detail', vars.productId] });
      void qc.invalidateQueries({ queryKey: [...PRODUCTS_KEY, 'match-order'] });
    },
  });
}

export function useProductMatch(productClass: string) {
  return useQuery({
    queryKey: [...PRODUCTS_KEY, 'match', productClass],
    queryFn: () => api.match(productClass),
    enabled: Boolean(productClass.trim()),
  });
}

export function useMatchOrder() {
  return useQuery({
    queryKey: [...PRODUCTS_KEY, 'match-order'],
    queryFn: () => api.matchOrder(),
  });
}

export function useOrphanDevices(limit = 200) {
  return useQuery({
    queryKey: [...PRODUCTS_KEY, 'orphan-devices', limit],
    queryFn: () => api.listOrphan(limit),
  });
}

export function useRematchOrphan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api.rematchOrphan(),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: [...PRODUCTS_KEY, 'orphan-devices'] });
      void qc.invalidateQueries({ queryKey: [...PRODUCTS_KEY, 'list'] });
    },
  });
}

export function useBindOrphan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ deviceId, productId }: { deviceId: string; productId: string }) =>
      api.bindOrphan(deviceId, productId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: [...PRODUCTS_KEY, 'orphan-devices'] });
      void qc.invalidateQueries({ queryKey: [...PRODUCTS_KEY, 'list'] });
    },
  });
}

export function useProductCacheRefresh() {
  return useMutation({
    mutationFn: () => api.cacheRefresh(),
  });
}

export function useProductImportDirectory() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api.importDirectory(),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: PRODUCTS_KEY });
    },
  });
}

export function useIndicatorPlatforms(deviceType: string | undefined) {
  return useQuery({
    queryKey: [...PRODUCTS_KEY, 'indicator-platforms', deviceType],
    queryFn: () => api.listIndicatorPlatforms(deviceType as string),
    enabled: Boolean(deviceType && deviceType.trim()),
    staleTime: 5 * 60 * 1000,
  });
}

export function useAlarmNeTypes() {
  return useQuery({
    queryKey: [...PRODUCTS_KEY, 'alarm-ne-types'],
    queryFn: () => api.listAlarmNeTypes(),
    staleTime: 5 * 60 * 1000,
  });
}
