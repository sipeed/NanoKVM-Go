import { useEffect, useState } from 'react';
import { Button, Modal, notification, Typography } from 'antd';
import clsx from 'clsx';
import {
  ArrowBigDownDashIcon,
  ArrowBigUpDashIcon,
  DiscIcon,
  LoaderCircleIcon,
  PackageIcon,
  Trash2Icon
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/storage.ts';
import { feedback } from '@/lib/feedback.tsx';
import { client } from '@/lib/websocket.ts';
import { useProtectionChallengeGate } from '@/hooks/useProtectionChallengeGate.tsx';
import { ButtonBase } from '@/components/ui/button-base.tsx';

type ImagesProps = {
  isOpen: boolean;
  cdrom: boolean;
  readOnly: boolean;
  refreshVersion: number;
  setIsMounted: (isMounted: boolean) => void;
};

export const Images = ({ isOpen, cdrom, readOnly, refreshVersion, setIsMounted }: ImagesProps) => {
  const { t } = useTranslation();
  const [notify, contextHolder] = notification.useNotification();
  const { runWithChallenge, modal: protectionModal, isGateBusy } = useProtectionChallengeGate();

  const [isLoading, setIsLoading] = useState(false);
  const [images, setImages] = useState<string[]>([]);
  const [mountingImage, setMountingImage] = useState('');
  const [mountedImage, setMountedImage] = useState('');
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [selectedImage, setSelectedImage] = useState('');
  const [deletingImage, setDeletingImage] = useState('');
  const [storageWritable, setStorageWritable] = useState(false);

  function getStorageState() {
    api.imageEnabled().then((rsp) => setStorageWritable(rsp.code === 0 && !!rsp.data?.enabled));
  }

  useEffect(() => {
    if (isOpen) {
      getImages();
      getStorageState();
    }
  }, [isOpen, refreshVersion]);

  // get image list
  function getImages() {
    if (isLoading) return;
    setIsLoading(true);

    api
      .getImages()
      .then((rsp) => {
        if (rsp.code !== 0) {
          return;
        }

        const files = rsp.data?.files;

        if (files?.length > 0) {
          setImages(files);
          getMountedImage();
        } else {
          setImages([]);
        }
      })
      .finally(() => {
        setIsLoading(false);
      });
  }

  // get mounted image
  function getMountedImage() {
    api.getMountedImage().then((rsp) => {
      if (rsp.code !== 0) return;

      const file = rsp.data?.file;
      setMountedImage(file);
      setIsMounted(!!file);
    });
  }

  // mount or unmount image
  function mountImage(image: string) {
    if (mountingImage) return;
    setMountingImage(image);

    client.close();

    const isMounted = mountedImage === image;
    const filename = isMounted ? '' : image;

    api
      .mountImage(filename, cdrom, readOnly)
      .then((rsp) => {
        if (rsp.code !== 0) {
          openNotification(isMounted);
          return;
        }

        setMountedImage(filename);
        setIsMounted(!!filename);
      })
      .finally(() => {
        setMountingImage('');
        client.connect();
        getStorageState();
      });
  }

  // show delete image modal
  function showDeleteModal(e: any, image: string) {
    e.stopPropagation();

    const isMounted = mountedImage === image;
    const isDeleting = deletingImage !== '' || isGateBusy;

    if (isMounted || isDeleting || !storageWritable) {
      return;
    }

    setSelectedImage(image);
    setIsModalOpen(true);
  }

  // delete image (optional advanced 2FA challenge)
  function deleteImage() {
    if (!selectedImage || !!deletingImage || isGateBusy || !storageWritable) return;

    const file = selectedImage;
    setIsModalOpen(false);

    void runWithChallenge('delete_image', 'deleteImage', async (challenge) => {
      setDeletingImage(file);
      try {
        const rsp = await api.deleteImage(file, challenge);
        if (rsp.code === 0) {
          getImages();
          setSelectedImage('');
        } else {
          console.log(rsp.msg);
          feedback.error(t('image.deleteFailed'));
        }
        return rsp;
      } catch {
        feedback.error(t('image.deleteFailed'));
        return undefined;
      } finally {
        setDeletingImage('');
      }
    });
  }

  // show mount/unmount failed notification
  function openNotification(isMounted: boolean) {
    if (isMounted) {
      notify.open({
        message: t('image.unmountFailed'),
        description: t('image.unmountDesc'),
        duration: 10
      });
    } else {
      notify.open({
        message: t('image.mountFailed'),
        description: t('image.mountDesc'),
        duration: 10
      });
    }
  }

  //  loading
  if (isLoading) {
    return (
      <div className="flex items-center justify-center gap-2 rounded-xl border border-dashed border-neutral-700 py-10 text-neutral-400">
        <LoaderCircleIcon className="animate-spin" size={18} />
        <span className="text-sm">{t('image.loading')}</span>
      </div>
    );
  }

  // empty image
  if (images.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-2.5 rounded-xl border border-dashed border-neutral-700 py-10">
        <div className="flex h-11 w-11 items-center justify-center rounded-full bg-neutral-800">
          <DiscIcon size={22} strokeWidth={1.5} className="text-neutral-500" />
        </div>
        <span className="text-sm text-neutral-500">{t('image.empty')}</span>
      </div>
    );
  }

  // image list
  return (
    <>
      <div className="flex max-h-[340px] flex-col gap-1.5 overflow-y-auto pr-0.5">
        {images.map((image) => {
          const mounted = mountedImage === image;
          const deleteDisabled = mounted || !storageWritable;

          return (
            <div
              key={image}
              className={clsx(
                'group flex cursor-pointer select-none items-center gap-3 rounded-xl border px-3 py-2.5 transition-colors',
                mounted
                  ? 'border-blue-500/50 bg-blue-500/10'
                  : 'border-transparent hover:border-neutral-700 hover:bg-neutral-800/80'
              )}
              onClick={() => mountImage(image)}
            >
              <div
                className={clsx(
                  'flex h-9 w-9 shrink-0 items-center justify-center rounded-lg transition-colors',
                  mounted
                    ? 'bg-blue-500/20 text-blue-400'
                    : 'bg-neutral-800 text-neutral-400 group-hover:bg-neutral-700/70 group-hover:text-neutral-200'
                )}
              >
                {mountingImage === image ? (
                  <LoaderCircleIcon className="animate-spin" size={18} />
                ) : (
                  <PackageIcon size={18} strokeWidth={1.8} />
                )}
              </div>

              <div className="flex min-w-0 flex-1 flex-col">
                <span
                  className={clsx(
                    'truncate text-sm',
                    mounted ? 'text-blue-300' : 'text-neutral-200'
                  )}
                >
                  {image.replace(/^.*[\\/]/, '')}
                </span>
                {mounted && (
                  <span className="mt-0.5 flex items-center gap-1.5 text-xs text-blue-400/90">
                    <span className="h-1.5 w-1.5 rounded-full bg-blue-400" />
                    {t('image.mounted')}
                  </span>
                )}
              </div>

              <div className="flex h-7 w-7 shrink-0 items-center justify-center">
                {mounted ? (
                  <ArrowBigDownDashIcon
                    size={20}
                    className="text-neutral-600 transition-colors group-hover:text-red-400"
                  />
                ) : (
                  <ArrowBigUpDashIcon
                    size={20}
                    className="text-neutral-700 transition-colors group-hover:text-blue-400"
                  />
                )}
              </div>

              <ButtonBase
                className={clsx(
                  'flex h-7 w-7 shrink-0 items-center justify-center rounded-md transition-colors',
                  deleteDisabled
                    ? 'cursor-not-allowed text-neutral-600'
                    : 'text-neutral-500 hover:bg-red-500/15 hover:text-red-400'
                )}
                onClick={(e) => showDeleteModal(e, image)}
              >
                {deletingImage === image ? (
                  <LoaderCircleIcon className="animate-spin text-red-400" size={15} />
                ) : (
                  <Trash2Icon size={15} />
                )}
              </ButtonBase>
            </div>
          );
        })}
      </div>

      <Modal
        title={t('image.attention')}
        open={isModalOpen}
        width={520}
        footer={null}
        onCancel={() => setIsModalOpen(false)}
      >
        <div className="flex flex-col items-center pb-10">
          <p>{t('image.deleteConfirm')}</p>
          <Typography.Text code>{selectedImage}</Typography.Text>
        </div>

        <div className="flex justify-center space-x-3 pb-3">
          <Button
            type="primary"
            danger
            disabled={isGateBusy || !!deletingImage}
            onClick={deleteImage}
          >
            {t('image.okBtn')}
          </Button>
          <Button disabled={isGateBusy || !!deletingImage} onClick={() => setIsModalOpen(false)}>
            {t('image.cancelBtn')}
          </Button>
        </div>
      </Modal>

      {protectionModal}
      {contextHolder}
    </>
  );
};
