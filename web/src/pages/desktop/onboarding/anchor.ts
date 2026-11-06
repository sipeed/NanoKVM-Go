export type UnionRect = {
  left: number;
  top: number;
  width: number;
  height: number;
};

// Union of the rendered children rects of every wrapper carrying
// data-menu-group="<group>". The wrappers are display:contents (their own
// box is 0x0) and a group can span two non-contiguous wrappers (automation),
// so measure the children — never the wrappers, and never with a single-node
// query. Returns null when the group renders nothing measurable.
export function measureMenuGroupRect(group: string): UnionRect | null {
  const wrappers = document.querySelectorAll(`[data-menu-group="${group}"]`);

  let left = Infinity;
  let top = Infinity;
  let right = -Infinity;
  let bottom = -Infinity;

  wrappers.forEach((wrapper) => {
    for (const child of Array.from(wrapper.children)) {
      const rect = child.getBoundingClientRect();
      if (rect.width === 0 && rect.height === 0) continue;
      left = Math.min(left, rect.left);
      top = Math.min(top, rect.top);
      right = Math.max(right, rect.right);
      bottom = Math.max(bottom, rect.bottom);
    }
  });

  if (!Number.isFinite(left) || !Number.isFinite(right)) return null;
  return { left, top, width: right - left, height: bottom - top };
}

// Whole-box rect of the expanded mobile rail (a real element, unlike the
// display:contents group wrappers). Used by the mobile tour's rail step and
// its placement decision (left vs right of the rail).
export function measureMenuRailRect(): UnionRect | null {
  const node = document.querySelector('[data-menu-rail="mobile"]');
  if (!node) return null;
  const rect = node.getBoundingClientRect();
  if (rect.width === 0 && rect.height === 0) return null;
  return { left: rect.left, top: rect.top, width: rect.width, height: rect.height };
}
