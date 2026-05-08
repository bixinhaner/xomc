import type { ParamModel, ParamMapping, StandardParam } from '../../types/paramModel';

export const mockParamModels: ParamModel[] = [
  {
    id: 'pm-001',
    name: 'CombaPico_LTE_v3',
    totalEntries: 528,
    totalObjects: 78,
    totalParams: 450,
    description: 'Comba 皮基站 LTE 参数模型 v3',
    isActive: true,
    loadedFrom: 'CombaPico_LTE_v3.xml',
  },
  {
    id: 'pm-002',
    name: 'BaicellsMicro_NR_v1',
    totalEntries: 612,
    totalObjects: 95,
    totalParams: 517,
    description: 'Baicells 5G 微基站参数模型 v1',
    isActive: true,
    loadedFrom: 'BaicellsMicro_NR_v1.xml',
  },
];

export const mockMappings: Record<string, ParamMapping[]> = {
  CombaPico_LTE_v3: [
    {
      id: 'map-001',
      paramModelId: 'pm-001',
      standardPath: 'Device.Cellular.AccessPoint.{i}.PLMN',
      privatePath: 'X_VENDOR_AccessPoint.{i}.PLMNID',
      entryType: 'parameter',
      access: 'readWrite',
      dataType: 'string',
      changeApplies: 'reload',
      isStorable: true,
      isActive: true,
      softwareVersion: '3.0.1',
    },
    {
      id: 'map-002',
      paramModelId: 'pm-001',
      standardPath: 'Device.Cellular.LTE.RAN.S1.AMFAddress',
      privatePath: 'X_VENDOR_LTE.RAN.S1.AMFAddr',
      entryType: 'parameter',
      access: 'readWrite',
      dataType: 'string',
      changeApplies: 'immediate',
      isStorable: true,
      isActive: true,
    },
  ],
  BaicellsMicro_NR_v1: [
    {
      id: 'map-101',
      paramModelId: 'pm-002',
      standardPath: 'Device.Cellular.NR.gNB.{i}.CellID',
      privatePath: 'X_BAICELLS_NR.gNB.{i}.CellID',
      entryType: 'parameter',
      access: 'readWrite',
      dataType: 'unsignedInt',
      changeApplies: 'reload',
      isStorable: true,
      isActive: true,
    },
  ],
};

export const mockStandardParams: StandardParam[] = [
  {
    standardPath: 'Device.Cellular.AccessPoint.{i}.PLMN',
    entryType: 'parameter',
    access: 'readWrite',
    dataType: 'string',
    changeApplies: 'reload',
  },
  {
    standardPath: 'Device.Cellular.LTE.RAN.S1.AMFAddress',
    entryType: 'parameter',
    access: 'readWrite',
    dataType: 'string',
    changeApplies: 'immediate',
  },
  {
    standardPath: 'Device.Cellular.NR.gNB.{i}.CellID',
    entryType: 'parameter',
    access: 'readWrite',
    dataType: 'unsignedInt',
    changeApplies: 'reload',
  },
];
