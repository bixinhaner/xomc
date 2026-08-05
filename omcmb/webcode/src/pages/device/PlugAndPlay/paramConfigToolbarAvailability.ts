/**
 * Parameter-configuration actions follow the module switch. Product-specific
 * prerequisites are checked by each action and reported with a user message.
 */
export function isParamConfigToolbarEnabled(selfConfigEnabled: boolean | undefined): boolean {
  return selfConfigEnabled === true;
}
