import { useCallback, useEffect, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { deviceApi } from '../services/api/deviceApi';
import type { AntennaEditableField, AntennaSector } from '../types/map';
import { calculateAntennaCoverage } from '../utils/antennaCoverage';

export function useAntennaSectorEditor(deviceId: string | undefined, sectors: AntennaSector[]) {
  const queryClient = useQueryClient();
  const [previewSectors, setPreviewSectors] = useState<AntennaSector[]>(sectors);
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    setPreviewSectors(sectors);
  }, [deviceId, sectors]);

  const updatePreview = useCallback((sectorNumber: number, field: AntennaEditableField, value: number | null) => {
    if (value === null || !Number.isFinite(value)) return;
    setPreviewSectors((current) => current.map((sector) => (
      sector.number === sectorNumber
        ? calculateAntennaCoverage({ ...sector, [field]: value })
        : sector
    )));
  }, []);

  const discardPreview = useCallback(() => {
    setPreviewSectors(sectors);
  }, [sectors]);

  const saveSector = useCallback(async (sectorNumber: number) => {
    if (!deviceId) throw new Error('device ID is required');
    const sector = previewSectors.find((item) => item.number === sectorNumber);
    if (!sector) throw new Error('antenna sector not found');

    setIsSaving(true);
    try {
      const saved = await deviceApi.updateAntennaSectorPlan(deviceId, sectorNumber, {
        azimuth: sector.azimuth,
        antennaHeight: sector.antennaHeight,
        mechanicalDowntilt: sector.mechanicalDowntilt,
        horizontalBeamwidth: sector.horizontalBeamwidth,
        verticalBeamwidth: sector.verticalBeamwidth,
      });
      queryClient.setQueryData<AntennaSector[]>(
        ['devices', deviceId, 'antenna-sectors'],
        (current = []) => current.map((item) => item.number === sectorNumber ? saved : item),
      );
      setPreviewSectors((current) => current.map((item) => item.number === sectorNumber ? saved : item));
      return saved;
    } finally {
      setIsSaving(false);
    }
  }, [deviceId, previewSectors, queryClient]);

  return {
    previewSectors,
    updatePreview,
    discardPreview,
    saveSector,
    isSaving,
  };
}
