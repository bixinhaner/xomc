import { isValidElement } from 'react';
import { describe, expect, it } from 'vitest';
import PrivateRoute from '../PrivateRoute';
import { routes } from '../routes';

function findAppRoute(path: string) {
  const appRoute = routes.find((route) => route.path === '/');
  return appRoute?.children?.find((route) => route.path === path);
}

describe('product KPI library route governance', () => {
  it('requires super_admin on the official product KPI library route', () => {
    const route = findAppRoute('product/kpi-library');
    const element = route?.element;

    expect(isValidElement(element)).toBe(true);
    if (!isValidElement<{ requireSuperAdmin?: boolean }>(element)) {
      throw new Error('product KPI library route must render a React element');
    }
    expect(element.type).toBe(PrivateRoute);
    expect(element.props.requireSuperAdmin).toBe(true);
  });

  it('keeps the hidden performance KPI route for backward compatibility', () => {
    const route = findAppRoute('performance/kpi-standard');

    expect(route?.element).toBeDefined();
  });
});
