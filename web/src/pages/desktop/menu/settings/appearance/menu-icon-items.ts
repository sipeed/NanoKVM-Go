export type MenuIconItem = {
  key: string;
};

export type MenuIconFilterOptions = {
  isMobilePortrait: boolean;
  supportedServiceIDs: ReadonlySet<string>;
};

const requiredServiceByItemKey = new Map([['screenTimelapse', 'screenTimelapse']]);
const hiddenOnMobileItemKeys = new Set(['fullscreen', 'collapse']);

export function getSupportedServiceIDs(services: unknown): Set<string> {
  if (!Array.isArray(services)) {
    return new Set();
  }

  return new Set(
    services.flatMap((service) => {
      if (typeof service !== 'object' || service === null) {
        return [];
      }

      const { id, supported } = service as { id?: unknown; supported?: unknown };
      return typeof id === 'string' && supported === true ? [id] : [];
    })
  );
}

export function filterMenuIconItems<T extends MenuIconItem>(
  items: readonly T[],
  { isMobilePortrait, supportedServiceIDs }: MenuIconFilterOptions
): T[] {
  return items.filter((item) => {
    const requiredServiceID = requiredServiceByItemKey.get(item.key);
    if (requiredServiceID && !supportedServiceIDs.has(requiredServiceID)) {
      return false;
    }

    return !isMobilePortrait || !hiddenOnMobileItemKeys.has(item.key);
  });
}
