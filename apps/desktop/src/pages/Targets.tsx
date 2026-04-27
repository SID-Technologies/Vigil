import { useEffect, useMemo, useState } from 'react';
import { Plus, Trash, Lock, Check } from '@phosphor-icons/react';
import { Button, Input, Select, XStack, YStack, Text } from 'tamagui';

import { Card } from '../components/Card';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { FormField } from '../components/FormField';
import { Modal } from '../components/Modal';
import { PageHeader } from '../components/PageHeader';
import { RowSkeleton } from '../components/Skeleton';
import { Toggle } from '../components/Toggle';
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
      {selected.size > 0 ? (
        <SelectionHeader
          selectedCount={selected.size}
          deletableCount={deletableSelected.length}
          onEnable={() => bulkSetEnabled(true)}
          onDisable={() => bulkSetEnabled(false)}
          onDelete={() => setBulkDeleteOpen(true)}
          onClear={() => setSelected(new Set())}
        />
      ) : (
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
      )}

      <YStack padding="$4" gap="$3" maxWidth={1100} width="100%" alignSelf="center">
        <Card>
          <YStack gap="$1">
            <XStack paddingVertical="$1" paddingHorizontal="$2" gap="$3" alignItems="center">
              <YStack width={32} alignItems="center">
                <Checkbox checked={allSelected} onPress={toggleAll} />
              </YStack>
              <YStack width={56}>
                <HeaderCell>ON</HeaderCell>
              </YStack>
              <YStack flexBasis={0} flexGrow={2} minWidth={180}>
                <HeaderCell>LABEL</HeaderCell>
              </YStack>
              <YStack width={80}>
                <HeaderCell>KIND</HeaderCell>
              </YStack>
              <YStack flexBasis={0} flexGrow={2} minWidth={180}>
                <HeaderCell>HOST:PORT</HeaderCell>
              </YStack>
              <YStack width={120}>
                <HeaderCell>ACTIONS</HeaderCell>
              </YStack>
            </XStack>
            {targets.isLoading && !targets.data ? (
              <YStack gap="$1.5" padding="$1">
                <RowSkeleton />
                <RowSkeleton />
                <RowSkeleton />
              </YStack>
            ) : sorted.length === 0 ? (
              <Text fontSize={11} color="$color8" padding="$2">
                No targets yet. Click "Add target" to start monitoring something.
              </Text>
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

/**
 * SelectionHeader — replaces the page's normal header when one or more
 * targets are checkbox-selected. Same vertical footprint as PageHeader so
 * there's zero layout shift when toggling between them.
 *
 * Pattern matches Linear / Gmail / Notion: when you start selecting rows,
 * the page header transforms to show "N selected" + bulk actions. Way more
 * discoverable than a separate floating bar, and the user knows exactly
 * how to exit selection mode (Clear, top-right).
 */
function SelectionHeader({
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
      paddingHorizontal="$4"
      paddingVertical="$3"
      borderBottomWidth={1}
      borderBottomColor="$accentBackground"
      backgroundColor="$color3"
      alignItems="center"
      justifyContent="space-between"
      gap="$3"
      animation="quick"
    >
      <YStack gap="$0.5" flex={1}>
        <Text fontSize={20} fontWeight="700" color="$color12" fontFamily="$heading">
          {selectedCount} target{selectedCount === 1 ? '' : 's'} selected
        </Text>
        <Text fontSize={12} color="$color9">
          {deletableCount === selectedCount
            ? 'All selected items can be deleted.'
            : `${selectedCount - deletableCount} builtin item(s) will be skipped on delete.`}
        </Text>
      </YStack>
      <XStack gap="$2" alignItems="center">
        <Button size="$3" chromeless onPress={onEnable}>
          Enable
        </Button>
        <Button size="$3" chromeless onPress={onDisable}>
          Disable
        </Button>
        <Button
          size="$3"
          chromeless
          onPress={onDelete}
          disabled={deletableCount === 0}
          opacity={deletableCount === 0 ? 0.4 : 1}
        >
          <Text fontSize={13} color="$red10" fontWeight="600">
            Delete
          </Text>
        </Button>
        <YStack width={1} height={20} backgroundColor="$borderColor" marginHorizontal="$1" />
        <Button
          size="$3"
          backgroundColor="$accentBackground"
          color="$accentColor"
          onPress={onClear}
        >
          Clear
        </Button>
      </XStack>
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
        <Toggle checked={target.enabled} onCheckedChange={onToggleEnabled} />
      </YStack>
      <XStack flexBasis={0} flexGrow={2} minWidth={180} alignItems="center" gap="$1.5">
        <Text fontSize={12} color="$color12" fontWeight="600" numberOfLines={1}>
          {target.label}
        </Text>
        {target.is_builtin ? (
          <Lock size={11} color="var(--color8)" weight="fill" />
        ) : null}
      </XStack>
      <YStack width={80}>
        <Text fontSize={11} color="$color9" letterSpacing={0.5}>
          {target.kind.toUpperCase()}
        </Text>
      </YStack>
      <YStack flexBasis={0} flexGrow={2} minWidth={180}>
        <Text fontSize={11} color="$color11" fontFamily="$body" numberOfLines={1}>
          {target.host}{target.port ? `:${target.port}` : ''}
        </Text>
      </YStack>
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

/**
 * Header cell typography. Wrapped in YStack of fixed/flex width by the
 * caller so the column constraints match the row exactly.
 */
function HeaderCell({ children }: { children: React.ReactNode }) {
  return (
    <Text fontSize={10} color="$color8" letterSpacing={0.5} fontWeight="600" numberOfLines={1}>
      {children}
    </Text>
  );
}

// Label rules: short identifier shown in chips and used as a query
// discriminator. Whitespace would break URL/event-name encoding; the 40-char
// cap keeps chip widths sane. Snake_case is a recommendation, not a hard rule.
const LABEL_MAX = 40;
const LABEL_PATTERN = /^[A-Za-z0-9_\-.]+$/;

// Host: don't try to fully parse — the sidecar resolves DNS and surfaces
// real errors. Just block empties and obvious typos (spaces, schemes).
function validateHost(raw: string): string | undefined {
  const v = raw.trim();
  if (!v) return 'Required.';
  if (/\s/.test(v)) return 'No spaces allowed.';
  if (/^[a-z]+:\/\//i.test(v)) return 'Drop the scheme — host only.';
  if (v.length > 253) return 'Too long.';
  return undefined;
}

function validateLabel(raw: string): string | undefined {
  const v = raw.trim();
  if (!v) return 'Required.';
  if (v.length > LABEL_MAX) return `At most ${LABEL_MAX} characters.`;
  if (!LABEL_PATTERN.test(v)) return 'Letters, numbers, _ - . only.';
  return undefined;
}

function validatePort(raw: string): string | undefined {
  const v = raw.trim();
  if (!v) return 'Required.';
  if (!/^\d+$/.test(v)) return 'Whole numbers only.';
  const n = Number.parseInt(v, 10);
  if (n < 1 || n > 65535) return 'Range 1–65535.';
  return undefined;
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
  // Server-side errors (e.g. duplicate label) live separately from
  // client-side validation since they only land after a submit attempt.
  const [serverError, setServerError] = useState<string | null>(null);
  // Track which fields the user has interacted with — avoids screaming
  // "Required" on a freshly-opened, empty modal.
  const [touched, setTouched] = useState<Record<string, boolean>>({});

  const kindOpt = KIND_OPTIONS.find((k) => k.value === kind)!;

  const labelErr = validateLabel(label);
  const hostErr = validateHost(host);
  const portErr = kindOpt.needsPort ? validatePort(port) : undefined;
  const hasErrors = !!(labelErr || hostErr || portErr);

  const reset = () => {
    setLabel('');
    setKind('icmp');
    setHost('');
    setPort('');
    setServerError(null);
    setTouched({});
  };

  const submit = () => {
    setServerError(null);
    if (hasErrors) {
      // Reveal all errors at once on failed submit.
      setTouched({ label: true, host: true, port: true });
      return;
    }
    const payload: { label: string; kind: ProbeKind; host: string; port?: number } = {
      label: label.trim(),
      kind,
      host: host.trim(),
    };
    if (kindOpt.needsPort) {
      payload.port = Number.parseInt(port.trim(), 10);
    }
    create.mutate(payload, {
      onSuccess: () => {
        reset();
        onOpenChange(false);
      },
      onError: (e: any) => setServerError(e.message ?? 'Create failed'),
    });
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
            backgroundColor={hasErrors ? '$color5' : '$accentBackground'}
            color={hasErrors ? '$color9' : '$accentColor'}
            onPress={submit}
            disabled={create.isPending || hasErrors}
          >
            {create.isPending ? 'Adding…' : 'Add target'}
          </Button>
        </>
      }
    >
      <YStack gap="$3">
        <FormField
          label="Label"
          hint="Unique short name. Lowercase, snake_case recommended."
          error={touched.label ? labelErr : undefined}
        >
          <Input
            size="$3"
            placeholder="e.g. my_office_router"
            value={label}
            maxLength={LABEL_MAX}
            borderColor={touched.label && labelErr ? '$red8' : undefined}
            onChangeText={setLabel}
            onBlur={() => setTouched((t) => ({ ...t, label: true }))}
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

        <FormField
          label="Host"
          hint="IP or hostname."
          error={touched.host ? hostErr : undefined}
        >
          <Input
            size="$3"
            placeholder="e.g. 192.168.1.1 or example.com"
            value={host}
            borderColor={touched.host && hostErr ? '$red8' : undefined}
            onChangeText={setHost}
            onBlur={() => setTouched((t) => ({ ...t, host: true }))}
          />
        </FormField>

        {kindOpt.needsPort ? (
          <FormField
            label="Port"
            hint="Required for TCP/UDP. 1–65535."
            error={touched.port ? portErr : undefined}
          >
            <Input
              size="$3"
              placeholder="e.g. 443"
              keyboardType="number-pad"
              value={port}
              borderColor={touched.port && portErr ? '$red8' : undefined}
              onChangeText={setPort}
              onBlur={() => setTouched((t) => ({ ...t, port: true }))}
            />
          </FormField>
        ) : null}

        {serverError ? (
          <Text fontSize={11} color="$red10">
            {serverError}
          </Text>
        ) : null}
      </YStack>
    </Modal>
  );
}
