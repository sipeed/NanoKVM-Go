type BarcodeDetectorLike = {
  detect: (source: HTMLCanvasElement) => Promise<{ rawValue: string }[]>;
};

type BarcodeDetectorConstructor = {
  new (options: { formats: string[] }): BarcodeDetectorLike;
  getSupportedFormats: () => Promise<string[]>;
};

// Phone photos are 12 MP or more; QR detection needs far less. Try a detailed
// pass first, then a coarser one, which helps when the code fills the frame.
const DECODE_SIZES = [1600, 800];

async function nativeDetector(): Promise<BarcodeDetectorLike | null> {
  const Detector = (window as unknown as { BarcodeDetector?: BarcodeDetectorConstructor })
    .BarcodeDetector;
  if (!Detector) return null;

  try {
    const formats = await Detector.getSupportedFormats();
    return formats.includes('qr_code') ? new Detector({ formats: ['qr_code'] }) : null;
  } catch {
    return null;
  }
}

function drawScaled(image: ImageBitmap, maxSize: number): HTMLCanvasElement | null {
  const scale = Math.min(1, maxSize / Math.max(image.width, image.height));
  const canvas = document.createElement('canvas');
  canvas.width = Math.max(1, Math.round(image.width * scale));
  canvas.height = Math.max(1, Math.round(image.height * scale));

  const context = canvas.getContext('2d', { willReadFrequently: true });
  if (!context) return null;
  context.drawImage(image, 0, 0, canvas.width, canvas.height);
  return canvas;
}

/** Decode the first QR code in an image file, or return null if none is found. */
export async function decodeQrImage(file: Blob): Promise<string | null> {
  const image = await createImageBitmap(file);

  try {
    const detector = await nativeDetector();
    if (detector) {
      for (const size of DECODE_SIZES) {
        const canvas = drawScaled(image, size);
        if (!canvas) break;
        try {
          const [code] = await detector.detect(canvas);
          if (code?.rawValue) return code.rawValue;
        } catch {
          break;
        }
      }
    }

    // Native detection is missing on iOS Safari and may miss dense codes.
    const { default: jsQR } = await import('jsqr');
    for (const size of DECODE_SIZES) {
      const canvas = drawScaled(image, size);
      if (!canvas) return null;

      const context = canvas.getContext('2d', { willReadFrequently: true });
      if (!context) return null;
      const pixels = context.getImageData(0, 0, canvas.width, canvas.height);
      const code = jsQR(pixels.data, pixels.width, pixels.height, {
        inversionAttempts: 'attemptBoth'
      });
      if (code?.data) return code.data;
    }

    return null;
  } finally {
    image.close();
  }
}
