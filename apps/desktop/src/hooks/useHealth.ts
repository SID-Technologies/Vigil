import { useQuery } from '@tanstack/react-query';
import { healthCheck } from '../lib/ipc';

export const useHealth = () =>
  useQuery({
    queryKey: ['health-check'],
    queryFn: healthCheck,
    retry: 3,
    retryDelay: 500,
  });
