export const LARGE_APP_UPLOAD_THRESHOLD_BYTES = 100 * 1024 * 1024;

export function requiresLargeAppUploadConfirmation(fileSize: number) {
  return fileSize > LARGE_APP_UPLOAD_THRESHOLD_BYTES;
}
