import { useEffect, useRef, useState } from 'react';
import type { ChangeEvent } from 'react';
import { UploadOutlined } from '@ant-design/icons';
import { Button, Empty } from 'antd';
import clsx from 'clsx';
import { ImageIcon, LoaderCircleIcon, WallpaperIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';
import type { DialBackgroundImage as DialBackgroundImageType } from '@/api/vm.ts';
import { feedback } from '@/lib/feedback.tsx';

export const DialBackground = () => {
  const { t } = useTranslation();

  const [images, setImages] = useState<DialBackgroundImageType[]>([]);
  const [currentImage, setCurrentImage] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [switchingImage, setSwitchingImage] = useState('');
  const [errMsg, setErrMsg] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    getImages();
  }, []);

  async function getImages() {
    setIsLoading(true);
    setErrMsg('');

    try {
      const rsp = await api.getDialBackgroundImages();
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        setErrMsg(t('settings.device.dialBackground.errors.load'));
        return;
      }

      setImages(rsp.data?.files ?? []);
      setCurrentImage(rsp.data?.current ?? '');
    } catch (err) {
      console.log(err);
      setErrMsg(t('settings.device.dialBackground.errors.load'));
    } finally {
      setIsLoading(false);
    }
  }

  function selectFile() {
    if (inputRef.current) {
      inputRef.current.value = '';
    }
    inputRef.current?.click();
  }

  async function uploadFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file || isUploading) return;

    if (!file.name.toLowerCase().endsWith('.bmp')) {
      feedback.error(t('settings.device.dialBackground.onlyBmp'));
      return;
    }

    setIsUploading(true);
    setErrMsg('');

    const formData = new FormData();
    formData.append('file', file);

    try {
      const rsp = await api.uploadDialBackgroundImage(formData);
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        setErrMsg(t('settings.device.dialBackground.errors.upload'));
        return;
      }

      feedback.success(t('settings.device.dialBackground.uploaded'));
      await getImages();
    } catch (err) {
      console.log(err);
      setErrMsg(t('settings.device.dialBackground.errors.upload'));
    } finally {
      setIsUploading(false);
    }
  }

  async function switchImage(image: DialBackgroundImageType) {
    if (switchingImage || image.name === currentImage) return;

    setSwitchingImage(image.name);
    setErrMsg('');

    try {
      const rsp = await api.switchDialBackground(image.name);
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        setErrMsg(t('settings.device.dialBackground.errors.switch'));
        return;
      }

      setCurrentImage(image.name);
      feedback.success(t('settings.device.dialBackground.switched'));
    } catch (err) {
      console.log(err);
      setErrMsg(t('settings.device.dialBackground.errors.switch'));
    } finally {
      setSwitchingImage('');
    }
  }

  return (
    <div className="flex flex-col space-y-3 rounded-xl border border-neutral-700/70 bg-neutral-900/50 p-3 shadow-inner shadow-black/20">
      <div className="flex items-start justify-between gap-3">
        <div className="flex flex-col space-y-1">
          <div className="flex items-center space-x-2">
            <ImageIcon size={16} className="text-blue-400" />
            <span>{t('settings.device.dialBackground.title')}</span>
          </div>
          <span className="text-xs text-neutral-500">
            {t('settings.device.dialBackground.description')}
          </span>
        </div>

        <div>
          <input
            ref={inputRef}
            type="file"
            accept=".bmp,image/bmp"
            className="hidden"
            onChange={uploadFile}
          />
          <Button
            ghost
            type="primary"
            size="small"
            icon={<UploadOutlined />}
            loading={isUploading}
            onClick={selectFile}
          >
            {t('settings.device.dialBackground.upload')}
          </Button>
        </div>
      </div>

      <div className="max-h-[230px] overflow-y-auto pr-1">
        {isLoading ? (
          <div className="flex items-center justify-center space-x-2 py-8 text-neutral-400">
            <LoaderCircleIcon className="animate-spin" size={18} />
            <span className="text-sm">{t('settings.device.dialBackground.loading')}</span>
          </div>
        ) : images.length === 0 ? (
          <Empty
            className="py-8"
            image={
              <div className="flex h-full items-center justify-center">
                <WallpaperIcon size={40} strokeWidth={1.2} className="text-neutral-600" />
              </div>
            }
            description={t('settings.device.dialBackground.empty')}
          />
        ) : (
          <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
            {images.map((image) => {
              const isCurrent = image.name === currentImage;
              const isSwitching = image.name === switchingImage;

              return (
                <button
                  key={image.name}
                  type="button"
                  className={clsx(
                    'group flex min-w-0 flex-col overflow-hidden rounded-lg border bg-neutral-800/70 text-left transition hover:border-blue-400/70 hover:bg-neutral-800',
                    isCurrent
                      ? 'border-blue-500/80 ring-1 ring-blue-500/40'
                      : 'border-neutral-700/80',
                    switchingImage && !isSwitching && 'cursor-wait opacity-60'
                  )}
                  disabled={!!switchingImage}
                  onClick={() => switchImage(image)}
                >
                  <div className="relative aspect-[27/17] w-full overflow-hidden bg-neutral-950">
                    <img
                      src={`${image.url}?v=${image.modTime}`}
                      alt={image.name}
                      className="h-full w-full object-cover transition duration-200 group-hover:scale-105"
                      loading="lazy"
                    />
                    {isSwitching && (
                      <div className="absolute inset-0 flex items-center justify-center bg-black/50">
                        <LoaderCircleIcon className="animate-spin text-white" size={20} />
                      </div>
                    )}
                    {isCurrent && (
                      <div className="absolute right-1.5 top-1.5 rounded-full bg-blue-500 px-2 py-0.5 text-[10px] font-medium text-white shadow">
                        {t('settings.device.dialBackground.current')}
                      </div>
                    )}
                  </div>

                  <div className="min-w-0 px-2 py-1.5">
                    <div className="truncate text-xs text-neutral-200">{image.name}</div>
                    <div className="text-[10px] text-neutral-500">{formatSize(image.size)}</div>
                  </div>
                </button>
              );
            })}
          </div>
        )}
      </div>

      {errMsg && <span className="break-words text-xs text-red-500">{errMsg}</span>}
    </div>
  );
};

function formatSize(size: number): string {
  if (size < 1024) {
    return `${size} B`;
  }

  if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(1)} KB`;
  }

  return `${(size / 1024 / 1024).toFixed(1)} MB`;
}
