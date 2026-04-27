import { useEffect, useState } from 'react';
import { Folder } from '@phosphor-icons/react';
import { Button, Input, Switch, XStack, YStack, Text } from 'tamagui';

import { Card } from '../components/Card';
import { FormField } from '../components/FormField';
import { PageHeader } from '../components/PageHeader';
import { useAppConfig, useUpdateConfig } from '../hooks/useAppConfig';
import type { AppConfig } from '../lib/ipc';

/**
 * Settings — config form for the singleton app_config row.
 *
 * Important caveat in the UI: changes don't hot-reload the running monitor.
 * Probe interval, timeout, and Wi-Fi sample toggle are read once at startup.
 * After saving, the user has to quit + relaunch (or right-click tray → Quit
 * → reopen) for the new values to take effect. We tell them this clearly.
 *
 * Retention values DO take effect on the next pruner cycle (within an hour)
 * since the pruner re-reads app_config every wakeup — flagged in the form.
 */
export function SettingsPage() {
  const cfg = useAppConfig();
  const update = useUpdateConfig();

  const [draft, setDraft] = useState<AppConfig | null>(null);
  const [savedAt, setSavedAt] = useState<Date | null>(null);

  // Seed draft when fetch lands.
  useEffect(() => {
    if (cfg.data && draft == null) setDraft(cfg.data);
  }, [cfg.data, draft]);

  if (!draft) {
    return (
      <YStack flex={1}>
        <PageHeader title="Settings" />
        <YStack padding="$4">
          <Text fontSize={11} color="$color9">Loading…</Text>
        </YStack>
      </YStack>
    );
  }

  const dirty = JSON.stringify(draft) !== JSON.stringify(cfg.data);

  const setField = <K extends keyof AppConfig>(key: K, value: AppConfig[K]) =>
    setDraft((prev) => (prev ? { ...prev, [key]: value } : prev));

  const onSave = () => {
    update.mutate(draft, {
      onSuccess: () => setSavedAt(new Date()),
    });
  };

  const onRevert = () => {
    if (cfg.data) setDraft(cfg.data);
  };

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
              backgroundColor={dirty ? '$accentBackground' : '$color5'}
              color={dirty ? '$accentColor' : '$color9'}
              onPress={onSave}
              disabled={!dirty || update.isPending}
            >
              {update.isPending ? 'Saving…' : 'Save'}
            </Button>
          </XStack>
        }
      />

      <YStack padding="$4" gap="$3" maxWidth={760} width="100%" alignSelf="center">
        {savedAt && !dirty ? (
          <Card>
            <XStack gap="$2" alignItems="center">
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
          </Card>
        ) : null}

        <Card title="Probe loop">
          <YStack gap="$3">
            <FormField
              label="Ping interval (seconds)"
              hint="Time between probe cycles. Lower = denser data, higher load. Applies on the next cycle."
            >
              <Input
                size="$3"
                keyboardType="decimal-pad"
                value={String(draft.ping_interval_sec)}
                onChangeText={(v) =>
                  setField('ping_interval_sec', Number.parseFloat(v) || draft.ping_interval_sec)
                }
              />
            </FormField>
            <FormField
              label="Per-probe timeout (ms)"
              hint="How long a probe waits before failing. Applies on the next cycle."
            >
              <Input
                size="$3"
                keyboardType="number-pad"
                value={String(draft.ping_timeout_ms)}
                onChangeText={(v) =>
                  setField('ping_timeout_ms', Number.parseInt(v, 10) || draft.ping_timeout_ms)
                }
              />
            </FormField>
            <FormField
              label="Flush interval (seconds)"
              hint="How often the in-memory buffer is written to SQLite. Applies immediately."
            >
              <Input
                size="$3"
                keyboardType="number-pad"
                value={String(draft.flush_interval_sec)}
                onChangeText={(v) =>
                  setField('flush_interval_sec', Number.parseInt(v, 10) || draft.flush_interval_sec)
                }
              />
            </FormField>
          </YStack>
        </Card>

        <Card title="Retention">
          <YStack gap="$3">
            <Text fontSize={11} color="$color8">
              Pruner runs every hour and reads these values fresh — changes apply on the next
              wakeup, no restart needed.
            </Text>
            <FormField
              label="Raw samples — keep for (days)"
              hint="Per-probe rows. Default 7. Each day ~70K rows on the 12-target default config."
            >
              <Input
                size="$3"
                keyboardType="number-pad"
                value={String(draft.retention_raw_days)}
                onChangeText={(v) =>
                  setField('retention_raw_days', Number.parseInt(v, 10) || draft.retention_raw_days)
                }
              />
            </FormField>
            <FormField
              label="5-minute buckets — keep for (days)"
              hint="Aggregated rollups. Default 90. ~12 buckets/hour × N targets."
            >
              <Input
                size="$3"
                keyboardType="number-pad"
                value={String(draft.retention_5min_days)}
                onChangeText={(v) =>
                  setField('retention_5min_days', Number.parseInt(v, 10) || draft.retention_5min_days)
                }
              />
            </FormField>
            <Text fontSize={11} color="$color8">
              1-hour buckets and outage records are kept forever — they're tiny and the whole
              point of long-term Vigil is the historical record.
            </Text>
          </YStack>
        </Card>

        <Card title="Wi-Fi sampling">
          <XStack alignItems="center" gap="$3">
            <Switch
              size="$3"
              checked={draft.wifi_sample_enabled}
              onCheckedChange={(v) => setField('wifi_sample_enabled', v)}
              backgroundColor={draft.wifi_sample_enabled ? '$accentBackground' : '$color5'}
            >
              <Switch.Thumb animation="quick" />
            </Switch>
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
        </Card>

        <DataFolderCard />
      </YStack>
    </YStack>
  );
}

/**
 * Open the OS file manager to the data folder. Uses the shell plugin's
 * `open` API which we already permitted in capabilities/default.json.
 */
function DataFolderCard() {
  const [path, setPath] = useState<string | null>(null);

  // Tauri 2's path API exposes app_data_dir via JS — fetch it once.
  useEffect(() => {
    import('@tauri-apps/api/path')
      .then(({ appDataDir }) => appDataDir())
      .then(setPath)
      .catch(() => setPath(null));
  }, []);

  const openFolder = async () => {
    if (!path) return;
    try {
      const { open } = await import('@tauri-apps/plugin-shell');
      await open(path);
    } catch (err) {
      // Plugin not permitted or shell.open denied. Surface but don't crash.
      console.warn('Failed to open data folder:', err);
    }
  };

  return (
    <Card title="Data folder">
      <YStack gap="$2">
        <Text fontSize={11} color="$color8">
          SQLite database, log file, and cached settings live here.
        </Text>
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
      </YStack>
    </Card>
  );
}

