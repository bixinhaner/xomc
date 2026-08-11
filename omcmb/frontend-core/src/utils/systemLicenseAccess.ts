/** Returns whether a route belongs to the system-license recovery surface. */
export function isSystemLicensePath(pathname: string): boolean {
  return pathname === '/license' || pathname.startsWith('/license/');
}