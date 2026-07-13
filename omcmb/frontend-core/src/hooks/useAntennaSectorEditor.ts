import { useCallback, useEffect, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useUpdateParameters } from './api/useDeviceParameters';
import type { AntennaSector } from '../types/map';
import type { ParameterUpdateRequest } from '../types/deviceParameter';

export type AntennaEditableField = 'azimuth' | 'mechanicalDowntilt';

const fieldSourceKeys: Record<AntennaEditableField, string> = {
  azimuth: 'azimuth',
  mechanicalDowntilt: 'downtilt',
};

export function isAntennaSectorFieldEditable(sector: AntennaSector, field: AntennaEditableField): boolean {
  return Boolean(sector.fieldSources[fieldSourceKeys[field]]);
}

export function useAntennaSectorEditor(deviceId: string | undefined, sectors: AntennaSector[]) {
  const queryClient = useQueryClient();
  const updateParameters = useUpdateParameters();
  const [previewSectors, setPreviewSectors] = useState<AntennaSector[]>(sectors);

  useEffect(() => {
    setPreviewSectors(sectors);
  }, [deviceId, sectors]);

  const updatePreview = useCallback((sectorNumber: number, field: AntennaEditableField, value: number | null) => {
    if (value === null || !Number.isFinite(value)) return;
    setPreviewSectors((current) => current.map((sector) => (
      sector.number === sectorNumber ? { ...sector, [field]: value } : sector
    )));
  }, []);

  const discardPreview = useCallback(() => {
    setPreviewSectors(sectors);
  }, [sectors]);

  const saveSector = useCallback(async (sectorNumber: number) => {
    if (!deviceId) throw new Error('device ID is required');
    const sector = previewSectors.find((item) => item.number === sectorNumber);
    const originalSector = sectors.find((item) => item.number === sectorNumber);
    if (!sector || !originalSector) throw new Error('antenna sector not found');

    const fields: AntennaEditableField[] = ['azimuth', 'mechanicalDowntilt'];
    const parameters: ParameterUpdateRequest[] = fields.flatMap((field) => {
      const path = sector.fieldSources[fieldSourceKeys[field]];
      const value = sector[field];
      if (!path || value === undefined || value === originalSector[field]) return [];
      return [{ parameterPath: path, parameterValue: String(value), parameterType: 'int' }];
    });
    if (parameters.length === 0) throw new Error('no editable antenna parameters');

    const result = await updateParameters.mutateAsync({ deviceId, parameters });
    // SetParameterValues 异步等待 CPE 回执，保留预览值；避免立即查询旧上报值而回跳。
    await queryClient.invalidateQueries({
      queryKey: ['devices', deviceId, 'antenna-sectors'],
      refetchType: 'none',
    });
    return result;
  }, [deviceId, previewSectors, queryClient, sectors, updateParameters]);

  return {
    previewSectors,
    updatePreview,
    discardPreview,
    saveSector,
    isSaving: updateParameters.isPending,
  };
}