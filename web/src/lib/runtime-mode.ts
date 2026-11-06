/**
 * Returns whether the frontend is running the opt-in, browser-only mock runtime.
 *
 * Keeping the Vite environment reads at call sites makes this helper usable from
 * Node-based contract tests without teaching those tests about import.meta.
 */
export function isMockedDevelopment(mode: string, isDev: boolean): boolean {
  return isDev && mode === 'mocked';
}
