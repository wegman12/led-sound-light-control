import { useState, useEffect } from 'react';
import { checkHealth } from '../services';

export function useHealthCheck() {
  const [isHealthy, setIsHealthy] = useState<boolean | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    async function performHealthCheck() {
      try {
        const response = await checkHealth();
        setIsHealthy(response.status === 'ok');
      } catch {
        setIsHealthy(false);
      } finally {
        setIsLoading(false);
      }
    }

    performHealthCheck();
  }, []);

  return { isHealthy, isLoading };
}
