import type { AntennaSector } from '../types/map';

const radians = (degrees: number) => degrees * Math.PI / 180;

export function calculateAntennaCoverage(sector: AntennaSector): AntennaSector {
  const missingFields: string[] = [];
  const directionAvailable = sector.azimuth !== undefined
    && Number.isFinite(sector.azimuth)
    && sector.azimuth >= 0
    && sector.azimuth < 360;

  if (!directionAvailable) missingFields.push('azimuth');
  if (sector.antennaHeight === undefined || sector.antennaHeight <= 0) missingFields.push('antennaHeight');
  if (sector.mechanicalDowntilt === undefined || sector.mechanicalDowntilt < 0 || sector.mechanicalDowntilt >= 90) {
    missingFields.push('mechanicalDowntilt');
  }
  if (sector.horizontalBeamwidth === undefined || sector.horizontalBeamwidth <= 0 || sector.horizontalBeamwidth >= 180) {
    missingFields.push('horizontalBeamwidth');
  }
  if (sector.verticalBeamwidth === undefined || sector.verticalBeamwidth <= 0 || sector.verticalBeamwidth >= 180) {
    missingFields.push('verticalBeamwidth');
  }

  if (missingFields.length > 0) {
    return {
      ...sector,
      directionAvailable,
      coverageAvailable: false,
      coverageStatus: 'incomplete',
      coverageIssue: undefined,
      nearRadiusMeters: undefined,
      farRadiusMeters: undefined,
      missingFields,
    };
  }

  const nearAngle = sector.mechanicalDowntilt! + sector.verticalBeamwidth! / 2;
  const farAngle = sector.mechanicalDowntilt! - sector.verticalBeamwidth! / 2;
  if (farAngle <= 0) {
    return {
      ...sector,
      directionAvailable,
      coverageAvailable: false,
      coverageStatus: 'invalid_geometry',
      coverageIssue: 'far_angle_not_positive',
      nearRadiusMeters: undefined,
      farRadiusMeters: undefined,
      missingFields: [],
    };
  }
  if (nearAngle >= 90) {
    return {
      ...sector,
      directionAvailable,
      coverageAvailable: false,
      coverageStatus: 'invalid_geometry',
      coverageIssue: 'near_angle_out_of_range',
      nearRadiusMeters: undefined,
      farRadiusMeters: undefined,
      missingFields: [],
    };
  }

  const nearRadiusMeters = sector.antennaHeight! / Math.tan(radians(nearAngle));
  const farRadiusMeters = sector.antennaHeight! / Math.tan(radians(farAngle));
  const radiiFinite = Number.isFinite(nearRadiusMeters) && Number.isFinite(farRadiusMeters);
  const coverageAvailable = radiiFinite && nearRadiusMeters > 0 && nearRadiusMeters < farRadiusMeters;
  return {
    ...sector,
    directionAvailable,
    coverageAvailable,
    coverageStatus: coverageAvailable ? 'available' : 'invalid_geometry',
    coverageIssue: coverageAvailable
      ? undefined
      : radiiFinite ? 'radius_order_invalid' : 'radius_not_finite',
    nearRadiusMeters: coverageAvailable ? nearRadiusMeters : undefined,
    farRadiusMeters: coverageAvailable ? farRadiusMeters : undefined,
    missingFields: [],
  };
}
