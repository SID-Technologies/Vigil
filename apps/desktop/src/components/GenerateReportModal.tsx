import { useState } from 'react';
import { Folder, FileCsv, FileText, FileHtml, Check } from '@phosphor-icons/react';
import { Button, XStack, YStack, Text } from 'tamagui';

import { Toggle } from './Toggle';

import { Modal } from './Modal';
import { reportGenerate, type ReportFormat } from '../lib/ipc';

interface GenerateReportModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  fromMs: number;
  toMs: number;
  /** If empty, all targets are included. */
  targets: string[];
  /** Human-readable description of the window for the modal title. */
  windowLabel: string;
}

/**
 * "Generate report" modal. Three format toggles (CSV/JSON/HTML), an output
 * folder picker (Tauri dialog plugin), and a Generate button that calls the
 * report.generate IPC method.
 *
 * On success: shows the list of written file paths and a "Reveal in Finder"
 * button that opens the folder via shell.open.
 */
export function GenerateReportModal({
  open,
  onOpenChange,
  fromMs,
  toMs,
  targets,
  windowLabel,
}: GenerateReportModalProps) {
  const [formats, setFormats] = useState<Set<ReportFormat>>(new Set(['html', 'csv']));
  const [outDir, setOutDir] = useState<string | null>(null);
  const [generating, setGenerating] = useState(false);
  const [result, setResult] = useState<{ paths: string[] } | null>(null);
  const [error, setError] = useState<string | null>(null);

  const reset = () => {
    setResult(null);
    setError(null);
    setGenerating(false);
  };

  const pickFolder = async () => {
    try {
      const { open: openDialog } = await import('@tauri-apps/plugin-dialog');
      const picked = await openDialog({
        directory: true,
        multiple: false,
        title: 'Where should Vigil save the report?',
      });
      if (typeof picked === 'string') setOutDir(picked);
    } catch (e) {
      setError(`Folder picker failed: ${e instanceof Error ? e.message : String(e)}`);
    }
  };

  const toggleFormat = (f: ReportFormat) => {
    setFormats((prev) => {
      const next = new Set(prev);
      if (next.has(f)) next.delete(f);
      else next.add(f);
      return next;
    });
  };

  const generate = async () => {
    if (!outDir) {
      setError('Pick an output folder first.');
      return;
    }
    if (formats.size === 0) {
      setError('Pick at least one format.');
      return;
    }
    setGenerating(true);
    setError(null);
    try {
      const res = await reportGenerate({
        out_dir: outDir,
        from_ms: fromMs,
        to_ms: toMs,
        targets: targets.length > 0 ? targets : undefined,
        formats: Array.from(formats),
      });
      setResult(res);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setGenerating(false);
    }
  };

  const reveal = async () => {
    if (!outDir) return;
    try {
      const { openPath } = await import('@tauri-apps/plugin-opener');
      await openPath(outDir);
    } catch (e) {
      console.warn('Failed to open folder:', e);
    }
  };

  return (
    <Modal
      open={open}
      onOpenChange={(o) => {
        onOpenChange(o);
        if (!o) reset();
      }}
      title="Generate report"
      description={`Window: ${windowLabel} · ${targets.length === 0 ? 'all targets' : `${targets.length} target${targets.length === 1 ? '' : 's'}`}`}
      width={520}
      footer={
        result ? (
          <>
            <Button size="$3" chromeless onPress={() => onOpenChange(false)}>
              Done
            </Button>
            <Button
              size="$3"
              backgroundColor="$accentBackground"
              color="$accentColor"
              onPress={reveal}
              icon={<Folder size={14} color="var(--accentColor)" />}
            >
              Reveal in Finder
            </Button>
          </>
        ) : (
          <>
            <Button size="$3" chromeless onPress={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button
              size="$3"
              backgroundColor="$accentBackground"
              color="$accentColor"
              onPress={generate}
              disabled={generating || !outDir || formats.size === 0}
            >
              {generating ? 'Generating…' : 'Generate'}
            </Button>
          </>
        )
      }
    >
      {result ? (
        <YStack gap="$2">
          <XStack gap="$2" alignItems="center">
            <Check size={16} color="var(--accentColor)" weight="bold" />
            <Text fontSize={13} color="$color12" fontWeight="600">
              Wrote {result.paths.length} file{result.paths.length === 1 ? '' : 's'}
            </Text>
          </XStack>
          <YStack gap="$1">
            {result.paths.map((p) => (
              <Text key={p} fontSize={11} color="$color9" fontFamily="$body" numberOfLines={1}>
                {p}
              </Text>
            ))}
          </YStack>
        </YStack>
      ) : (
        <YStack gap="$3">
          <YStack gap="$2">
            <Text fontSize={11} color="$color10" letterSpacing={0.4} fontWeight="600">
              FORMATS
            </Text>
            <FormatRow
              icon={<FileHtml size={16} color="var(--color11)" />}
              format="html"
              label="HTML — shareable dashboard"
              hint="Self-contained file with charts, tables, and verdict. Open in any browser."
              checked={formats.has('html')}
              onToggle={() => toggleFormat('html')}
            />
            <FormatRow
              icon={<FileCsv size={16} color="var(--color11)" />}
              format="csv"
              label="CSV — one row per probe"
              hint="Spreadsheet-friendly. Use this if your ISP wants raw data."
              checked={formats.has('csv')}
              onToggle={() => toggleFormat('csv')}
            />
            <FormatRow
              icon={<FileText size={16} color="var(--color11)" />}
              format="json"
              label="JSON — structured payload"
              hint="Includes summary, per-target stats, hourly buckets, outages, raw probes."
              checked={formats.has('json')}
              onToggle={() => toggleFormat('json')}
            />
          </YStack>

          <YStack gap="$1.5">
            <Text fontSize={11} color="$color10" letterSpacing={0.4} fontWeight="600">
              OUTPUT FOLDER
            </Text>
            <XStack
              padding="$2.5"
              borderRadius="$2"
              backgroundColor="$color3"
              borderWidth={1}
              borderColor="$borderColor"
              gap="$2"
              alignItems="center"
              cursor="pointer"
              hoverStyle={{ borderColor: '$color8' }}
              onPress={pickFolder}
            >
              <Folder size={14} color="var(--color9)" />
              <Text fontSize={11} color={outDir ? '$color12' : '$color8'} flex={1} numberOfLines={1}>
                {outDir ?? 'Click to choose…'}
              </Text>
              <Text fontSize={10} color="$color8">change</Text>
            </XStack>
          </YStack>

          {error ? (
            <Text fontSize={11} color="$red10">
              {error}
            </Text>
          ) : null}
        </YStack>
      )}
    </Modal>
  );
}

function FormatRow({
  icon,
  format,
  label,
  hint,
  checked,
  onToggle,
}: {
  icon: React.ReactNode;
  format: ReportFormat;
  label: string;
  hint: string;
  checked: boolean;
  onToggle: () => void;
}) {
  return (
    <XStack
      padding="$2.5"
      borderRadius="$2"
      borderWidth={1}
      borderColor={checked ? '$accentBackground' : '$borderColor'}
      backgroundColor={checked ? '$color3' : 'transparent'}
      gap="$2"
      alignItems="center"
      cursor="pointer"
      hoverStyle={{ backgroundColor: '$color3' }}
      onPress={onToggle}
      animation="quick"
    >
      {icon}
      <YStack flex={1} gap="$0.5">
        <Text fontSize={12} color="$color12" fontWeight="600">
          {label}
        </Text>
        <Text fontSize={10} color="$color9">
          {hint}
        </Text>
      </YStack>
      <Toggle checked={checked} onCheckedChange={onToggle} />
    </XStack>
  );
}
