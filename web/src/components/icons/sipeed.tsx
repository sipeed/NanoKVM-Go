type SipeedIconProps = {
  /** Pixel size of the brand mark. */
  size?: number;
};

/** Sipeed brand mark, shared with the login screen asset. */
export const Sipeed = ({ size = 18 }: SipeedIconProps) => {
  return <img src="/sipeed.ico" style={{ width: size, height: size }} alt="" aria-hidden="true" />;
};
