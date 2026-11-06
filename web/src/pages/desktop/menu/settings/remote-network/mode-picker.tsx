import clsx from 'clsx';
import type { ComponentType } from 'react';

export type RemoteNetworkModeOption = {
  id: string;
  label: string;
  description?: string;
  badge?: string;
  Icon: ComponentType<{ size?: number }>;
};

type RemoteNetworkModePickerProps = {
  /** Radio group name, unique per picker instance. */
  name: string;
  /** Accessible label of the radio group. */
  legend: string;
  options: RemoteNetworkModeOption[];
  value: string;
  onChange: (id: string) => void;
};

/**
 * Selectable mode cards shared by remote network components.
 *
 * Presentational only: callers pass resolved text and own what a selection means.
 * Native radio inputs keep arrow keys, grouping and screen reader semantics free.
 */
export const RemoteNetworkModePicker = ({
  name,
  legend,
  options,
  value,
  onChange
}: RemoteNetworkModePickerProps) => {
  return (
    <div aria-label={legend} className="grid grid-cols-1 gap-3 sm:grid-cols-2" role="radiogroup">
      {options.map((option) => {
        const isSelected = option.id === value;

        return (
          <label
            key={option.id}
            className={clsx(
              'flex cursor-pointer items-start gap-3 rounded-[10px] border p-4 transition-colors has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-blue-500/80',
              isSelected
                ? 'border-blue-500/70 bg-blue-500/10'
                : 'border-neutral-700 bg-neutral-800 hover:border-neutral-600 hover:bg-neutral-700/80'
            )}
          >
            <input
              checked={isSelected}
              className="sr-only"
              name={name}
              onChange={() => onChange(option.id)}
              type="radio"
              value={option.id}
            />

            <span
              aria-hidden="true"
              className={clsx(
                'flex size-12 shrink-0 items-center justify-center rounded-xl',
                isSelected ? 'bg-blue-500/20 text-blue-300' : 'bg-neutral-700 text-neutral-100'
              )}
            >
              <option.Icon size={26} />
            </span>

            <span className="flex min-w-0 flex-1 flex-col gap-1">
              <span className="flex flex-wrap items-center gap-2">
                <span className="text-[15px] font-semibold text-neutral-100">{option.label}</span>
                {option.badge && (
                  <span className="rounded-full border border-neutral-600 px-2 py-0.5 text-[11px] leading-4 text-neutral-300">
                    {option.badge}
                  </span>
                )}
              </span>

              {option.description && (
                <span className="text-xs leading-5 text-neutral-400">{option.description}</span>
              )}
            </span>

            <span
              aria-hidden="true"
              className={clsx(
                'mt-1 flex size-4 shrink-0 items-center justify-center rounded-full border',
                isSelected ? 'border-blue-500' : 'border-neutral-500'
              )}
            >
              {isSelected && <span className="size-2 rounded-full bg-blue-500" />}
            </span>
          </label>
        );
      })}
    </div>
  );
};
