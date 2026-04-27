import { useEffect, useMemo, useState } from 'react';
import { Plus, Trash, Lock, Check } from '@phosphor-icons/react';
import { Button, Input, Select, XStack, YStack, Text, Switch } from 'tamagui';

import { Card } from '../components/Card';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { FormField } from '../components/FormField';
import { Modal } from '../components/Modal';
import { PageHeader } from '../components/PageHeader';
import {
  useCreateTarget,
  useDeleteTarget,
  useTargets,
  useUpdateTarget,
  type Target,
} from '../hooks/useTargets';
import type { ProbeKind } from '../lib/ipc';

const KIND_OPTIONS: { value: ProbeKind; label: string; needsPort: boolean }[] = [
  { value: 'icmp', label: 'ICMP — ping reachability', needsPort: false },
  { value: 'tcp', label: 'TCP handshake', needsPort: true },
  { value: 'udp_dns', label: 'UDP — DNS query', needsPort: true },
  { value: 'udp_stun', label: 'UDP — STUN binding', needsPort: true },
];

/**
 * Targets — manage the probe target list. Builtin targets (the 12 seeded
 * defaults plus the dynamic gateway) can be enabled/disabled but not edited
 * or deleted; that's enforced at the sidecar layer too.
 *
 * Bulk operations: a checkbox column lets the user select multiple targets;
 * a sticky toolbar appears with Enable / Disable / Delete actions. Delete
 * skips builtins automatically (they're protected). Confirms via the
 * Tamagui ConfirmDialog — no native confirm() shocks.
 */
export function TargetsPage() {
  const targets = useTargets();
  const update = useUpdateTarget();
  const del = useDeleteTarget();

  const [addOpen, setAddOpen] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [singleDelete, setSingleDelete] = useState<Target | null>(null);
  const [bulkDeleteOpen, setBulkDeleteOpen] = useState(false);

  const sorted = useMemo(() => {
    const rows = (targets.data ?? []).slice();
    rows.sort((a, b) => {
      if (a.is_builtin !== b.is_builtin) return a.is_builtin ? -1 : 1;
      if (a.kind !== b.kind) return a.kind.localeCompare(b.kind);
      return a.label.localeCompare(b.label);
    });
    return rows;
  }, [targets.data]);

  // Drop selections that no longer correspond to existing targets — happens
  // after bulk delete or external mutations.
  useEffect(() => {
    if (!targets.data) return;
    const present = new Set(targets.data.map((t) => t.id));
    setSelected((prev) => {
      let changed = false;
      const next = new Set<string>();
      for (const id of prev) {
        if (present.has(id)) next.add(id);
        else changed = true;
      }
      return changed ? next : prev;
    });
  }, [targets.data]);

  const selectableIds = sorted.map((t) => t.id);
  const allSelected = selectableIds.length > 0 && selectableIds.every((id) => selected.has(id));

  const toggleOne = (id: string) =>
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });

  const toggleAll = () =>
    setSelected((prev) => (prev.size === sorted.length ? new Set() : new Set(selectableIds)));

  const selectedTargets = sorted.filter((t) => selected.has(t.id));
  const deletableSelected = selectedTargets.filter((t) => !t.is_builtin);

  const bulkSetEnabled = (enabled: boolean) => {
    for (const t of selectedTargets) {
      update.mutate({ id: t.id, enabled });
    }
  };

  const performBulkDelete = () => {
    for (const t of deletableSelected) {
      del.mutate(t.id);
    }
    setSelected(new Set());
    setBulkDeleteOpen(false);
  };

  return (
    <YStack flex={1}>
      <PageHeader
        title="Targets"
        blurb="Probe destinations. 12 built-in targets ship with Vigil; add your own to test routes that matter to you."
        trailing={
          <Button
            size="$3"
            backgroundColor="$accentBackground"
            color="$accentColor"
            icon={<Plus size={14} color="var(--accentColor)" />}
            onPress={() => setAddOpen(true)}
            hoverStyle={{ opacity: 0.9 }}
          >
            Add target
          </Button>
        }
      />

      <YStack padding="$4" gap="$3" maxWidth={1100} width="100%" alignSelf="center">
        {selected.size > 0 ? (
          <BulkActionBar
            selectedCount={selected.size}
            deletableCount={deletableSelected.length}
            onEnable={() => bulkSetEnabled(true)}
            onDisable={() => bulkSetEnabled(false)}
            onDelete={() => setBulkDeleteOpen(true)}
            onClear={() => setSelected(new Set())}
          />
        ) : null}

        <Card>
          <YStack gap="$1">
            <XStack paddingVertical="$1" paddingHorizontal="$2" gap="$3" alignItems="center">
              <YStack width={32} alignItems="center">
                <Checkbox checked={allSelected} onPress={toggleAll} />
              </YStack>
              <Text width={56} fontSize={10} color="$color8" letterSpacing={0.5} fontWeight="600">
                ON
              </Text>
              <Text flex={2} fontSize={10} color="$color8" letterSpacing={0.5} fontWeight="600">
                LABEL
              </Text>
              <Text width={80} fontSize={10} color="$color8" letterSpacing={0.5} fontWeight="600">
                KIND
              </Text>
              <Text flex={2} fontSize={10} color="$color8" letterSpacing={0.5} fontWeight="600">
                HOST:PORT
              </Text>
              <Text width={120} fontSize={10} color="$color8" letterSpacing={0.5} fontWeight="600">
                ACTIONS
              </Text>
            </XStack>
            {targets.isLoading && !targets.data ? (
              <Text fontSize={11} color="$color8" padding="$2">Loading…</Text>
            ) : sorted.length === 0 ? (
              <Text fontSize={11} color="$color8" padding="$2">No targets configured.</Text>
            ) : (
              sorted.map((t) => (
                <TargetRow
                  key={t.id}
                  target={t}
                  selected={selected.has(t.id)}
                  onToggleSelected={() => toggleOne(t.id)}
                  onToggleEnabled={(enabled) => update.mutate({ id: t.id, enabled })}
                  onDelete={() => setSingleDelete(t)}
                />
              ))
            )}
          </YStack>
        </Card>

        <Text fontSize={11} color="$color8" textAlign="center">
          Builtin targets are protected — they can be disabled but not edited or deleted.
          Disable / re-enable to control which run on each cycle. Sidecar restart needed
          to pick up newly-added targets in the active probe loop.
        </Text>
      </YStack>

      <AddTargetModal open={addOpen} onOpenChange={setAddOpen} />

      <ConfirmDialog
        open={singleDelete != null}
        onOpenChange={(o) => !o && setSingleDelete(null)}
        title={singleDelete ? `Delete "${singleDelete.label}"?` : 'Delete target?'}
        description="This cannot be undone. The probe will stop on the next cycle and historical samples for this target will remain."
        confirmLabel="Delete"
        destructive
        loading={del.isPending}
        onConfirm={() => {
          if (singleDelete) del.mutate(singleDelete.id);
          setSingleDelete(null);
        }}
      />

      <ConfirmDialog
        open={bulkDeleteOpen}
        onOpenChange={setBulkDeleteOpen}
        title={`Delete ${deletableSelected.length} target${deletableSelected.length === 1 ? '' : 's'}?`}
        description={
          deletableSelected.length < selected.size
            ? `${selected.size - deletableSelected.length} builtin target(s) in your selection will be skipped — they can't be deleted.`
            : 'This cannot be undone. Probes stop on the next cycle; historical samples remain.'
        }
        confirmLabel={`Delete ${deletableSelected.length}`}
        destructive
        loading={del.isPending}
        onConfirm={performBulkDelete}
      />
    </YStack>
  );
}

function BulkActionBar({
  selectedCount,
  deletableCount,
  onEnable,
  onDisable,
  onDelete,
  onClear,
}: {
  selectedCount: number;
  deletableCount: number;
  onEnable: () => void;
  onDisable: () => void;
  onDelete: () => void;
  onClear: () => void;
}) {
  return (
    <XStack
      backgroundColor="$color3"
      borderWidth={1}
      borderColor="$accentBackground"
      borderRadius="$3"
      paddingHorizontal="$3"
      paddingVertical="$2"
      alignItems="center"
      gap="$2"
      animation="quick"
    >
      <Text fontSize={12} color="$color12" fontWeight="600">
        {selectedCount} selected
      </Text>
      <YStack flex={1} />
      <Button size="$2" chromeless onPress={onEnable}>
        Enable
      </Button>
      <Button size="$2" chromeless onPress={onDisable}>
        Disable
      </Button>
      <Button
        size="$2"
        chromeless
        onPress={onDelete}
        disabled={deletableCount === 0}
        opacity={deletableCount === 0 ? 0.4 : 1}
      >
        <Text fontSize={11} color="$red10" fontWeight="600">
          Delete
        </Text>
      </Button>
      <YStack width={1} height={20} backgroundColor="$borderColor" marginHorizontal="$1" />
      <Button size="$2" chromeless onPress={onClear}>
        Clear
      </Button>
    </XStack>
  );
}

function Checkbox({ checked, onPress }: { checked: boolean; onPress: () => void }) {
  return (
    <XStack
      width={18}
      height={18}
      borderRadius="$1"
      borderWidth={1}
      borderColor={checked ? '$accentBackground' : '$color7'}
      backgroundColor={checked ? '$accentBackground' : 'transparent'}
      alignItems="center"
      justifyContent="center"
      cursor="pointer"
      hoverStyle={{ borderColor: '$accentBackground' }}
      animation="quick"
      onPress={(e: any) => {
        e?.stopPropagation?.();
        onPress();
      }}
    >
      {checked ? <Check size={11} color="var(--accentColor)" weight="bold" /> : null}
    </XStack>
  );
}

function TargetRow({
  target,
  selected,
  onToggleSelected,
  onToggleEnabled,
  onDelete,
}: {
  target: Target;
  selected: boolean;
  onToggleSelected: () => void;
  onToggleEnabled: (enabled: boolean) => void;
  onDelete: () => void;
}) {
  return (
    <XStack
      paddingVertical="$2"
      paddingHorizontal="$2"
      gap="$3"
      borderRadius="$2"
      hoverStyle={{ backgroundColor: '$color3' }}
      backgroundColor={selected ? '$color3' : 'transparent'}
      alignItems="center"
      animation="quick"
    >
      <YStack width={32} alignItems="center">
        <Checkbox checked={selected} onPress={onToggleSelected} />
      </YStack>
      <YStack width={56}>
        <Switch
          size="$2"
          checked={target.enabled}
          onCheckedChange={onToggleEnabled}
          backgroundColor={target.enabled ? '$accentBackground' : '$color5'}
        >
          <Switch.Thumb animation="quick" />
        </Switch>
      </YStack>
      <XStack flex={2} alignItems="center" gap="$1.5">
        <Text fontSize={12} color="$color12" fontWeight="600" numberOfLines={1}>
          {target.label}
        </Text>
        {target.is_builtin ? (
          <Lock size={11} color="var(--color8)" weight="fill" />
        ) : null}
      </XStack>
      <Text width={80} fontSize={11} color="$color9" letterSpacing={0.5}>
        {target.kind.toUpperCase()}
      </Text>
      <Text flex={2} fontSize={11} color="$color11" fontFamily="$body" numberOfLines={1}>
        {target.host}{target.port ? `:${target.port}` : ''}
      </Text>
      <XStack width={120} gap="$1">
        {target.is_builtin ? (
          <Text fontSize={10} color="$color8" fontStyle="italic">
            built-in
          </Text>
        ) : (
          <Button
            size="$2"
            chromeless
            icon={<Trash size={12} color="var(--red10)" />}
            onPress={onDelete}
          >
            <Text fontSize={11} color="$red10">
              Delete
            </Text>
          </Button>
        )}
      </XStack>
    </XStack>
  );
}

function AddTargetModal({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const create = useCreateTarget();
  const [label, setLabel] = useState('');
  const [kind, setKind] = useState<ProbeKind>('icmp');
  const [host, setHost] = useState('');
  const [port, setPort] = useState('');
  const [error, setError] = useState<string | null>(null);

  const kindOpt = KIND_OPTIONS.find((k) => k.value === kind)!;

  const reset = () => {
    setLabel('');
    setKind('icmp');
    setHost('');
    setPort('');
    setError(null);
  };

  const submit = () => {
    if (!label.trim() || !host.trim()) {
      setError('Label and host are required.');
      return;
    }
    if (kindOpt.needsPort) {
      const p = Number.parseInt(port, 10);
      if (!Number.isFinite(p) || p <= 0 || p > 65535) {
        setError('Port must be between 1 and 65535.');
        return;
      }
      create.mutate(
        { label: label.trim(), kind, host: host.trim(), port: p },
        {
          onSuccess: () => {
            reset();
            onOpenChange(false);
          },
          onError: (e: any) => setError(e.message ?? 'Create failed'),
        },
      );
      return;
    }
    create.mutate(
      { label: label.trim(), kind, host: host.trim() },
      {
        onSuccess: () => {
          reset();
          onOpenChange(false);
        },
        onError: (e: any) => setError(e.message ?? 'Create failed'),
      },
    );
  };

  return (
    <Modal
      open={open}
      onOpenChange={(o) => {
        onOpenChange(o);
        if (!o) reset();
      }}
      title="Add probe target"
      description="Label is for display; host is the destination."
      footer={
        <>
          <Button size="$3" chromeless onPress={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            size="$3"
            backgroundColor="$accentBackground"
            color="$accentColor"
            onPress={submit}
            disabled={create.isPending}
          >
            {create.isPending ? 'Adding…' : 'Add target'}
          </Button>
        </>
      }
    >
      <YStack gap="$3">
        <FormField label="Label" hint="Unique short name. Lowercase, snake_case recommended.">
          <Input
            size="$3"
            placeholder="e.g. my_office_router"
            value={label}
            onChangeText={setLabel}
          />
        </FormField>

        <FormField label="Kind">
          <Select value={kind} onValueChange={(v) => setKind(v as ProbeKind)}>
            <Select.Trigger>
              <Select.Value placeholder="Choose…" />
            </Select.Trigger>
            <Select.Content>
              <Select.Viewport>
                <Select.Group>
                  {KIND_OPTIONS.map((k, i) => (
                    <Select.Item index={i} key={k.value} value={k.value}>
                      <Select.ItemText>{k.label}</Select.ItemText>
                    </Select.Item>
                  ))}
                </Select.Group>
              </Select.Viewport>
            </Select.Content>
          </Select>
        </FormField>

        <FormField label="Host" hint="IP or hostname.">
          <Input
            size="$3"
            placeholder="e.g. 192.168.1.1 or example.com"
            value={host}
            onChangeText={setHost}
          />
        </FormField>

        {kindOpt.needsPort ? (
          <FormField label="Port" hint="Required for TCP/UDP. 1–65535.">
            <Input
              size="$3"
              placeholder="e.g. 443"
              keyboardType="number-pad"
              value={port}
              onChangeText={setPort}
            />
          </FormField>
        ) : null}

        {error ? (
          <Text fontSize={11} color="$red10">
            {error}
          </Text>
        ) : null}
      </YStack>
    </Modal>
  );
}
