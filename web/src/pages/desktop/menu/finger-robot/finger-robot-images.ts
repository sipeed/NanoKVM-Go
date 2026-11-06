import type { FingerRobotArm } from '@/api/vm.ts';

export const fingerRobotImageLevels = [1, 2, 3, 4, 5] as const;

export type FingerRobotImageLevel = (typeof fingerRobotImageLevels)[number];

const armImageModules = import.meta.glob<string>(
  [
    '/src/assets/images/finger-robot/short-l{1,2,3,4,5}.webp',
    '/src/assets/images/finger-robot/long-l{1,2,3,4,5}.webp'
  ],
  {
    eager: true,
    import: 'default',
    query: '?url'
  }
);

function resolveArmImage(arm: FingerRobotArm, level: FingerRobotImageLevel): string | undefined {
  return armImageModules[`/src/assets/images/finger-robot/${arm}-l${level}.webp`];
}

export const armImageMap: Record<
  FingerRobotArm,
  Record<FingerRobotImageLevel, string | undefined>
> = {
  short: {
    1: resolveArmImage('short', 1),
    2: resolveArmImage('short', 2),
    3: resolveArmImage('short', 3),
    4: resolveArmImage('short', 4),
    5: resolveArmImage('short', 5)
  },
  long: {
    1: resolveArmImage('long', 1),
    2: resolveArmImage('long', 2),
    3: resolveArmImage('long', 3),
    4: resolveArmImage('long', 4),
    5: resolveArmImage('long', 5)
  }
};
