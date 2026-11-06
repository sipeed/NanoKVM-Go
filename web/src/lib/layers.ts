/**
 * Global stacking (z-index) scale for the web UI.
 *
 * Ant Design anchors the scale: its seed token `zIndexPopupBase` stays at the
 * default 1000. Standalone (first-level) Ant Design popups take their z-index
 * from the component stylesheet: Modal/Drawer at 1000, Popover at 1030,
 * Tooltip/Popconfirm at 1070, Select/Dropdown/DatePicker popups at 1050,
 * Image preview at 1001. From the second nesting level, `useZIndex` applies
 * inline stacking via zindexContext: a Modal/Popover inside a Modal renders at
 * 1200, a Select dropdown inside a Modal at 1150 (both browser-verified).
 * Feedback components are governed by theme tokens in `app.tsx`: Message and
 * Notification are both pinned to 1100 — above standalone Modals (1000) and
 * the menu rail, below nested containers (1200).
 *
 * Hand-written layers MUST come from this module instead of bare numbers.
 *
 * Band map (low → high):
 *
 *   screen-local (not exported, live inside screen stacking contexts):
 *     video viewport (auto) → no-signal screen (z-10) → input capture (z-20)
 *
 *   998  LAYER_VIDEO_STATUS_MESSAGE  persistent video-status notice host
 *                                    (pages/desktop/message.tsx); below the menu
 *                                    rail by design, constrained by its own
 *                                    relative stacking context
 *   999  LAYER_DESKTOP_MENU_POPOVER  desktop menu popovers; above the
 *                                    video-status host (998), below the rail
 *        LAYER_VIRTUAL_KEYBOARD      virtual keyboard drawer (vaul); below the
 *                                    rail so the menu stays reachable
 *   1000 LAYER_MENU_RAIL             desktop top bar + mobile side rail; equals
 *                                    Ant Design zIndexPopupBase by intent
 *   1040 LAYER_ONBOARDING_MOBILE_PROMPT  mobile portrait onboarding soft-prompt
 *                                    card; above the rail (1000) and standalone
 *                                    Popovers (1030), below Select/Dropdown
 *                                    popups (1050) so the card's LanguagePicker
 *                                    menu stays clickable
 *   1100 LAYER_MOBILE_MENU_POPOVER   mobile rail popovers; above the rail so
 *                                    panels stay interactive; same altitude as
 *                                    Message/Notification feedback tokens
 *   1199 LAYER_OVERLAY_BACKDROP      calibration transparent capture layer
 *   1200 LAYER_OVERLAY               full-screen interaction overlays that own
 *                                    input while active: calibration dim/frame,
 *                                    portal selection, timelapse loading veil
 *                                    (must cover the settings Modal at 1100)
 *   1201 LAYER_OVERLAY_HINT          calibration hint bar
 *   1202 LAYER_ONBOARDING_BACKDROP   onboarding transparent pointer shield;
 *                                    fully transparent (the tour mask owns
 *                                    dimming), swallows pointer/wheel/context
 *                                    events outside the tour panel
 *   1210 LAYER_ONBOARDING            onboarding tour mask + popover (antd Tour
 *                                    zIndex); above every overlay and menu
 *                                    surface, below the recorder
 *   1211 LAYER_ONBOARDING_RING       mobile tour's self-rendered spotlight
 *                                    ring; above the mask so it stays bright
 *   9999 LAYER_RECORDER              keyboard macro recorder indicator;
 *                                    deliberately topmost, above feedback and
 *                                    every Ant Design layer
 *
 * Collision note: Ant Design's second nesting level also lands on 1200
 * (Modal/Popover inside a Modal). LAYER_OVERLAY surfaces are modal-blocking
 * full-screen interactions; when they activate, any Ant Design popup mounted
 * later wins by DOM order, which is acceptable because the overlay owns the
 * input anyway.
 */

/** Persistent video-status notice host (pages/desktop/message.tsx). */
export const LAYER_VIDEO_STATUS_MESSAGE = 998;

/** Desktop menu popovers: above the video-status host (998), below the rail (1000). */
export const LAYER_DESKTOP_MENU_POPOVER = 999;

/** Virtual keyboard drawer content; below the menu rail so the menu stays reachable. */
export const LAYER_VIRTUAL_KEYBOARD = 999;

/** Menu rail (desktop top bar + mobile side rail); equals Ant Design zIndexPopupBase. */
export const LAYER_MENU_RAIL = 1000;

/**
 * Mobile portrait onboarding soft-prompt card: above the rail, below Ant Design
 * Select/Dropdown popups (1050) so the card's own LanguagePicker menu is not
 * painted over (the desktop prompt card keeps the same invariant at 999).
 */
export const LAYER_ONBOARDING_MOBILE_PROMPT = 1040;

/** Mobile rail popovers sit above the rail itself so panels stay interactive. */
export const LAYER_MOBILE_MENU_POPOVER = 1100;

/** Calibration overlay: transparent pointer-capture layer. */
export const LAYER_OVERLAY_BACKDROP = 1199;

/** Full-screen interaction overlays: calibration dim/frame, portal selection, timelapse veil. */
export const LAYER_OVERLAY = 1200;

/** Calibration overlay: bottom hint bar. */
export const LAYER_OVERLAY_HINT = 1201;

/** Onboarding tour: fully transparent full-screen pointer shield below the tour mask. */
export const LAYER_ONBOARDING_BACKDROP = 1202;

/** Onboarding tour mask and popover (antd Tour zIndex). */
export const LAYER_ONBOARDING = 1210;

/** Mobile tour spotlight ring (self-rendered; above the tour mask). */
export const LAYER_ONBOARDING_RING = 1211;

/** Keyboard macro recorder indicator; deliberately topmost. */
export const LAYER_RECORDER = 9999;
