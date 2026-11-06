import { message } from 'antd';
import { CircleCheckIcon, CircleXIcon, InfoIcon, TriangleAlertIcon } from 'lucide-react';
import type { CSSProperties, ReactNode } from 'react';

/**
 * Unified top-center transient feedback toasts.
 *
 * Every transient toast renders through the single global static holder
 * configured in app.tsx (dark algorithm, z-index 1100), with lucide outline
 * icons matching the persistent video-status bar in pages/desktop/message.tsx.
 * See .trellis/spec/frontend/feedback-notifications.md for the full contract.
 */
/**
 * Ant Design only spaces an icon that carries its own `.anticon` class; these
 * Lucide glyphs are plain SVGs, so the gap to the text is added here.
 */
const spaced = (icon: ReactNode) => <span className="mr-2 flex items-center">{icon}</span>;

const icons = {
  success: spaced(<CircleCheckIcon size={16} className="text-green-500/80" />),
  error: spaced(<CircleXIcon size={16} className="text-red-500/80" />),
  warning: spaced(<TriangleAlertIcon size={16} className="text-yellow-500/80" />),
  info: spaced(<InfoIcon size={16} className="text-blue-500/80" />)
} as const;

type FeedbackType = keyof typeof icons;

type FeedbackOptions = {
  /** Seconds before auto-dismiss; defaults to 3. */
  duration?: number;
  /** Dedupe key: opening a toast with an existing key replaces it in place. */
  key?: string;
  /** Reserved for the mid-screen pointer-lock prompt; do not use elsewhere. */
  style?: CSSProperties;
};

const open = (type: FeedbackType, content: ReactNode, options?: number | FeedbackOptions) => {
  const { duration, key, style }: FeedbackOptions =
    typeof options === 'number' ? { duration: options } : (options ?? {});

  message.open({ type, icon: icons[type], content, duration: duration ?? 3, key, style });
};

export const feedback = {
  success: (content: ReactNode, options?: number | FeedbackOptions) =>
    open('success', content, options),
  error: (content: ReactNode, options?: number | FeedbackOptions) =>
    open('error', content, options),
  warning: (content: ReactNode, options?: number | FeedbackOptions) =>
    open('warning', content, options),
  info: (content: ReactNode, options?: number | FeedbackOptions) =>
    open('info', content, options)
};
