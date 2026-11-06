const officialModeId = 'official';

/**
 * The authenticated backend is the only capability authority. Missing server
 * capability is treated as the legacy official-only API, so an old backend
 * never unlocks the managed mode by accident.
 */
export function getAllowedModeIds(backendModes: readonly string[] | undefined): string[] {
  return backendModes?.length ? [...backendModes] : [officialModeId];
}
