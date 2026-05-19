import { useQuery } from '@tanstack/react-query';

import { systemGateway } from '../lib/ipc';

// Gateway detection runs once at mount and once a minute — gateway changes
// during a session are rare (Wi-Fi swap, VPN connect/disconnect) and the
// caller refetches manually after toggling the router-probe switch.
export const useGatewayInfo = () =>
  useQuery({
    queryKey: ['system-gateway'] as const,
    queryFn: systemGateway,
    refetchInterval: 60_000,
    staleTime: 30_000,
  });
