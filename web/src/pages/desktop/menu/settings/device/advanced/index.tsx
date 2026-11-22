import { Collapse } from 'antd';
import { useTranslation } from 'react-i18next';

import { DiskInquiryCdrom } from './disk-inquiry-cdrom.tsx';
import { DiskInquiry } from './disk-inquiry.tsx';
import { LowPower } from './low-power.tsx';
import { RebootToFlashing } from './reboot-to-flashing.tsx';
import { StaticIp } from './static-ip.tsx';
import { Swap } from './swap.tsx';
import { Uac2FunctionName } from './uac2-function-name.tsx';
import { UsbId } from './usb-id.tsx';
import { WebcamFunctionName } from './webcam-function-name.tsx';

export const Advanced = () => {
  const { t } = useTranslation();

  return (
    <Collapse
      ghost
      expandIconPosition="end"
      items={[
        {
          key: 'advanced',
          label: t('settings.device.advanced'),
          children: (
            <div className="mt-5 flex flex-col space-y-8">
              <LowPower />
              <Swap />
              <StaticIp />
              <UsbId />
              <DiskInquiry />
              <DiskInquiryCdrom />
              <Uac2FunctionName />
              <WebcamFunctionName />
              <RebootToFlashing />
            </div>
          )
        }
      ]}
    />
  );
};
