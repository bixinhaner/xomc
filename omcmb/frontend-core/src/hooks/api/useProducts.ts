import { useMemo } from 'react';
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
      isActive,
    }: {
      productId: string;
      patternId: string;
      productClass?: string;
      isActive?: boolean;
    }) => api.updatePattern(productId, patternId, { productClass, isActive }),
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

/** 2026-05-28: server-side 分页 + SN 模糊搜索。
 *  staleTime=0 + refetchOnMount 让每次进入页面都拉最新数据(用户决策:取消"刷新"
 *  按钮,实时性靠 hook 自身保证)。
 */
export function useOrphanDevices(params: {
  page?: number;
  pageSize?: number;
  search?: string;
} = {}) {
  return useQuery({
    queryKey: [
      ...PRODUCTS_KEY,
      'orphan-devices',
      params.page ?? 1,
      params.pageSize ?? 50,
      params.search ?? '',
    ],
    queryFn: () => api.listOrphan(params),
    staleTime: 0,
    refetchOnMount: 'always',
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

// #602：产品名称反解析。后端表里 device.product_class 是上报字面值（如 `FAP/BSQ7258L254`），
// product.patterns[] 是**正则字符串**（如 `^FAP/BSQ7258L254$`）。文件管理列表把
// product_class 渲染为「产品名称」时必须用 RegExp.test 匹配，不能 Map.get 字面比对。
// 用法：
//   const resolveProductName = useProductNameResolver()
//   <span>{resolveProductName(row.productClass) || '—'}</span>
export function useProductNameResolver(): (raw: string | null | undefined) => string {
  const { data } = useProductList();
  return useMemo(() => {
    const compiled = (data?.items ?? []).map((p) => ({
      name: p.name,
      regexes: (p.patterns ?? [])
        .map((pat) => {
          try {
            return new RegExp(pat);
          } catch {
            return null;
          }
        })
        .filter((r): r is RegExp => r !== null),
    }));
    return (raw: string | null | undefined): string => {
      const v = (raw ?? '').trim();
      if (!v) return '';
      for (const { name, regexes } of compiled) {
        for (const re of regexes) {
          if (re.test(v)) return name;
        }
      }
      return v; // 无匹配回退原始 product_class，避免空白
    };
  }, [data]);
}

