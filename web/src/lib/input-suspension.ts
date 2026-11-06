import { setInputRouterEnabled } from '@/lib/input-router.ts';

// Owner-counted suspension layered over the input router's base enable state.
//
// The router reports enabled only when the pointer-mode base state allows input
// AND no owner holds a suspension. Base-state writers (pointer-mode controller,
// pointer-mode selection) express the device-driven desired state; a suspension
// holder (for example the onboarding tour) force-disables every HID path —
// browser keyboard/pointer, virtual keyboard, menu action senders, and WebHID —
// until it releases. Neither side can stomp the other: a base re-derive or a
// controller unmount cleanup can never re-enable input while a suspension is
// held, and releasing a suspension only restores the current base state.

const suspensionOwners = new Set<string>();
let baseEnabled = true;

function publishInputRouterState() {
  setInputRouterEnabled(baseEnabled && suspensionOwners.size === 0);
}

// Drives the device-authoritative base state. Former direct writers of
// setInputRouterEnabled (pointer-mode controller/selection) call this instead.
export function setBaseInputEnabled(enabled: boolean) {
  if (baseEnabled === enabled) return;
  baseEnabled = enabled;
  publishInputRouterState();
}

// Acquires an owner-counted suspension. The returned release function is
// idempotent; an owner should hold at most one suspension at a time.
export function suspendInput(owner: string): () => void {
  if (owner.length === 0) throw new Error('suspendInput requires a non-empty owner');

  suspensionOwners.add(owner);
  publishInputRouterState();

  let held = true;
  return () => {
    if (!held) return;
    held = false;
    suspensionOwners.delete(owner);
    publishInputRouterState();
  };
}

export function isInputSuspended() {
  return suspensionOwners.size > 0;
}
