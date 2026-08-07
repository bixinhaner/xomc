import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import LogRetentionSection from './LogRetentionSection';
import PmRetentionSection from './PmRetentionSection';
import RetentionBackpressureSection from './RetentionBackpressureSection';

const hookMocks = vi.hoisted(() => ({
  query: vi.fn(),
  mutateAsync: vi.fn(),
  refetch: vi.fn(),
}));

vi.mock('@core/hooks/api/useSystem', () => ({
  useSysConfigsByCategory: (category: string) => hookMocks.query(category),
  useBatchUpdateSysConfigs: () => ({ mutateAsync: hookMocks.mutateAsync }),
  useSysConfigApplyBatch: () => ({ data: undefined }),
}));

vi.mock('@/hooks/useT', () => ({
  useT: () => (key: string) => key,
}));

const failedQuery = {
  data: undefined,
  isLoading: false,
  isFetching: false,
  isError: true,
  isSuccess: false,
  refetch: hookMocks.refetch,
};

describe('self-managed system config load guard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    hookMocks.query.mockReturnValue(failedQuery);
  });

  it('PM 保留策略加载失败时禁止保存和重置', () => {
    render(<PmRetentionSection />);

    expect(screen.getByText('empty.loadFailed')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'pmRetention.save' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'pmRetention.reset' })).toBeDisabled();

    fireEvent.click(screen.getByRole('button', { name: 'common.retry' }));
    expect(hookMocks.refetch).toHaveBeenCalledTimes(1);
    expect(hookMocks.mutateAsync).not.toHaveBeenCalled();
  });

  it('资源保留与背压各分类加载失败时分别禁止保存', () => {
    render(<RetentionBackpressureSection />);

    expect(screen.getAllByText('empty.loadFailed')).toHaveLength(4);
    for (const button of screen.getAllByRole('button', { name: 'retentionBp.save' })) {
      expect(button).toBeDisabled();
    }
    expect(screen.getAllByRole('button', { name: 'common.retry' })).toHaveLength(4);
    expect(hookMocks.mutateAsync).not.toHaveBeenCalled();
  });

  it('日志配置各分类加载失败时分别禁止保存', () => {
    render(<LogRetentionSection />);

    expect(screen.getAllByText('empty.loadFailed')).toHaveLength(2);
    for (const button of screen.getAllByRole('button', { name: 'logCfg.save' })) {
      expect(button).toBeDisabled();
    }
    expect(screen.getAllByRole('button', { name: 'common.retry' })).toHaveLength(2);
    expect(hookMocks.mutateAsync).not.toHaveBeenCalled();
  });

  it('PM 保留策略加载成功后只保存管理员实际修改的字段', async () => {
    const user = userEvent.setup();
    const values: Record<string, string> = {
      raw_15min_days: '30',
      hourly_days: '180',
      daily_days: '730',
      weekly_days: '730',
      monthly_days: '1825',
    };
    hookMocks.query.mockReturnValue({
      data: Object.entries(values).map(([key, value]) => ({
        id: key,
        category: 'pm.retention',
        key,
        value,
        valueType: 'int',
        isPublic: false,
        isSecret: false,
      })),
      isLoading: false,
      isFetching: false,
      isError: false,
      isSuccess: true,
      refetch: hookMocks.refetch,
    });
    hookMocks.mutateAsync.mockResolvedValue({
      batch: { id: 'batch-pm', category: 'pm.retention', status: 'applied', targets: [] },
    });

    render(<PmRetentionSection />);

    const firstInput = screen.getAllByRole('spinbutton')[0];
    await waitFor(() => expect(firstInput).toHaveValue('30'));
    await user.clear(firstInput);
    await user.type(firstInput, '31');
    await user.click(screen.getByRole('button', { name: 'pmRetention.save' }));

    await waitFor(() => {
      expect(hookMocks.mutateAsync).toHaveBeenCalledWith({
        category: 'pm.retention',
        items: [{ key: 'raw_15min_days', value: '31', value_type: 'int' }],
      });
    });
  });

  it('PM 保留策略分类为空时允许编辑，但未修改不提交', async () => {
    const user = userEvent.setup();
    hookMocks.query.mockReturnValue({
      data: [],
      isLoading: false,
      isFetching: false,
      isError: false,
      isSuccess: true,
      refetch: hookMocks.refetch,
    });
    hookMocks.mutateAsync.mockResolvedValue({
      batch: { id: 'batch-pm-empty', category: 'pm.retention', status: 'applied', targets: [] },
    });

    render(<PmRetentionSection />);

    const saveButton = screen.getByRole('button', { name: 'pmRetention.save' });
    expect(saveButton).toBeEnabled();
    await user.click(saveButton);
    expect(hookMocks.mutateAsync).not.toHaveBeenCalled();

    const firstInput = screen.getAllByRole('spinbutton')[0];
    await user.clear(firstInput);
    await user.type(firstInput, '31');
    await user.click(saveButton);
    await waitFor(() => {
      expect(hookMocks.mutateAsync).toHaveBeenCalledWith({
        category: 'pm.retention',
        items: [{ key: 'raw_15min_days', value: '31', value_type: 'int' }],
      });
    });
  });

  it('资源保留与背压只提交当前卡片内实际修改的字段', async () => {
    const user = userEvent.setup();
    const valuesByCategory: Record<string, Record<string, string>> = {
      'acs.backpressure': {
        enabled: 'true',
        disk_high_pct: '90',
        disk_low_pct: '70',
        check_interval_sec: '60',
      },
      'minio.retention': { raw_object_days: '30' },
      'stationlog.retention': {
        max_retention_days: '30',
        cleanup_interval_minutes: '60',
        max_file_count: '0',
        max_file_count_per_device: '0',
      },
      raw_archive: { compress_after_ingest: 'false' },
    };
    hookMocks.query.mockImplementation((category: string) => ({
      data: Object.entries(valuesByCategory[category]).map(([key, value]) => ({
        id: `${category}-${key}`,
        category,
        key,
        value,
        valueType: key === 'enabled' || key === 'compress_after_ingest' ? 'bool' : 'int',
        isPublic: false,
        isSecret: false,
      })),
      isLoading: false,
      isFetching: false,
      isError: false,
      isSuccess: true,
      refetch: hookMocks.refetch,
    }));
    hookMocks.mutateAsync.mockResolvedValue({});

    render(<RetentionBackpressureSection />);

    const card = screen.getByText('retentionBp.bp.title').closest('.ant-card');
    expect(card).not.toBeNull();
    const highInput = within(card as HTMLElement).getByRole('spinbutton', {
      name: 'retentionBp.field.disk_high_pct',
    });
    await waitFor(() => expect(highInput).toHaveValue('90'));
    await user.clear(highInput);
    await user.type(highInput, '91');
    await user.click(within(card as HTMLElement).getByRole('button', { name: 'retentionBp.save' }));

    await waitFor(() => {
      expect(hookMocks.mutateAsync).toHaveBeenCalledWith({
        category: 'acs.backpressure',
        items: [{ key: 'disk_high_pct', value: '91', value_type: 'int' }],
      });
    });
  });

  it('资源保留与背压支持保存基站日志清理周期并校验范围', async () => {
    const user = userEvent.setup();
    const valuesByCategory: Record<string, Record<string, string>> = {
      'acs.backpressure': {
        enabled: 'true',
        disk_high_pct: '90',
        disk_low_pct: '70',
        check_interval_sec: '60',
      },
      'minio.retention': { raw_object_days: '30' },
      'stationlog.retention': {
        max_retention_days: '30',
        cleanup_interval_minutes: '60',
        max_file_count: '0',
        max_file_count_per_device: '0',
      },
      raw_archive: { compress_after_ingest: 'false' },
    };
    hookMocks.query.mockImplementation((category: string) => ({
      data: Object.entries(valuesByCategory[category]).map(([key, value]) => ({
        id: `${category}-${key}`,
        category,
        key,
        value,
        valueType: key === 'enabled' || key === 'compress_after_ingest' ? 'bool' : 'int',
        isPublic: false,
        isSecret: false,
      })),
      isLoading: false,
      isFetching: false,
      isError: false,
      isSuccess: true,
      refetch: hookMocks.refetch,
    }));
    hookMocks.mutateAsync.mockResolvedValue({});

    render(<RetentionBackpressureSection />);

    const card = screen.getByText('retentionBp.stationlog.title').closest('.ant-card');
    expect(card).not.toBeNull();
    const cardText = (card as HTMLElement).textContent ?? '';
    expect(cardText.indexOf('retentionBp.field.cleanup_interval_minutes')).toBeGreaterThan(
      cardText.indexOf('retentionBp.field.max_file_count_per_device'),
    );
    const intervalInput = within(card as HTMLElement).getByRole('spinbutton', {
      name: 'retentionBp.field.cleanup_interval_minutes',
    });
    await waitFor(() => expect(intervalInput).toHaveValue('60'));
    expect(intervalInput).toHaveAttribute('aria-valuemin', '10');
    expect(intervalInput).toHaveAttribute('aria-valuemax', '1440');

    await user.clear(intervalInput);
    await user.type(intervalInput, '45');
    await user.click(within(card as HTMLElement).getByRole('button', { name: 'retentionBp.save' }));

    await waitFor(() => {
      expect(hookMocks.mutateAsync).toHaveBeenCalledWith({
        category: 'stationlog.retention',
        items: [{ key: 'cleanup_interval_minutes', value: '45', value_type: 'int' }],
      });
    });
  });

  it('日志配置只提交当前卡片内实际修改的字段', async () => {
    const user = userEvent.setup();
    hookMocks.query.mockImplementation((category: string) => {
      const values = category === 'log.retention'
        ? {
            enabled: 'true',
            database_days: '180',
        }
        : {
            service_days: '30',
            max_size_mb: '100',
            rotate_interval_minutes: '60',
            keep_files: '10',
          };
      return {
        data: Object.entries(values).map(([key, value]) => ({
          id: `${category}-${key}`,
          category,
          key,
          value,
          valueType: key === 'enabled' ? 'bool' : 'int',
          isPublic: false,
          isSecret: false,
        })),
        isLoading: false,
        isFetching: false,
        isError: false,
        isSuccess: true,
        refetch: hookMocks.refetch,
      };
    });
    hookMocks.mutateAsync.mockResolvedValue({});

    render(<LogRetentionSection />);

    const card = screen.getByText('logCfg.retention.title').closest('.ant-card');
    expect(card).not.toBeNull();
    const databaseInput = within(card as HTMLElement).getByRole('spinbutton', {
      name: 'logCfg.field.database_days',
    });
    await waitFor(() => expect(databaseInput).toHaveValue('180'));
    await user.clear(databaseInput);
    await user.type(databaseInput, '365');
    await user.click(within(card as HTMLElement).getByRole('button', { name: 'logCfg.save' }));

    await waitFor(() => {
      expect(hookMocks.mutateAsync).toHaveBeenCalledWith({
        category: 'log.retention',
        items: [{ key: 'database_days', value: '365', value_type: 'int' }],
      });
    });

    const rotationCard = screen.getByText('logCfg.rotation.title').closest('.ant-card');
    expect(rotationCard).not.toBeNull();
    const serviceInput = within(rotationCard as HTMLElement).getByRole('spinbutton', {
      name: 'logCfg.field.service_days',
    });
    await waitFor(() => expect(serviceInput).toHaveValue('30'));
    await user.clear(serviceInput);
    await user.type(serviceInput, '45');
    await user.click(within(rotationCard as HTMLElement).getByRole('button', { name: 'logCfg.save' }));

    await waitFor(() => {
      expect(hookMocks.mutateAsync).toHaveBeenLastCalledWith({
        category: 'log.rotation',
        items: [{ key: 'service_days', value: '45', value_type: 'int' }],
      });
    });
  });
});
