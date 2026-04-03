import { useState, useEffect } from 'react';
import { useUserStore } from './userStore';

/**
 * Hook to check if zustand persist has finished hydration.
 * This is the recommended way to handle hydration in zustand 5.
 */
export function useHydration() {
  const [hydrated, setHydrated] = useState(() => {
    // Check if already hydrated on mount
    return useUserStore.persist?.hasHydrated?.() ?? false;
  });

  useEffect(() => {
    // Subscribe to hydration finish event
    const unsubHydrate = useUserStore.persist?.onHydrate?.(() => {
      setHydrated(false);
    });

    const unsubFinish = useUserStore.persist?.onFinishHydration?.(() => {
      setHydrated(true);
    });

    // Check again in case hydration finished before subscription
    if (useUserStore.persist?.hasHydrated?.()) {
      setHydrated(true);
    }

    return () => {
      unsubHydrate?.();
      unsubFinish?.();
    };
  }, []);

  return hydrated;
}
