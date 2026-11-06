#include <sound/asound.h>
#include <sys/ioctl.h>

#if defined(__arm__) && defined(__GLIBC__)
_Static_assert(__TIMESIZE == 32,
               "NanoKVM ARM32 ALSA must use the legacy time32 ABI");
_Static_assert(sizeof(struct snd_pcm_sync_ptr) == 132,
               "Unexpected ARM32 snd_pcm_sync_ptr ABI size");
_Static_assert(SNDRV_PCM_IOCTL_SYNC_PTR == 0xC0844123UL,
               "Unexpected ARM32 SNDRV_PCM_IOCTL_SYNC_PTR encoding");
#endif
