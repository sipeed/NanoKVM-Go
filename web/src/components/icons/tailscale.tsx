import icon from '@/assets/images/tailscale.svg';

type TailscaleIconProps = {
  /** Pixel size of the brand mark. */
  size?: number;
};

export const Tailscale = ({ size = 18 }: TailscaleIconProps) => {
  return <img src={icon} style={{ width: size, height: size }} alt="" aria-hidden="true" />;
};
