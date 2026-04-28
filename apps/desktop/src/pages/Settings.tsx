import { useEffect, useMemo, useState } from 'react';
import { Folder, Compass, Keyboard } from '@phosphor-icons/react';
import { Button, Input, XStack, YStack, Text } from 'tamagui';

import { FormField } from '../components/FormField';
import { openShortcuts } from '../components/KeyboardShortcuts';
import { PageHeader } from '../components/PageHeader';
import { Section } from '../components/Section';
import { Skeleton } from '../components/Skeleton';
import { Toggle } from '../components/Toggle';
import { resetWelcomeTour } from '../components/WelcomeTour';
import { useAppConfig, useUpdateConfig } from '../hooks/useAppConfig';
import {
  CONFIG_BOUNDS,
  validateField,
  type ConfigNumericKey,
} from '../lib/configBounds';
import type { AppConfig } from '../lib/ipc';

type RawNumeric = Record<ConfigNumericKey, string>;

const NUMERIC_KEYS: ConfigNumericKey[] = [
  'ping_interval_sec',
  'ping_timeout_ms',
  'flush_interval_sec',
  'retention_raw_days',
  'retention_1min_days',
  'retention_5min_days',
];

function seedRaw(cfg: AppConfig): RawNumeric {
  return {
    ping_interval_sec: String(cfg.ping_interval_sec),
    ping_timeout_ms: String(cfg.ping_timeout_ms),
    flush_interval_sec: String(cfg.flush_interval_sec),
    retention_raw_days: String(cfg.retention_raw_days),
    retention_1min_days: String(cfg.retention_1min_days),
    retention_5min_days: String(cfg.retention_5min_days),
  };
}

/**
 * Settings — config form for the singleton app_config row.
 *
 * Numeric fields keep their own raw text state alongside the parsed AppConfig
 * draft. This lets users type freely (including transient invalid states like
 * "0." while typing "0.5") without snap-back, and surfaces validation errors
 * inline as they type. Save is disabled until every field validates.
 */
export function SettingsPage() {
  const cfg = useAppConfig();
  const update = useUpdateConfig();

  const [draft, setDraft] = useState<AppConfig | null>(null);
  const [raw, setRaw] = useState<RawNumeric | null>(null);
  const [savedAt, setSavedAt] = useState<Date | null>(null);

  // Seed both draft and raw text when fetch lands.
  useEffect(() => {
    if (cfg.data && draft == null) {
      setDraft(cfg.data);
      setRaw(seedRaw(cfg.data));
    }
  }, [cfg.data, draft]);

  // Validate every numeric field on each render — cheap, derived state.
  const errors = useMemo(() => {
    if (!raw) return {} as Partial<Record<ConfigNumericKey, string>>;
    const out: Partial<Record<ConfigNumericKey, string>> = {};
    for (const key of NUMERIC_KEYS) {
      const r = validateField(key, raw[key]);
      if (r.error) out[key] = r.error;
    }
    return out;
  }, [raw]);

  const hasErrors = Object.keys(errors).length > 0;
  // `dirty` has to be computed in a way that's safe before the config has
  // landed — otherwise the next hook (⌘S effect) would sit behind a
  // conditional early return and break Rules of Hooks.
  const dirty = !!(cfg.data && draft && JSON.stringify(draft) !== JSON.stringify(cfg.data));

  // ⌘S / Ctrl+S to save — matches macOS / Windows native conventions.
  // preventDefault stops the WebKit "save page as HTML" fallback.
  //
  // CRITICAL: This effect must live above any conditional early-return so
  // its hook-call position is stable across renders. The handler itself
  // guards against the pre-config-load state (draft == null).
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 's') {
        e.preventDefault();
        if (!draft || !dirty || hasErrors || update.isPending) return;
        update.mutate(draft, {
          onSuccess: () => setSavedAt(new Date()),
        });
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [draft, dirty, hasErrors, update.isPending]);

  if (!draft || !raw) {
    return (
      <YStack flex={1}>
        <PageHeader title="Settings" />
        <YStack
          padding="$4"
          paddingTop="$2"
          maxWidth={760}
          width="100%"
          alignSelf="center"
          gap="$5"
        >
          {[0, 1, 2].map((s) => (
            <YStack key={s} paddingTop="$4" gap="$3">
              <YStack
                gap="$1"
                paddingBottom="$2"
                borderBottomWidth={1}
                borderBottomColor="$borderColor"
              >
                <Skeleton width={120} height={11} />
              </YStack>
              <YStack gap="$3">
                <YStack gap="$1.5">
                  <Skeleton width={140} height={11} />
                  <Skeleton width="100%" height={36} borderRadius={6} />
                </YStack>
                <YStack gap="$1.5">
                  <Skeleton width={160} height={11} />
                  <Skeleton width="100%" height={36} borderRadius={6} />
                </YStack>
              </YStack>
            </YStack>
          ))}
        </YStack>
      </YStack>
    );
  }

  const setNumeric = (key: ConfigNumericKey, text: string) => {
    setRaw((prev) => (prev ? { ...prev, [key]: text } : prev));
    const r = validateField(key, text);
    if (r.value !== undefined) {
      setDraft((prev) => (prev ? { ...prev, [key]: r.value as number } : prev));
    }
  };

  const setBool = <K extends keyof AppConfig>(key: K, value: AppConfig[K]) =>
    setDraft((prev) => (prev ? { ...prev, [key]: value } : prev));

  const onSave = () => {
    if (hasErrors) return;
    update.mutate(draft, {
      onSuccess: () => setSavedAt(new Date()),
    });
  };

  const onRevert = () => {
    if (cfg.data) {
      setDraft(cfg.data);
      setRaw(seedRaw(cfg.data));
    }
  };

  const saveDisabled = !dirty || hasErrors || update.isPending;

  return (
    <YStack flex={1}>
      <PageHeader
        title="Settings"
        blurb="Probe cadence, timeouts, retention. All changes apply live — no restart needed."
        trailing={
          <XStack gap="$2">
            <Button size="$3" chromeless onPress={onRevert} disabled={!dirty}>
              Revert
            </Button>
            <Button
              size="$3"
              backgroundColor={!saveDisabled ? '$accentBackground' : '$color5'}
              color={!saveDisabled ? '$accentColor' : '$color9'}
              onPress={onSave}
              disabled={saveDisabled}
            >
              {update.isPending ? 'Saving…' : 'Save'}
            </Button>
          </XStack>
        }
      />

      <YStack padding="$4" paddingTop="$2" maxWidth={760} width="100%" alignSelf="center">
        {savedAt && !dirty ? (
          <XStack
            gap="$2"
            alignItems="center"
            paddingVertical="$2"
            paddingHorizontal="$3"
            backgroundColor="$color2"
            borderRadius="$2"
            borderLeftWidth={3}
            borderLeftColor="$accentBackground"
            marginTop="$3"
          >
            <YStack
              width={8}
              height={8}
              borderRadius={999}
              backgroundColor="$accentBackground"
            />
            <Text fontSize={12} color="$color11">
              Saved {savedAt.toLocaleTimeString()}. Changes apply on the next probe cycle — no
              restart needed.
            </Text>
          </XStack>
        ) : null}

        {hasErrors && dirty ? (
          <XStack
            gap="$2"
            alignItems="center"
            paddingVertical="$2"
            paddingHorizontal="$3"
            backgroundColor="$color2"
            borderRadius="$2"
            borderLeftWidth={3}
            borderLeftColor="$red9"
            marginTop="$3"
          >
            <Text fontSize={12} color="$color11">
              Fix the highlighted fields before saving.
            </Text>
          </XStack>
        ) : null}

        <Section title="Probe loop">
          <FormField
            label="Ping interval (seconds)"
            hint={`How often Vigil checks each target. Smaller numbers mean richer data; larger numbers are gentler on your network. Default 2.5. Range ${CONFIG_BOUNDS.ping_interval_sec.min}–${CONFIG_BOUNDS.ping_interval_sec.max}.`}
            error={errors.ping_interval_sec}
          >
            <Input
              size="$3"
              keyboardType="decimal-pad"
              value={raw.ping_interval_sec}
              borderColor={errors.ping_interval_sec ? '$red8' : undefined}
              onChangeText={(v) => setNumeric('ping_interval_sec', v)}
            />
          </FormField>
          <FormField
            label="Per-probe timeout (ms)"
            hint={`How long a single probe waits for a response before counting it as failed. Default 2000 ms. Range ${CONFIG_BOUNDS.ping_timeout_ms.min}–${CONFIG_BOUNDS.ping_timeout_ms.max}.`}
            error={errors.ping_timeout_ms}
          >
            <Input
              size="$3"
              keyboardType="number-pad"
              value={raw.ping_timeout_ms}
              borderColor={errors.ping_timeout_ms ? '$red8' : undefined}
              onChangeText={(v) => setNumeric('ping_timeout_ms', v)}
            />
          </FormField>
          <FormField
            label="Flush interval (seconds)"
            hint={`How often Vigil saves new probe data to disk. Smaller values mean less data lost if Vigil crashes; larger values reduce disk writes. Range ${CONFIG_BOUNDS.flush_interval_sec.min}–${CONFIG_BOUNDS.flush_interval_sec.max}.`}
            error={errors.flush_interval_sec}
          >
            <Input
              size="$3"
              keyboardType="number-pad"
              value={raw.flush_interval_sec}
              borderColor={errors.flush_interval_sec ? '$red8' : undefined}
              onChangeText={(v) => setNumeric('flush_interval_sec', v)}
            />
          </FormField>
        </Section>

        <Section
          title="Retention"
          description="The pruner runs every hour and reads these values fresh — changes apply on the next wakeup, no restart needed."
        >
          <FormField
            label="Raw samples — keep for (days)"
            hint={`How many days of detailed per-probe records to keep. After this, only the 1-minute and coarser summaries remain — still plenty for charts, but you can't drill down to individual probes. Default 7. Range ${CONFIG_BOUNDS.retention_raw_days.min}–${CONFIG_BOUNDS.retention_raw_days.max}.`}
            error={errors.retention_raw_days}
          >
            <Input
              size="$3"
              keyboardType="number-pad"
              value={raw.retention_raw_days}
              borderColor={errors.retention_raw_days ? '$red8' : undefined}
              onChangeText={(v) => setNumeric('retention_raw_days', v)}
            />
          </FormField>
          <FormField
            label="1-minute buckets — keep for (days)"
            hint={`How many days of 1-minute summaries to keep. These power History charts in the 1h–6h band where 5-minute buckets are too coarse. Default 14. Range ${CONFIG_BOUNDS.retention_1min_days.min}–${CONFIG_BOUNDS.retention_1min_days.max}.`}
            error={errors.retention_1min_days}
          >
            <Input
              size="$3"
              keyboardType="number-pad"
              value={raw.retention_1min_days}
              borderColor={errors.retention_1min_days ? '$red8' : undefined}
              onChangeText={(v) => setNumeric('retention_1min_days', v)}
            />
          </FormField>
          <FormField
            label="5-minute buckets — keep for (days)"
            hint={`How many days of 5-minute summaries to keep. These power the History page charts at week-and-longer windows. Default 90. Range ${CONFIG_BOUNDS.retention_5min_days.min}–${CONFIG_BOUNDS.retention_5min_days.max}.`}
            error={errors.retention_5min_days}
          >
            <Input
              size="$3"
              keyboardType="number-pad"
              value={raw.retention_5min_days}
              borderColor={errors.retention_5min_days ? '$red8' : undefined}
              onChangeText={(v) => setNumeric('retention_5min_days', v)}
            />
          </FormField>
          <Text fontSize={11} color="$color8">
            1-hour buckets and outage records are kept forever — they're tiny and the whole
            point of long-term Vigil is the historical record.
          </Text>
        </Section>

        <Section title="Wi-Fi sampling">
          <XStack alignItems="center" gap="$3">
            <Toggle
              checked={draft.wifi_sample_enabled}
              onCheckedChange={(v) => setBool('wifi_sample_enabled', v)}
              size="md"
            />
            <YStack flex={1} gap="$0.5">
              <Text fontSize={13} color="$color12">
                Capture Wi-Fi state every flush
              </Text>
              <Text fontSize={11} color="$color9">
                On macOS, runs `system_profiler SPAirPortDataType` once per minute. On Linux,
                netlink. On Windows, `netsh wlan`. Disable if you're on wired Ethernet only.
                Applies on the next flush.
              </Text>
            </YStack>
          </XStack>
        </Section>

        <Section
          title="Data folder"
          description="SQLite database, log file, and cached settings live here."
        >
          <DataFolderRow />
        </Section>

        <Section
          title="Help"
          description="Replay the welcome tour or peek at the keyboard shortcuts."
        >
          <HelpRow />
        </Section>
      </YStack>
    </YStack>
  );
}

/**
 * Two-button row for the Help section. Replays the welcome tour (clears the
 * dismissed flag, then reloads so the App-level tour fires fresh) and opens
 * the keyboard shortcut overlay (mirroring Shift+?).
 */
function HelpRow() {
  const replayTour = () => {
    resetWelcomeTour();
    window.location.reload();
  };
  return (
    <XStack gap="$2" flexWrap="wrap">
      <Button
        size="$3"
        chromeless
        icon={<Compass size={14} color="var(--color9)" />}
        onPress={replayTour}
      >
        <Text fontSize={12} color="$color11">Show welcome tour again</Text>
      </Button>
      <Button
        size="$3"
        chromeless
        icon={<Keyboard size={14} color="var(--color9)" />}
        onPress={openShortcuts}
      >
        <Text fontSize={12} color="$color11">Keyboard shortcuts</Text>
      </Button>
    </XStack>
  );
}

/**
 * Path display + Open button. Wrapped by a Section in the parent so we
 * don't duplicate the title — the Section provides the heading and
 * description, this row just renders the path and the action.
 */
function DataFolderRow() {
  const [path, setPath] = useState<string | null>(null);

  useEffect(() => {
    import('@tauri-apps/api/path')
      .then(({ appDataDir }) => appDataDir())
      .then(setPath)
      .catch(() => setPath(null));
  }, []);

  const openFolder = async () => {
    if (!path) return;
    try {
      const { openPath } = await import('@tauri-apps/plugin-opener');
      await openPath(path);
    } catch (err) {
      console.warn('Failed to open data folder:', err);
    }
  };

  return (
    <XStack gap="$2" alignItems="center" flexWrap="wrap">
      <Text fontSize={11} color="$color11" fontFamily="$body" flex={1} numberOfLines={1}>
        {path ?? '…'}
      </Text>
      <Button
        size="$2"
        chromeless
        icon={<Folder size={12} color="var(--color9)" />}
        onPress={openFolder}
        disabled={!path}
      >
        <Text fontSize={11} color="$color11">Open</Text>
      </Button>
    </XStack>
  );
}
