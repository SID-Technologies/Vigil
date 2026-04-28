import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { configGet, configUpdate, type AppConfig } from '../lib/ipc';

const KEY = ['app-config'] as const;

export const useAppConfig = () =>
  useQuery({
    queryKey: KEY,
    queryFn: configGet,
  });

export const useUpdateConfig = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (patch: Partial<AppConfig>) => configUpdate(patch),
    onSuccess: () => qc.invalidateQueries({ queryKey: KEY }),
  });
};
