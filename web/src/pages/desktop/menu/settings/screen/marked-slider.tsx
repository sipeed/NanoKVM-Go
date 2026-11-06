import { type ReactNode } from 'react';
import { Slider } from 'antd';
import type { SliderSingleProps } from 'antd/es/slider';
import clsx from 'clsx';

import { useResponsiveDevice } from '@/hooks/useResponsiveDevice.ts';

type MarkedSliderProps = Omit<SliderSingleProps, 'marks' | 'range'> & {
  min: number;
  max: number;
  minLabel: ReactNode;
  maxLabel: ReactNode;
};

// Renders endpoint values below the rail in mobile portrait so long labels such as
// "20000" cannot be clipped by the page edge; desktop keeps the built-in antd marks.
export const MarkedSlider = ({ min, max, minLabel, maxLabel, className, ...sliderProps }: MarkedSliderProps) => {
  const { isMobilePortrait } = useResponsiveDevice();

  if (!isMobilePortrait) {
    return (
      <Slider
        min={min}
        max={max}
        marks={{ [min]: minLabel, [max]: maxLabel }}
        className={className}
        {...sliderProps}
      />
    );
  }

  return (
    <div className="flex flex-col">
      {/* Desktop keeps the slider's bottom margin (11px with the app's static theme tokens)
          plus the extra with-marks reservation. In mobile portrait the label row replaces
          the marks, so drop the bottom margin and let the row start at the slider box
          bottom; the rail sits 4px above it, matching the 4px rail-to-mark gap of antd
          marks exactly. */}
      <Slider min={min} max={max} className={clsx(className, '!mb-0')} {...sliderProps} />
      {/* antd Slider keeps a horizontal margin of controlHeightLG / 8 (= 5px with the
          app's default theme tokens) so the handle clears the rail ends; inset the label
          row by the same amount so labels stay flush with the visible rail edges. */}
      <div className="mx-[5px] flex items-center justify-between text-xs text-neutral-500">
        <span>{minLabel}</span>
        <span>{maxLabel}</span>
      </div>
    </div>
  );
};
