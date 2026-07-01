import type { AlarmRule, AlarmRuleAction, AlarmRuleCondition } from '../types/alarm';

export type AlarmRuleSelectionMode = 'devices' | 'groups';

export interface AlarmRuleSelectionState {
  deviceSelectionMode: AlarmRuleSelectionMode;
  selectedDevices: string[];
  selectedGroups: string[];
  selectedAlarms: string[];
}

export function getAlarmRuleConditionValues(
  rule: AlarmRule | null | undefined,
  field: string,
): string[] {
  if (!rule) {
    return [];
  }

  return rule.conditions
    .filter((condition) => condition.field === field)
    .flatMap((condition) => (Array.isArray(condition.value) ? condition.value : [condition.value]))
    .map((value) => String(value))
    .filter((value) => value.length > 0);
}

export function getAlarmRuleSelection(rule: AlarmRule | null | undefined): AlarmRuleSelectionState {
  const selectedDevices = getAlarmRuleConditionValues(rule, 'device_id');
  const selectedGroups = getAlarmRuleConditionValues(rule, 'device_group_id');
  const selectedAlarms = getAlarmRuleConditionValues(rule, 'alarm_identifier');

  return {
    deviceSelectionMode: selectedGroups.length > 0 && selectedDevices.length === 0 ? 'groups' : 'devices',
    selectedDevices,
    selectedGroups,
    selectedAlarms,
  };
}

export function buildAlarmRuleConditions(selection: AlarmRuleSelectionState): AlarmRuleCondition[] {
  const conditions: AlarmRuleCondition[] = [];

  if (selection.selectedAlarms.length > 0) {
    conditions.push({
      field: 'alarm_identifier',
      operator: 'contains',
      value: selection.selectedAlarms,
    });
  }

  if (selection.deviceSelectionMode === 'devices' && selection.selectedDevices.length > 0) {
    conditions.push({
      field: 'device_id',
      operator: 'contains',
      value: selection.selectedDevices,
    });
  }

  if (selection.deviceSelectionMode === 'groups' && selection.selectedGroups.length > 0) {
    conditions.push({
      field: 'device_group_id',
      operator: 'contains',
      value: selection.selectedGroups,
    });
  }

  return conditions;
}

export function buildAlarmRuleActions(ruleType: string): AlarmRuleAction[] {
  if (!ruleType) {
    return [];
  }

  return [{
    type: 'suppress',
    target: ruleType,
  }];
}