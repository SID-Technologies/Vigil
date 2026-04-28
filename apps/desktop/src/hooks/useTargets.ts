import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  targetsCreate,
  targetsDelete,
  targetsList,
  targetsUpdate,
  type Target,
} from '../lib/ipc';

const KEY = ['targets-list'] as const;

export const useTargets = () =>
  useQuery({
    queryKey: KEY,
    queryFn: targetsList,
  });

export const useCreateTarget = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: targetsCreate,
    onSuccess: () => qc.invalidateQueries({ queryKey: KEY }),
  });
};

export const useUpdateTarget = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: targetsUpdate,
    onSuccess: () => qc.invalidateQueries({ queryKey: KEY }),
  });
};

export const useDeleteTarget = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: targetsDelete,
    onSuccess: () => qc.invalidateQueries({ queryKey: KEY }),
  });
};

export type { Target };
