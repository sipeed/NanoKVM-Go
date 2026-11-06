const dotSegmentPattern = /(^|\/)\.{1,2}(\/|$)/;
const scriptDestinationPattern = /^\/\s*(?:javascript|data|vbscript):/i;

function hasControlCharacter(value: string): boolean {
  return Array.from(value).some((character) => {
    const codePoint = character.codePointAt(0) ?? 0;
    return codePoint <= 0x1f || (codePoint >= 0x7f && codePoint <= 0x9f);
  });
}

function isSafeInternalRoute(value: string): boolean {
  if (!value.startsWith('/') || value.startsWith('//')) return false;
  if (value.includes('\\') || hasControlCharacter(value)) return false;

  const path = value.split(/[?#]/, 1)[0];
  if (dotSegmentPattern.test(path) || scriptDestinationPattern.test(path)) return false;

  return true;
}

/**
 * Accepts only application-internal routes. URLSearchParams has already decoded
 * one layer, so validate every remaining encoded form before returning the
 * original route to React Router.
 */
export function safeLoginRedirect(candidate: string | null | undefined): string {
  if (!candidate) return '/';

  let inspected = candidate;
  for (let depth = 0; depth < 4; depth += 1) {
    if (!isSafeInternalRoute(inspected)) return '/';

    let decoded: string;
    try {
      decoded = decodeURIComponent(inspected);
    } catch {
      return '/';
    }

    if (decoded === inspected) return candidate;
    inspected = decoded;
  }

  return '/';
}
