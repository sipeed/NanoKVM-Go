import { webcamFrames } from './webcam-frames';
import { sendWebcam, type WebcamReport } from './webcam-sender';

type Command = { stream: ReadableStream<VideoFrame>; url: string };
self.onmessage = ({ data }: MessageEvent<Command>) => {
  const report = (message: WebcamReport) => self.postMessage(message);
  try {
    sendWebcam(data.url, webcamFrames(data.stream), report);
  } catch (error) {
    report({
      error: error instanceof Error ? error.message : 'Could not start the webcam worker.'
    });
  }
};
