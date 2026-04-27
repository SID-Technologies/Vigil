import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Eye, ArrowSquareOut, Clock, ArrowRight, ArrowLeft } from '@phosphor-icons/react';
import { Button, Dialog, Unspaced, XStack, YStack, Text } from 'tamagui';

const STORAGE_KEY = 'vigil:welcome-seen';

interface Step {
  title: string;
  body: string;
  Icon: typeof Eye;
}

// Three steps — anything more is a manual unless someone insists. The job is
// to land Jordan-the-friend on the dashboard with the three things they need
// to know to use Vigil at all: what they're looking at, how to drill in, and
// why outages are recorded the way they are.
const STEPS: Step[] = [
  {
    title: 'Vigil watches your network',
    body:
      "Every couple of seconds, Vigil checks a handful of places — your router, public DNS, the wider internet. Green tiles mean the path's healthy. Red tiles mean it's not. The chart shows latency over time so you can spot when things slow down.",
    Icon: Eye,
  },
  {
    title: 'Click any tile for the full story',
    body:
      "Each tile on the dashboard is a doorway. Click one to jump to History — same target, longer window, full latency curve and percentiles. Use it when you want to know whether tonight's stutter is a pattern or a fluke.",
    Icon: ArrowSquareOut,
  },
  {
    title: 'Outages get recorded automatically',
    body:
      'When three checks in a row fail, Vigil opens an outage and timestamps it. When the next check succeeds, it closes. The Outages page is your receipt — exact start, exact end, duration, error breakdown. The kind of evidence ISPs and property managers actually answer to.',
    Icon: Clock,
  },
];

/**
 * WelcomeTour — first-launch 3-step modal explaining what Vigil does, how to
 * navigate it, and why outages matter. Mounted at the App level so it shows
 * regardless of which route the user lands on first.
 *
 * Persistence: a single localStorage flag (vigil:welcome-seen). Set when the
 * user finishes or skips. We deliberately don't tie this to sample data
 * presence — even on a clean reinstall, returning users shouldn't see the
 * tour again unless they manually clear it.
 */
export function WelcomeTour() {
  const [open, setOpen] = useState(false);
  const [step, setStep] = useState(0);
  const navigate = useNavigate();

  // Decide on mount whether to show the tour. Defer one tick so the rest of
  // the app paints first — the modal arriving 50ms after first paint feels
  // intentional rather than blocking.
  useEffect(() => {
    const seen = (() => {
      try {
        return window.localStorage.getItem(STORAGE_KEY);
      } catch {
        return 'unknown';
      }
    })();
    if (seen) return;
    const t = setTimeout(() => setOpen(true), 200);
    return () => clearTimeout(t);
  }, []);

  const dismiss = () => {
    try {
      window.localStorage.setItem(STORAGE_KEY, String(Date.now()));
    } catch {
      // localStorage disabled — the tour will reappear next launch, which
      // is acceptable.
    }
    setOpen(false);
    setStep(0);
  };

  const finish = () => {
    dismiss();
    // Land on Dashboard so the user has somewhere to start. If they're
    // already there, navigate is a no-op.
    navigate('/');
  };

  const isLast = step === STEPS.length - 1;
  const current = STEPS[step];
  const Icon = current.Icon;

  return (
    <Dialog modal open={open} onOpenChange={(o) => (o ? setOpen(true) : dismiss())}>
      <Dialog.Portal>
        <Dialog.Overlay
          key="overlay"
          animation="quick"
          opacity={0.7}
          backgroundColor="$color1"
          enterStyle={{ opacity: 0 }}
          exitStyle={{ opacity: 0 }}
        />
        <Dialog.Content
          key="content"
          backgroundColor="$color2"
          borderColor="$borderColor"
          borderWidth={1}
          borderRadius="$3"
          padding="$5"
          width={520}
          gap="$4"
          animation="quick"
          enterStyle={{ y: -8, opacity: 0 }}
          exitStyle={{ y: -8, opacity: 0 }}
        >
          {/* Step indicator pills + skip button */}
          <XStack alignItems="center" justifyContent="space-between">
            <XStack gap="$1.5" alignItems="center">
              {STEPS.map((_, i) => (
                <YStack
                  key={i}
                  width={i === step ? 24 : 8}
                  height={4}
                  borderRadius={999}
                  backgroundColor={i <= step ? '$accentBackground' : '$color5'}
                  animation="quick"
                />
              ))}
            </XStack>
            <Unspaced>
              <Button size="$2" chromeless onPress={dismiss}>
                <Text fontSize={11} color="$color9">Skip tour</Text>
              </Button>
            </Unspaced>
          </XStack>

          {/* Hero icon — large, amber, sets the watchman tone */}
          <YStack
            alignItems="center"
            justifyContent="center"
            width={56}
            height={56}
            borderRadius="$3"
            backgroundColor="$color3"
            borderWidth={1}
            borderColor="$accentBackground"
          >
            <Icon size={28} color="var(--accentColor)" weight="duotone" />
          </YStack>

          <YStack gap="$2">
            <Dialog.Title>
              <Text fontSize={20} fontWeight="700" color="$color12" fontFamily="$heading">
                {current.title}
              </Text>
            </Dialog.Title>
            <Dialog.Description>
              <Text fontSize={13} color="$color11" lineHeight={20}>
                {current.body}
              </Text>
            </Dialog.Description>
          </YStack>

          <XStack justifyContent="space-between" alignItems="center" paddingTop="$2">
            <Text fontSize={11} color="$color8">
              Step {step + 1} of {STEPS.length}
            </Text>
            <XStack gap="$2">
              {step > 0 ? (
                <Button
                  size="$3"
                  chromeless
                  icon={<ArrowLeft size={12} color="var(--color9)" />}
                  onPress={() => setStep((s) => Math.max(0, s - 1))}
                >
                  <Text fontSize={12} color="$color11">Back</Text>
                </Button>
              ) : null}
              <Button
                size="$3"
                backgroundColor="$accentBackground"
                color="$accentColor"
                iconAfter={
                  isLast ? undefined : (
                    <ArrowRight size={12} color="var(--accentColor)" />
                  )
                }
                onPress={() => {
                  if (isLast) finish();
                  else setStep((s) => s + 1);
                }}
              >
                {isLast ? "Got it — let's go" : 'Next'}
              </Button>
            </XStack>
          </XStack>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog>
  );
}

/**
 * Programmatic helper — clears the welcome flag so the tour fires on next
 * launch. Wire this to a "Show welcome tour" button in Settings if/when we
 * want users to be able to replay it.
 */
export function resetWelcomeTour() {
  try {
    window.localStorage.removeItem(STORAGE_KEY);
  } catch {
    // ignore
  }
}
