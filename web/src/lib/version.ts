import semver from 'semver';

export function normalizeVersion(version: string) {
  return version.trim().replace(/^[vV]/, '');
}

export function isNewerVersion(latest: string, current: string) {
  const normalizedLatest = normalizeVersion(latest);
  const normalizedCurrent = normalizeVersion(current);

  if (!semver.valid(normalizedLatest) || !semver.valid(normalizedCurrent)) {
    return false;
  }

  return semver.gt(normalizedLatest, normalizedCurrent);
}
