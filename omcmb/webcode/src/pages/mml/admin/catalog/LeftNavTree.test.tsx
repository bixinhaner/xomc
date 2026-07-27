import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { GroupTreeNode } from '@core/types/mmlConsole';
import LeftNavTree from './LeftNavTree';

vi.mock('@/hooks/useT', () => ({
  useT: () => (id: string) => id,
}));

describe('LeftNavTree search', () => {
  it('matches command target paths in the MML catalog search', () => {
    const groups = [{
      id: 'group-management-server',
      groupCode: 'MANAGEMENT_SERVER',
      path: 'MANAGEMENT_SERVER',
      displayName: '基站网关连接',
      displayOrder: 1,
      commands: [{
        id: 'command-lst-management-server',
        commandCode: 'LST MANAGEMENT_SERVER',
        logicalCode: 'MANAGEMENT_SERVER',
        operationType: 'LST',
        displayName: '查询基站网关连接',
        requireConfirm: false,
        targetPaths: [
          'Device.ManagementServer.sslStatus.endDate',
          'Device.ManagementServer.sslStatus.startDate',
        ],
      }],
      children: [],
    }] as unknown as GroupTreeNode[];

    render(
      <LeftNavTree
        groups={groups}
        selectedKey={null}
        expandedKeys={[]}
        search="Device.ManagementServer.sslStatus.endDate"
        onSelect={vi.fn()}
        onExpand={vi.fn()}
        onGroupAction={vi.fn()}
        onCommandAction={vi.fn()}
      />,
    );

    expect(screen.getByText('查询基站网关连接')).toBeInTheDocument();
  });
});
