import icon from '@/assets/images/netbird.svg';

type NetBirdIconProps = {
  /** Pixel size of the brand mark. */
  size?: number;
};

/** Official NetBird v0.80.0 client brand mark. */
export const NetBird = ({ size = 18 }: NetBirdIconProps) => {
  return <img src={icon} style={{ width: size, height: size }} alt="" aria-hidden="true" />;
};
