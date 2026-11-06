// Pure derivation of the onboarding tour steps. Mirrors the visibility rules
// of renderMenuItems in pages/desktop/menu/index.tsx exactly, so the tour can
// never name or spotlight an item that is not on the rail.

export const ONBOARDING_VERSION = 'v1';

export type OnboardingGroup = 'display' | 'media' | 'input' | 'tools' | 'automation' | 'system';

export type OnboardingStepDef = {
  key: 'welcome' | OnboardingGroup | 'finish';
  /** Rail group to spotlight; absent for the centered welcome/finish cards. */
  group?: OnboardingGroup;
  /** Item line keys under onboarding.steps.<group>.items (visible items only). */
  items: string[];
};

export type OnboardingStepInput = {
  disabledItems: string[];
  videoMode: string | null;
  screenTimelapseEnabled: boolean;
  /**
   * The rail renders its mobile variant (no Fullscreen/Collapse) in mobile
   * portrait even on devices where the owner gate still allows the tour
   * (e.g. a large rotated touchscreen), so the system step must mirror it.
   */
  isMobilePortrait: boolean;
};

export function deriveOnboardingSteps(input: OnboardingStepInput): OnboardingStepDef[] {
  const isEnabled = (key: string) => !input.disabledItems.includes(key);
  const isWebRTCVideoMode = input.videoMode === 'h264-webrtc' || input.videoMode === 'h265-webrtc';

  const steps: OnboardingStepDef[] = [{ key: 'welcome', items: [] }];

  // display: Screen + Portal are unconditional; Picture-in-Picture is gated.
  steps.push({
    key: 'display',
    group: 'display',
    items: ['screen', 'portal', ...(isEnabled('pictureInPicture') ? ['pictureInPicture'] : [])]
  });

  // media: Volume and Microphone only exist in WebRTC video modes.
  const mediaItems = [
    ...(isEnabled('volume') && isWebRTCVideoMode ? ['volume'] : []),
    ...(isEnabled('microphone') && isWebRTCVideoMode ? ['microphone'] : [])
  ];
  if (mediaItems.length > 0) {
    steps.push({ key: 'media', group: 'media', items: mediaItems });
  }

  // input: Keyboard, Mouse, and Preview are unconditional.
  steps.push({ key: 'input', group: 'input', items: ['keyboard', 'mouse', 'preview'] });

  // tools: each tool is gated individually; the step exists when any shows.
  const toolItems = ['image', 'script', 'terminal', 'wol'].filter(isEnabled);
  if (toolItems.length > 0) {
    steps.push({ key: 'tools', group: 'tools', items: toolItems });
  }

  // automation: Screen Timelapse needs the AI service flag; the power key
  // (Finger Robot) is disabled by default.
  const automationItems = [
    ...(input.screenTimelapseEnabled && isEnabled('screenTimelapse') ? ['screenTimelapse'] : []),
    ...(isEnabled('power') ? ['power'] : [])
  ];
  if (automationItems.length > 0) {
    steps.push({ key: 'automation', group: 'automation', items: automationItems });
  }

  // system: Settings is unconditional; Fullscreen/Collapse render only in the
  // rail's desktop variant (variant === 'desktop' in renderMenuItems) and are
  // individually gated.
  steps.push({
    key: 'system',
    group: 'system',
    items: [
      'settings',
      ...(!input.isMobilePortrait && isEnabled('fullscreen') ? ['fullscreen'] : []),
      ...(!input.isMobilePortrait && isEnabled('collapse') ? ['collapse'] : [])
    ]
  });

  steps.push({ key: 'finish', items: [] });
  return steps;
}
