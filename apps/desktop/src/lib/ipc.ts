// IPC client for the Vigil sidecar.
//
// The Tauri Rust shell exposes a single command, `ipc_call`, which accepts a
// method name + params object and forwards them to the sidecar's stdio JSON-
// RPC server. Responses come back as the resolved promise; events arrive on
// the parallel `lib/events.ts` listener.

import { invoke } from '@tauri-apps/api/core';

import { translateError } from './errorMessages';

export interface IpcError {
  code: string;
  message: string;
}

/**
 * Error thrown when a sidecar IPC call fails. The `.message` is a
 * user-friendly translation; the original technical message is on
 * `.technicalDetail` for debugging / logging.
 */
export class VigilIpcError extends Error {
  code: string;
  technicalDetail: string;
  constructor(err: IpcError) {
    super(translateError(err.code, err.message));
    this.name = 'VigilIpcError';
    this.code = err.code;
    this.technicalDetail = err.message;
  }
}

export async function ipcCall<TResult, TParams = unknown>(
  method: string,
  params?: TParams,
): Promise<TResult> {
  try {
    return await invoke<TResult>('ipc_call', { method, params: params ?? {} });
  } catch (raw) {
    if (typeof raw === 'object' && raw !== null && 'code' in raw && 'message' in raw) {
      throw new VigilIpcError(raw as IpcError);
    }
    throw raw;
  }
}

// =============================================================================
// Method-typed helpers
// =============================================================================

export interface HealthCheckResult {
  status: string;
  version: string;
  commit?: string;
}

export const healthCheck = () => ipcCall<HealthCheckResult>('health.check');

// ---------- Targets ----------

export type ProbeKind = 'icmp' | 'tcp' | 'udp_dns' | 'udp_stun';

export interface Target {
  id: string;
  label: string;
  kind: ProbeKind;
  host: string;
  port?: number;
  enabled: boolean;
  is_builtin: boolean;
}

export const targetsList = () => ipcCall<Target[]>('targets.list');

export const targetsCreate = (params: {
  label: string;
  kind: ProbeKind;
  host: string;
  port?: number;
}) => ipcCall<Target>('targets.create', params);

export const targetsUpdate = (params: {
  id: string;
  enabled?: boolean;
  host?: string;
  port?: number;
}) => ipcCall<Target>('targets.update', params);

export const targetsDelete = (id: string) =>
  ipcCall<{ ok: true }>('targets.delete', { id });

// ---------- Samples ----------

export type Granularity = 'auto' | 'raw' | '1min' | '5min' | '1h';

export interface RawSample {
  ts_unix_ms: number;
  target_label: string;
  target_kind: string;
  target_host: string;
  target_port?: number;
  success: boolean;
  rtt_ms?: number;
  error?: string;
}

export interface AggregatedRow {
  bucket_start_unix_ms: number;
  target_label: string;
  count: number;
  success_count: number;
  fail_count: number;
  rtt_p50_ms?: number;
  rtt_p95_ms?: number;
  rtt_p99_ms?: number;
  rtt_max_ms?: number;
  rtt_mean_ms?: number;
  jitter_ms?: number;
  errors?: Record<string, number>;
}

// Discriminated union — `granularity` is the discriminator. Frontend charts
// switch on this to pick the right field names.
export type SamplesQueryResult =
  | { granularity: 'raw'; rows: RawSample[] }
  | { granularity: '1min'; rows: AggregatedRow[] }
  | { granularity: '5min'; rows: AggregatedRow[] }
  | { granularity: '1h'; rows: AggregatedRow[] };

export const samplesQuery = (params: {
  from_ms?: number;
  to_ms?: number;
  target_labels?: string[];
  limit?: number;
  granularity?: Granularity;
}) => ipcCall<SamplesQueryResult>('samples.query', params);

// ---------- Wi-Fi ----------

export interface WifiSample {
  ts_unix_ms: number;
  ssid?: string;
  bssid?: string;
  signal_percent?: number;
  rssi_dbm?: number;
  rx_rate_mbps?: number;
  tx_rate_mbps?: number;
  channel?: string;
}

export const wifiList = (params: { from_ms?: number; to_ms?: number } = {}) =>
  ipcCall<WifiSample[]>('wifi.list', params);

// ---------- Outages ----------

export interface Outage {
  id: string;
  scope: string; // "network" | "target:<label>"
  start_ts_unix_ms: number;
  end_ts_unix_ms?: number;
  consecutive_failures: number;
  errors?: Record<string, number>;
}

export const outagesList = (params: {
  from_ms?: number;
  to_ms?: number;
  scope?: string;
  only_open?: boolean;
} = {}) => ipcCall<Outage[]>('outages.list', params);

// ---------- Config ----------

export interface AppConfig {
  ping_interval_sec: number;
  flush_interval_sec: number;
  ping_timeout_ms: number;
  retention_raw_days: number;
  retention_1min_days: number;
  retention_5min_days: number;
  wifi_sample_enabled: boolean;
}

export const configGet = () => ipcCall<AppConfig>('config.get');

export const configUpdate = (patch: Partial<AppConfig>) =>
  ipcCall<AppConfig>('config.update', patch);

// ---------- Reports ----------

export type ReportFormat = 'csv' | 'json' | 'html';

export interface ReportResult {
  paths: string[];
}

export const reportGenerate = (params: {
  out_dir: string;
  from_ms: number;
  to_ms: number;
  targets?: string[];
  formats: ReportFormat[];
  base_name?: string;
}) => ipcCall<ReportResult>('report.generate', params);
