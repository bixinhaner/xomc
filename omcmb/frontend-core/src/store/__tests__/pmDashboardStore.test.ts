import { describe, expect, it, beforeEach } from 'vitest';
import { usePmDashboardStore } from '../pmDashboardStore';
import type { Dashboard, Panel } from '../../types/pmDashboard';

const mockDash: Dashboard = {
  id: 'd1',
  name: 'test',
  ownerId: 'u1',
  sharedWith: [],
  technology: 'lte',
  layout: { panels: [{ i: 'p1', x: 0, y: 0, w: 6, h: 4 }] },
  createdAt: '2026-05-23T00:00:00Z',
  updatedAt: '2026-05-23T00:00:00Z',
};

const mockPanel: Panel = {
  id: 'p1',
  dashboardId: 'd1',
  panelType: 'kpi_card',
  title: 'KPI',
  metricPaths: ['M1'],
  granularity: 'hourly',
  dimension: 'device',
  timeRange: { start_offset: '-1h' },
  config: {},
};

describe('pmDashboardStore', () => {
  beforeEach(() => {
    usePmDashboardStore.getState().reset();
  });

  it('loadDashboard initializes state + clears dirty', () => {
    usePmDashboardStore.getState().markDirty();
    usePmDashboardStore.getState().loadDashboard(mockDash, [mockPanel]);
    const s = usePmDashboardStore.getState();
    expect(s.currentDashboard?.id).toBe('d1');
    expect(s.currentPanels).toHaveLength(1);
    expect(s.unsavedChanges).toBe(false);
    expect(s.editMode).toBe(false);
  });

  it('toggleEditMode flips flag', () => {
    expect(usePmDashboardStore.getState().editMode).toBe(false);
    usePmDashboardStore.getState().toggleEditMode();
    expect(usePmDashboardStore.getState().editMode).toBe(true);
  });

  it('addPanel appends + marks dirty', () => {
    usePmDashboardStore.getState().loadDashboard(mockDash, []);
    const newPanel: Panel = { ...mockPanel, id: 'p2' };
    usePmDashboardStore.getState().addPanel(newPanel);
    const s = usePmDashboardStore.getState();
    expect(s.currentPanels).toHaveLength(1);
    expect(s.currentDashboard?.layout.panels).toHaveLength(2);
    expect(s.unsavedChanges).toBe(true);
  });

  it('removePanel filters + marks dirty', () => {
    usePmDashboardStore.getState().loadDashboard(mockDash, [mockPanel]);
    usePmDashboardStore.getState().removePanel('p1');
    const s = usePmDashboardStore.getState();
    expect(s.currentPanels).toHaveLength(0);
    expect(s.currentDashboard?.layout.panels).toHaveLength(0);
    expect(s.unsavedChanges).toBe(true);
  });

  it('updatePanelLayout patches single item', () => {
    usePmDashboardStore.getState().loadDashboard(mockDash, [mockPanel]);
    usePmDashboardStore.getState().updatePanelLayout('p1', { x: 6, y: 2 });
    const item = usePmDashboardStore.getState().currentDashboard?.layout.panels[0];
    expect(item?.x).toBe(6);
    expect(item?.y).toBe(2);
    expect(usePmDashboardStore.getState().unsavedChanges).toBe(true);
  });

  it('markClean clears dirty', () => {
    usePmDashboardStore.getState().markDirty();
    usePmDashboardStore.getState().markClean();
    expect(usePmDashboardStore.getState().unsavedChanges).toBe(false);
  });
});
