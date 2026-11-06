import { useEffect, useRef } from 'react';
import { Modal } from 'antd';

const UPDATE_COUNTDOWN_SECONDS = 5;
const COUNTDOWN_TICK_INTERVAL_MS = 250;

type ConfirmAction = () => void | Promise<unknown>;

interface CountdownConfirmationLabels {
  title: string;
  description: string;
  ok: string;
  okCountdown: (seconds: number) => string;
  cancel: string;
}

export function useCountdownConfirmation(labels: CountdownConfirmationLabels) {
  const [modal, contextHolder] = Modal.useModal();
  const countdownTimerRef = useRef<number | null>(null);
  const activeDialogRef = useRef<ReturnType<typeof modal.confirm> | null>(null);

  useEffect(() => {
    return () => {
      if (countdownTimerRef.current !== null) {
        window.clearInterval(countdownTimerRef.current);
      }
      activeDialogRef.current?.destroy();
    };
  }, []);

  function clearCountdown() {
    if (countdownTimerRef.current === null) return;

    window.clearInterval(countdownTimerRef.current);
    countdownTimerRef.current = null;
  }

  function confirm(onConfirm: ConfirmAction) {
    if (activeDialogRef.current !== null) return;

    clearCountdown();

    let remainingSeconds = UPDATE_COUNTDOWN_SECONDS;
    const countdownDeadline = Date.now() + UPDATE_COUNTDOWN_SECONDS * 1000;
    let dialog: ReturnType<typeof modal.confirm>;

    dialog = modal.confirm({
      title: labels.title,
      content: <span className="text-xs text-neutral-500">{labels.description}</span>,
      okText: labels.okCountdown(remainingSeconds),
      okButtonProps: { disabled: true },
      cancelText: labels.cancel,
      autoFocusButton: 'cancel',
      onOk: () => {
        clearCountdown();
        return onConfirm();
      },
      onCancel: clearCountdown,
      afterClose: () => {
        if (activeDialogRef.current !== dialog) return;

        clearCountdown();
        activeDialogRef.current = null;
      }
    });
    activeDialogRef.current = dialog;

    countdownTimerRef.current = window.setInterval(() => {
      if (activeDialogRef.current !== dialog) {
        clearCountdown();
        return;
      }

      const nextRemainingSeconds = Math.max(0, Math.ceil((countdownDeadline - Date.now()) / 1000));
      if (nextRemainingSeconds === remainingSeconds) return;

      remainingSeconds = nextRemainingSeconds;
      const countdownFinished = remainingSeconds <= 0;

      dialog.update((previous) => ({
        ...previous,
        okText: countdownFinished ? labels.ok : labels.okCountdown(remainingSeconds),
        okButtonProps: {
          ...previous.okButtonProps,
          disabled: !countdownFinished
        }
      }));

      if (countdownFinished) {
        clearCountdown();
      }
    }, COUNTDOWN_TICK_INTERVAL_MS);
  }

  return { confirm, contextHolder };
}
