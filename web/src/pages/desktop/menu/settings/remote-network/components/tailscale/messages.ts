import type { SupportedLanguage } from '@/i18n/locale.ts';

import type { RemoteNetworkComponentMessages } from '../../types.ts';

const sipeedManagedFallbackMessages = {
  'settings.tailscale.sipeed.disclaimerTitle': 'Disclaimer',
  'settings.tailscale.sipeed.enableDesc':
    'Enabling Sipeed Remote Network sends requests to Sipeed servers and provides connection information such as the device identifier, device name, and virtual network IP address. This information is used for device registration, authentication, connection coordination, and network management.\n\nRemote access depends on network conditions, device configuration, and service availability. Connections may be delayed, interrupted, or temporarily unavailable. Assess the risks of using this service and keep an alternative way to access your device when needed.\n\nUse this service only with devices and networks you own or are authorized to access. Keep login credentials and pairing codes secure, and grant access to others with care.\n\nBy clicking “Agree and enable”, you confirm that you have read and understood this notice and consent to the communication and information processing described above for providing remote connections. Your consent is saved on this device, so you do not need to confirm it each time you enable the service.\n\nYou can withdraw consent from the top-right menu on the connected page. Withdrawal exits Sipeed Remote Network and requests deletion of the associated cloud network and its user. Once deleted, other devices in that network can no longer use it for remote access. Enabling the service again requires renewed consent.',
  'settings.tailscale.sipeed.acceptAndEnable': 'Agree and enable',
  'settings.tailscale.sipeed.revokeDisclaimer': 'Withdraw disclaimer consent',
  'settings.tailscale.sipeed.revokeDisclaimerDesc':
    'This will exit Sipeed Remote Network, delete the cloud network and its user, and clear your saved consent. All devices in this network will be disconnected. Continue?',
  'settings.tailscale.sipeed.revokeDisclaimerFailed':
    'Could not complete withdrawal. Please try again.',
  'settings.tailscale.sipeed.enabling': 'Enabling remote network…',
  'settings.tailscale.sipeed.enableFailed': 'Could not enable the remote network.',
  'settings.tailscale.sipeed.officialActive':
    'An official Tailscale identity is active. This version does not switch identities directly.',
  'settings.tailscale.sipeed.invalidIdentity':
    'The remote network identity is no longer valid. Retry to create a new identity.',
  'settings.tailscale.sipeed.identityMissing':
    'The remote-network identity is not available. Enable it to continue.',
  'settings.tailscale.sipeed.rateLimited': 'Too many attempts. Retry in {{seconds}} seconds.',
  'settings.tailscale.sipeed.pairingUnavailable':
    'The pairing service is temporarily unavailable. Try again later.',
  'settings.tailscale.sipeed.pairingUpstreamError':
    'The pairing service could not reach its network controller. Try again later.',
  'settings.tailscale.sipeed.pairingRejected':
    'The pairing service rejected this device. Try again or contact support.',
  'settings.tailscale.sipeed.nodeNotFound':
    'The managed device was not found. Retry to refresh its network identity.',
  'settings.tailscale.sipeed.protocolError':
    'The pairing service returned an unexpected response. Try again later.',
  'settings.tailscale.sipeed.localAPIUnavailable':
    'The local remote-network service is unavailable. Start it and try again.',
  'settings.tailscale.sipeed.stateFileInvalid':
    'The saved remote-network state is invalid. Reset it and try again.',
  'settings.tailscale.sipeed.stateWriteFailed':
    'The remote-network state could not be saved. Check local storage and try again.',
  'settings.tailscale.sipeed.installFailed':
    'The remote-network client could not be installed. Try again later.',
  'settings.tailscale.sipeed.profileReadFailed':
    'The local remote-network profile could not be read. Try again.',
  'settings.tailscale.sipeed.hostnameUnavailable':
    'The device hostname is unavailable. Set a hostname and try again.',
  'settings.tailscale.sipeed.profileSetupFailed':
    'The managed remote-network profile could not be prepared. Try again.',
  'settings.tailscale.sipeed.profileStartFailed':
    'The managed remote-network profile could not be started. Try again.',
  'settings.tailscale.sipeed.connectionFailed':
    'The managed remote-network connection could not be established. Try again.',
  'settings.tailscale.sipeed.profileVerificationFailed':
    'The managed remote-network profile could not be verified. Try again.',
  'settings.tailscale.sipeed.rollbackFailed':
    'The previous remote-network profile could not be restored. Check Tailscale and try again.',
  'settings.tailscale.sipeed.interrupted':
    'The previous remote-network operation was interrupted. Retry to continue.',
  'settings.tailscale.sipeed.identityGenerationFailed':
    'A remote-network identity could not be created locally. Try again.',
  'settings.tailscale.sipeed.retry': 'Retry',
  'settings.tailscale.sipeed.localStatus': 'Local connection',
  'settings.tailscale.sipeed.connected': 'Connected',
  'settings.tailscale.sipeed.copyIP': 'Copy device IP',
  'settings.tailscale.sipeed.ipCopied': 'Device IP copied',
  'settings.tailscale.sipeed.devices': 'Devices',
  'settings.tailscale.sipeed.emptyDevices': 'No other devices have been added yet.',
  'settings.tailscale.sipeed.exit': 'Exit',
  'settings.tailscale.sipeed.exitDesc':
    'Remove this device from the Sipeed network and return to the network selection page.',
  'settings.tailscale.sipeed.exitConfirm': 'Exit the Sipeed network?',
  'settings.tailscale.sipeed.exitWarning':
    'The local profile was removed, but the cloud device could not be deleted. You can remove it later from another device.',
  'settings.tailscale.sipeed.exitFailed': 'Could not exit the Sipeed network.',
  'settings.tailscale.sipeed.pendingTitle': 'Pending device',
  'settings.tailscale.sipeed.pendingDesc': 'Verify this device before it joins your network.',
  'settings.tailscale.sipeed.pendingSas': 'Verification numbers',
  'settings.tailscale.sipeed.pendingMismatch':
    'If these numbers do not match the joining device, reject the request.',
  'settings.tailscale.sipeed.pendingClient': 'Client',
  'settings.tailscale.sipeed.pendingAllow': 'Allow',
  'settings.tailscale.sipeed.pendingReject': 'Reject',
  'settings.tailscale.sipeed.pendingName': 'Device name (optional)',
  'settings.tailscale.sipeed.backupCode': 'Backup code',
  'settings.tailscale.sipeed.backupCodeDesc': 'Enter the 8-character backup code from the device.',
  'settings.tailscale.sipeed.authorize': 'Use backup code',
  'settings.tailscale.sipeed.authorizeFailed': 'The backup code could not be accepted.',
  'settings.tailscale.sipeed.managementFailed': 'Device management failed. Try again.',
  'settings.tailscale.sipeed.identityExpired':
    'The Sipeed identity expired. Choose a network mode again.',
  'settings.tailscale.sipeed.deviceOnline': 'Online',
  'settings.tailscale.sipeed.deviceOffline': 'Offline',
  'settings.tailscale.sipeed.lastSeen': 'Last seen {{time}}',
  'settings.tailscale.sipeed.justNow': 'just now',
  'settings.tailscale.sipeed.rename': 'Rename',
  'settings.tailscale.sipeed.save': 'Save',
  'settings.tailscale.sipeed.cancel': 'Cancel',
  'settings.tailscale.sipeed.remove': 'Remove',
  'settings.tailscale.sipeed.removeConfirm': 'Remove this device from the network?',
  'settings.tailscale.sipeed.removeSelf': 'The current device can only be removed by exiting.',
  'settings.tailscale.sipeed.refreshFailed': 'Could not refresh the device list.'
};

function prefixSipeedMessages(messages: Record<string, string>): Record<string, string> {
  return Object.fromEntries(
    Object.entries(messages).map(([key, value]) => [`settings.tailscale.sipeed.${key}`, value])
  );
}

type SipeedRuntimeMessageKey =
  | 'enabling'
  | 'enableFailed'
  | 'officialActive'
  | 'invalidIdentity'
  | 'identityMissing'
  | 'rateLimited'
  | 'pairingUnavailable'
  | 'pairingUpstreamError'
  | 'pairingRejected'
  | 'nodeNotFound'
  | 'protocolError'
  | 'localAPIUnavailable'
  | 'stateFileInvalid'
  | 'stateWriteFailed'
  | 'installFailed'
  | 'profileReadFailed'
  | 'hostnameUnavailable'
  | 'profileSetupFailed'
  | 'profileStartFailed'
  | 'connectionFailed'
  | 'profileVerificationFailed'
  | 'rollbackFailed'
  | 'interrupted'
  | 'identityGenerationFailed'
  | 'retry'
  | 'localStatus'
  | 'connected'
  | 'copyIP'
  | 'ipCopied'
  | 'devices'
  | 'emptyDevices';

type NonEnglishLanguage = Exclude<SupportedLanguage, 'en'>;

function defineSipeedRuntimeMessages(
  messages: Record<SipeedRuntimeMessageKey, string>
): Record<string, string> {
  return prefixSipeedMessages(messages);
}

/**
 * Runtime and error strings shown while enabling the Sipeed-managed network.
 * Keep this separate from the English safety fallback so a missing translation
 * is a type error instead of silently leaking English into another locale.
 */
const sipeedRuntimeTranslations: Record<NonEnglishLanguage, Record<string, string>> = {
  nl: defineSipeedRuntimeMessages({
    enabling: 'Netwerk op afstand inschakelen…',
    enableFailed: 'Het netwerk op afstand kon niet worden ingeschakeld.',
    officialActive:
      'Er is een officiële Tailscale-identiteit actief. Deze versie schakelt niet rechtstreeks tussen identiteiten.',
    invalidIdentity:
      'De identiteit voor het netwerk op afstand is niet meer geldig. Probeer opnieuw om een nieuwe identiteit aan te maken.',
    identityMissing:
      'De identiteit voor het netwerk op afstand is niet beschikbaar. Schakel het netwerk in om door te gaan.',
    rateLimited: 'Te veel pogingen. Probeer het over {{seconds}} seconden opnieuw.',
    pairingUnavailable:
      'De koppelingsservice is tijdelijk niet beschikbaar. Probeer het later opnieuw.',
    pairingUpstreamError:
      'De koppelingsservice kan de netwerkcontroller niet bereiken. Probeer het later opnieuw.',
    pairingRejected:
      'De koppelingsservice heeft dit apparaat geweigerd. Probeer het opnieuw of neem contact op met de ondersteuning.',
    nodeNotFound:
      'Het beheerde apparaat is niet gevonden. Probeer opnieuw om de netwerkidentiteit te vernieuwen.',
    protocolError:
      'De koppelingsservice heeft een onverwacht antwoord gegeven. Probeer het later opnieuw.',
    localAPIUnavailable:
      'De lokale service voor het netwerk op afstand is niet beschikbaar. Start de service en probeer het opnieuw.',
    stateFileInvalid:
      'De opgeslagen status van het netwerk op afstand is ongeldig. Stel deze opnieuw in en probeer het opnieuw.',
    stateWriteFailed:
      'De status van het netwerk op afstand kon niet worden opgeslagen. Controleer de lokale opslag en probeer het opnieuw.',
    installFailed:
      'De client voor het netwerk op afstand kon niet worden geïnstalleerd. Probeer het later opnieuw.',
    profileReadFailed:
      'Het lokale profiel voor het netwerk op afstand kon niet worden gelezen. Probeer het opnieuw.',
    hostnameUnavailable:
      'De hostnaam van het apparaat is niet beschikbaar. Stel een hostnaam in en probeer het opnieuw.',
    profileSetupFailed:
      'Het beheerde profiel voor het netwerk op afstand kon niet worden voorbereid. Probeer het opnieuw.',
    profileStartFailed:
      'Het beheerde profiel voor het netwerk op afstand kon niet worden gestart. Probeer het opnieuw.',
    connectionFailed:
      'De beheerde verbinding met het netwerk op afstand kon niet tot stand worden gebracht. Probeer het opnieuw.',
    profileVerificationFailed:
      'Het beheerde profiel voor het netwerk op afstand kon niet worden geverifieerd. Probeer het opnieuw.',
    rollbackFailed:
      'Het vorige profiel voor het netwerk op afstand kon niet worden hersteld. Controleer Tailscale en probeer het opnieuw.',
    interrupted:
      'De vorige bewerking voor het netwerk op afstand is onderbroken. Probeer opnieuw om door te gaan.',
    identityGenerationFailed:
      'Er kon lokaal geen identiteit voor het netwerk op afstand worden aangemaakt. Probeer het opnieuw.',
    retry: 'Opnieuw proberen',
    localStatus: 'Lokale verbinding',
    connected: 'Verbonden',
    copyIP: 'IP-adres van apparaat kopiëren',
    ipCopied: 'IP-adres van apparaat gekopieerd',
    devices: 'Apparaten',
    emptyDevices: 'Er zijn nog geen andere apparaten toegevoegd.'
  }),
  da: defineSipeedRuntimeMessages({
    enabling: 'Aktiverer fjernnetværk…',
    enableFailed: 'Fjernnetværket kunne ikke aktiveres.',
    officialActive:
      'En officiel Tailscale-identitet er aktiv. Denne version skifter ikke direkte mellem identiteter.',
    invalidIdentity:
      'Fjernnetværksidentiteten er ikke længere gyldig. Prøv igen for at oprette en ny identitet.',
    identityMissing: 'Fjernnetværksidentiteten er ikke tilgængelig. Aktivér den for at fortsætte.',
    rateLimited: 'For mange forsøg. Prøv igen om {{seconds}} sekunder.',
    pairingUnavailable: 'Parringstjenesten er midlertidigt utilgængelig. Prøv igen senere.',
    pairingUpstreamError: 'Parringstjenesten kunne ikke nå netværkscontrolleren. Prøv igen senere.',
    pairingRejected: 'Parringstjenesten afviste denne enhed. Prøv igen, eller kontakt support.',
    nodeNotFound:
      'Den administrerede enhed blev ikke fundet. Prøv igen for at forny dens netværksidentitet.',
    protocolError: 'Parringstjenesten returnerede et uventet svar. Prøv igen senere.',
    localAPIUnavailable:
      'Den lokale fjernnetværkstjeneste er ikke tilgængelig. Start den, og prøv igen.',
    stateFileInvalid: 'Den gemte fjernnetværkstilstand er ugyldig. Nulstil den, og prøv igen.',
    stateWriteFailed:
      'Fjernnetværkstilstanden kunne ikke gemmes. Kontrollér det lokale lager, og prøv igen.',
    installFailed: 'Fjernnetværksklienten kunne ikke installeres. Prøv igen senere.',
    profileReadFailed: 'Den lokale fjernnetværksprofil kunne ikke læses. Prøv igen.',
    hostnameUnavailable:
      'Enhedens værtsnavn er ikke tilgængeligt. Angiv et værtsnavn, og prøv igen.',
    profileSetupFailed: 'Den administrerede fjernnetværksprofil kunne ikke klargøres. Prøv igen.',
    profileStartFailed: 'Den administrerede fjernnetværksprofil kunne ikke startes. Prøv igen.',
    connectionFailed: 'Den administrerede fjernnetværksforbindelse kunne ikke oprettes. Prøv igen.',
    profileVerificationFailed:
      'Den administrerede fjernnetværksprofil kunne ikke bekræftes. Prøv igen.',
    rollbackFailed:
      'Den tidligere fjernnetværksprofil kunne ikke gendannes. Kontrollér Tailscale, og prøv igen.',
    interrupted: 'Den forrige fjernnetværkshandling blev afbrudt. Prøv igen for at fortsætte.',
    identityGenerationFailed:
      'Der kunne ikke oprettes en fjernnetværksidentitet lokalt. Prøv igen.',
    retry: 'Prøv igen',
    localStatus: 'Lokal forbindelse',
    connected: 'Forbundet',
    copyIP: 'Kopiér enhedens IP-adresse',
    ipCopied: 'Enhedens IP-adresse er kopieret',
    devices: 'Enheder',
    emptyDevices: 'Der er endnu ikke tilføjet andre enheder.'
  }),
  de: defineSipeedRuntimeMessages({
    enabling: 'Remote-Netzwerk wird aktiviert…',
    enableFailed: 'Das Remote-Netzwerk konnte nicht aktiviert werden.',
    officialActive:
      'Eine offizielle Tailscale-Identität ist aktiv. Diese Version wechselt Identitäten nicht direkt.',
    invalidIdentity:
      'Die Identität des Remote-Netzwerks ist nicht mehr gültig. Versuchen Sie erneut, eine neue Identität zu erstellen.',
    identityMissing:
      'Die Identität des Remote-Netzwerks ist nicht verfügbar. Aktivieren Sie sie, um fortzufahren.',
    rateLimited: 'Zu viele Versuche. Versuchen Sie es in {{seconds}} Sekunden erneut.',
    pairingUnavailable:
      'Der Kopplungsdienst ist vorübergehend nicht verfügbar. Versuchen Sie es später erneut.',
    pairingUpstreamError:
      'Der Kopplungsdienst konnte den Netzwerk-Controller nicht erreichen. Versuchen Sie es später erneut.',
    pairingRejected:
      'Der Kopplungsdienst hat dieses Gerät abgelehnt. Versuchen Sie es erneut oder wenden Sie sich an den Support.',
    nodeNotFound:
      'Das verwaltete Gerät wurde nicht gefunden. Versuchen Sie erneut, seine Netzwerkidentität zu erneuern.',
    protocolError:
      'Der Kopplungsdienst hat eine unerwartete Antwort zurückgegeben. Versuchen Sie es später erneut.',
    localAPIUnavailable:
      'Der lokale Remote-Netzwerkdienst ist nicht verfügbar. Starten Sie ihn und versuchen Sie es erneut.',
    stateFileInvalid:
      'Der gespeicherte Status des Remote-Netzwerks ist ungültig. Setzen Sie ihn zurück und versuchen Sie es erneut.',
    stateWriteFailed:
      'Der Status des Remote-Netzwerks konnte nicht gespeichert werden. Prüfen Sie den lokalen Speicher und versuchen Sie es erneut.',
    installFailed:
      'Der Remote-Netzwerkclient konnte nicht installiert werden. Versuchen Sie es später erneut.',
    profileReadFailed:
      'Das lokale Remote-Netzwerkprofil konnte nicht gelesen werden. Versuchen Sie es erneut.',
    hostnameUnavailable:
      'Der Hostname des Geräts ist nicht verfügbar. Legen Sie einen Hostnamen fest und versuchen Sie es erneut.',
    profileSetupFailed:
      'Das verwaltete Remote-Netzwerkprofil konnte nicht vorbereitet werden. Versuchen Sie es erneut.',
    profileStartFailed:
      'Das verwaltete Remote-Netzwerkprofil konnte nicht gestartet werden. Versuchen Sie es erneut.',
    connectionFailed:
      'Die verwaltete Remote-Netzwerkverbindung konnte nicht hergestellt werden. Versuchen Sie es erneut.',
    profileVerificationFailed:
      'Das verwaltete Remote-Netzwerkprofil konnte nicht überprüft werden. Versuchen Sie es erneut.',
    rollbackFailed:
      'Das vorherige Remote-Netzwerkprofil konnte nicht wiederhergestellt werden. Prüfen Sie Tailscale und versuchen Sie es erneut.',
    interrupted:
      'Der vorherige Remote-Netzwerkvorgang wurde unterbrochen. Versuchen Sie erneut, um fortzufahren.',
    identityGenerationFailed:
      'Es konnte lokal keine Remote-Netzwerkidentität erstellt werden. Versuchen Sie es erneut.',
    retry: 'Erneut versuchen',
    localStatus: 'Lokale Verbindung',
    connected: 'Verbunden',
    copyIP: 'Geräte-IP kopieren',
    ipCopied: 'Geräte-IP kopiert',
    devices: 'Geräte',
    emptyDevices: 'Es wurden noch keine weiteren Geräte hinzugefügt.'
  }),
  es: defineSipeedRuntimeMessages({
    enabling: 'Activando la red remota…',
    enableFailed: 'No se pudo activar la red remota.',
    officialActive:
      'Hay una identidad oficial de Tailscale activa. Esta versión no cambia de identidad directamente.',
    invalidIdentity:
      'La identidad de la red remota ya no es válida. Inténtalo de nuevo para crear una identidad nueva.',
    identityMissing: 'La identidad de la red remota no está disponible. Actívala para continuar.',
    rateLimited: 'Demasiados intentos. Vuelve a intentarlo en {{seconds}} segundos.',
    pairingUnavailable:
      'El servicio de vinculación no está disponible temporalmente. Inténtalo más tarde.',
    pairingUpstreamError:
      'El servicio de vinculación no pudo comunicarse con el controlador de red. Inténtalo más tarde.',
    pairingRejected:
      'El servicio de vinculación rechazó este dispositivo. Inténtalo de nuevo o contacta con soporte.',
    nodeNotFound:
      'No se encontró el dispositivo administrado. Inténtalo de nuevo para renovar su identidad de red.',
    protocolError:
      'El servicio de vinculación devolvió una respuesta inesperada. Inténtalo más tarde.',
    localAPIUnavailable:
      'El servicio local de red remota no está disponible. Inícialo e inténtalo de nuevo.',
    stateFileInvalid:
      'El estado guardado de la red remota no es válido. Restablécelo e inténtalo de nuevo.',
    stateWriteFailed:
      'No se pudo guardar el estado de la red remota. Comprueba el almacenamiento local e inténtalo de nuevo.',
    installFailed: 'No se pudo instalar el cliente de red remota. Inténtalo más tarde.',
    profileReadFailed: 'No se pudo leer el perfil local de la red remota. Inténtalo de nuevo.',
    hostnameUnavailable:
      'El nombre de host del dispositivo no está disponible. Configura uno e inténtalo de nuevo.',
    profileSetupFailed:
      'No se pudo preparar el perfil administrado de la red remota. Inténtalo de nuevo.',
    profileStartFailed:
      'No se pudo iniciar el perfil administrado de la red remota. Inténtalo de nuevo.',
    connectionFailed:
      'No se pudo establecer la conexión administrada de la red remota. Inténtalo de nuevo.',
    profileVerificationFailed:
      'No se pudo verificar el perfil administrado de la red remota. Inténtalo de nuevo.',
    rollbackFailed:
      'No se pudo restaurar el perfil anterior de la red remota. Comprueba Tailscale e inténtalo de nuevo.',
    interrupted:
      'La operación anterior de la red remota se interrumpió. Inténtalo de nuevo para continuar.',
    identityGenerationFailed:
      'No se pudo crear localmente una identidad de red remota. Inténtalo de nuevo.',
    retry: 'Reintentar',
    localStatus: 'Conexión local',
    connected: 'Conectado',
    copyIP: 'Copiar IP del dispositivo',
    ipCopied: 'IP del dispositivo copiada',
    devices: 'Dispositivos',
    emptyDevices: 'Todavía no se han añadido otros dispositivos.'
  }),
  fr: defineSipeedRuntimeMessages({
    enabling: 'Activation du réseau distant…',
    enableFailed: 'Impossible d’activer le réseau distant.',
    officialActive:
      'Une identité Tailscale officielle est active. Cette version ne permet pas de changer directement d’identité.',
    invalidIdentity:
      'L’identité du réseau distant n’est plus valide. Réessayez pour créer une nouvelle identité.',
    identityMissing:
      'L’identité du réseau distant n’est pas disponible. Activez-la pour continuer.',
    rateLimited: 'Trop de tentatives. Réessayez dans {{seconds}} secondes.',
    pairingUnavailable:
      'Le service d’association est temporairement indisponible. Réessayez plus tard.',
    pairingUpstreamError:
      'Le service d’association n’a pas pu joindre le contrôleur réseau. Réessayez plus tard.',
    pairingRejected:
      'Le service d’association a refusé cet appareil. Réessayez ou contactez l’assistance.',
    nodeNotFound: 'L’appareil géré est introuvable. Réessayez pour renouveler son identité réseau.',
    protocolError:
      'Le service d’association a renvoyé une réponse inattendue. Réessayez plus tard.',
    localAPIUnavailable:
      'Le service local de réseau distant est indisponible. Démarrez-le et réessayez.',
    stateFileInvalid:
      'L’état enregistré du réseau distant n’est pas valide. Réinitialisez-le et réessayez.',
    stateWriteFailed:
      'Impossible d’enregistrer l’état du réseau distant. Vérifiez le stockage local et réessayez.',
    installFailed: 'Impossible d’installer le client de réseau distant. Réessayez plus tard.',
    profileReadFailed: 'Impossible de lire le profil local du réseau distant. Réessayez.',
    hostnameUnavailable:
      'Le nom d’hôte de l’appareil est indisponible. Définissez-en un et réessayez.',
    profileSetupFailed: 'Impossible de préparer le profil géré du réseau distant. Réessayez.',
    profileStartFailed: 'Impossible de démarrer le profil géré du réseau distant. Réessayez.',
    connectionFailed: 'Impossible d’établir la connexion gérée au réseau distant. Réessayez.',
    profileVerificationFailed:
      'Impossible de vérifier le profil géré du réseau distant. Réessayez.',
    rollbackFailed:
      'Impossible de restaurer le profil précédent du réseau distant. Vérifiez Tailscale et réessayez.',
    interrupted:
      'L’opération précédente sur le réseau distant a été interrompue. Réessayez pour continuer.',
    identityGenerationFailed:
      'Impossible de créer localement une identité de réseau distant. Réessayez.',
    retry: 'Réessayer',
    localStatus: 'Connexion locale',
    connected: 'Connecté',
    copyIP: 'Copier l’adresse IP de l’appareil',
    ipCopied: 'Adresse IP de l’appareil copiée',
    devices: 'Appareils',
    emptyDevices: 'Aucun autre appareil n’a encore été ajouté.'
  }),
  id: defineSipeedRuntimeMessages({
    enabling: 'Mengaktifkan jaringan jarak jauh…',
    enableFailed: 'Jaringan jarak jauh tidak dapat diaktifkan.',
    officialActive:
      'Identitas Tailscale resmi sedang aktif. Versi ini tidak dapat beralih identitas secara langsung.',
    invalidIdentity:
      'Identitas jaringan jarak jauh sudah tidak valid. Coba lagi untuk membuat identitas baru.',
    identityMissing: 'Identitas jaringan jarak jauh tidak tersedia. Aktifkan untuk melanjutkan.',
    rateLimited: 'Terlalu banyak percobaan. Coba lagi dalam {{seconds}} detik.',
    pairingUnavailable: 'Layanan pemasangan untuk sementara tidak tersedia. Coba lagi nanti.',
    pairingUpstreamError:
      'Layanan pemasangan tidak dapat menjangkau pengontrol jaringan. Coba lagi nanti.',
    pairingRejected: 'Layanan pemasangan menolak perangkat ini. Coba lagi atau hubungi dukungan.',
    nodeNotFound:
      'Perangkat terkelola tidak ditemukan. Coba lagi untuk memperbarui identitas jaringannya.',
    protocolError: 'Layanan pemasangan mengembalikan respons yang tidak terduga. Coba lagi nanti.',
    localAPIUnavailable:
      'Layanan jaringan jarak jauh lokal tidak tersedia. Jalankan layanan lalu coba lagi.',
    stateFileInvalid:
      'Status jaringan jarak jauh yang tersimpan tidak valid. Atur ulang lalu coba lagi.',
    stateWriteFailed:
      'Status jaringan jarak jauh tidak dapat disimpan. Periksa penyimpanan lokal lalu coba lagi.',
    installFailed: 'Klien jaringan jarak jauh tidak dapat dipasang. Coba lagi nanti.',
    profileReadFailed: 'Profil jaringan jarak jauh lokal tidak dapat dibaca. Coba lagi.',
    hostnameUnavailable: 'Nama host perangkat tidak tersedia. Tetapkan nama host lalu coba lagi.',
    profileSetupFailed: 'Profil jaringan jarak jauh terkelola tidak dapat disiapkan. Coba lagi.',
    profileStartFailed: 'Profil jaringan jarak jauh terkelola tidak dapat dijalankan. Coba lagi.',
    connectionFailed: 'Koneksi jaringan jarak jauh terkelola tidak dapat dibuat. Coba lagi.',
    profileVerificationFailed:
      'Profil jaringan jarak jauh terkelola tidak dapat diverifikasi. Coba lagi.',
    rollbackFailed:
      'Profil jaringan jarak jauh sebelumnya tidak dapat dipulihkan. Periksa Tailscale lalu coba lagi.',
    interrupted: 'Operasi jaringan jarak jauh sebelumnya terputus. Coba lagi untuk melanjutkan.',
    identityGenerationFailed:
      'Identitas jaringan jarak jauh tidak dapat dibuat secara lokal. Coba lagi.',
    retry: 'Coba lagi',
    localStatus: 'Koneksi lokal',
    connected: 'Terhubung',
    copyIP: 'Salin IP perangkat',
    ipCopied: 'IP perangkat disalin',
    devices: 'Perangkat',
    emptyDevices: 'Belum ada perangkat lain yang ditambahkan.'
  }),
  it: defineSipeedRuntimeMessages({
    enabling: 'Attivazione della rete remota…',
    enableFailed: 'Impossibile attivare la rete remota.',
    officialActive:
      'È attiva un’identità Tailscale ufficiale. Questa versione non consente di cambiare direttamente identità.',
    invalidIdentity:
      'L’identità della rete remota non è più valida. Riprova per creare una nuova identità.',
    identityMissing: 'L’identità della rete remota non è disponibile. Attivala per continuare.',
    rateLimited: 'Troppi tentativi. Riprova tra {{seconds}} secondi.',
    pairingUnavailable:
      'Il servizio di associazione non è temporaneamente disponibile. Riprova più tardi.',
    pairingUpstreamError:
      'Il servizio di associazione non ha potuto raggiungere il controller di rete. Riprova più tardi.',
    pairingRejected:
      'Il servizio di associazione ha rifiutato questo dispositivo. Riprova o contatta l’assistenza.',
    nodeNotFound:
      'Il dispositivo gestito non è stato trovato. Riprova per rinnovarne l’identità di rete.',
    protocolError:
      'Il servizio di associazione ha restituito una risposta imprevista. Riprova più tardi.',
    localAPIUnavailable:
      'Il servizio locale della rete remota non è disponibile. Avvialo e riprova.',
    stateFileInvalid: 'Lo stato salvato della rete remota non è valido. Reimpostalo e riprova.',
    stateWriteFailed:
      'Impossibile salvare lo stato della rete remota. Controlla l’archiviazione locale e riprova.',
    installFailed: 'Impossibile installare il client della rete remota. Riprova più tardi.',
    profileReadFailed: 'Impossibile leggere il profilo locale della rete remota. Riprova.',
    hostnameUnavailable: 'Il nome host del dispositivo non è disponibile. Impostane uno e riprova.',
    profileSetupFailed: 'Impossibile preparare il profilo gestito della rete remota. Riprova.',
    profileStartFailed: 'Impossibile avviare il profilo gestito della rete remota. Riprova.',
    connectionFailed: 'Impossibile stabilire la connessione gestita alla rete remota. Riprova.',
    profileVerificationFailed:
      'Impossibile verificare il profilo gestito della rete remota. Riprova.',
    rollbackFailed:
      'Impossibile ripristinare il profilo precedente della rete remota. Controlla Tailscale e riprova.',
    interrupted:
      'L’operazione precedente sulla rete remota è stata interrotta. Riprova per continuare.',
    identityGenerationFailed: 'Impossibile creare localmente un’identità di rete remota. Riprova.',
    retry: 'Riprova',
    localStatus: 'Connessione locale',
    connected: 'Connesso',
    copyIP: 'Copia IP del dispositivo',
    ipCopied: 'IP del dispositivo copiato',
    devices: 'Dispositivi',
    emptyDevices: 'Non sono ancora stati aggiunti altri dispositivi.'
  }),
  pl: defineSipeedRuntimeMessages({
    enabling: 'Włączanie sieci zdalnej…',
    enableFailed: 'Nie udało się włączyć sieci zdalnej.',
    officialActive:
      'Aktywna jest oficjalna tożsamość Tailscale. Ta wersja nie przełącza bezpośrednio między tożsamościami.',
    invalidIdentity:
      'Tożsamość sieci zdalnej utraciła ważność. Spróbuj ponownie, aby utworzyć nową tożsamość.',
    identityMissing: 'Tożsamość sieci zdalnej jest niedostępna. Włącz ją, aby kontynuować.',
    rateLimited: 'Zbyt wiele prób. Spróbuj ponownie za {{seconds}} s.',
    pairingUnavailable: 'Usługa parowania jest tymczasowo niedostępna. Spróbuj ponownie później.',
    pairingUpstreamError:
      'Usługa parowania nie mogła połączyć się z kontrolerem sieci. Spróbuj ponownie później.',
    pairingRejected:
      'Usługa parowania odrzuciła to urządzenie. Spróbuj ponownie lub skontaktuj się z pomocą techniczną.',
    nodeNotFound:
      'Nie znaleziono zarządzanego urządzenia. Spróbuj ponownie, aby odnowić jego tożsamość sieciową.',
    protocolError: 'Usługa parowania zwróciła nieoczekiwaną odpowiedź. Spróbuj ponownie później.',
    localAPIUnavailable:
      'Lokalna usługa sieci zdalnej jest niedostępna. Uruchom ją i spróbuj ponownie.',
    stateFileInvalid:
      'Zapisany stan sieci zdalnej jest nieprawidłowy. Zresetuj go i spróbuj ponownie.',
    stateWriteFailed:
      'Nie udało się zapisać stanu sieci zdalnej. Sprawdź pamięć lokalną i spróbuj ponownie.',
    installFailed: 'Nie udało się zainstalować klienta sieci zdalnej. Spróbuj ponownie później.',
    profileReadFailed: 'Nie udało się odczytać lokalnego profilu sieci zdalnej. Spróbuj ponownie.',
    hostnameUnavailable:
      'Nazwa hosta urządzenia jest niedostępna. Ustaw nazwę hosta i spróbuj ponownie.',
    profileSetupFailed:
      'Nie udało się przygotować zarządzanego profilu sieci zdalnej. Spróbuj ponownie.',
    profileStartFailed:
      'Nie udało się uruchomić zarządzanego profilu sieci zdalnej. Spróbuj ponownie.',
    connectionFailed:
      'Nie udało się nawiązać zarządzanego połączenia z siecią zdalną. Spróbuj ponownie.',
    profileVerificationFailed:
      'Nie udało się zweryfikować zarządzanego profilu sieci zdalnej. Spróbuj ponownie.',
    rollbackFailed:
      'Nie udało się przywrócić poprzedniego profilu sieci zdalnej. Sprawdź Tailscale i spróbuj ponownie.',
    interrupted:
      'Poprzednia operacja sieci zdalnej została przerwana. Spróbuj ponownie, aby kontynuować.',
    identityGenerationFailed:
      'Nie udało się lokalnie utworzyć tożsamości sieci zdalnej. Spróbuj ponownie.',
    retry: 'Spróbuj ponownie',
    localStatus: 'Połączenie lokalne',
    connected: 'Połączono',
    copyIP: 'Kopiuj adres IP urządzenia',
    ipCopied: 'Skopiowano adres IP urządzenia',
    devices: 'Urządzenia',
    emptyDevices: 'Nie dodano jeszcze żadnych innych urządzeń.'
  }),
  ru: defineSipeedRuntimeMessages({
    enabling: 'Включение удалённой сети…',
    enableFailed: 'Не удалось включить удалённую сеть.',
    officialActive:
      'Активна официальная учётная запись Tailscale. Эта версия не поддерживает прямое переключение учётных записей.',
    invalidIdentity:
      'Идентификатор удалённой сети больше недействителен. Повторите попытку, чтобы создать новый.',
    identityMissing: 'Идентификатор удалённой сети недоступен. Включите сеть, чтобы продолжить.',
    rateLimited: 'Слишком много попыток. Повторите через {{seconds}} сек.',
    pairingUnavailable: 'Сервис сопряжения временно недоступен. Повторите попытку позже.',
    pairingUpstreamError:
      'Сервис сопряжения не смог связаться с контроллером сети. Повторите попытку позже.',
    pairingRejected:
      'Сервис сопряжения отклонил это устройство. Повторите попытку или обратитесь в службу поддержки.',
    nodeNotFound:
      'Управляемое устройство не найдено. Повторите попытку, чтобы обновить его сетевой идентификатор.',
    protocolError: 'Сервис сопряжения вернул неожиданный ответ. Повторите попытку позже.',
    localAPIUnavailable:
      'Локальная служба удалённой сети недоступна. Запустите её и повторите попытку.',
    stateFileInvalid:
      'Сохранённое состояние удалённой сети недействительно. Сбросьте его и повторите попытку.',
    stateWriteFailed:
      'Не удалось сохранить состояние удалённой сети. Проверьте локальное хранилище и повторите попытку.',
    installFailed: 'Не удалось установить клиент удалённой сети. Повторите попытку позже.',
    profileReadFailed: 'Не удалось прочитать локальный профиль удалённой сети. Повторите попытку.',
    hostnameUnavailable: 'Имя хоста устройства недоступно. Задайте имя хоста и повторите попытку.',
    profileSetupFailed:
      'Не удалось подготовить управляемый профиль удалённой сети. Повторите попытку.',
    profileStartFailed:
      'Не удалось запустить управляемый профиль удалённой сети. Повторите попытку.',
    connectionFailed:
      'Не удалось установить управляемое соединение с удалённой сетью. Повторите попытку.',
    profileVerificationFailed:
      'Не удалось проверить управляемый профиль удалённой сети. Повторите попытку.',
    rollbackFailed:
      'Не удалось восстановить предыдущий профиль удалённой сети. Проверьте Tailscale и повторите попытку.',
    interrupted:
      'Предыдущая операция удалённой сети была прервана. Повторите попытку, чтобы продолжить.',
    identityGenerationFailed:
      'Не удалось локально создать идентификатор удалённой сети. Повторите попытку.',
    retry: 'Повторить',
    localStatus: 'Локальное подключение',
    connected: 'Подключено',
    copyIP: 'Копировать IP-адрес устройства',
    ipCopied: 'IP-адрес устройства скопирован',
    devices: 'Устройства',
    emptyDevices: 'Другие устройства пока не добавлены.'
  }),
  ko: defineSipeedRuntimeMessages({
    enabling: '원격 네트워크를 활성화하는 중…',
    enableFailed: '원격 네트워크를 활성화하지 못했습니다.',
    officialActive:
      '공식 Tailscale ID가 활성화되어 있습니다. 이 버전에서는 ID를 직접 전환할 수 없습니다.',
    invalidIdentity:
      '원격 네트워크 ID가 더 이상 유효하지 않습니다. 다시 시도하여 새 ID를 만드세요.',
    identityMissing:
      '원격 네트워크 ID를 사용할 수 없습니다. 계속하려면 원격 네트워크를 활성화하세요.',
    rateLimited: '시도 횟수가 너무 많습니다. {{seconds}}초 후 다시 시도하세요.',
    pairingUnavailable: '페어링 서비스를 일시적으로 사용할 수 없습니다. 나중에 다시 시도하세요.',
    pairingUpstreamError:
      '페어링 서비스가 네트워크 컨트롤러에 연결하지 못했습니다. 나중에 다시 시도하세요.',
    pairingRejected: '페어링 서비스가 이 기기를 거부했습니다. 다시 시도하거나 지원팀에 문의하세요.',
    nodeNotFound: '관리 대상 기기를 찾지 못했습니다. 다시 시도하여 네트워크 ID를 갱신하세요.',
    protocolError: '페어링 서비스에서 예기치 않은 응답을 반환했습니다. 나중에 다시 시도하세요.',
    localAPIUnavailable:
      '로컬 원격 네트워크 서비스를 사용할 수 없습니다. 서비스를 시작한 후 다시 시도하세요.',
    stateFileInvalid: '저장된 원격 네트워크 상태가 올바르지 않습니다. 초기화한 후 다시 시도하세요.',
    stateWriteFailed:
      '원격 네트워크 상태를 저장하지 못했습니다. 로컬 저장소를 확인한 후 다시 시도하세요.',
    installFailed: '원격 네트워크 클라이언트를 설치하지 못했습니다. 나중에 다시 시도하세요.',
    profileReadFailed: '로컬 원격 네트워크 프로필을 읽지 못했습니다. 다시 시도하세요.',
    hostnameUnavailable:
      '기기 호스트 이름을 사용할 수 없습니다. 호스트 이름을 설정한 후 다시 시도하세요.',
    profileSetupFailed: '관리형 원격 네트워크 프로필을 준비하지 못했습니다. 다시 시도하세요.',
    profileStartFailed: '관리형 원격 네트워크 프로필을 시작하지 못했습니다. 다시 시도하세요.',
    connectionFailed: '관리형 원격 네트워크 연결을 설정하지 못했습니다. 다시 시도하세요.',
    profileVerificationFailed:
      '관리형 원격 네트워크 프로필을 확인하지 못했습니다. 다시 시도하세요.',
    rollbackFailed:
      '이전 원격 네트워크 프로필을 복원하지 못했습니다. Tailscale을 확인한 후 다시 시도하세요.',
    interrupted: '이전 원격 네트워크 작업이 중단되었습니다. 계속하려면 다시 시도하세요.',
    identityGenerationFailed: '로컬에서 원격 네트워크 ID를 만들지 못했습니다. 다시 시도하세요.',
    retry: '다시 시도',
    localStatus: '로컬 연결',
    connected: '연결됨',
    copyIP: '기기 IP 복사',
    ipCopied: '기기 IP를 복사했습니다',
    devices: '기기',
    emptyDevices: '아직 추가된 다른 기기가 없습니다.'
  }),
  zh: defineSipeedRuntimeMessages({
    enabling: '正在启用远程网络…',
    enableFailed: '远程网络启用失败。',
    officialActive: '当前已有官方 Tailscale 身份，本版本暂不支持直接切换身份。',
    invalidIdentity: '远程网络身份已失效，请重试以创建新身份。',
    identityMissing: '远程网络身份不可用，请先启用后再继续。',
    rateLimited: '请求过于频繁，请在 {{seconds}} 秒后重试。',
    pairingUnavailable: '配对服务暂时不可用，请稍后重试。',
    pairingUpstreamError: '配对服务无法连接网络控制器，请稍后重试。',
    pairingRejected: '配对服务拒绝了此设备，请重试或联系技术支持。',
    nodeNotFound: '未找到已托管设备，请重试以刷新其网络身份。',
    protocolError: '配对服务返回了异常响应，请稍后重试。',
    localAPIUnavailable: '本地远程网络服务不可用，请启动服务后重试。',
    stateFileInvalid: '已保存的远程网络状态无效，请重置后重试。',
    stateWriteFailed: '远程网络状态保存失败，请检查本地存储后重试。',
    installFailed: '远程网络客户端安装失败，请稍后重试。',
    profileReadFailed: '本地远程网络配置读取失败，请重试。',
    hostnameUnavailable: '无法获取设备主机名，请设置主机名后重试。',
    profileSetupFailed: '托管远程网络配置准备失败，请重试。',
    profileStartFailed: '托管远程网络配置启动失败，请重试。',
    connectionFailed: '托管远程网络连接建立失败，请重试。',
    profileVerificationFailed: '托管远程网络配置验证失败，请重试。',
    rollbackFailed: '之前的远程网络配置恢复失败，请检查 Tailscale 后重试。',
    interrupted: '上一次远程网络操作已中断，请重试以继续。',
    identityGenerationFailed: '无法在本地创建远程网络身份，请重试。',
    retry: '重试',
    localStatus: '本机连接',
    connected: '已连接',
    copyIP: '复制设备地址',
    ipCopied: '设备地址已复制',
    devices: '设备',
    emptyDevices: '尚未添加其他设备。'
  }),
  zh_tw: defineSipeedRuntimeMessages({
    enabling: '正在啟用遠端網路…',
    enableFailed: '遠端網路啟用失敗。',
    officialActive: '目前已有官方 Tailscale 身分，此版本暫不支援直接切換身分。',
    invalidIdentity: '遠端網路身分已失效，請重試以建立新身分。',
    identityMissing: '遠端網路身分無法使用，請先啟用後再繼續。',
    rateLimited: '要求過於頻繁，請在 {{seconds}} 秒後重試。',
    pairingUnavailable: '配對服務暫時無法使用，請稍後重試。',
    pairingUpstreamError: '配對服務無法連線至網路控制器，請稍後重試。',
    pairingRejected: '配對服務拒絕了此裝置，請重試或聯絡技術支援。',
    nodeNotFound: '找不到已託管裝置，請重試以重新整理其網路身分。',
    protocolError: '配對服務傳回非預期的回應，請稍後重試。',
    localAPIUnavailable: '本機遠端網路服務無法使用，請啟動服務後重試。',
    stateFileInvalid: '已儲存的遠端網路狀態無效，請重設後重試。',
    stateWriteFailed: '遠端網路狀態儲存失敗，請檢查本機儲存空間後重試。',
    installFailed: '遠端網路用戶端安裝失敗，請稍後重試。',
    profileReadFailed: '無法讀取本機遠端網路設定檔，請重試。',
    hostnameUnavailable: '無法取得裝置主機名稱，請設定主機名稱後重試。',
    profileSetupFailed: '無法準備託管遠端網路設定檔，請重試。',
    profileStartFailed: '無法啟動託管遠端網路設定檔，請重試。',
    connectionFailed: '無法建立託管遠端網路連線，請重試。',
    profileVerificationFailed: '無法驗證託管遠端網路設定檔，請重試。',
    rollbackFailed: '無法還原先前的遠端網路設定檔，請檢查 Tailscale 後重試。',
    interrupted: '上一次遠端網路操作已中斷，請重試以繼續。',
    identityGenerationFailed: '無法在本機建立遠端網路身分，請重試。',
    retry: '重試',
    localStatus: '本機連線',
    connected: '已連線',
    copyIP: '複製裝置 IP',
    ipCopied: '已複製裝置 IP',
    devices: '裝置',
    emptyDevices: '尚未新增其他裝置。'
  }),
  hu: defineSipeedRuntimeMessages({
    enabling: 'Távoli hálózat engedélyezése…',
    enableFailed: 'Nem sikerült engedélyezni a távoli hálózatot.',
    officialActive:
      'Egy hivatalos Tailscale-azonosító aktív. Ez a verzió nem vált közvetlenül az azonosítók között.',
    invalidIdentity:
      'A távoli hálózati azonosító már nem érvényes. Próbálja újra egy új azonosító létrehozásához.',
    identityMissing: 'A távoli hálózati azonosító nem érhető el. A folytatáshoz engedélyezze.',
    rateLimited: 'Túl sok próbálkozás. Próbálja újra {{seconds}} másodperc múlva.',
    pairingUnavailable:
      'A párosítási szolgáltatás átmenetileg nem érhető el. Próbálja újra később.',
    pairingUpstreamError:
      'A párosítási szolgáltatás nem érte el a hálózati vezérlőt. Próbálja újra később.',
    pairingRejected:
      'A párosítási szolgáltatás elutasította ezt az eszközt. Próbálja újra, vagy forduljon az ügyfélszolgálathoz.',
    nodeNotFound:
      'A felügyelt eszköz nem található. Próbálja újra a hálózati azonosító megújításához.',
    protocolError: 'A párosítási szolgáltatás váratlan választ adott. Próbálja újra később.',
    localAPIUnavailable:
      'A helyi távoli hálózati szolgáltatás nem érhető el. Indítsa el, majd próbálja újra.',
    stateFileInvalid:
      'A távoli hálózat mentett állapota érvénytelen. Állítsa alaphelyzetbe, majd próbálja újra.',
    stateWriteFailed:
      'Nem sikerült menteni a távoli hálózat állapotát. Ellenőrizze a helyi tárhelyet, majd próbálja újra.',
    installFailed: 'Nem sikerült telepíteni a távoli hálózati klienst. Próbálja újra később.',
    profileReadFailed: 'Nem sikerült beolvasni a helyi távoli hálózati profilt. Próbálja újra.',
    hostnameUnavailable:
      'Az eszköz állomásneve nem érhető el. Állítson be egy állomásnevet, majd próbálja újra.',
    profileSetupFailed:
      'Nem sikerült előkészíteni a felügyelt távoli hálózati profilt. Próbálja újra.',
    profileStartFailed:
      'Nem sikerült elindítani a felügyelt távoli hálózati profilt. Próbálja újra.',
    connectionFailed:
      'Nem sikerült létrehozni a felügyelt távoli hálózati kapcsolatot. Próbálja újra.',
    profileVerificationFailed:
      'Nem sikerült ellenőrizni a felügyelt távoli hálózati profilt. Próbálja újra.',
    rollbackFailed:
      'Nem sikerült visszaállítani az előző távoli hálózati profilt. Ellenőrizze a Tailscale-t, majd próbálja újra.',
    interrupted: 'Az előző távoli hálózati művelet megszakadt. A folytatáshoz próbálja újra.',
    identityGenerationFailed:
      'Nem sikerült helyben létrehozni a távoli hálózati azonosítót. Próbálja újra.',
    retry: 'Újrapróbálkozás',
    localStatus: 'Helyi kapcsolat',
    connected: 'Csatlakoztatva',
    copyIP: 'Eszköz IP-címének másolása',
    ipCopied: 'Az eszköz IP-címe másolva',
    devices: 'Eszközök',
    emptyDevices: 'Még nincs másik eszköz hozzáadva.'
  }),
  vi: defineSipeedRuntimeMessages({
    enabling: 'Đang bật mạng từ xa…',
    enableFailed: 'Không thể bật mạng từ xa.',
    officialActive:
      'Một danh tính Tailscale chính thức đang hoạt động. Phiên bản này không chuyển đổi trực tiếp giữa các danh tính.',
    invalidIdentity: 'Danh tính mạng từ xa không còn hợp lệ. Hãy thử lại để tạo danh tính mới.',
    identityMissing: 'Không có danh tính mạng từ xa. Hãy bật mạng để tiếp tục.',
    rateLimited: 'Quá nhiều lần thử. Hãy thử lại sau {{seconds}} giây.',
    pairingUnavailable: 'Dịch vụ ghép nối tạm thời không khả dụng. Hãy thử lại sau.',
    pairingUpstreamError:
      'Dịch vụ ghép nối không thể kết nối với bộ điều khiển mạng. Hãy thử lại sau.',
    pairingRejected: 'Dịch vụ ghép nối đã từ chối thiết bị này. Hãy thử lại hoặc liên hệ hỗ trợ.',
    nodeNotFound:
      'Không tìm thấy thiết bị được quản lý. Hãy thử lại để làm mới danh tính mạng của thiết bị.',
    protocolError: 'Dịch vụ ghép nối trả về phản hồi không mong đợi. Hãy thử lại sau.',
    localAPIUnavailable:
      'Dịch vụ mạng từ xa cục bộ không khả dụng. Hãy khởi động dịch vụ rồi thử lại.',
    stateFileInvalid: 'Trạng thái mạng từ xa đã lưu không hợp lệ. Hãy đặt lại rồi thử lại.',
    stateWriteFailed:
      'Không thể lưu trạng thái mạng từ xa. Hãy kiểm tra bộ nhớ cục bộ rồi thử lại.',
    installFailed: 'Không thể cài đặt ứng dụng mạng từ xa. Hãy thử lại sau.',
    profileReadFailed: 'Không thể đọc hồ sơ mạng từ xa cục bộ. Hãy thử lại.',
    hostnameUnavailable:
      'Tên máy chủ của thiết bị không khả dụng. Hãy đặt tên máy chủ rồi thử lại.',
    profileSetupFailed: 'Không thể chuẩn bị hồ sơ mạng từ xa được quản lý. Hãy thử lại.',
    profileStartFailed: 'Không thể khởi động hồ sơ mạng từ xa được quản lý. Hãy thử lại.',
    connectionFailed: 'Không thể thiết lập kết nối mạng từ xa được quản lý. Hãy thử lại.',
    profileVerificationFailed: 'Không thể xác minh hồ sơ mạng từ xa được quản lý. Hãy thử lại.',
    rollbackFailed:
      'Không thể khôi phục hồ sơ mạng từ xa trước đó. Hãy kiểm tra Tailscale rồi thử lại.',
    interrupted: 'Thao tác mạng từ xa trước đó đã bị gián đoạn. Hãy thử lại để tiếp tục.',
    identityGenerationFailed: 'Không thể tạo danh tính mạng từ xa trên thiết bị. Hãy thử lại.',
    retry: 'Thử lại',
    localStatus: 'Kết nối cục bộ',
    connected: 'Đã kết nối',
    copyIP: 'Sao chép IP thiết bị',
    ipCopied: 'Đã sao chép IP thiết bị',
    devices: 'Thiết bị',
    emptyDevices: 'Chưa có thiết bị nào khác được thêm.'
  }),
  ja: defineSipeedRuntimeMessages({
    enabling: 'リモートネットワークを有効化しています…',
    enableFailed: 'リモートネットワークを有効化できませんでした。',
    officialActive:
      '公式 Tailscale ID が有効です。このバージョンでは ID を直接切り替えられません。',
    invalidIdentity:
      'リモートネットワーク ID は無効になっています。再試行して新しい ID を作成してください。',
    identityMissing: 'リモートネットワーク ID を利用できません。続行するには有効化してください。',
    rateLimited: '試行回数が多すぎます。{{seconds}} 秒後に再試行してください。',
    pairingUnavailable: 'ペアリングサービスは一時的に利用できません。後でもう一度お試しください。',
    pairingUpstreamError:
      'ペアリングサービスがネットワークコントローラーに接続できませんでした。後でもう一度お試しください。',
    pairingRejected:
      'ペアリングサービスがこのデバイスを拒否しました。再試行するか、サポートにお問い合わせください。',
    nodeNotFound:
      '管理対象デバイスが見つかりません。再試行してネットワーク ID を更新してください。',
    protocolError:
      'ペアリングサービスから予期しない応答が返されました。後でもう一度お試しください。',
    localAPIUnavailable:
      'ローカルのリモートネットワークサービスを利用できません。サービスを起動して再試行してください。',
    stateFileInvalid:
      '保存されたリモートネットワークの状態が無効です。リセットして再試行してください。',
    stateWriteFailed:
      'リモートネットワークの状態を保存できませんでした。ローカルストレージを確認して再試行してください。',
    installFailed:
      'リモートネットワーククライアントをインストールできませんでした。後でもう一度お試しください。',
    profileReadFailed:
      'ローカルのリモートネットワークプロファイルを読み取れませんでした。再試行してください。',
    hostnameUnavailable:
      'デバイスのホスト名を取得できません。ホスト名を設定して再試行してください。',
    profileSetupFailed:
      '管理対象のリモートネットワークプロファイルを準備できませんでした。再試行してください。',
    profileStartFailed:
      '管理対象のリモートネットワークプロファイルを起動できませんでした。再試行してください。',
    connectionFailed:
      '管理対象のリモートネットワーク接続を確立できませんでした。再試行してください。',
    profileVerificationFailed:
      '管理対象のリモートネットワークプロファイルを確認できませんでした。再試行してください。',
    rollbackFailed:
      '以前のリモートネットワークプロファイルを復元できませんでした。Tailscale を確認して再試行してください。',
    interrupted: '前回のリモートネットワーク操作が中断されました。続行するには再試行してください。',
    identityGenerationFailed:
      'リモートネットワーク ID をローカルで作成できませんでした。再試行してください。',
    retry: '再試行',
    localStatus: 'ローカル接続',
    connected: '接続済み',
    copyIP: 'デバイス IP をコピー',
    ipCopied: 'デバイス IP をコピーしました',
    devices: 'デバイス',
    emptyDevices: '他のデバイスはまだ追加されていません。'
  }),
  cz: defineSipeedRuntimeMessages({
    enabling: 'Zapínání vzdálené sítě…',
    enableFailed: 'Vzdálenou síť se nepodařilo zapnout.',
    officialActive:
      'Je aktivní oficiální identita Tailscale. Tato verze neumí identity přepínat přímo.',
    invalidIdentity:
      'Identita vzdálené sítě již není platná. Zkuste to znovu a vytvořte novou identitu.',
    identityMissing: 'Identita vzdálené sítě není k dispozici. Chcete-li pokračovat, zapněte ji.',
    rateLimited: 'Příliš mnoho pokusů. Zkuste to znovu za {{seconds}} sekund.',
    pairingUnavailable: 'Služba párování je dočasně nedostupná. Zkuste to znovu později.',
    pairingUpstreamError:
      'Služba párování se nemohla spojit se síťovým řadičem. Zkuste to znovu později.',
    pairingRejected:
      'Služba párování toto zařízení odmítla. Zkuste to znovu nebo kontaktujte podporu.',
    nodeNotFound:
      'Spravované zařízení nebylo nalezeno. Zkuste znovu obnovit jeho síťovou identitu.',
    protocolError: 'Služba párování vrátila neočekávanou odpověď. Zkuste to znovu později.',
    localAPIUnavailable:
      'Místní služba vzdálené sítě není k dispozici. Spusťte ji a zkuste to znovu.',
    stateFileInvalid: 'Uložený stav vzdálené sítě je neplatný. Resetujte jej a zkuste to znovu.',
    stateWriteFailed:
      'Stav vzdálené sítě se nepodařilo uložit. Zkontrolujte místní úložiště a zkuste to znovu.',
    installFailed: 'Klienta vzdálené sítě se nepodařilo nainstalovat. Zkuste to znovu později.',
    profileReadFailed: 'Místní profil vzdálené sítě se nepodařilo načíst. Zkuste to znovu.',
    hostnameUnavailable:
      'Název hostitele zařízení není k dispozici. Nastavte jej a zkuste to znovu.',
    profileSetupFailed: 'Spravovaný profil vzdálené sítě se nepodařilo připravit. Zkuste to znovu.',
    profileStartFailed: 'Spravovaný profil vzdálené sítě se nepodařilo spustit. Zkuste to znovu.',
    connectionFailed:
      'Spravované připojení ke vzdálené síti se nepodařilo navázat. Zkuste to znovu.',
    profileVerificationFailed:
      'Spravovaný profil vzdálené sítě se nepodařilo ověřit. Zkuste to znovu.',
    rollbackFailed:
      'Předchozí profil vzdálené sítě se nepodařilo obnovit. Zkontrolujte Tailscale a zkuste to znovu.',
    interrupted: 'Předchozí operace vzdálené sítě byla přerušena. Pokračujte dalším pokusem.',
    identityGenerationFailed:
      'Identitu vzdálené sítě se nepodařilo vytvořit místně. Zkuste to znovu.',
    retry: 'Zkusit znovu',
    localStatus: 'Místní připojení',
    connected: 'Připojeno',
    copyIP: 'Kopírovat IP adresu zařízení',
    ipCopied: 'IP adresa zařízení zkopírována',
    devices: 'Zařízení',
    emptyDevices: 'Zatím nebyla přidána žádná další zařízení.'
  }),
  uk: defineSipeedRuntimeMessages({
    enabling: 'Увімкнення віддаленої мережі…',
    enableFailed: 'Не вдалося ввімкнути віддалену мережу.',
    officialActive:
      'Активна офіційна ідентичність Tailscale. Ця версія не перемикає ідентичності безпосередньо.',
    invalidIdentity:
      'Ідентичність віддаленої мережі більше не дійсна. Повторіть спробу, щоб створити нову.',
    identityMissing: 'Ідентичність віддаленої мережі недоступна. Увімкніть її, щоб продовжити.',
    rateLimited: 'Забагато спроб. Повторіть через {{seconds}} с.',
    pairingUnavailable: 'Служба сполучення тимчасово недоступна. Повторіть спробу пізніше.',
    pairingUpstreamError:
      'Службі сполучення не вдалося зв’язатися з контролером мережі. Повторіть спробу пізніше.',
    pairingRejected:
      'Служба сполучення відхилила цей пристрій. Повторіть спробу або зверніться до служби підтримки.',
    nodeNotFound:
      'Керований пристрій не знайдено. Повторіть спробу, щоб оновити його мережеву ідентичність.',
    protocolError: 'Служба сполучення повернула неочікувану відповідь. Повторіть спробу пізніше.',
    localAPIUnavailable:
      'Локальна служба віддаленої мережі недоступна. Запустіть її та повторіть спробу.',
    stateFileInvalid:
      'Збережений стан віддаленої мережі недійсний. Скиньте його та повторіть спробу.',
    stateWriteFailed:
      'Не вдалося зберегти стан віддаленої мережі. Перевірте локальне сховище та повторіть спробу.',
    installFailed: 'Не вдалося встановити клієнт віддаленої мережі. Повторіть спробу пізніше.',
    profileReadFailed:
      'Не вдалося прочитати локальний профіль віддаленої мережі. Повторіть спробу.',
    hostnameUnavailable: 'Ім’я хоста пристрою недоступне. Задайте ім’я хоста та повторіть спробу.',
    profileSetupFailed:
      'Не вдалося підготувати керований профіль віддаленої мережі. Повторіть спробу.',
    profileStartFailed:
      'Не вдалося запустити керований профіль віддаленої мережі. Повторіть спробу.',
    connectionFailed:
      'Не вдалося встановити кероване з’єднання з віддаленою мережею. Повторіть спробу.',
    profileVerificationFailed:
      'Не вдалося перевірити керований профіль віддаленої мережі. Повторіть спробу.',
    rollbackFailed:
      'Не вдалося відновити попередній профіль віддаленої мережі. Перевірте Tailscale та повторіть спробу.',
    interrupted:
      'Попередню операцію віддаленої мережі було перервано. Повторіть спробу, щоб продовжити.',
    identityGenerationFailed:
      'Не вдалося локально створити ідентичність віддаленої мережі. Повторіть спробу.',
    retry: 'Повторити',
    localStatus: 'Локальне підключення',
    connected: 'Підключено',
    copyIP: 'Копіювати IP-адресу пристрою',
    ipCopied: 'IP-адресу пристрою скопійовано',
    devices: 'Пристрої',
    emptyDevices: 'Інших пристроїв ще не додано.'
  }),
  nb: defineSipeedRuntimeMessages({
    enabling: 'Aktiverer fjernnettverk…',
    enableFailed: 'Fjernnettverket kunne ikke aktiveres.',
    officialActive:
      'En offisiell Tailscale-identitet er aktiv. Denne versjonen bytter ikke identitet direkte.',
    invalidIdentity:
      'Fjernnettverksidentiteten er ikke lenger gyldig. Prøv på nytt for å opprette en ny identitet.',
    identityMissing: 'Fjernnettverksidentiteten er ikke tilgjengelig. Aktiver den for å fortsette.',
    rateLimited: 'For mange forsøk. Prøv på nytt om {{seconds}} sekunder.',
    pairingUnavailable: 'Sammenkoblingstjenesten er midlertidig utilgjengelig. Prøv igjen senere.',
    pairingUpstreamError:
      'Sammenkoblingstjenesten kunne ikke nå nettverkskontrolleren. Prøv igjen senere.',
    pairingRejected:
      'Sammenkoblingstjenesten avviste denne enheten. Prøv på nytt eller kontakt kundestøtte.',
    nodeNotFound:
      'Den administrerte enheten ble ikke funnet. Prøv på nytt for å fornye nettverksidentiteten.',
    protocolError: 'Sammenkoblingstjenesten returnerte et uventet svar. Prøv igjen senere.',
    localAPIUnavailable:
      'Den lokale fjernnettverkstjenesten er utilgjengelig. Start den og prøv på nytt.',
    stateFileInvalid:
      'Den lagrede fjernnettverkstilstanden er ugyldig. Tilbakestill den og prøv på nytt.',
    stateWriteFailed:
      'Fjernnettverkstilstanden kunne ikke lagres. Kontroller lokal lagring og prøv på nytt.',
    installFailed: 'Fjernnettverksklienten kunne ikke installeres. Prøv igjen senere.',
    profileReadFailed: 'Den lokale fjernnettverksprofilen kunne ikke leses. Prøv på nytt.',
    hostnameUnavailable: 'Enhetens vertsnavn er utilgjengelig. Angi et vertsnavn og prøv på nytt.',
    profileSetupFailed:
      'Den administrerte fjernnettverksprofilen kunne ikke klargjøres. Prøv på nytt.',
    profileStartFailed:
      'Den administrerte fjernnettverksprofilen kunne ikke startes. Prøv på nytt.',
    connectionFailed:
      'Den administrerte fjernnettverkstilkoblingen kunne ikke opprettes. Prøv på nytt.',
    profileVerificationFailed:
      'Den administrerte fjernnettverksprofilen kunne ikke bekreftes. Prøv på nytt.',
    rollbackFailed:
      'Den forrige fjernnettverksprofilen kunne ikke gjenopprettes. Kontroller Tailscale og prøv på nytt.',
    interrupted: 'Den forrige fjernnettverksoperasjonen ble avbrutt. Prøv på nytt for å fortsette.',
    identityGenerationFailed:
      'En fjernnettverksidentitet kunne ikke opprettes lokalt. Prøv på nytt.',
    retry: 'Prøv på nytt',
    localStatus: 'Lokal tilkobling',
    connected: 'Tilkoblet',
    copyIP: 'Kopier enhetens IP-adresse',
    ipCopied: 'Enhetens IP-adresse er kopiert',
    devices: 'Enheter',
    emptyDevices: 'Ingen andre enheter er lagt til ennå.'
  }),
  th: defineSipeedRuntimeMessages({
    enabling: 'กำลังเปิดใช้งานเครือข่ายระยะไกล…',
    enableFailed: 'ไม่สามารถเปิดใช้งานเครือข่ายระยะไกลได้',
    officialActive:
      'มีข้อมูลประจำตัว Tailscale อย่างเป็นทางการที่ใช้งานอยู่ เวอร์ชันนี้ไม่รองรับการสลับข้อมูลประจำตัวโดยตรง',
    invalidIdentity:
      'ข้อมูลประจำตัวเครือข่ายระยะไกลใช้ไม่ได้แล้ว โปรดลองอีกครั้งเพื่อสร้างข้อมูลประจำตัวใหม่',
    identityMissing: 'ไม่มีข้อมูลประจำตัวเครือข่ายระยะไกล โปรดเปิดใช้งานเพื่อดำเนินการต่อ',
    rateLimited: 'ลองหลายครั้งเกินไป โปรดลองอีกครั้งใน {{seconds}} วินาที',
    pairingUnavailable: 'บริการจับคู่ไม่พร้อมใช้งานชั่วคราว โปรดลองอีกครั้งภายหลัง',
    pairingUpstreamError:
      'บริการจับคู่ไม่สามารถเชื่อมต่อกับตัวควบคุมเครือข่ายได้ โปรดลองอีกครั้งภายหลัง',
    pairingRejected: 'บริการจับคู่ปฏิเสธอุปกรณ์นี้ โปรดลองอีกครั้งหรือติดต่อฝ่ายสนับสนุน',
    nodeNotFound: 'ไม่พบอุปกรณ์ที่มีการจัดการ โปรดลองอีกครั้งเพื่อรีเฟรชข้อมูลประจำตัวเครือข่าย',
    protocolError: 'บริการจับคู่ส่งคืนการตอบกลับที่ไม่คาดคิด โปรดลองอีกครั้งภายหลัง',
    localAPIUnavailable:
      'บริการเครือข่ายระยะไกลในเครื่องไม่พร้อมใช้งาน โปรดเริ่มบริการแล้วลองอีกครั้ง',
    stateFileInvalid: 'สถานะเครือข่ายระยะไกลที่บันทึกไว้ไม่ถูกต้อง โปรดรีเซ็ตแล้วลองอีกครั้ง',
    stateWriteFailed:
      'ไม่สามารถบันทึกสถานะเครือข่ายระยะไกลได้ โปรดตรวจสอบที่จัดเก็บในเครื่องแล้วลองอีกครั้ง',
    installFailed: 'ไม่สามารถติดตั้งไคลเอนต์เครือข่ายระยะไกลได้ โปรดลองอีกครั้งภายหลัง',
    profileReadFailed: 'ไม่สามารถอ่านโปรไฟล์เครือข่ายระยะไกลในเครื่องได้ โปรดลองอีกครั้ง',
    hostnameUnavailable: 'ไม่มีชื่อโฮสต์ของอุปกรณ์ โปรดตั้งชื่อโฮสต์แล้วลองอีกครั้ง',
    profileSetupFailed: 'ไม่สามารถเตรียมโปรไฟล์เครือข่ายระยะไกลที่มีการจัดการได้ โปรดลองอีกครั้ง',
    profileStartFailed: 'ไม่สามารถเริ่มโปรไฟล์เครือข่ายระยะไกลที่มีการจัดการได้ โปรดลองอีกครั้ง',
    connectionFailed: 'ไม่สามารถสร้างการเชื่อมต่อเครือข่ายระยะไกลที่มีการจัดการได้ โปรดลองอีกครั้ง',
    profileVerificationFailed:
      'ไม่สามารถตรวจสอบโปรไฟล์เครือข่ายระยะไกลที่มีการจัดการได้ โปรดลองอีกครั้ง',
    rollbackFailed:
      'ไม่สามารถกู้คืนโปรไฟล์เครือข่ายระยะไกลก่อนหน้าได้ โปรดตรวจสอบ Tailscale แล้วลองอีกครั้ง',
    interrupted:
      'การดำเนินการเครือข่ายระยะไกลก่อนหน้าถูกขัดจังหวะ โปรดลองอีกครั้งเพื่อดำเนินการต่อ',
    identityGenerationFailed:
      'ไม่สามารถสร้างข้อมูลประจำตัวเครือข่ายระยะไกลในเครื่องได้ โปรดลองอีกครั้ง',
    retry: 'ลองอีกครั้ง',
    localStatus: 'การเชื่อมต่อในเครื่อง',
    connected: 'เชื่อมต่อแล้ว',
    copyIP: 'คัดลอก IP อุปกรณ์',
    ipCopied: 'คัดลอก IP อุปกรณ์แล้ว',
    devices: 'อุปกรณ์',
    emptyDevices: 'ยังไม่มีการเพิ่มอุปกรณ์อื่น'
  })
};

const sipeedManagementTranslations = {
  nl: prefixSipeedMessages({
    exit: 'Verlaten',
    exitDesc: 'Verwijder dit apparaat uit het Sipeed-netwerk en ga terug naar de netwerkkeuze.',
    exitConfirm: 'Het Sipeed-netwerk verlaten?',
    exitWarning:
      'Het lokale profiel is verwijderd, maar het cloudapparaat kon niet worden verwijderd. U kunt het later vanaf een ander apparaat verwijderen.',
    exitFailed: 'Het Sipeed-netwerk kon niet worden verlaten.',
    pendingTitle: 'Apparaat wacht op goedkeuring',
    pendingDesc: 'Controleer dit apparaat voordat het tot uw netwerk toetreedt.',
    pendingSas: 'Verificatiecijfers',
    pendingMismatch:
      'Als deze cijfers niet overeenkomen met het apparaat dat verbinding maakt, weiger dan het verzoek.',
    pendingClient: 'Client',
    pendingAllow: 'Toestaan',
    pendingReject: 'Weigeren',
    pendingName: 'Apparaatnaam (optioneel)',
    backupCode: 'Reservecode',
    backupCodeDesc: 'Voer de reservecode van 8 tekens van het apparaat in.',
    authorize: 'Reservecode gebruiken',
    authorizeFailed: 'De reservecode kon niet worden geaccepteerd.',
    managementFailed: 'Apparaatbeheer is mislukt. Probeer het opnieuw.',
    identityExpired: 'De Sipeed-identiteit is verlopen. Kies opnieuw een netwerkmodus.',
    deviceOnline: 'Online',
    deviceOffline: 'Offline',
    lastSeen: 'Laatst gezien {{time}}',
    justNow: 'zojuist',
    rename: 'Naam wijzigen',
    save: 'Opslaan',
    cancel: 'Annuleren',
    remove: 'Verwijderen',
    removeConfirm: 'Dit apparaat uit het netwerk verwijderen?',
    removeSelf: 'Het huidige apparaat kan alleen worden verwijderd door het netwerk te verlaten.',
    refreshFailed: 'De apparatenlijst kon niet worden vernieuwd.'
  }),
  da: prefixSipeedMessages({
    exit: 'Forlad',
    exitDesc: 'Fjern denne enhed fra Sipeed-netværket, og gå tilbage til netværksvalget.',
    exitConfirm: 'Forlad Sipeed-netværket?',
    exitWarning:
      'Den lokale profil blev fjernet, men skyenheden kunne ikke slettes. Du kan fjerne den senere fra en anden enhed.',
    exitFailed: 'Sipeed-netværket kunne ikke forlades.',
    pendingTitle: 'Enhed afventer godkendelse',
    pendingDesc: 'Kontrollér denne enhed, før den tilsluttes dit netværk.',
    pendingSas: 'Bekræftelsescifre',
    pendingMismatch:
      'Hvis cifrene ikke matcher enheden, der tilsluttes, skal du afvise anmodningen.',
    pendingClient: 'Klient',
    pendingAllow: 'Tillad',
    pendingReject: 'Afvis',
    pendingName: 'Enhedsnavn (valgfrit)',
    backupCode: 'Reservekode',
    backupCodeDesc: 'Indtast enhedens reservekode på 8 tegn.',
    authorize: 'Brug reservekode',
    authorizeFailed: 'Reservekoden kunne ikke accepteres.',
    managementFailed: 'Enhedsadministrationen mislykkedes. Prøv igen.',
    identityExpired: 'Sipeed-identiteten er udløbet. Vælg en netværkstilstand igen.',
    deviceOnline: 'Online',
    deviceOffline: 'Offline',
    lastSeen: 'Senest set {{time}}',
    justNow: 'lige nu',
    rename: 'Omdøb',
    save: 'Gem',
    cancel: 'Annuller',
    remove: 'Fjern',
    removeConfirm: 'Fjern denne enhed fra netværket?',
    removeSelf: 'Den aktuelle enhed kan kun fjernes ved at forlade netværket.',
    refreshFailed: 'Enhedslisten kunne ikke opdateres.'
  }),
  de: prefixSipeedMessages({
    exit: 'Verlassen',
    exitDesc:
      'Dieses Gerät aus dem Sipeed-Netzwerk entfernen und zur Netzwerkauswahl zurückkehren.',
    exitConfirm: 'Sipeed-Netzwerk verlassen?',
    exitWarning:
      'Das lokale Profil wurde entfernt, das Cloud-Gerät konnte jedoch nicht gelöscht werden. Sie können es später von einem anderen Gerät entfernen.',
    exitFailed: 'Das Sipeed-Netzwerk konnte nicht verlassen werden.',
    pendingTitle: 'Gerät wartet auf Freigabe',
    pendingDesc: 'Prüfen Sie dieses Gerät, bevor es Ihrem Netzwerk beitritt.',
    pendingSas: 'Bestätigungsziffern',
    pendingMismatch:
      'Wenn diese Ziffern nicht mit dem beitretenden Gerät übereinstimmen, lehnen Sie die Anfrage ab.',
    pendingClient: 'Client',
    pendingAllow: 'Zulassen',
    pendingReject: 'Ablehnen',
    pendingName: 'Gerätename (optional)',
    backupCode: 'Ersatzcode',
    backupCodeDesc: 'Geben Sie den 8-stelligen Ersatzcode des Geräts ein.',
    authorize: 'Ersatzcode verwenden',
    authorizeFailed: 'Der Ersatzcode konnte nicht akzeptiert werden.',
    managementFailed: 'Die Geräteverwaltung ist fehlgeschlagen. Versuchen Sie es erneut.',
    identityExpired: 'Die Sipeed-Identität ist abgelaufen. Wählen Sie erneut einen Netzwerkmodus.',
    deviceOnline: 'Online',
    deviceOffline: 'Offline',
    lastSeen: 'Zuletzt gesehen: {{time}}',
    justNow: 'gerade eben',
    rename: 'Umbenennen',
    save: 'Speichern',
    cancel: 'Abbrechen',
    remove: 'Entfernen',
    removeConfirm: 'Dieses Gerät aus dem Netzwerk entfernen?',
    removeSelf: 'Das aktuelle Gerät kann nur durch Verlassen des Netzwerks entfernt werden.',
    refreshFailed: 'Die Geräteliste konnte nicht aktualisiert werden.'
  }),
  es: prefixSipeedMessages({
    exit: 'Salir',
    exitDesc: 'Quita este dispositivo de la red Sipeed y vuelve a la selección de red.',
    exitConfirm: '¿Salir de la red Sipeed?',
    exitWarning:
      'Se eliminó el perfil local, pero no se pudo borrar el dispositivo de la nube. Puedes eliminarlo más tarde desde otro dispositivo.',
    exitFailed: 'No se pudo salir de la red Sipeed.',
    pendingTitle: 'Dispositivo pendiente',
    pendingDesc: 'Verifica este dispositivo antes de que se una a tu red.',
    pendingSas: 'Dígitos de verificación',
    pendingMismatch:
      'Si estos dígitos no coinciden con el dispositivo que se está uniendo, rechaza la solicitud.',
    pendingClient: 'Cliente',
    pendingAllow: 'Permitir',
    pendingReject: 'Rechazar',
    pendingName: 'Nombre del dispositivo (opcional)',
    backupCode: 'Código de respaldo',
    backupCodeDesc: 'Introduce el código de respaldo de 8 caracteres del dispositivo.',
    authorize: 'Usar código de respaldo',
    authorizeFailed: 'No se pudo aceptar el código de respaldo.',
    managementFailed: 'No se pudo gestionar el dispositivo. Inténtalo de nuevo.',
    identityExpired: 'La identidad de Sipeed caducó. Vuelve a elegir un modo de red.',
    deviceOnline: 'En línea',
    deviceOffline: 'Sin conexión',
    lastSeen: 'Visto por última vez {{time}}',
    justNow: 'hace un momento',
    rename: 'Cambiar nombre',
    save: 'Guardar',
    cancel: 'Cancelar',
    remove: 'Eliminar',
    removeConfirm: '¿Eliminar este dispositivo de la red?',
    removeSelf: 'El dispositivo actual solo se puede eliminar saliendo de la red.',
    refreshFailed: 'No se pudo actualizar la lista de dispositivos.'
  }),
  fr: prefixSipeedMessages({
    exit: 'Quitter',
    exitDesc: 'Retirez cet appareil du réseau Sipeed et revenez au choix du réseau.',
    exitConfirm: 'Quitter le réseau Sipeed ?',
    exitWarning:
      'Le profil local a été supprimé, mais l’appareil cloud n’a pas pu l’être. Vous pourrez le supprimer plus tard depuis un autre appareil.',
    exitFailed: 'Impossible de quitter le réseau Sipeed.',
    pendingTitle: 'Appareil en attente',
    pendingDesc: 'Vérifiez cet appareil avant de l’autoriser à rejoindre votre réseau.',
    pendingSas: 'Chiffres de vérification',
    pendingMismatch:
      'Si ces chiffres ne correspondent pas à ceux de l’appareil en cours d’ajout, refusez la demande.',
    pendingClient: 'Client',
    pendingAllow: 'Autoriser',
    pendingReject: 'Refuser',
    pendingName: 'Nom de l’appareil (facultatif)',
    backupCode: 'Code de secours',
    backupCodeDesc: 'Saisissez le code de secours à 8 caractères affiché sur l’appareil.',
    authorize: 'Utiliser le code de secours',
    authorizeFailed: 'Le code de secours n’a pas pu être accepté.',
    managementFailed: 'La gestion de l’appareil a échoué. Réessayez.',
    identityExpired: 'L’identité Sipeed a expiré. Choisissez de nouveau un mode réseau.',
    deviceOnline: 'En ligne',
    deviceOffline: 'Hors ligne',
    lastSeen: 'Dernière activité : {{time}}',
    justNow: 'à l’instant',
    rename: 'Renommer',
    save: 'Enregistrer',
    cancel: 'Annuler',
    remove: 'Supprimer',
    removeConfirm: 'Supprimer cet appareil du réseau ?',
    removeSelf: 'L’appareil actuel ne peut être supprimé qu’en quittant le réseau.',
    refreshFailed: 'Impossible d’actualiser la liste des appareils.'
  }),
  id: prefixSipeedMessages({
    exit: 'Keluar',
    exitDesc: 'Hapus perangkat ini dari jaringan Sipeed dan kembali ke pilihan jaringan.',
    exitConfirm: 'Keluar dari jaringan Sipeed?',
    exitWarning:
      'Profil lokal telah dihapus, tetapi perangkat cloud tidak dapat dihapus. Anda dapat menghapusnya nanti dari perangkat lain.',
    exitFailed: 'Tidak dapat keluar dari jaringan Sipeed.',
    pendingTitle: 'Perangkat menunggu persetujuan',
    pendingDesc: 'Verifikasi perangkat ini sebelum bergabung ke jaringan Anda.',
    pendingSas: 'Angka verifikasi',
    pendingMismatch:
      'Jika angka ini tidak cocok dengan perangkat yang bergabung, tolak permintaannya.',
    pendingClient: 'Klien',
    pendingAllow: 'Izinkan',
    pendingReject: 'Tolak',
    pendingName: 'Nama perangkat (opsional)',
    backupCode: 'Kode cadangan',
    backupCodeDesc: 'Masukkan kode cadangan 8 karakter dari perangkat.',
    authorize: 'Gunakan kode cadangan',
    authorizeFailed: 'Kode cadangan tidak dapat diterima.',
    managementFailed: 'Pengelolaan perangkat gagal. Coba lagi.',
    identityExpired: 'Identitas Sipeed telah kedaluwarsa. Pilih kembali mode jaringan.',
    deviceOnline: 'Daring',
    deviceOffline: 'Luring',
    lastSeen: 'Terakhir terlihat {{time}}',
    justNow: 'baru saja',
    rename: 'Ubah nama',
    save: 'Simpan',
    cancel: 'Batal',
    remove: 'Hapus',
    removeConfirm: 'Hapus perangkat ini dari jaringan?',
    removeSelf: 'Perangkat saat ini hanya dapat dihapus dengan keluar dari jaringan.',
    refreshFailed: 'Tidak dapat memperbarui daftar perangkat.'
  }),
  it: prefixSipeedMessages({
    exit: 'Esci',
    exitDesc: 'Rimuovi questo dispositivo dalla rete Sipeed e torna alla selezione della rete.',
    exitConfirm: 'Uscire dalla rete Sipeed?',
    exitWarning:
      'Il profilo locale è stato rimosso, ma non è stato possibile eliminare il dispositivo cloud. Potrai rimuoverlo in seguito da un altro dispositivo.',
    exitFailed: 'Impossibile uscire dalla rete Sipeed.',
    pendingTitle: 'Dispositivo in attesa',
    pendingDesc: 'Verifica questo dispositivo prima che entri nella rete.',
    pendingSas: 'Cifre di verifica',
    pendingMismatch:
      'Se queste cifre non corrispondono al dispositivo che sta entrando, rifiuta la richiesta.',
    pendingClient: 'Client',
    pendingAllow: 'Consenti',
    pendingReject: 'Rifiuta',
    pendingName: 'Nome dispositivo (facoltativo)',
    backupCode: 'Codice di riserva',
    backupCodeDesc: 'Inserisci il codice di riserva di 8 caratteri mostrato dal dispositivo.',
    authorize: 'Usa codice di riserva',
    authorizeFailed: 'Il codice di riserva non è stato accettato.',
    managementFailed: 'Gestione del dispositivo non riuscita. Riprova.',
    identityExpired: 'L’identità Sipeed è scaduta. Scegli di nuovo una modalità di rete.',
    deviceOnline: 'Online',
    deviceOffline: 'Offline',
    lastSeen: 'Ultimo accesso {{time}}',
    justNow: 'poco fa',
    rename: 'Rinomina',
    save: 'Salva',
    cancel: 'Annulla',
    remove: 'Rimuovi',
    removeConfirm: 'Rimuovere questo dispositivo dalla rete?',
    removeSelf: 'Il dispositivo corrente può essere rimosso solo uscendo dalla rete.',
    refreshFailed: 'Impossibile aggiornare l’elenco dei dispositivi.'
  }),
  pl: prefixSipeedMessages({
    exit: 'Opuść',
    exitDesc: 'Usuń to urządzenie z sieci Sipeed i wróć do wyboru sieci.',
    exitConfirm: 'Opuścić sieć Sipeed?',
    exitWarning:
      'Profil lokalny został usunięty, ale nie udało się usunąć urządzenia z chmury. Możesz usunąć je później z innego urządzenia.',
    exitFailed: 'Nie udało się opuścić sieci Sipeed.',
    pendingTitle: 'Urządzenie oczekujące',
    pendingDesc: 'Zweryfikuj to urządzenie, zanim dołączy do Twojej sieci.',
    pendingSas: 'Cyfry weryfikacyjne',
    pendingMismatch: 'Jeśli cyfry nie zgadzają się z urządzeniem, które dołącza, odrzuć żądanie.',
    pendingClient: 'Klient',
    pendingAllow: 'Zezwól',
    pendingReject: 'Odrzuć',
    pendingName: 'Nazwa urządzenia (opcjonalnie)',
    backupCode: 'Kod zapasowy',
    backupCodeDesc: 'Wprowadź 8-znakowy kod zapasowy z urządzenia.',
    authorize: 'Użyj kodu zapasowego',
    authorizeFailed: 'Kod zapasowy nie został zaakceptowany.',
    managementFailed: 'Zarządzanie urządzeniem nie powiodło się. Spróbuj ponownie.',
    identityExpired: 'Tożsamość Sipeed wygasła. Ponownie wybierz tryb sieci.',
    deviceOnline: 'Online',
    deviceOffline: 'Offline',
    lastSeen: 'Ostatnio widziano: {{time}}',
    justNow: 'przed chwilą',
    rename: 'Zmień nazwę',
    save: 'Zapisz',
    cancel: 'Anuluj',
    remove: 'Usuń',
    removeConfirm: 'Usunąć to urządzenie z sieci?',
    removeSelf: 'Bieżące urządzenie można usunąć tylko przez opuszczenie sieci.',
    refreshFailed: 'Nie udało się odświeżyć listy urządzeń.'
  }),
  ru: prefixSipeedMessages({
    exit: 'Выйти',
    exitDesc: 'Удалить это устройство из сети Sipeed и вернуться к выбору сети.',
    exitConfirm: 'Выйти из сети Sipeed?',
    exitWarning:
      'Локальный профиль удалён, но удалить облачное устройство не удалось. Его можно удалить позже с другого устройства.',
    exitFailed: 'Не удалось выйти из сети Sipeed.',
    pendingTitle: 'Устройство ожидает подтверждения',
    pendingDesc: 'Проверьте устройство, прежде чем разрешить ему подключиться к сети.',
    pendingSas: 'Проверочные цифры',
    pendingMismatch:
      'Если эти цифры не совпадают с цифрами на подключаемом устройстве, отклоните запрос.',
    pendingClient: 'Клиент',
    pendingAllow: 'Разрешить',
    pendingReject: 'Отклонить',
    pendingName: 'Имя устройства (необязательно)',
    backupCode: 'Резервный код',
    backupCodeDesc: 'Введите 8-значный резервный код с устройства.',
    authorize: 'Использовать резервный код',
    authorizeFailed: 'Не удалось принять резервный код.',
    managementFailed: 'Не удалось управлять устройством. Повторите попытку.',
    identityExpired: 'Срок действия идентификатора Sipeed истёк. Снова выберите режим сети.',
    deviceOnline: 'В сети',
    deviceOffline: 'Не в сети',
    lastSeen: 'Последняя активность: {{time}}',
    justNow: 'только что',
    rename: 'Переименовать',
    save: 'Сохранить',
    cancel: 'Отмена',
    remove: 'Удалить',
    removeConfirm: 'Удалить это устройство из сети?',
    removeSelf: 'Текущее устройство можно удалить только путём выхода из сети.',
    refreshFailed: 'Не удалось обновить список устройств.'
  }),
  ko: prefixSipeedMessages({
    exit: '나가기',
    exitDesc: '이 기기를 Sipeed 네트워크에서 제거하고 네트워크 선택 화면으로 돌아갑니다.',
    exitConfirm: 'Sipeed 네트워크에서 나갈까요?',
    exitWarning:
      '로컬 프로필은 제거했지만 클라우드 기기를 삭제하지 못했습니다. 나중에 다른 기기에서 제거할 수 있습니다.',
    exitFailed: 'Sipeed 네트워크에서 나가지 못했습니다.',
    pendingTitle: '승인 대기 중인 기기',
    pendingDesc: '네트워크에 참여시키기 전에 이 기기를 확인하세요.',
    pendingSas: '확인 숫자',
    pendingMismatch: '참여하려는 기기의 숫자와 일치하지 않으면 요청을 거부하세요.',
    pendingClient: '클라이언트',
    pendingAllow: '허용',
    pendingReject: '거부',
    pendingName: '기기 이름(선택 사항)',
    backupCode: '백업 코드',
    backupCodeDesc: '기기에 표시된 8자리 백업 코드를 입력하세요.',
    authorize: '백업 코드 사용',
    authorizeFailed: '백업 코드를 승인하지 못했습니다.',
    managementFailed: '기기 관리에 실패했습니다. 다시 시도하세요.',
    identityExpired: 'Sipeed ID가 만료되었습니다. 네트워크 모드를 다시 선택하세요.',
    deviceOnline: '온라인',
    deviceOffline: '오프라인',
    lastSeen: '마지막 접속 {{time}}',
    justNow: '방금',
    rename: '이름 변경',
    save: '저장',
    cancel: '취소',
    remove: '제거',
    removeConfirm: '이 기기를 네트워크에서 제거할까요?',
    removeSelf: '현재 기기는 네트워크에서 나가야만 제거할 수 있습니다.',
    refreshFailed: '기기 목록을 새로 고치지 못했습니다.'
  }),
  zh: prefixSipeedMessages({
    exit: '退出',
    exitDesc: '将此设备移出 Sipeed 网络，并返回网络选择页面。',
    exitConfirm: '退出 Sipeed 网络？',
    exitWarning: '本机配置已移除，但云端设备删除失败。你可以稍后在其他设备上将其移除。',
    exitFailed: '退出 Sipeed 网络失败。',
    pendingTitle: '待批准设备',
    pendingDesc: '请先核对此设备，再允许它加入你的网络。',
    pendingSas: '核对数字',
    pendingMismatch: '如果这些数字与待加入设备上显示的不一致，请拒绝此请求。',
    pendingClient: '对方设备',
    pendingAllow: '允许',
    pendingReject: '拒绝',
    pendingName: '设备名称（可选）',
    backupCode: '备用码',
    backupCodeDesc: '请输入对方设备显示的 8 位备用码。',
    authorize: '使用备用码',
    authorizeFailed: '备用码验证失败。',
    managementFailed: '设备管理失败，请重试。',
    identityExpired: 'Sipeed 身份已失效，请重新选择网络模式。',
    deviceOnline: '在线',
    deviceOffline: '离线',
    lastSeen: '最后在线：{{time}}',
    justNow: '刚刚',
    rename: '重命名',
    save: '保存',
    cancel: '取消',
    remove: '移除',
    removeConfirm: '将此设备移出网络？',
    removeSelf: '当前设备只能通过退出网络来移除。',
    refreshFailed: '刷新设备列表失败。'
  }),
  zh_tw: prefixSipeedMessages({
    exit: '退出',
    exitDesc: '將此裝置移出 Sipeed 網路，並返回網路選擇頁面。',
    exitConfirm: '退出 Sipeed 網路？',
    exitWarning: '本機設定已移除，但雲端裝置刪除失敗。你可以稍後在其他裝置上將其移除。',
    exitFailed: '退出 Sipeed 網路失敗。',
    pendingTitle: '待核准裝置',
    pendingDesc: '請先核對此裝置，再允許它加入你的網路。',
    pendingSas: '核對數字',
    pendingMismatch: '如果這些數字與待加入裝置上顯示的不一致，請拒絕此要求。',
    pendingClient: '對方裝置',
    pendingAllow: '允許',
    pendingReject: '拒絕',
    pendingName: '裝置名稱（選填）',
    backupCode: '備用碼',
    backupCodeDesc: '請輸入對方裝置顯示的 8 位備用碼。',
    authorize: '使用備用碼',
    authorizeFailed: '備用碼驗證失敗。',
    managementFailed: '裝置管理失敗，請重試。',
    identityExpired: 'Sipeed 身分已失效，請重新選擇網路模式。',
    deviceOnline: '線上',
    deviceOffline: '離線',
    lastSeen: '最後上線：{{time}}',
    justNow: '剛剛',
    rename: '重新命名',
    save: '儲存',
    cancel: '取消',
    remove: '移除',
    removeConfirm: '將此裝置移出網路？',
    removeSelf: '目前裝置只能透過退出網路來移除。',
    refreshFailed: '重新整理裝置清單失敗。'
  }),
  hu: prefixSipeedMessages({
    exit: 'Kilépés',
    exitDesc: 'Az eszköz eltávolítása a Sipeed-hálózatból és visszatérés a hálózatválasztáshoz.',
    exitConfirm: 'Kilép a Sipeed-hálózatból?',
    exitWarning:
      'A helyi profil törlődött, de a felhőbeli eszköz nem. Később egy másik eszközről eltávolíthatja.',
    exitFailed: 'Nem sikerült kilépni a Sipeed-hálózatból.',
    pendingTitle: 'Jóváhagyásra váró eszköz',
    pendingDesc: 'Ellenőrizze az eszközt, mielőtt csatlakozik a hálózathoz.',
    pendingSas: 'Ellenőrző számjegyek',
    pendingMismatch:
      'Ha a számjegyek nem egyeznek a csatlakozó eszközön láthatókkal, utasítsa el a kérést.',
    pendingClient: 'Kliens',
    pendingAllow: 'Engedélyezés',
    pendingReject: 'Elutasítás',
    pendingName: 'Eszköznév (nem kötelező)',
    backupCode: 'Tartalékkód',
    backupCodeDesc: 'Adja meg az eszköz 8 karakteres tartalékkódját.',
    authorize: 'Tartalékkód használata',
    authorizeFailed: 'A tartalékkódot nem sikerült elfogadni.',
    managementFailed: 'Az eszköz kezelése sikertelen. Próbálja újra.',
    identityExpired: 'A Sipeed-azonosító lejárt. Válasszon újra hálózati módot.',
    deviceOnline: 'Online',
    deviceOffline: 'Offline',
    lastSeen: 'Utoljára elérhető: {{time}}',
    justNow: 'az imént',
    rename: 'Átnevezés',
    save: 'Mentés',
    cancel: 'Mégse',
    remove: 'Eltávolítás',
    removeConfirm: 'Eltávolítja ezt az eszközt a hálózatból?',
    removeSelf: 'A jelenlegi eszköz csak a hálózatból való kilépéssel távolítható el.',
    refreshFailed: 'Nem sikerült frissíteni az eszközlistát.'
  }),
  vi: prefixSipeedMessages({
    exit: 'Thoát',
    exitDesc: 'Xóa thiết bị này khỏi mạng Sipeed và quay lại trang chọn mạng.',
    exitConfirm: 'Rời mạng Sipeed?',
    exitWarning:
      'Hồ sơ cục bộ đã được xóa nhưng không thể xóa thiết bị trên đám mây. Bạn có thể xóa sau từ thiết bị khác.',
    exitFailed: 'Không thể rời mạng Sipeed.',
    pendingTitle: 'Thiết bị đang chờ duyệt',
    pendingDesc: 'Hãy xác minh thiết bị này trước khi cho phép tham gia mạng.',
    pendingSas: 'Các số xác minh',
    pendingMismatch: 'Nếu các số này không khớp với thiết bị đang tham gia, hãy từ chối yêu cầu.',
    pendingClient: 'Máy khách',
    pendingAllow: 'Cho phép',
    pendingReject: 'Từ chối',
    pendingName: 'Tên thiết bị (không bắt buộc)',
    backupCode: 'Mã dự phòng',
    backupCodeDesc: 'Nhập mã dự phòng 8 ký tự từ thiết bị.',
    authorize: 'Dùng mã dự phòng',
    authorizeFailed: 'Không thể chấp nhận mã dự phòng.',
    managementFailed: 'Quản lý thiết bị thất bại. Hãy thử lại.',
    identityExpired: 'Danh tính Sipeed đã hết hạn. Hãy chọn lại chế độ mạng.',
    deviceOnline: 'Trực tuyến',
    deviceOffline: 'Ngoại tuyến',
    lastSeen: 'Hoạt động lần cuối {{time}}',
    justNow: 'vừa xong',
    rename: 'Đổi tên',
    save: 'Lưu',
    cancel: 'Hủy',
    remove: 'Xóa',
    removeConfirm: 'Xóa thiết bị này khỏi mạng?',
    removeSelf: 'Chỉ có thể xóa thiết bị hiện tại bằng cách rời mạng.',
    refreshFailed: 'Không thể làm mới danh sách thiết bị.'
  }),
  ja: prefixSipeedMessages({
    exit: '退出',
    exitDesc: 'このデバイスを Sipeed ネットワークから削除し、ネットワーク選択画面に戻ります。',
    exitConfirm: 'Sipeed ネットワークから退出しますか？',
    exitWarning:
      'ローカルプロファイルは削除されましたが、クラウド上のデバイスを削除できませんでした。後で別のデバイスから削除できます。',
    exitFailed: 'Sipeed ネットワークから退出できませんでした。',
    pendingTitle: '承認待ちのデバイス',
    pendingDesc: 'ネットワークへの参加を許可する前に、このデバイスを確認してください。',
    pendingSas: '確認用数字',
    pendingMismatch: '参加するデバイスの数字と一致しない場合は、要求を拒否してください。',
    pendingClient: 'クライアント',
    pendingAllow: '許可',
    pendingReject: '拒否',
    pendingName: 'デバイス名（任意）',
    backupCode: 'バックアップコード',
    backupCodeDesc: 'デバイスに表示された 8 文字のバックアップコードを入力してください。',
    authorize: 'バックアップコードを使用',
    authorizeFailed: 'バックアップコードを受け付けられませんでした。',
    managementFailed: 'デバイスの管理に失敗しました。もう一度お試しください。',
    identityExpired: 'Sipeed ID の有効期限が切れました。ネットワークモードを選び直してください。',
    deviceOnline: 'オンライン',
    deviceOffline: 'オフライン',
    lastSeen: '最終接続：{{time}}',
    justNow: 'たった今',
    rename: '名前を変更',
    save: '保存',
    cancel: 'キャンセル',
    remove: '削除',
    removeConfirm: 'このデバイスをネットワークから削除しますか？',
    removeSelf: '現在のデバイスは、ネットワークから退出した場合にのみ削除できます。',
    refreshFailed: 'デバイス一覧を更新できませんでした。'
  }),
  cz: prefixSipeedMessages({
    exit: 'Opustit',
    exitDesc: 'Odeberte toto zařízení ze sítě Sipeed a vraťte se k výběru sítě.',
    exitConfirm: 'Opustit síť Sipeed?',
    exitWarning:
      'Místní profil byl odebrán, ale cloudové zařízení se nepodařilo odstranit. Později jej můžete odebrat z jiného zařízení.',
    exitFailed: 'Síť Sipeed se nepodařilo opustit.',
    pendingTitle: 'Zařízení čeká na schválení',
    pendingDesc: 'Před připojením zařízení k síti ověřte jeho totožnost.',
    pendingSas: 'Ověřovací číslice',
    pendingMismatch: 'Pokud se číslice neshodují s připojovaným zařízením, požadavek zamítněte.',
    pendingClient: 'Klient',
    pendingAllow: 'Povolit',
    pendingReject: 'Zamítnout',
    pendingName: 'Název zařízení (volitelný)',
    backupCode: 'Záložní kód',
    backupCodeDesc: 'Zadejte osmimístný záložní kód ze zařízení.',
    authorize: 'Použít záložní kód',
    authorizeFailed: 'Záložní kód se nepodařilo přijmout.',
    managementFailed: 'Správa zařízení se nezdařila. Zkuste to znovu.',
    identityExpired: 'Platnost identity Sipeed vypršela. Znovu vyberte režim sítě.',
    deviceOnline: 'Online',
    deviceOffline: 'Offline',
    lastSeen: 'Naposledy aktivní: {{time}}',
    justNow: 'právě teď',
    rename: 'Přejmenovat',
    save: 'Uložit',
    cancel: 'Zrušit',
    remove: 'Odebrat',
    removeConfirm: 'Odebrat toto zařízení ze sítě?',
    removeSelf: 'Aktuální zařízení lze odebrat pouze opuštěním sítě.',
    refreshFailed: 'Seznam zařízení se nepodařilo obnovit.'
  }),
  uk: prefixSipeedMessages({
    exit: 'Вийти',
    exitDesc: 'Видалити цей пристрій із мережі Sipeed і повернутися до вибору мережі.',
    exitConfirm: 'Вийти з мережі Sipeed?',
    exitWarning:
      'Локальний профіль видалено, але хмарний пристрій видалити не вдалося. Його можна видалити пізніше з іншого пристрою.',
    exitFailed: 'Не вдалося вийти з мережі Sipeed.',
    pendingTitle: 'Пристрій очікує схвалення',
    pendingDesc: 'Перевірте пристрій, перш ніж дозволити йому приєднатися до мережі.',
    pendingSas: 'Цифри перевірки',
    pendingMismatch:
      'Якщо ці цифри не збігаються з цифрами на пристрої, що приєднується, відхиліть запит.',
    pendingClient: 'Клієнт',
    pendingAllow: 'Дозволити',
    pendingReject: 'Відхилити',
    pendingName: 'Назва пристрою (необов’язково)',
    backupCode: 'Резервний код',
    backupCodeDesc: 'Введіть 8-символьний резервний код із пристрою.',
    authorize: 'Використати резервний код',
    authorizeFailed: 'Не вдалося прийняти резервний код.',
    managementFailed: 'Не вдалося керувати пристроєм. Спробуйте ще раз.',
    identityExpired: 'Термін дії ідентифікатора Sipeed минув. Знову виберіть режим мережі.',
    deviceOnline: 'У мережі',
    deviceOffline: 'Поза мережею',
    lastSeen: 'Остання активність: {{time}}',
    justNow: 'щойно',
    rename: 'Перейменувати',
    save: 'Зберегти',
    cancel: 'Скасувати',
    remove: 'Видалити',
    removeConfirm: 'Видалити цей пристрій із мережі?',
    removeSelf: 'Поточний пристрій можна видалити лише шляхом виходу з мережі.',
    refreshFailed: 'Не вдалося оновити список пристроїв.'
  }),
  nb: prefixSipeedMessages({
    exit: 'Forlat',
    exitDesc: 'Fjern denne enheten fra Sipeed-nettverket og gå tilbake til nettverksvalget.',
    exitConfirm: 'Forlate Sipeed-nettverket?',
    exitWarning:
      'Den lokale profilen ble fjernet, men skyenheten kunne ikke slettes. Du kan fjerne den senere fra en annen enhet.',
    exitFailed: 'Kunne ikke forlate Sipeed-nettverket.',
    pendingTitle: 'Enhet venter på godkjenning',
    pendingDesc: 'Kontroller denne enheten før den blir med i nettverket.',
    pendingSas: 'Bekreftelsessifre',
    pendingMismatch:
      'Hvis sifrene ikke samsvarer med enheten som kobles til, må du avvise forespørselen.',
    pendingClient: 'Klient',
    pendingAllow: 'Tillat',
    pendingReject: 'Avvis',
    pendingName: 'Enhetsnavn (valgfritt)',
    backupCode: 'Reservekode',
    backupCodeDesc: 'Skriv inn reservekoden på 8 tegn fra enheten.',
    authorize: 'Bruk reservekode',
    authorizeFailed: 'Reservekoden kunne ikke godtas.',
    managementFailed: 'Enhetsadministrasjonen mislyktes. Prøv igjen.',
    identityExpired: 'Sipeed-identiteten har utløpt. Velg nettverksmodus på nytt.',
    deviceOnline: 'Tilkoblet',
    deviceOffline: 'Frakoblet',
    lastSeen: 'Sist sett {{time}}',
    justNow: 'nettopp',
    rename: 'Gi nytt navn',
    save: 'Lagre',
    cancel: 'Avbryt',
    remove: 'Fjern',
    removeConfirm: 'Fjerne denne enheten fra nettverket?',
    removeSelf: 'Den gjeldende enheten kan bare fjernes ved å forlate nettverket.',
    refreshFailed: 'Kunne ikke oppdatere enhetslisten.'
  }),
  th: prefixSipeedMessages({
    exit: 'ออก',
    exitDesc: 'นำอุปกรณ์นี้ออกจากเครือข่าย Sipeed และกลับไปยังหน้าเลือกเครือข่าย',
    exitConfirm: 'ออกจากเครือข่าย Sipeed หรือไม่',
    exitWarning:
      'ลบโปรไฟล์ในเครื่องแล้ว แต่ไม่สามารถลบอุปกรณ์บนคลาวด์ได้ คุณสามารถลบภายหลังจากอุปกรณ์เครื่องอื่น',
    exitFailed: 'ไม่สามารถออกจากเครือข่าย Sipeed ได้',
    pendingTitle: 'อุปกรณ์รอการอนุมัติ',
    pendingDesc: 'ตรวจสอบอุปกรณ์นี้ก่อนอนุญาตให้เข้าร่วมเครือข่าย',
    pendingSas: 'ตัวเลขยืนยัน',
    pendingMismatch: 'หากตัวเลขไม่ตรงกับอุปกรณ์ที่กำลังเข้าร่วม ให้ปฏิเสธคำขอ',
    pendingClient: 'ไคลเอนต์',
    pendingAllow: 'อนุญาต',
    pendingReject: 'ปฏิเสธ',
    pendingName: 'ชื่ออุปกรณ์ (ไม่บังคับ)',
    backupCode: 'รหัสสำรอง',
    backupCodeDesc: 'ป้อนรหัสสำรอง 8 ตัวอักษรจากอุปกรณ์',
    authorize: 'ใช้รหัสสำรอง',
    authorizeFailed: 'ไม่สามารถยอมรับรหัสสำรองได้',
    managementFailed: 'จัดการอุปกรณ์ไม่สำเร็จ โปรดลองอีกครั้ง',
    identityExpired: 'ข้อมูลประจำตัว Sipeed หมดอายุ โปรดเลือกโหมดเครือข่ายอีกครั้ง',
    deviceOnline: 'ออนไลน์',
    deviceOffline: 'ออฟไลน์',
    lastSeen: 'ออนไลน์ล่าสุด {{time}}',
    justNow: 'เมื่อสักครู่',
    rename: 'เปลี่ยนชื่อ',
    save: 'บันทึก',
    cancel: 'ยกเลิก',
    remove: 'นำออก',
    removeConfirm: 'นำอุปกรณ์นี้ออกจากเครือข่ายหรือไม่',
    removeSelf: 'อุปกรณ์ปัจจุบันจะนำออกได้ด้วยการออกจากเครือข่ายเท่านั้น',
    refreshFailed: 'ไม่สามารถรีเฟรชรายการอุปกรณ์ได้'
  })
};

/**
 * Tailscale strings, shipped next to the component that renders them.
 *
 * Keys keep the `settings.tailscale.*` shape so the shared fallback in
 * `useRemoteNetworkMessages` can resolve generic keys such as `settings.back`
 * from the global i18n bundle.
 */
export const tailscaleMessages: RemoteNetworkComponentMessages = {
  nl: {
    'tailscale.continue': 'Doorgaan',
    'tailscale.start': 'Inschakelen',
    'tailscale.stop': 'Uitschakelen',
    'tailscale.restart': 'Herstarten',
    'tailscale.disconnect':
      'Tailscale wordt losgekoppeld en automatisch starten bij het opstarten wordt uitgeschakeld.',
    'tailscale.confirm': 'Bevestigen',
    'tailscale.cancel': 'Annuleren',
    'tailscale.stopped': 'Uitgeschakeld',
    'tailscale.connecting': 'Verbinden',
    'tailscale.error': 'Verbindingsfout',
    'tailscale.connected': 'Verbonden',
    'tailscale.offline': 'Offline',
    'tailscale.notLogin': 'Niet aangemeld',
    ...sipeedManagedFallbackMessages,
    ...sipeedRuntimeTranslations.nl,
    ...sipeedManagementTranslations.nl,
    'settings.tailscale.title': 'Tailscale',
    'settings.tailscale.restart': 'Weet u zeker dat u Tailscale opnieuw wilt opstarten?',
    'settings.tailscale.stop': 'Weet u zeker dat u Tailscale wilt stoppen?',
    'settings.tailscale.stopDesc':
      'Meld Tailscale af en schakel het automatisch opstarten bij het opstarten uit.',
    'settings.tailscale.loading': 'Laden...',
    'settings.tailscale.notInstall': 'Tailscale niet gevonden! Installeer a.u.b.',
    'settings.tailscale.install': 'Installeren',
    'settings.tailscale.installing': 'Installeren bezig',
    'settings.tailscale.failed': 'Installatie mislukt',
    'settings.tailscale.retry': 'Vernieuw en probeer opnieuw. Of probeer handmatig te installeren',
    'settings.tailscale.download': 'Download het',
    'settings.tailscale.package': 'installatiepakket',
    'settings.tailscale.unzip': 'en pak het uit',
    'settings.tailscale.upTailscale': 'Upload tailscale naar NanoKVM directory /usr/bin/',
    'settings.tailscale.upTailscaled': 'Upload tailscaled naar NanoKVM directory /usr/sbin/',
    'settings.tailscale.refresh': 'Vernieuw huidige pagina',
    'settings.tailscale.notRunning':
      'Tailscale is niet actief. Start het programma om door te gaan.',
    'settings.tailscale.run': 'Starten',
    'settings.tailscale.notLogin':
      'Het apparaat is nog niet gekoppeld. Log in en koppel dit apparaat aan uw account.',
    'settings.tailscale.urlPeriod': 'Deze url is 10 minuten geldig',
    'settings.tailscale.login': 'Inloggen',
    'settings.tailscale.loginSuccess': 'Inloggen gelukt',
    'settings.tailscale.enable': 'Tailscale inschakelen',
    'settings.tailscale.exitNode': 'Uitgangsknooppunt',
    'settings.tailscale.exitNodeDesc':
      'Kondig dit apparaat aan als internetuitgang voor uw tailnet.',
    'settings.tailscale.subnetRoute': 'Subnetroute',
    'settings.tailscale.subnetRouteDesc':
      'Kondig het direct verbonden LAN-subnet {{subnet}} aan in uw tailnet.',
    'settings.tailscale.subnetUnavailable': 'Er is geen direct verbonden LAN-subnet beschikbaar.',
    'settings.tailscale.routingUnavailable':
      'Tailscale-routeringsinstellingen zijn niet beschikbaar op dit apparaat.',
    'settings.tailscale.pendingApproval':
      'Keur na inschakeling de aangekondigde route goed in de Tailscale-beheerconsole.',
    'settings.tailscale.routingUpdateFailed':
      'De Tailscale-routeringsinstellingen konden niet worden bijgewerkt.',
    'settings.tailscale.deviceName': 'Apparaatnaam',
    'settings.tailscale.deviceIP': 'Apparaat IP',
    'settings.tailscale.account': 'Account',
    'settings.tailscale.logout': 'Uitloggen',
    'settings.tailscale.logoutDesc': 'Weet u zeker dat u wilt uitloggen?',
    'settings.tailscale.logoutFailed': 'Afmelden is mislukt.',
    'settings.tailscale.uninstall': 'Verwijderen Tailscale',
    'settings.tailscale.uninstallDesc': 'Weet u zeker dat u Tailscale wilt verwijderen?',
    'settings.tailscale.okBtn': 'Ja',
    'settings.tailscale.cancelBtn': 'Nee',
    'settings.tailscale.mode.official': 'Officiële Tailscale',
    'settings.tailscale.mode.officialDesc': 'Gebruik de officiële Tailscale-service.',
    'settings.tailscale.mode.sipeed': 'Sipeed-netwerk op afstand',
    'settings.tailscale.mode.sipeedDesc': 'Directe verbindingen tussen apparaten; geen relay.',
    'settings.tailscale.sipeed.disclaimerTitle': 'Vrijwaring',
    'settings.tailscale.sipeed.enableDesc':
      'Als je Sipeed Remote Network inschakelt, worden verzoeken naar Sipeed-servers verstuurd met verbindingsgegevens, zoals de apparaat-ID, apparaatnaam en het virtuele IP-adres. Deze gegevens worden gebruikt voor apparaatregistratie, authenticatie, verbindingscoördinatie en netwerkbeheer.\n\nToegang op afstand is afhankelijk van netwerkomstandigheden, apparaatinstellingen en de beschikbaarheid van diensten. Verbindingen kunnen vertraagd, onderbroken of tijdelijk niet beschikbaar zijn. Beoordeel de gebruiksrisico’s en zorg waar nodig voor een andere toegangsmogelijkheid.\n\nGebruik deze dienst alleen voor apparaten en netwerken die van jou zijn of waartoe je toegang mag hebben. Bewaar inloggegevens en koppelcodes veilig en wees voorzichtig met het verlenen van toegang aan anderen.\n\nDoor op “Akkoord en starten” te klikken, bevestig je dat je deze verklaring hebt gelezen en begrepen en instemt met de beschreven communicatie en gegevensverwerking voor externe verbindingen. Je toestemming wordt op dit apparaat opgeslagen, zodat je deze niet bij elke inschakeling hoeft te bevestigen.\n\nJe kunt de toestemming intrekken via het menu rechtsboven op de pagina met de actieve verbinding. Daarmee verlaat je Sipeed Remote Network en wordt verwijdering van het cloudnetwerk en de bijbehorende gebruiker aangevraagd. Na verwijdering kunnen ook andere apparaten dat netwerk niet meer gebruiken voor toegang op afstand. Opnieuw inschakelen vereist opnieuw toestemming.',
    'settings.tailscale.sipeed.acceptAndEnable': 'Akkoord en starten',
    'settings.tailscale.sipeed.revokeDisclaimer': 'Toestemming intrekken',
    'settings.tailscale.sipeed.revokeDisclaimerDesc':
      'Hiermee verlaat je Sipeed Remote Network, verwijder je het cloudnetwerk en de bijbehorende gebruiker en wis je de opgeslagen toestemming. Alle apparaten in dit netwerk worden losgekoppeld. Doorgaan?',
    'settings.tailscale.sipeed.revokeDisclaimerFailed':
      'Intrekken is niet voltooid. Probeer het opnieuw.',
    'settings.tailscale.sipeed.addDevice': 'Apparaat toevoegen',
    'settings.tailscale.sipeed.addDeviceDesc':
      'Er is nog geen apparaat toegevoegd. Kies ‘Apparaat toevoegen’ en rond het toevoegen af op de computer of telefoon die je wilt verbinden.',
    'settings.tailscale.sipeed.claimExpiry': 'Nog {{time}} · eenmalig gebruik',
    'settings.tailscale.sipeed.claimExpired': 'De code is verlopen',
    'settings.tailscale.sipeed.claimCopy': 'Code kopiëren',
    'settings.tailscale.sipeed.claimCopied': 'Gekopieerd',
    'settings.tailscale.sipeed.claimRenew': 'Nieuwe code aanvragen',
    'settings.tailscale.sipeed.claimHint':
      'Installeer en open de Tailscale-client op het apparaat dat je wilt toevoegen, kies de aangepaste server en vul {{server}} in. De browser opent de koppelingspagina; voer daar deze cijfers in.'
  },
  da: {
    'tailscale.continue': 'Fortsæt',
    'tailscale.start': 'Aktiver',
    'tailscale.stop': 'Deaktiver',
    'tailscale.restart': 'Genstart',
    'tailscale.disconnect': 'Dette afbryder Tailscale og deaktiverer automatisk start ved opstart.',
    'tailscale.confirm': 'Bekræft',
    'tailscale.cancel': 'Annuller',
    'tailscale.stopped': 'Deaktiveret',
    'tailscale.connecting': 'Opretter forbindelse',
    'tailscale.error': 'Forbindelsesfejl',
    'tailscale.connected': 'Forbundet',
    'tailscale.offline': 'Offline',
    'tailscale.notLogin': 'Ikke logget ind',
    ...sipeedManagedFallbackMessages,
    ...sipeedRuntimeTranslations.da,
    ...sipeedManagementTranslations.da,
    'settings.tailscale.title': 'Tailscale',
    'settings.tailscale.restart': 'Genstart Tailscale?',
    'settings.tailscale.stop': 'Stoppe Tailscale?',
    'settings.tailscale.stopDesc': 'Log Tailscale ud, og deaktiver automatisk start ved opstart.',
    'settings.tailscale.loading': 'Indlæser...',
    'settings.tailscale.notInstall':
      'Tailscale ikke fundet! Installer det for at fuldføre opsætningen.',
    'settings.tailscale.install': 'Installer',
    'settings.tailscale.installing': 'Installerer',
    'settings.tailscale.failed': 'Installation mislykkedes',
    'settings.tailscale.retry': 'Opdater siden og prøv igen. Ellers prøv at installere manuelt.',
    'settings.tailscale.download': 'Download',
    'settings.tailscale.package': 'installationspakken',
    'settings.tailscale.unzip': 'og udpak den',
    'settings.tailscale.upTailscale': 'Upload tailscale til følgende mappe på enheden: /usr/bin/',
    'settings.tailscale.upTailscaled':
      'Upload tailscaled til følgende mappe på enheden: /usr/sbin/',
    'settings.tailscale.refresh': 'Opdater sides',
    'settings.tailscale.notRunning': 'Tailscale kører ikke. Start det for at fortsætte.',
    'settings.tailscale.run': 'Start',
    'settings.tailscale.notLogin':
      'Enheden er ikke tilknyttet en Tailscale-konto endnu. Log ind for at fuldføre tilknytningen til din konto.',
    'settings.tailscale.urlPeriod': 'Denne URL er gyldig i 10 minutter',
    'settings.tailscale.login': 'Log ind',
    'settings.tailscale.loginSuccess': 'Log ind lykkedes',
    'settings.tailscale.enable': 'Aktiver Tailscale',
    'settings.tailscale.exitNode': 'Udgangsnode',
    'settings.tailscale.exitNodeDesc': 'Annoncér denne enhed som internetudgang for dit tailnet.',
    'settings.tailscale.subnetRoute': 'Undernet-rute',
    'settings.tailscale.subnetRouteDesc':
      'Annoncér det direkte tilsluttede LAN-undernet {{subnet}} til dit tailnet.',
    'settings.tailscale.subnetUnavailable':
      'Der er intet direkte tilsluttet LAN-undernet tilgængeligt.',
    'settings.tailscale.routingUnavailable':
      'Tailscale-routingindstillinger er ikke tilgængelige på denne enhed.',
    'settings.tailscale.pendingApproval':
      'Efter aktivering skal den annoncerede rute godkendes i Tailscale-administrationskonsollen.',
    'settings.tailscale.routingUpdateFailed':
      'Tailscale-routingindstillingerne kunne ikke opdateres.',
    'settings.tailscale.deviceName': 'Enhedens navn',
    'settings.tailscale.deviceIP': 'Enhedens IP',
    'settings.tailscale.account': 'Konto',
    'settings.tailscale.logout': 'Log ud',
    'settings.tailscale.logoutDesc': 'Er du sikker på, at du vil logge ud?',
    'settings.tailscale.logoutFailed': 'Log ud mislykkedes.',
    'settings.tailscale.uninstall': 'Afinstaller Tailscale',
    'settings.tailscale.uninstallDesc': 'Er du sikker på, at du vil afinstallere Tailscale?',
    'settings.tailscale.okBtn': 'Ja',
    'settings.tailscale.cancelBtn': 'Nej',
    'settings.tailscale.mode.official': 'Officiel Tailscale',
    'settings.tailscale.mode.officialDesc': 'Brug Tailscales officielle tjeneste.',
    'settings.tailscale.mode.sipeed': 'Sipeed-fjernnetværk',
    'settings.tailscale.mode.sipeedDesc': 'Direkte forbindelser mellem enheder; ingen relaying.',
    'settings.tailscale.sipeed.disclaimerTitle': 'Ansvarsfraskrivelse',
    'settings.tailscale.sipeed.enableDesc':
      'Når du aktiverer Sipeed Remote Network, sendes der anmodninger til Sipeeds servere med forbindelsesoplysninger såsom enheds-id, enhedsnavn og virtuel IP-adresse. Oplysningerne bruges til enhedsregistrering, godkendelse, koordinering af forbindelser og netværksadministration.\n\nFjernadgang afhænger af netværksforhold, enhedsindstillinger og tjenesternes tilgængelighed. Forbindelser kan blive forsinkede, afbrudt eller midlertidigt utilgængelige. Vurder risikoen ved brugen, og sørg om nødvendigt for en anden adgangsmulighed.\n\nBrug kun tjenesten med enheder og netværk, du ejer eller har tilladelse til at tilgå. Opbevar loginoplysninger og parringskoder sikkert, og vær omhyggelig med at give andre adgang.\n\nVed at klikke på “Acceptér og start” bekræfter du, at du har læst og forstået denne erklæring og accepterer den beskrevne kommunikation og behandling af oplysninger for at levere fjernforbindelser. Dit samtykke gemmes på enheden, så du ikke skal bekræfte det ved hver aktivering.\n\nDu kan trække dit samtykke tilbage i menuen øverst til højre på siden for den aktive forbindelse. Det forlader Sipeed Remote Network og anmoder om sletning af cloudnetværket og dets bruger. Efter sletningen kan andre enheder heller ikke bruge dette netværk til fjernadgang. Ny aktivering kræver nyt samtykke.',
    'settings.tailscale.sipeed.acceptAndEnable': 'Acceptér og start',
    'settings.tailscale.sipeed.revokeDisclaimer': 'Træk samtykke tilbage',
    'settings.tailscale.sipeed.revokeDisclaimerDesc':
      'Dette forlader Sipeed Remote Network, sletter cloudnetværket og dets bruger og fjerner dit gemte samtykke. Alle enheder i netværket afbrydes. Vil du fortsætte?',
    'settings.tailscale.sipeed.revokeDisclaimerFailed':
      'Tilbagetrækningen kunne ikke fuldføres. Prøv igen.',
    'settings.tailscale.sipeed.addDevice': 'Tilføj enhed',
    'settings.tailscale.sipeed.addDeviceDesc':
      'Der er endnu ikke tilføjet nogen enhed. Vælg »Tilføj enhed«, og fuldfør tilslutningen på den computer eller telefon, du vil forbinde.',
    'settings.tailscale.sipeed.claimExpiry': '{{time}} tilbage · kun én gang',
    'settings.tailscale.sipeed.claimExpired': 'Koden er udløbet',
    'settings.tailscale.sipeed.claimCopy': 'Kopiér kode',
    'settings.tailscale.sipeed.claimCopied': 'Kopieret',
    'settings.tailscale.sipeed.claimRenew': 'Hent en ny kode',
    'settings.tailscale.sipeed.claimHint':
      'Installer og åbn Tailscale-klienten på den enhed, du vil tilføje, vælg den brugerdefinerede server, og indtast {{server}}. Browseren åbner tilslutningssiden, hvor du indtaster disse cifre.'
  },
  de: {
    'tailscale.continue': 'Weiter',
    'tailscale.start': 'Aktivieren',
    'tailscale.stop': 'Deaktivieren',
    'tailscale.restart': 'Neu starten',
    'tailscale.disconnect':
      'Dadurch wird Tailscale getrennt und der automatische Start beim Systemstart deaktiviert.',
    'tailscale.confirm': 'Bestätigen',
    'tailscale.cancel': 'Abbrechen',
    'tailscale.stopped': 'Deaktiviert',
    'tailscale.connecting': 'Verbindung wird hergestellt',
    'tailscale.error': 'Verbindungsfehler',
    'tailscale.connected': 'Verbunden',
    'tailscale.offline': 'Offline',
    'tailscale.notLogin': 'Nicht angemeldet',
    ...sipeedManagedFallbackMessages,
    ...sipeedRuntimeTranslations.de,
    ...sipeedManagementTranslations.de,
    'settings.tailscale.title': 'Tailscale',
    'settings.tailscale.restart': 'Tailscale neu starten?',
    'settings.tailscale.stop': 'Tailscale stoppen?',
    'settings.tailscale.stopDesc':
      'Von Tailscale abmelden und automatischen Start beim Booten deaktivieren.',
    'settings.tailscale.loading': 'Lädt...',
    'settings.tailscale.notInstall': 'Tailscale nicht gefunden! Bitte installieren.',
    'settings.tailscale.install': 'Installieren',
    'settings.tailscale.installing': 'Installiere',
    'settings.tailscale.failed': 'Installation fehlgeschlagen',
    'settings.tailscale.retry':
      'Bitte Seite neu laden und erneut versuchen oder manuelle Installation versuchen.',
    'settings.tailscale.download': 'Laden Sie das',
    'settings.tailscale.package': 'Installations-Paket herunter',
    'settings.tailscale.unzip': 'und entpacken Sie es',
    'settings.tailscale.upTailscale': 'Tailscale nach /usr/bin/ auf NanoKVM hochladen',
    'settings.tailscale.upTailscaled': 'Tailscaled nach /usr/sbin/ auf NanoKVM hochladen',
    'settings.tailscale.refresh': 'Aktuelle Seite neu laden',
    'settings.tailscale.notRunning':
      'Tailscale läuft nicht. Bitte starten Sie es, um fortzufahren.',
    'settings.tailscale.run': 'Starten',
    'settings.tailscale.notLogin':
      'Das Gerät konnte noch nicht gefunden werden. Bitte melden Sie sich an und verknüpfen Sie dieses Gerät mit Ihrem Konto.',
    'settings.tailscale.urlPeriod': 'Diese URL ist für 10 Minuten gültig',
    'settings.tailscale.login': 'Anmelden',
    'settings.tailscale.loginSuccess': 'Anmeldung erfolgreich',
    'settings.tailscale.enable': 'Tailscale einschalten',
    'settings.tailscale.exitNode': 'Ausgangsknoten',
    'settings.tailscale.exitNodeDesc':
      'Dieses Gerät als Internetausgang für Ihr Tailnet ankündigen.',
    'settings.tailscale.subnetRoute': 'Subnetzroute',
    'settings.tailscale.subnetRouteDesc':
      'Das direkt verbundene LAN-Subnetz {{subnet}} in Ihrem Tailnet ankündigen.',
    'settings.tailscale.subnetUnavailable': 'Es ist kein direkt verbundenes LAN-Subnetz verfügbar.',
    'settings.tailscale.routingUnavailable':
      'Tailscale-Routing ist auf diesem Gerät nicht verfügbar.',
    'settings.tailscale.pendingApproval':
      'Genehmigen Sie die angekündigte Route nach dem Aktivieren in der Tailscale-Verwaltungskonsole.',
    'settings.tailscale.routingUpdateFailed':
      'Die Tailscale-Routing-Einstellungen konnten nicht aktualisiert werden.',
    'settings.tailscale.deviceName': 'Geräte Name',
    'settings.tailscale.deviceIP': 'Geräte IP',
    'settings.tailscale.account': 'Konto',
    'settings.tailscale.logout': 'Abmelden',
    'settings.tailscale.logoutDesc': 'Möchten Sie sich wirklich abmelden?',
    'settings.tailscale.logoutFailed': 'Abmeldung fehlgeschlagen.',
    'settings.tailscale.uninstall': 'Tailscale deinstallieren',
    'settings.tailscale.uninstallDesc':
      'Sind Sie sicher, dass Sie Tailscale deinstallieren möchten?',
    'settings.tailscale.okBtn': 'Ja',
    'settings.tailscale.cancelBtn': 'Nein',
    'settings.tailscale.mode.official': 'Offizielles Tailscale',
    'settings.tailscale.mode.officialDesc': 'Verwendet den offiziellen Tailscale-Dienst.',
    'settings.tailscale.mode.sipeed': 'Sipeed Remote-Netzwerk',
    'settings.tailscale.mode.sipeedDesc':
      'Direkte Verbindungen zwischen Geräten; keine Weiterleitung.',
    'settings.tailscale.sipeed.disclaimerTitle': 'Haftungsausschluss',
    'settings.tailscale.sipeed.enableDesc':
      'Beim Aktivieren von Sipeed Remote Network werden Anfragen mit Verbindungsinformationen wie Gerätekennung, Gerätename und virtueller IP-Adresse an Sipeed-Server gesendet. Diese Informationen dienen der Geräteregistrierung, Authentifizierung, Verbindungskoordination und Netzwerkverwaltung.\n\nDer Fernzugriff hängt von den Netzwerkbedingungen, der Gerätekonfiguration und der Verfügbarkeit der Dienste ab. Verbindungen können verzögert, unterbrochen oder vorübergehend nicht verfügbar sein. Bewerten Sie die Nutzungsrisiken und halten Sie bei Bedarf einen alternativen Zugangsweg bereit.\n\nNutzen Sie den Dienst nur mit eigenen Geräten und Netzwerken oder mit entsprechender Zugriffsberechtigung. Bewahren Sie Anmeldedaten und Kopplungscodes sicher auf und vergeben Sie Zugriffsrechte an andere mit Bedacht.\n\nMit „Zustimmen und starten“ bestätigen Sie, dass Sie diesen Hinweis gelesen und verstanden haben und der beschriebenen Kommunikation und Informationsverarbeitung für Fernverbindungen zustimmen. Ihre Zustimmung wird auf diesem Gerät gespeichert und muss bei späteren Aktivierungen nicht erneut bestätigt werden.\n\nSie können Ihre Zustimmung im Menü oben rechts auf der Seite der aktiven Verbindung widerrufen. Dadurch wird Sipeed Remote Network verlassen und die Löschung des zugehörigen Cloud-Netzwerks samt Benutzer angefordert. Nach der Löschung können auch andere Geräte dieses Netzwerk nicht mehr für den Fernzugriff nutzen. Eine erneute Aktivierung erfordert eine neue Zustimmung.',
    'settings.tailscale.sipeed.acceptAndEnable': 'Zustimmen und starten',
    'settings.tailscale.sipeed.revokeDisclaimer': 'Zustimmung widerrufen',
    'settings.tailscale.sipeed.revokeDisclaimerDesc':
      'Dadurch wird Sipeed Remote Network verlassen, das Cloud-Netzwerk samt Benutzer gelöscht und die gespeicherte Zustimmung entfernt. Alle Geräte dieses Netzwerks werden getrennt. Fortfahren?',
    'settings.tailscale.sipeed.revokeDisclaimerFailed':
      'Der Widerruf konnte nicht abgeschlossen werden. Bitte erneut versuchen.',
    'settings.tailscale.sipeed.addDevice': 'Gerät hinzufügen',
    'settings.tailscale.sipeed.addDeviceDesc':
      'Es wurde noch kein Gerät hinzugefügt. Wählen Sie „Gerät hinzufügen“ und schließen Sie den Beitritt auf dem Computer oder Smartphone ab.',
    'settings.tailscale.sipeed.claimExpiry': 'Noch {{time}} · nur einmal gültig',
    'settings.tailscale.sipeed.claimExpired': 'Der Anmeldecode ist abgelaufen',
    'settings.tailscale.sipeed.claimCopy': 'Anmeldecode kopieren',
    'settings.tailscale.sipeed.claimCopied': 'Kopiert',
    'settings.tailscale.sipeed.claimRenew': 'Neuen Code anfordern',
    'settings.tailscale.sipeed.claimHint':
      'Installiere und öffne den Tailscale-Client auf dem Gerät, das du hinzufügen möchtest, wähle den benutzerdefinierten Server und gib {{server}} ein. Der Browser öffnet die Beitrittsseite, wo du diese Ziffern eingibst.'
  },
  en: {
    'tailscale.continue': 'Continue',
    'tailscale.start': 'Enable',
    'tailscale.stop': 'Disable',
    'tailscale.restart': 'Restart',
    'tailscale.disconnect': 'This will disconnect Tailscale and disable automatic startup on boot.',
    'tailscale.confirm': 'Confirm',
    'tailscale.cancel': 'Cancel',
    'tailscale.stopped': 'Disabled',
    'tailscale.connecting': 'Connecting',
    'tailscale.error': 'Connection error',
    'tailscale.connected': 'Connected',
    'tailscale.offline': 'Offline',
    'tailscale.notLogin': 'Not logged in',
    ...sipeedManagedFallbackMessages,
    'settings.tailscale.title': 'Tailscale',
    'settings.tailscale.restart': 'Restart Tailscale?',
    'settings.tailscale.stop': 'Stop Tailscale?',
    'settings.tailscale.stopDesc': 'Log out Tailscale and disable automatic startup on boot.',
    'settings.tailscale.loading': 'Loading...',
    'settings.tailscale.notInstall': 'Tailscale not found! Please install.',
    'settings.tailscale.install': 'Install',
    'settings.tailscale.installing': 'Installing',
    'settings.tailscale.failed': 'Install failed',
    'settings.tailscale.retry': 'Please refresh and try again. Or try to install manually',
    'settings.tailscale.download': 'Download the',
    'settings.tailscale.package': 'installation package',
    'settings.tailscale.unzip': 'and unzip it',
    'settings.tailscale.upTailscale': 'Upload tailscale to NanoKVM directory /usr/bin/',
    'settings.tailscale.upTailscaled': 'Upload tailscaled to NanoKVM directory /usr/sbin/',
    'settings.tailscale.refresh': 'Refresh current page',
    'settings.tailscale.notRunning': 'Tailscale is not running. Please start it to continue.',
    'settings.tailscale.run': 'Start',
    'settings.tailscale.notLogin':
      'The device has not been bound yet. Please login and bind this device to your account.',
    'settings.tailscale.urlPeriod': 'This url is valid for 10 minutes',
    'settings.tailscale.login': 'Login',
    'settings.tailscale.loginSuccess': 'Login Success',
    'settings.tailscale.enable': 'Enable Tailscale',
    'settings.tailscale.exitNode': 'Exit node',
    'settings.tailscale.exitNodeDesc':
      'Advertise this device as an internet exit for your tailnet.',
    'settings.tailscale.subnetRoute': 'Subnet route',
    'settings.tailscale.subnetRouteDesc':
      'Advertise the directly connected LAN subnet {{subnet}} to your tailnet.',
    'settings.tailscale.subnetUnavailable': 'No directly connected LAN subnet is available.',
    'settings.tailscale.routingUnavailable':
      'Tailscale routing settings are unavailable on this device.',
    'settings.tailscale.pendingApproval':
      'After enabling, approve the advertised route in the Tailscale admin console.',
    'settings.tailscale.routingUpdateFailed': 'Failed to update Tailscale routing settings.',
    'settings.tailscale.deviceName': 'Device Name',
    'settings.tailscale.deviceIP': 'Device IP',
    'settings.tailscale.account': 'Account',
    'settings.tailscale.logout': 'Logout',
    'settings.tailscale.logoutDesc': 'Are you sure you want to logout?',
    'settings.tailscale.logoutFailed': 'Logout failed.',
    'settings.tailscale.uninstall': 'Uninstall Tailscale',
    'settings.tailscale.uninstallDesc': 'Are you sure you want to uninstall Tailscale?',
    'settings.tailscale.okBtn': 'Yes',
    'settings.tailscale.cancelBtn': 'No',
    'settings.tailscale.mode.official': 'Official Tailscale',
    'settings.tailscale.mode.officialDesc': 'Use the official Tailscale service.',
    'settings.tailscale.mode.sipeed': 'Sipeed Remote Network',
    'settings.tailscale.mode.sipeedDesc': 'Direct device-to-device connections; no relay.',
    'settings.tailscale.sipeed.disclaimerTitle': 'Disclaimer',
    'settings.tailscale.sipeed.enableDesc':
      'Enabling Sipeed Remote Network sends requests to Sipeed servers and provides connection information such as the device identifier, device name, and virtual network IP address. This information is used for device registration, authentication, connection coordination, and network management.\n\nRemote access depends on network conditions, device configuration, and service availability. Connections may be delayed, interrupted, or temporarily unavailable. Assess the risks of using this service and keep an alternative way to access your device when needed.\n\nUse this service only with devices and networks you own or are authorized to access. Keep login credentials and pairing codes secure, and grant access to others with care.\n\nBy clicking “Agree and enable”, you confirm that you have read and understood this notice and consent to the communication and information processing described above for providing remote connections. Your consent is saved on this device, so you do not need to confirm it each time you enable the service.\n\nYou can withdraw consent from the top-right menu on the connected page. Withdrawal exits Sipeed Remote Network and requests deletion of the associated cloud network and its user. Once deleted, other devices in that network can no longer use it for remote access. Enabling the service again requires renewed consent.',
    'settings.tailscale.sipeed.acceptAndEnable': 'Agree and enable',
    'settings.tailscale.sipeed.revokeDisclaimer': 'Withdraw disclaimer consent',
    'settings.tailscale.sipeed.revokeDisclaimerDesc':
      'This will exit Sipeed Remote Network, delete the cloud network and its user, and clear your saved consent. All devices in this network will be disconnected. Continue?',
    'settings.tailscale.sipeed.revokeDisclaimerFailed':
      'Could not complete withdrawal. Please try again.',
    'settings.tailscale.sipeed.addDevice': 'Add device',
    'settings.tailscale.sipeed.addDeviceDesc':
      'No device has been added yet. Select Add device and finish joining on the computer or phone you want to connect.',
    'settings.tailscale.sipeed.claimExpiry': '{{time}} left · single use',
    'settings.tailscale.sipeed.claimExpired': 'The claim code has expired',
    'settings.tailscale.sipeed.claimCopy': 'Copy claim code',
    'settings.tailscale.sipeed.claimCopied': 'Copied',
    'settings.tailscale.sipeed.claimRenew': 'Get a new code',
    'settings.tailscale.sipeed.claimHint':
      'Install and open the Tailscale client on the device you want to add, choose the custom server and enter {{server}}. The browser opens the join page, where you enter these digits.'
  },
  es: {
    'tailscale.continue': 'Continuar',
    'tailscale.start': 'Activar',
    'tailscale.stop': 'Desactivar',
    'tailscale.restart': 'Reiniciar',
    'tailscale.disconnect':
      'Se desconectará Tailscale y se desactivará el inicio automático al arrancar.',
    'tailscale.confirm': 'Confirmar',
    'tailscale.cancel': 'Cancelar',
    'tailscale.stopped': 'Desactivado',
    'tailscale.connecting': 'Conectando',
    'tailscale.error': 'Error de conexión',
    'tailscale.connected': 'Conectado',
    'tailscale.offline': 'Sin conexión',
    'tailscale.notLogin': 'Sesión no iniciada',
    ...sipeedManagedFallbackMessages,
    ...sipeedRuntimeTranslations.es,
    ...sipeedManagementTranslations.es,
    'settings.tailscale.title': 'Tailscale',
    'settings.tailscale.restart': '¿Reiniciar Tailscale?',
    'settings.tailscale.stop': '¿Detener Tailscale?',
    'settings.tailscale.stopDesc':
      'Cerrar sesión en Tailscale y deshabilitar el inicio automático al arrancar.',
    'settings.tailscale.loading': 'Cargando...',
    'settings.tailscale.notInstall': '¡Tailscale no encontrado! Por favor, instálalo.',
    'settings.tailscale.install': 'Instalar',
    'settings.tailscale.installing': 'Instalando',
    'settings.tailscale.failed': 'La instalación falló',
    'settings.tailscale.retry':
      'Por favor, actualiza la página e inténtalo de nuevo. O intenta instalarlo manualmente',
    'settings.tailscale.download': 'Descargar el',
    'settings.tailscale.package': 'paquete de instalación',
    'settings.tailscale.unzip': 'y descomprimirlo',
    'settings.tailscale.upTailscale': 'Sube tailscale al directorio /usr/bin/ del NanoKVM',
    'settings.tailscale.upTailscaled': 'Sube tailscaled al directorio /usr/sbin/ del NanoKVM',
    'settings.tailscale.refresh': 'Actualizar la página actual',
    'settings.tailscale.notRunning':
      'Tailscale no se está ejecutando. Por favor, inícialo para continuar.',
    'settings.tailscale.run': 'Iniciar',
    'settings.tailscale.notLogin':
      'El dispositivo aún no ha sido vinculado. Por favor, inicia sesión y vincula este dispositivo a tu cuenta.',
    'settings.tailscale.urlPeriod': 'Esta URL es válida por 10 minutos',
    'settings.tailscale.login': 'Iniciar sesión',
    'settings.tailscale.loginSuccess': 'Inicio de sesión exitoso',
    'settings.tailscale.enable': 'Habilitar Tailscale',
    'settings.tailscale.exitNode': 'Nodo de salida',
    'settings.tailscale.exitNodeDesc':
      'Anuncia este dispositivo como salida a Internet para tu tailnet.',
    'settings.tailscale.subnetRoute': 'Ruta de subred',
    'settings.tailscale.subnetRouteDesc':
      'Anuncia la subred LAN conectada directamente {{subnet}} en tu tailnet.',
    'settings.tailscale.subnetUnavailable':
      'No hay ninguna subred LAN conectada directamente disponible.',
    'settings.tailscale.routingUnavailable':
      'La configuración de enrutamiento de Tailscale no está disponible.',
    'settings.tailscale.pendingApproval':
      'Después de habilitarla, aprueba la ruta anunciada en la consola de administración de Tailscale.',
    'settings.tailscale.routingUpdateFailed':
      'No se pudo actualizar la configuración de enrutamiento de Tailscale.',
    'settings.tailscale.deviceName': 'Nombre del dispositivo',
    'settings.tailscale.deviceIP': 'IP del dispositivo',
    'settings.tailscale.account': 'Cuenta',
    'settings.tailscale.logout': 'Cerrar sesión',
    'settings.tailscale.logoutDesc': '¿Estás seguro de que deseas cerrar sesión?',
    'settings.tailscale.logoutFailed': 'No se pudo cerrar sesión.',
    'settings.tailscale.uninstall': 'Desinstalar Tailscale',
    'settings.tailscale.uninstallDesc': '¿Estás seguro de que deseas desinstalar Tailscale?',
    'settings.tailscale.okBtn': 'Sí',
    'settings.tailscale.cancelBtn': 'Cancelar',
    'settings.tailscale.mode.official': 'Tailscale oficial',
    'settings.tailscale.mode.officialDesc': 'Usa el servicio oficial de Tailscale.',
    'settings.tailscale.mode.sipeed': 'Red remota Sipeed',
    'settings.tailscale.mode.sipeedDesc':
      'Conexiones directas entre dispositivos; sin retransmisión.',
    'settings.tailscale.sipeed.disclaimerTitle': 'Aviso legal',
    'settings.tailscale.sipeed.enableDesc':
      'Al activar la red remota Sipeed, se envían solicitudes a los servidores de Sipeed con datos de conexión, como el identificador del dispositivo, su nombre y su dirección IP de red virtual. Estos datos se utilizan para el registro, la autenticación, la coordinación de conexiones y la gestión de la red.\n\nEl acceso remoto depende de las condiciones de la red, la configuración del dispositivo y la disponibilidad de los servicios. Puede haber retrasos, interrupciones o falta temporal de conexión. Evalúa los riesgos y conserva una forma alternativa de acceso cuando sea necesario.\n\nUtiliza el servicio solo con dispositivos y redes que te pertenezcan o para los que tengas autorización. Protege las credenciales de acceso y los códigos de emparejamiento, y concede permisos a otras personas con cuidado.\n\nAl pulsar «Aceptar e iniciar», confirmas que has leído y comprendido este aviso y aceptas las comunicaciones y el tratamiento de información descritos para proporcionar conexiones remotas. El consentimiento se guarda en este dispositivo y no se solicita en cada activación.\n\nPuedes retirar el consentimiento desde el menú superior derecho de la página de conexión activa. Se cerrará la sesión en la red remota Sipeed y se solicitará eliminar la red en la nube y su usuario. Tras la eliminación, los demás dispositivos tampoco podrán usar esa red para el acceso remoto. Para volver a activar el servicio, tendrás que aceptar de nuevo.',
    'settings.tailscale.sipeed.acceptAndEnable': 'Aceptar e iniciar',
    'settings.tailscale.sipeed.revokeDisclaimer': 'Retirar el consentimiento',
    'settings.tailscale.sipeed.revokeDisclaimerDesc':
      'Se cerrará la sesión en la red remota Sipeed, se eliminarán la red en la nube y su usuario y se borrará el consentimiento guardado. Todos los dispositivos de esta red se desconectarán. ¿Continuar?',
    'settings.tailscale.sipeed.revokeDisclaimerFailed':
      'No se pudo completar la retirada del consentimiento. Inténtalo de nuevo.',
    'settings.tailscale.sipeed.addDevice': 'Añadir dispositivo',
    'settings.tailscale.sipeed.addDeviceDesc':
      'Aún no se ha añadido ningún dispositivo. Elige «Añadir dispositivo» y completa la conexión en el ordenador o móvil que quieras conectar.',
    'settings.tailscale.sipeed.claimExpiry': 'Quedan {{time}} · un solo uso',
    'settings.tailscale.sipeed.claimExpired': 'El código ha caducado',
    'settings.tailscale.sipeed.claimCopy': 'Copiar código',
    'settings.tailscale.sipeed.claimCopied': 'Copiado',
    'settings.tailscale.sipeed.claimRenew': 'Obtener un código nuevo',
    'settings.tailscale.sipeed.claimHint':
      'Instala y abre el cliente de Tailscale en el dispositivo que quieras añadir, elige el servidor personalizado e introduce {{server}}. El navegador abrirá la página de acceso, donde deberás introducir estos dígitos.'
  },
  fr: {
    'tailscale.continue': 'Continuer',
    'tailscale.start': 'Activer',
    'tailscale.stop': 'Désactiver',
    'tailscale.restart': 'Redémarrer',
    'tailscale.disconnect':
      'Cette action déconnectera Tailscale et désactivera le démarrage automatique.',
    'tailscale.confirm': 'Confirmer',
    'tailscale.cancel': 'Annuler',
    'tailscale.stopped': 'Désactivé',
    'tailscale.connecting': 'Connexion en cours',
    'tailscale.error': 'Erreur de connexion',
    'tailscale.connected': 'Connecté',
    'tailscale.offline': 'Hors ligne',
    'tailscale.notLogin': 'Non connecté au compte',
    ...sipeedManagedFallbackMessages,
    ...sipeedRuntimeTranslations.fr,
    ...sipeedManagementTranslations.fr,
    'settings.tailscale.title': 'Tailscale',
    'settings.tailscale.restart': 'Êtes-vous sûr de vouloir redémarrer Tailscale ?',
    'settings.tailscale.stop': 'Êtes-vous sûr de vouloir arrêter Tailscale ?',
    'settings.tailscale.stopDesc': 'Arrêtez Tailscale et désactivez son démarrage automatique.',
    'settings.tailscale.loading': 'Chargement...',
    'settings.tailscale.notInstall': "Tailscale non trouvé ! Veuillez l'installer.",
    'settings.tailscale.install': 'Installer',
    'settings.tailscale.installing': 'Installation',
    'settings.tailscale.failed': 'Installation échouée',
    'settings.tailscale.retry':
      "Veuillez rafraîchir et réessayer. Ou essayez d'installer manuellement",
    'settings.tailscale.download': 'Télécharger le',
    'settings.tailscale.package': "paquet d'installation",
    'settings.tailscale.unzip': 'et décompressez-le',
    'settings.tailscale.upTailscale': 'Téléverser tailscale dans le répertoire NanoKVM /usr/bin/',
    'settings.tailscale.upTailscaled':
      'Téléverser tailscaled dans le répertoire NanoKVM /usr/sbin/',
    'settings.tailscale.refresh': 'Rafraîchir la page courante',
    'settings.tailscale.notRunning':
      "Tailscale n'est pas en cours d'exécution. Veuillez le démarrer pour continuer.",
    'settings.tailscale.run': 'Démarrer',
    'settings.tailscale.notLogin':
      "L'appareil n'est pas relié. Connectez-vous et liez cet appareil à votre compte.",
    'settings.tailscale.urlPeriod': "L'URL est valide pendant 10 minutes",
    'settings.tailscale.login': 'Connexion',
    'settings.tailscale.loginSuccess': 'Connexion réussie',
    'settings.tailscale.enable': 'Démarrer Tailscale',
    'settings.tailscale.exitNode': 'Nœud de sortie',
    'settings.tailscale.exitNodeDesc':
      'Publier cet appareil comme sortie Internet pour votre tailnet.',
    'settings.tailscale.subnetRoute': 'Route de sous-réseau',
    'settings.tailscale.subnetRouteDesc':
      'Publier le sous-réseau LAN directement connecté {{subnet}} dans votre tailnet.',
    'settings.tailscale.subnetUnavailable':
      "Aucun sous-réseau LAN directement connecté n'est disponible.",
    'settings.tailscale.routingUnavailable':
      'Les paramètres de routage Tailscale ne sont pas disponibles sur cet appareil.',
    'settings.tailscale.pendingApproval':
      "Après l'activation, approuvez la route publiée dans la console d'administration Tailscale.",
    'settings.tailscale.routingUpdateFailed':
      'Impossible de mettre à jour les paramètres de routage Tailscale.',
    'settings.tailscale.deviceName': "Nom de l'appareil",
    'settings.tailscale.deviceIP': "IP de l'appareil",
    'settings.tailscale.account': 'Compte',
    'settings.tailscale.logout': 'Déconnexion',
    'settings.tailscale.logoutDesc': 'Êtes-vous sûr de vouloir vous déconnecter?',
    'settings.tailscale.logoutFailed': 'Échec de la déconnexion.',
    'settings.tailscale.uninstall': 'Désinstaller Tailscale',
    'settings.tailscale.uninstallDesc': 'Êtes-vous sûr de vouloir désinstaller Tailscale?',
    'settings.tailscale.okBtn': 'Oui',
    'settings.tailscale.cancelBtn': 'Non',
    'settings.tailscale.mode.official': 'Tailscale officiel',
    'settings.tailscale.mode.officialDesc': 'Utiliser le service officiel Tailscale.',
    'settings.tailscale.mode.sipeed': 'Réseau distant Sipeed',
    'settings.tailscale.mode.sipeedDesc': 'Connexions directes entre appareils ; sans relais.',
    'settings.tailscale.sipeed.disclaimerTitle': 'Avertissement',
    'settings.tailscale.sipeed.enableDesc':
      'Activer le réseau distant Sipeed envoie des requêtes aux serveurs Sipeed avec des informations de connexion, notamment l’identifiant de l’appareil, son nom et son adresse IP de réseau virtuel. Ces informations servent à l’enregistrement, à l’authentification, à la coordination des connexions et à la gestion du réseau.\n\nL’accès à distance dépend des conditions réseau, de la configuration de l’appareil et de la disponibilité des services. Des retards, interruptions ou indisponibilités temporaires peuvent survenir. Évaluez les risques et conservez un autre moyen d’accès si nécessaire.\n\nUtilisez ce service uniquement avec des appareils et réseaux qui vous appartiennent ou auxquels vous êtes autorisé à accéder. Protégez vos identifiants de connexion et codes d’association, et accordez les accès à autrui avec prudence.\n\nEn cliquant sur « Accepter et démarrer », vous confirmez avoir lu et compris cet avertissement et acceptez les communications et le traitement des informations décrits pour fournir les connexions à distance. Votre consentement est enregistré sur cet appareil et ne sera pas redemandé à chaque activation.\n\nVous pouvez retirer votre consentement dans le menu en haut à droite de la page de connexion active. Cette action quitte le réseau distant Sipeed et demande la suppression du réseau cloud associé et de son utilisateur. Après suppression, les autres appareils ne pourront plus utiliser ce réseau pour l’accès à distance. Une nouvelle activation nécessite un nouveau consentement.',
    'settings.tailscale.sipeed.acceptAndEnable': 'Accepter et démarrer',
    'settings.tailscale.sipeed.revokeDisclaimer': 'Retirer le consentement',
    'settings.tailscale.sipeed.revokeDisclaimerDesc':
      'Cette action quitte le réseau distant Sipeed, supprime le réseau cloud et son utilisateur, puis efface votre consentement enregistré. Tous les appareils de ce réseau seront déconnectés. Continuer ?',
    'settings.tailscale.sipeed.revokeDisclaimerFailed':
      'Le retrait du consentement a échoué. Veuillez réessayer.',
    'settings.tailscale.sipeed.addDevice': 'Ajouter un appareil',
    'settings.tailscale.sipeed.addDeviceDesc':
      'Aucun appareil ajouté pour l’instant. Choisissez « Ajouter un appareil » et terminez l’association sur l’ordinateur ou le téléphone à connecter.',
    'settings.tailscale.sipeed.claimExpiry': '{{time}} restantes · usage unique',
    'settings.tailscale.sipeed.claimExpired': 'Le code d’association a expiré',
    'settings.tailscale.sipeed.claimCopy': 'Copier le code',
    'settings.tailscale.sipeed.claimCopied': 'Copié',
    'settings.tailscale.sipeed.claimRenew': 'Obtenir un nouveau code',
    'settings.tailscale.sipeed.claimHint':
      'Installez et ouvrez le client Tailscale sur l’appareil à ajouter, choisissez le serveur personnalisé et saisissez {{server}}. Le navigateur ouvre la page d’association, où vous saisirez ces chiffres.'
  },
  id: {
    'tailscale.continue': 'Lanjutkan',
    'tailscale.start': 'Aktifkan',
    'tailscale.stop': 'Nonaktifkan',
    'tailscale.restart': 'Mulai ulang',
    'tailscale.disconnect':
      'Tindakan ini memutuskan koneksi Tailscale dan menonaktifkan mulai otomatis saat boot.',
    'tailscale.confirm': 'Konfirmasi',
    'tailscale.cancel': 'Batal',
    'tailscale.stopped': 'Dinonaktifkan',
    'tailscale.connecting': 'Menghubungkan',
    'tailscale.error': 'Kesalahan koneksi',
    'tailscale.connected': 'Terhubung',
    'tailscale.offline': 'Luring',
    'tailscale.notLogin': 'Belum masuk',
    ...sipeedManagedFallbackMessages,
    ...sipeedRuntimeTranslations.id,
    ...sipeedManagementTranslations.id,
    'settings.tailscale.title': 'Tailscale',
    'settings.tailscale.restart': 'Mulai ulang Tailscale?',
    'settings.tailscale.stop': 'Hentikan Tailscale?',
    'settings.tailscale.stopDesc':
      'Keluar dari Tailscale dan nonaktifkan mulai otomatis saat boot.',
    'settings.tailscale.loading': 'Memuat...',
    'settings.tailscale.notInstall': 'Tailscale tidak ditemukan! Silahkan pasang.',
    'settings.tailscale.install': 'Memasang',
    'settings.tailscale.installing': 'Memasangkan',
    'settings.tailscale.failed': 'Gagal memasangkan',
    'settings.tailscale.retry': 'Harap segarkan dan coba lagi. Atau coba instal secara manual',
    'settings.tailscale.download': 'Mengunduh',
    'settings.tailscale.package': 'paket instalasi',
    'settings.tailscale.unzip': 'dan unzip itu',
    'settings.tailscale.upTailscale': 'Unggah tailscale ke direktori NanoKVM /usr/bin/',
    'settings.tailscale.upTailscaled': 'Unggah tailscaled ke direktori NanoKVM /usr/sbin/',
    'settings.tailscale.refresh': 'Segarkan halaman ini',
    'settings.tailscale.notRunning': 'Tailscale tidak berjalan. Silakan mulai untuk melanjutkan.',
    'settings.tailscale.run': 'Mulai',
    'settings.tailscale.notLogin':
      'Perangkat belum ditautkan. Silakan masuk dan tautkan perangkat ini ke akun Anda.',
    'settings.tailscale.urlPeriod': 'Url ini berlaku selama 10 menit',
    'settings.tailscale.login': 'Masuk',
    'settings.tailscale.loginSuccess': 'Berhasil masuk',
    'settings.tailscale.enable': 'Aktifkan Tailscale',
    'settings.tailscale.exitNode': 'Simpul keluar',
    'settings.tailscale.exitNodeDesc':
      'Iklankan perangkat ini sebagai jalur keluar internet untuk tailnet Anda.',
    'settings.tailscale.subnetRoute': 'Rute subnet',
    'settings.tailscale.subnetRouteDesc':
      'Iklankan subnet LAN yang terhubung langsung {{subnet}} ke tailnet Anda.',
    'settings.tailscale.subnetUnavailable': 'Tidak ada subnet LAN yang terhubung langsung.',
    'settings.tailscale.routingUnavailable':
      'Pengaturan perutean Tailscale tidak tersedia di perangkat ini.',
    'settings.tailscale.pendingApproval':
      'Setelah diaktifkan, setujui rute yang diiklankan di konsol admin Tailscale.',
    'settings.tailscale.routingUpdateFailed': 'Gagal memperbarui pengaturan perutean Tailscale.',
    'settings.tailscale.deviceName': 'Nama Perangkat',
    'settings.tailscale.deviceIP': 'IP Perangkat',
    'settings.tailscale.account': 'Akun',
    'settings.tailscale.logout': 'Keluar',
    'settings.tailscale.logoutDesc': 'Apakah Anda yakin ingin logout?',
    'settings.tailscale.logoutFailed': 'Gagal keluar.',
    'settings.tailscale.uninstall': 'Copot pemasangan Tailscale',
    'settings.tailscale.uninstallDesc': 'Apakah Anda yakin ingin menghapus instalan Tailscale?',
    'settings.tailscale.okBtn': 'Ya',
    'settings.tailscale.cancelBtn': 'Tidak',
    'settings.tailscale.mode.official': 'Tailscale Resmi',
    'settings.tailscale.mode.officialDesc': 'Gunakan layanan resmi Tailscale.',
    'settings.tailscale.mode.sipeed': 'Jaringan Jarak Jauh Sipeed',
    'settings.tailscale.mode.sipeedDesc': 'Koneksi langsung antar perangkat; tanpa relai.',
    'settings.tailscale.sipeed.disclaimerTitle': 'Penafian',
    'settings.tailscale.sipeed.enableDesc':
      'Mengaktifkan Jaringan Jarak Jauh Sipeed akan mengirim permintaan ke server Sipeed beserta informasi koneksi, seperti pengenal perangkat, nama perangkat, dan alamat IP jaringan virtual. Informasi ini digunakan untuk pendaftaran perangkat, autentikasi, koordinasi koneksi, dan pengelolaan jaringan.\n\nAkses jarak jauh bergantung pada kondisi jaringan, konfigurasi perangkat, dan ketersediaan layanan. Koneksi dapat mengalami penundaan, terputus, atau tidak tersedia sementara. Pertimbangkan risikonya dan siapkan cara akses alternatif bila diperlukan.\n\nGunakan layanan hanya pada perangkat dan jaringan milik Anda atau yang boleh Anda akses. Jaga keamanan kredensial login dan kode pemasangan, serta berhati-hatilah saat memberikan akses kepada orang lain.\n\nDengan mengeklik “Setuju dan mulai”, Anda menyatakan telah membaca dan memahami pemberitahuan ini serta menyetujui komunikasi dan pemrosesan informasi yang dijelaskan untuk menyediakan koneksi jarak jauh. Persetujuan disimpan pada perangkat ini sehingga tidak perlu dikonfirmasi setiap kali layanan diaktifkan.\n\nAnda dapat menarik persetujuan melalui menu kanan atas pada halaman koneksi aktif. Tindakan ini keluar dari Jaringan Jarak Jauh Sipeed dan meminta penghapusan jaringan cloud beserta penggunanya. Setelah dihapus, perangkat lain juga tidak dapat menggunakan jaringan tersebut untuk akses jarak jauh. Pengaktifan kembali memerlukan persetujuan baru.',
    'settings.tailscale.sipeed.acceptAndEnable': 'Setuju dan mulai',
    'settings.tailscale.sipeed.revokeDisclaimer': 'Tarik persetujuan',
    'settings.tailscale.sipeed.revokeDisclaimerDesc':
      'Tindakan ini akan keluar dari Jaringan Jarak Jauh Sipeed, menghapus jaringan cloud beserta penggunanya, dan menghapus persetujuan yang tersimpan. Semua perangkat di jaringan ini akan terputus. Lanjutkan?',
    'settings.tailscale.sipeed.revokeDisclaimerFailed':
      'Penarikan persetujuan belum selesai. Silakan coba lagi.',
    'settings.tailscale.sipeed.addDevice': 'Tambah perangkat',
    'settings.tailscale.sipeed.addDeviceDesc':
      'Belum ada perangkat yang ditambahkan. Pilih "Tambah perangkat" dan selesaikan penggabungan di komputer atau ponsel yang ingin dihubungkan.',
    'settings.tailscale.sipeed.claimExpiry': 'Tersisa {{time}} · sekali pakai',
    'settings.tailscale.sipeed.claimExpired': 'Kode klaim sudah kedaluwarsa',
    'settings.tailscale.sipeed.claimCopy': 'Salin kode klaim',
    'settings.tailscale.sipeed.claimCopied': 'Disalin',
    'settings.tailscale.sipeed.claimRenew': 'Minta kode baru',
    'settings.tailscale.sipeed.claimHint':
      'Pasang dan buka klien Tailscale di perangkat yang ingin ditambahkan, pilih server khusus lalu isi {{server}}. Browser akan membuka halaman pendaftaran; masukkan angka ini di sana.'
  },
  it: {
    'tailscale.continue': 'Continua',
    'tailscale.start': 'Abilita',
    'tailscale.stop': 'Disabilita',
    'tailscale.restart': 'Riavvia',
    'tailscale.disconnect':
      'Tailscale verrà disconnesso e il suo avvio automatico verrà disabilitato.',
    'tailscale.confirm': 'Conferma',
    'tailscale.cancel': 'Annulla',
    'tailscale.stopped': 'Disabilitato',
    'tailscale.connecting': 'Connessione in corso',
    'tailscale.error': 'Errore di connessione',
    'tailscale.connected': 'Connesso',
    'tailscale.offline': 'Non in linea',
    'tailscale.notLogin': 'Accesso non effettuato',
    ...sipeedManagedFallbackMessages,
    ...sipeedRuntimeTranslations.it,
    ...sipeedManagementTranslations.it,
    'settings.tailscale.title': 'Tailscale',
    'settings.tailscale.restart': 'Riavviare Tailscale?',
    'settings.tailscale.stop': 'Arrestare Tailscale?',
    'settings.tailscale.stopDesc': 'Esci da Tailscale e disabilita l’avvio automatico al boot.',
    'settings.tailscale.loading': 'Caricamento...',
    'settings.tailscale.notInstall': 'Tailscale non trovato! Per favore, installa.',
    'settings.tailscale.install': 'Installa',
    'settings.tailscale.installing': 'Installazione in corso',
    'settings.tailscale.failed': 'Installazione fallita',
    'settings.tailscale.retry': 'Riprova aggiornando la pagina o installa manualmente',
    'settings.tailscale.download': 'Scarica il',
    'settings.tailscale.package': 'pacchetto di installazione',
    'settings.tailscale.unzip': 'e decomprimilo',
    'settings.tailscale.upTailscale': 'Carica tailscale nella directory /usr/bin/ del NanoKVM',
    'settings.tailscale.upTailscaled': 'Carica tailscaled nella directory /usr/sbin/ del NanoKVM',
    'settings.tailscale.refresh': 'Aggiorna la pagina corrente',
    'settings.tailscale.notRunning':
      'Tailscale non è in esecuzione. Per favore avvialo per continuare.',
    'settings.tailscale.run': 'Avvia',
    'settings.tailscale.notLogin':
      'Il dispositivo non è ancora stato associato. Effettua il login e associa questo dispositivo al tuo account.',
    'settings.tailscale.urlPeriod': 'Questo URL è valido per 10 minuti',
    'settings.tailscale.login': 'Accedi',
    'settings.tailscale.loginSuccess': 'Accesso riuscito',
    'settings.tailscale.enable': 'Abilita Tailscale',
    'settings.tailscale.exitNode': 'Nodo di uscita',
    'settings.tailscale.exitNodeDesc':
      'Pubblica questo dispositivo come uscita Internet per la tua tailnet.',
    'settings.tailscale.subnetRoute': 'Route di sottorete',
    'settings.tailscale.subnetRouteDesc':
      'Pubblica la sottorete LAN collegata direttamente {{subnet}} nella tua tailnet.',
    'settings.tailscale.subnetUnavailable':
      'Non è disponibile alcuna sottorete LAN collegata direttamente.',
    'settings.tailscale.routingUnavailable':
      'Le impostazioni di routing Tailscale non sono disponibili sul dispositivo.',
    'settings.tailscale.pendingApproval':
      'Dopo l’attivazione, approva la route pubblicata nella console di amministrazione Tailscale.',
    'settings.tailscale.routingUpdateFailed':
      'Impossibile aggiornare le impostazioni di routing Tailscale.',
    'settings.tailscale.deviceName': 'Nome Dispositivo',
    'settings.tailscale.deviceIP': 'IP Dispositivo',
    'settings.tailscale.account': 'Account',
    'settings.tailscale.logout': 'Disconnetti',
    'settings.tailscale.logoutDesc': 'Sei sicuro di voler uscire?',
    'settings.tailscale.logoutFailed': 'Disconnessione non riuscita.',
    'settings.tailscale.uninstall': 'Disinstalla Tailscale',
    'settings.tailscale.uninstallDesc': 'Sei sicuro di voler disinstallare Tailscale?',
    'settings.tailscale.okBtn': 'Sì',
    'settings.tailscale.cancelBtn': 'Annulla',
    'settings.tailscale.mode.official': 'Tailscale ufficiale',
    'settings.tailscale.mode.officialDesc': 'Usa il servizio ufficiale di Tailscale.',
    'settings.tailscale.mode.sipeed': 'Rete remota Sipeed',
    'settings.tailscale.mode.sipeedDesc': 'Connessioni dirette tra dispositivi; nessun relay.',
    'settings.tailscale.sipeed.disclaimerTitle': 'Avvertenza',
    'settings.tailscale.sipeed.enableDesc':
      'Attivando la rete remota Sipeed, vengono inviate richieste ai server Sipeed con informazioni di connessione, come identificativo del dispositivo, nome e indirizzo IP della rete virtuale. Queste informazioni servono per la registrazione, l’autenticazione, il coordinamento delle connessioni e la gestione della rete.\n\nL’accesso remoto dipende dalle condizioni della rete, dalla configurazione del dispositivo e dalla disponibilità dei servizi. Le connessioni possono subire ritardi, interruzioni o indisponibilità temporanee. Valuta i rischi e mantieni un accesso alternativo quando necessario.\n\nUsa il servizio solo con dispositivi e reti di tua proprietà o a cui sei autorizzato ad accedere. Proteggi le credenziali di accesso e i codici di abbinamento e concedi l’accesso ad altri con attenzione.\n\nFacendo clic su “Accetta e avvia”, confermi di aver letto e compreso questo avviso e acconsenti alle comunicazioni e al trattamento delle informazioni descritti per fornire connessioni remote. Il consenso viene salvato su questo dispositivo e non va confermato a ogni attivazione.\n\nPuoi revocare il consenso dal menu in alto a destra nella pagina della connessione attiva. Verrà effettuata l’uscita dalla rete remota Sipeed e richiesta l’eliminazione della rete cloud associata e del relativo utente. Dopo l’eliminazione, anche gli altri dispositivi non potranno più usare questa rete per l’accesso remoto. Una nuova attivazione richiede un nuovo consenso.',
    'settings.tailscale.sipeed.acceptAndEnable': 'Accetta e avvia',
    'settings.tailscale.sipeed.revokeDisclaimer': 'Revoca il consenso',
    'settings.tailscale.sipeed.revokeDisclaimerDesc':
      'Questa operazione esce dalla rete remota Sipeed, elimina la rete cloud e il relativo utente e cancella il consenso salvato. Tutti i dispositivi di questa rete verranno disconnessi. Continuare?',
    'settings.tailscale.sipeed.revokeDisclaimerFailed':
      'Impossibile completare la revoca. Riprova.',
    'settings.tailscale.sipeed.addDevice': 'Aggiungi dispositivo',
    'settings.tailscale.sipeed.addDeviceDesc':
      'Non è ancora stato aggiunto alcun dispositivo. Scegli «Aggiungi dispositivo» e completa l’accesso sul computer o telefono da collegare.',
    'settings.tailscale.sipeed.claimExpiry': 'Restano {{time}} · uso singolo',
    'settings.tailscale.sipeed.claimExpired': 'Il codice è scaduto',
    'settings.tailscale.sipeed.claimCopy': 'Copia codice',
    'settings.tailscale.sipeed.claimCopied': 'Copiato',
    'settings.tailscale.sipeed.claimRenew': 'Ottieni un nuovo codice',
    'settings.tailscale.sipeed.claimHint':
      'Installa e apri il client Tailscale sul dispositivo da aggiungere, scegli il server personalizzato e inserisci {{server}}. Il browser aprirà la pagina di ingresso, dove inserire queste cifre.'
  },
  pl: {
    'tailscale.continue': 'Dalej',
    'tailscale.start': 'Włącz',
    'tailscale.stop': 'Wyłącz',
    'tailscale.restart': 'Uruchom ponownie',
    'tailscale.disconnect':
      'Spowoduje to rozłączenie Tailscale i wyłączenie automatycznego uruchamiania przy starcie systemu.',
    'tailscale.confirm': 'Potwierdź',
    'tailscale.cancel': 'Anuluj',
    'tailscale.stopped': 'Wyłączono',
    'tailscale.connecting': 'Łączenie',
    'tailscale.error': 'Błąd połączenia',
    'tailscale.connected': 'Połączono',
    'tailscale.offline': 'Offline',
    'tailscale.notLogin': 'Nie zalogowano',
    ...sipeedManagedFallbackMessages,
    ...sipeedRuntimeTranslations.pl,
    ...sipeedManagementTranslations.pl,
    'settings.tailscale.title': 'Tailscale',
    'settings.tailscale.restart': 'Uruchomić ponownie Tailscale?',
    'settings.tailscale.stop': 'Zatrzymać Tailscale?',
    'settings.tailscale.stopDesc':
      'Wyloguj Tailscale i wyłącz automatyczne uruchamianie przy starcie systemu.',
    'settings.tailscale.loading': 'Ładowanie...',
    'settings.tailscale.notInstall': 'Nie znaleziono Tailscale! Proszę zainstalować.',
    'settings.tailscale.install': 'Instaluj',
    'settings.tailscale.installing': 'Instalowanie',
    'settings.tailscale.failed': 'Instalowanie nie powiodło się',
    'settings.tailscale.retry':
      'Odśwież stronę i spróbuj ponownie, albo spróbuj zainstalować manualnie.',
    'settings.tailscale.download': 'Pobierz',
    'settings.tailscale.package': 'pakiet instalacyjny',
    'settings.tailscale.unzip': 'i wypakuj pliki',
    'settings.tailscale.upTailscale': 'Prześlij tailscale do NanoKVM w katalogu /usr/bin/',
    'settings.tailscale.upTailscaled': 'Prześlij tailscaled do NanoKVM w katalogu /usr/sbin/',
    'settings.tailscale.refresh': 'Odśwież obecną stronę',
    'settings.tailscale.notRunning': 'Tailscale nie działa. Rozpocznij, aby kontynuować.',
    'settings.tailscale.run': 'Uruchom',
    'settings.tailscale.notLogin':
      'Urządzenie nie zostało jeszcze powiązane. Zaloguj się i powiąż to urządzenie ze swoim kontem.',
    'settings.tailscale.urlPeriod': 'Ten URL jest ważny przez 10 minut',
    'settings.tailscale.login': 'Zaloguj',
    'settings.tailscale.loginSuccess': 'Zalogowanie pomyślne',
    'settings.tailscale.enable': 'Włącz Tailscale',
    'settings.tailscale.exitNode': 'Węzeł wyjściowy',
    'settings.tailscale.exitNodeDesc':
      'Ogłaszaj to urządzenie jako wyjście internetowe dla sieci tailnet.',
    'settings.tailscale.subnetRoute': 'Trasa podsieci',
    'settings.tailscale.subnetRouteDesc':
      'Ogłaszaj bezpośrednio połączoną podsieć LAN {{subnet}} w sieci tailnet.',
    'settings.tailscale.subnetUnavailable': 'Brak dostępnej bezpośrednio połączonej podsieci LAN.',
    'settings.tailscale.routingUnavailable':
      'Ustawienia routingu Tailscale są niedostępne na tym urządzeniu.',
    'settings.tailscale.pendingApproval':
      'Po włączeniu zatwierdź ogłaszaną trasę w konsoli administracyjnej Tailscale.',
    'settings.tailscale.routingUpdateFailed':
      'Nie udało się zaktualizować ustawień routingu Tailscale.',
    'settings.tailscale.deviceName': 'Nazwa urządzenia',
    'settings.tailscale.deviceIP': 'Adres IP urządzenia',
    'settings.tailscale.account': 'Konto',
    'settings.tailscale.logout': 'Wyloguj',
    'settings.tailscale.logoutDesc': 'Czy na pewno chcesz się wylogować?',
    'settings.tailscale.logoutFailed': 'Wylogowanie nie powiodło się.',
    'settings.tailscale.uninstall': 'Odinstaluj Tailscale',
    'settings.tailscale.uninstallDesc': 'Czy na pewno chcesz odinstalować Tailscale?',
    'settings.tailscale.okBtn': 'Tak',
    'settings.tailscale.cancelBtn': 'Nie',
    'settings.tailscale.mode.official': 'Oficjalny Tailscale',
    'settings.tailscale.mode.officialDesc': 'Użyj oficjalnej usługi Tailscale.',
    'settings.tailscale.mode.sipeed': 'Zdalna sieć Sipeed',
    'settings.tailscale.mode.sipeedDesc':
      'Bezpośrednie połączenia między urządzeniami; bez przekazywania.',
    'settings.tailscale.sipeed.disclaimerTitle': 'Zastrzeżenie',
    'settings.tailscale.sipeed.enableDesc':
      'Włączenie sieci zdalnej Sipeed powoduje wysyłanie żądań do serwerów Sipeed wraz z danymi połączenia, takimi jak identyfikator urządzenia, jego nazwa i adres IP sieci wirtualnej. Dane służą do rejestracji urządzeń, uwierzytelniania, koordynacji połączeń i zarządzania siecią.\n\nDostęp zdalny zależy od warunków sieciowych, konfiguracji urządzenia i dostępności usług. Mogą występować opóźnienia, przerwy lub czasowa niedostępność. Oceń ryzyko korzystania z usługi i w razie potrzeby zachowaj alternatywny sposób dostępu.\n\nKorzystaj z usługi tylko na własnych urządzeniach i w sieciach, do których masz uprawnienia. Chroń dane logowania i kody parowania oraz ostrożnie przyznawaj innym dostęp.\n\nKlikając „Zgadzam się, uruchom”, potwierdzasz przeczytanie i zrozumienie tego oświadczenia oraz zgodę na opisaną komunikację i przetwarzanie informacji w celu zapewnienia połączeń zdalnych. Zgoda jest zapisywana na tym urządzeniu, więc nie trzeba jej potwierdzać przy każdym uruchomieniu.\n\nZgodę można wycofać w menu w prawym górnym rogu strony aktywnego połączenia. Spowoduje to opuszczenie sieci zdalnej Sipeed i wysłanie żądania usunięcia powiązanej sieci w chmurze oraz jej użytkownika. Po usunięciu inne urządzenia również utracą możliwość zdalnego dostępu przez tę sieć. Ponowne włączenie wymaga ponownej zgody.',
    'settings.tailscale.sipeed.acceptAndEnable': 'Zgadzam się, uruchom',
    'settings.tailscale.sipeed.revokeDisclaimer': 'Wycofaj zgodę',
    'settings.tailscale.sipeed.revokeDisclaimerDesc':
      'Spowoduje to opuszczenie sieci zdalnej Sipeed, usunięcie sieci w chmurze wraz z jej użytkownikiem oraz zapisanej zgody. Wszystkie urządzenia w tej sieci zostaną rozłączone. Kontynuować?',
    'settings.tailscale.sipeed.revokeDisclaimerFailed':
      'Nie udało się dokończyć wycofywania zgody. Spróbuj ponownie.',
    'settings.tailscale.sipeed.addDevice': 'Dodaj urządzenie',
    'settings.tailscale.sipeed.addDeviceDesc':
      'Nie dodano jeszcze żadnego urządzenia. Wybierz „Dodaj urządzenie” i dokończ łączenie na komputerze lub telefonie, który chcesz podłączyć.',
    'settings.tailscale.sipeed.claimExpiry': 'Pozostało {{time}} · jednorazowo',
    'settings.tailscale.sipeed.claimExpired': 'Kod wygasł',
    'settings.tailscale.sipeed.claimCopy': 'Kopiuj kod',
    'settings.tailscale.sipeed.claimCopied': 'Skopiowano',
    'settings.tailscale.sipeed.claimRenew': 'Pobierz nowy kod',
    'settings.tailscale.sipeed.claimHint':
      'Zainstaluj i otwórz klienta Tailscale na urządzeniu, które chcesz dodać, wybierz niestandardowy serwer i wpisz {{server}}. Przeglądarka otworzy stronę dołączania, gdzie wpiszesz te cyfry.'
  },
  ru: {
    'tailscale.continue': 'Продолжить',
    'tailscale.start': 'Включить',
    'tailscale.stop': 'Отключить',
    'tailscale.restart': 'Перезапустить',
    'tailscale.disconnect':
      'Это отключит Tailscale и его автоматический запуск при загрузке системы.',
    'tailscale.confirm': 'Подтвердить',
    'tailscale.cancel': 'Отмена',
    'tailscale.stopped': 'Отключено',
    'tailscale.connecting': 'Подключение',
    'tailscale.error': 'Ошибка подключения',
    'tailscale.connected': 'Подключено',
    'tailscale.offline': 'Не в сети',
    'tailscale.notLogin': 'Вход не выполнен',
    ...sipeedManagedFallbackMessages,
    ...sipeedRuntimeTranslations.ru,
    ...sipeedManagementTranslations.ru,
    'settings.tailscale.title': 'Tailscale',
    'settings.tailscale.restart': 'Вы уверены что хотите перезагрузить Tailscale?',
    'settings.tailscale.stop': 'Вы уверены что хотите остановить Tailscale?',
    'settings.tailscale.stopDesc':
      'Выйти из Tailscale и отключить его автоматический запуск при включении.',
    'settings.tailscale.loading': 'Загрузка...',
    'settings.tailscale.notInstall': 'Tailscale не найден! Пожалуйста, установите.',
    'settings.tailscale.install': 'Установить',
    'settings.tailscale.installing': 'Установка',
    'settings.tailscale.failed': 'Не удалось установить',
    'settings.tailscale.retry':
      'Пожалуйста, обновите и попробуйте снова, или попробуйте установить вручную',
    'settings.tailscale.download': 'Скачайте',
    'settings.tailscale.package': 'установочный пакет',
    'settings.tailscale.unzip': 'и разархивируйте его',
    'settings.tailscale.upTailscale': 'Переместите tailscale в каталог /usr/bin/ на NanoKVM',
    'settings.tailscale.upTailscaled': 'Переместите tailscaled в каталог /usr/sbin/ на NanoKVM',
    'settings.tailscale.refresh': 'Обновите текущую страницу',
    'settings.tailscale.notRunning':
      'Tailscale не запущен. Пожалуйста, чтобы продолжить, запустите его.',
    'settings.tailscale.run': 'Запустить',
    'settings.tailscale.notLogin':
      'Устройство не привязано. Войдите, чтобы привязать его к аккаунту.',
    'settings.tailscale.urlPeriod': 'Этот адрес действителен в течение 10 минут',
    'settings.tailscale.login': 'Войти',
    'settings.tailscale.loginSuccess': 'Вход выполнен',
    'settings.tailscale.enable': 'Включить Tailscale',
    'settings.tailscale.exitNode': 'Выходной узел',
    'settings.tailscale.exitNodeDesc':
      'Объявить это устройство интернет-шлюзом для вашей сети tailnet.',
    'settings.tailscale.subnetRoute': 'Маршрут подсети',
    'settings.tailscale.subnetRouteDesc':
      'Объявить напрямую подключённую LAN-подсеть {{subnet}} в вашей сети tailnet.',
    'settings.tailscale.subnetUnavailable': 'Нет доступной напрямую подключённой LAN-подсети.',
    'settings.tailscale.routingUnavailable':
      'Настройки маршрутизации Tailscale недоступны на этом устройстве.',
    'settings.tailscale.pendingApproval':
      'После включения одобрите объявленный маршрут в консоли администрирования Tailscale.',
    'settings.tailscale.routingUpdateFailed':
      'Не удалось обновить настройки маршрутизации Tailscale.',
    'settings.tailscale.deviceName': 'Имя устройства',
    'settings.tailscale.deviceIP': 'IP-адрес устройства',
    'settings.tailscale.account': 'Учетная запись',
    'settings.tailscale.logout': 'Выход',
    'settings.tailscale.logoutDesc': 'Вы действительно хотите выйти?',
    'settings.tailscale.logoutFailed': 'Не удалось выйти.',
    'settings.tailscale.uninstall': 'Удалить Tailscale',
    'settings.tailscale.uninstallDesc': 'Вы уверены, что хотите удалить Tailscale?',
    'settings.tailscale.okBtn': 'Да',
    'settings.tailscale.cancelBtn': 'Нет',
    'settings.tailscale.mode.official': 'Официальный Tailscale',
    'settings.tailscale.mode.officialDesc': 'Использовать официальный сервис Tailscale.',
    'settings.tailscale.mode.sipeed': 'Удалённая сеть Sipeed',
    'settings.tailscale.mode.sipeedDesc': 'Прямые соединения между устройствами; без ретрансляции.',
    'settings.tailscale.sipeed.disclaimerTitle': 'Отказ от ответственности',
    'settings.tailscale.sipeed.enableDesc':
      'При включении удалённой сети Sipeed на серверы Sipeed отправляются запросы с данными подключения, такими как идентификатор устройства, его имя и IP-адрес виртуальной сети. Эти данные используются для регистрации устройств, аутентификации, координации соединений и управления сетью.\n\nУдалённый доступ зависит от состояния сети, настроек устройства и доступности сервисов. Возможны задержки, прерывания и временная недоступность соединений. Оцените риски использования и при необходимости сохраните альтернативный способ доступа.\n\nИспользуйте сервис только с собственными устройствами и сетями либо при наличии разрешения на доступ. Защищайте учётные данные и коды сопряжения и с осторожностью предоставляйте доступ другим людям.\n\nНажимая «Согласиться и запустить», вы подтверждаете, что прочитали и поняли это уведомление, и соглашаетесь на описанный обмен данными и их обработку для обеспечения удалённых соединений. Согласие сохраняется на этом устройстве и не требует подтверждения при каждом включении.\n\nВы можете отозвать согласие в меню в правом верхнем углу страницы активного подключения. Будет выполнен выход из удалённой сети Sipeed и отправлен запрос на удаление связанной облачной сети и её пользователя. После удаления другие устройства также не смогут использовать эту сеть для удалённого доступа. Для повторного включения нужно снова дать согласие.',
    'settings.tailscale.sipeed.acceptAndEnable': 'Согласиться и запустить',
    'settings.tailscale.sipeed.revokeDisclaimer': 'Отозвать согласие',
    'settings.tailscale.sipeed.revokeDisclaimerDesc':
      'Будет выполнен выход из удалённой сети Sipeed, удалены облачная сеть и её пользователь, а также сохранённое согласие. Все устройства этой сети будут отключены. Продолжить?',
    'settings.tailscale.sipeed.revokeDisclaimerFailed':
      'Не удалось завершить отзыв согласия. Повторите попытку.',
    'settings.tailscale.sipeed.addDevice': 'Добавить устройство',
    'settings.tailscale.sipeed.addDeviceDesc':
      'Устройства пока не добавлены. Выберите «Добавить устройство» и завершите подключение на компьютере или телефоне.',
    'settings.tailscale.sipeed.claimExpiry': 'Осталось {{time}} · одноразовый код',
    'settings.tailscale.sipeed.claimExpired': 'Срок действия кода истёк',
    'settings.tailscale.sipeed.claimCopy': 'Скопировать код',
    'settings.tailscale.sipeed.claimCopied': 'Скопировано',
    'settings.tailscale.sipeed.claimRenew': 'Получить новый код',
    'settings.tailscale.sipeed.claimHint':
      'Установите и откройте клиент Tailscale на устройстве, которое нужно добавить, выберите собственный сервер и укажите {{server}}. Браузер откроет страницу подключения, где нужно ввести эти цифры.'
  },
  ko: {
    'tailscale.continue': '계속',
    'tailscale.start': '활성화',
    'tailscale.stop': '비활성화',
    'tailscale.restart': '다시 시작',
    'tailscale.disconnect': 'Tailscale 연결을 끊고 부팅 시 자동 시작을 비활성화합니다.',
    'tailscale.confirm': '확인',
    'tailscale.cancel': '취소',
    'tailscale.stopped': '비활성화됨',
    'tailscale.connecting': '연결 중',
    'tailscale.error': '연결 오류',
    'tailscale.connected': '연결됨',
    'tailscale.offline': '오프라인',
    'tailscale.notLogin': '로그인되지 않음',
    ...sipeedManagedFallbackMessages,
    ...sipeedRuntimeTranslations.ko,
    ...sipeedManagementTranslations.ko,
    'settings.tailscale.title': 'Tailscale',
    'settings.tailscale.restart': '정말로 Tailscale을 다시 시작하시겠습니까?',
    'settings.tailscale.stop': '정말로 Tailscale을 중지하시겠습니까?',
    'settings.tailscale.stopDesc': 'Tailscale에서 로그아웃하고 자동 시작을 비활성화합니다.',
    'settings.tailscale.loading': '불러오는 중...',
    'settings.tailscale.notInstall': 'Tailscale이 없습니다. 설치해주세요.',
    'settings.tailscale.install': '설치',
    'settings.tailscale.installing': '설치중',
    'settings.tailscale.failed': '설치 실패',
    'settings.tailscale.retry': '새로고침하고 다시 시도하거나, 수동으로 설치하세요',
    'settings.tailscale.download': '다운로드 중 :',
    'settings.tailscale.package': '패키지 설치',
    'settings.tailscale.unzip': '압축 해제',
    'settings.tailscale.upTailscale':
      'tailscale을 NanoKVM 의 다음 경로에 업로드 했습니다. : /usr/bin/',
    'settings.tailscale.upTailscaled':
      'tailscaled을 NanoKVM 의 다음 경로에 업로드 했습니다. :  /usr/sbin/',
    'settings.tailscale.refresh': '현재 페이지 새로고침',
    'settings.tailscale.notRunning':
      'Tailscale이 실행되고 있지 않습니다. 계속하려면 시작해 주세요.',
    'settings.tailscale.run': '시작',
    'settings.tailscale.notLogin':
      '이 기기는 현재 연동 되지 않았습니다. 로그인해서 계정에 이 장치를 연동하세요.',
    'settings.tailscale.urlPeriod': '이 주소는 10분간 유효합니다.',
    'settings.tailscale.login': '로그인',
    'settings.tailscale.loginSuccess': '로그인 성공',
    'settings.tailscale.enable': 'Tailscale 활성화',
    'settings.tailscale.exitNode': '출구 노드',
    'settings.tailscale.exitNodeDesc': '이 장치를 tailnet의 인터넷 출구로 알립니다.',
    'settings.tailscale.subnetRoute': '서브넷 경로',
    'settings.tailscale.subnetRouteDesc': '직접 연결된 LAN 서브넷 {{subnet}}을 tailnet에 알립니다.',
    'settings.tailscale.subnetUnavailable': '사용 가능한 직접 연결 LAN 서브넷이 없습니다.',
    'settings.tailscale.routingUnavailable':
      '이 장치에서는 Tailscale 라우팅 설정을 사용할 수 없습니다.',
    'settings.tailscale.pendingApproval':
      '활성화한 후 Tailscale 관리 콘솔에서 알린 경로를 승인하세요.',
    'settings.tailscale.routingUpdateFailed': 'Tailscale 라우팅 설정을 업데이트하지 못했습니다.',
    'settings.tailscale.deviceName': '장치 이름',
    'settings.tailscale.deviceIP': '장치 IP',
    'settings.tailscale.account': '계정',
    'settings.tailscale.logout': '로그아웃',
    'settings.tailscale.logoutDesc': '정말로 로그아웃 하시겠습니까?',
    'settings.tailscale.logoutFailed': '로그아웃하지 못했습니다.',
    'settings.tailscale.uninstall': 'Tailscale 제거',
    'settings.tailscale.uninstallDesc': '정말로 Tailscale을 제거할까요?',
    'settings.tailscale.okBtn': '네',
    'settings.tailscale.cancelBtn': '아니오',
    'settings.tailscale.mode.official': '공식 Tailscale',
    'settings.tailscale.mode.officialDesc': 'Tailscale 공식 서비스를 사용합니다.',
    'settings.tailscale.mode.sipeed': 'Sipeed 원격 네트워크',
    'settings.tailscale.mode.sipeedDesc': '기기 간 P2P 직접 연결이며 중계를 사용하지 않습니다.',
    'settings.tailscale.sipeed.disclaimerTitle': '면책 고지',
    'settings.tailscale.sipeed.enableDesc':
      'Sipeed 원격 네트워크를 활성화하면 장치 식별자, 장치 이름, 가상 네트워크 IP 주소 등의 연결 정보와 함께 Sipeed 서버로 요청을 보냅니다. 이 정보는 장치 등록, 인증, 연결 조정 및 네트워크 관리에 사용됩니다.\n\n원격 액세스는 네트워크 환경, 장치 설정 및 서비스 가용성에 따라 달라집니다. 연결이 지연되거나 중단되거나 일시적으로 불가능할 수 있습니다. 사용에 따른 위험을 검토하고 필요할 때 사용할 대체 액세스 방법을 마련하세요.\n\n본인이 소유하거나 액세스 권한을 부여받은 장치 및 네트워크에서만 사용하세요. 로그인 자격 증명과 페어링 코드를 안전하게 보관하고 다른 사람에게 액세스 권한을 신중하게 부여하세요.\n\n“동의하고 시작”을 클릭하면 이 고지를 읽고 이해했으며, 원격 연결 제공을 위한 위 통신 및 정보 처리에 동의한 것으로 간주됩니다. 동의 상태는 이 장치에 저장되므로 활성화할 때마다 다시 확인할 필요가 없습니다.\n\n연결된 화면의 오른쪽 위 메뉴에서 동의를 철회할 수 있습니다. 철회하면 Sipeed 원격 네트워크에서 로그아웃하고 관련 클라우드 네트워크와 사용자 삭제를 요청합니다. 삭제되면 다른 장치도 이 네트워크를 통한 원격 액세스를 사용할 수 없습니다. 다시 활성화하려면 새로 동의해야 합니다.',
    'settings.tailscale.sipeed.acceptAndEnable': '동의하고 시작',
    'settings.tailscale.sipeed.revokeDisclaimer': '면책 고지 동의 철회',
    'settings.tailscale.sipeed.revokeDisclaimerDesc':
      '동의를 철회하면 Sipeed 원격 네트워크에서 로그아웃하고 클라우드 네트워크와 해당 사용자를 삭제하며 저장된 동의 상태를 지웁니다. 이 네트워크의 모든 장치 연결이 끊어집니다. 계속하시겠습니까?',
    'settings.tailscale.sipeed.revokeDisclaimerFailed':
      '동의 철회를 완료하지 못했습니다. 다시 시도하세요.',
    'settings.tailscale.sipeed.addDevice': '기기 추가',
    'settings.tailscale.sipeed.addDeviceDesc':
      '아직 추가된 기기가 없습니다. "기기 추가"를 선택하고 연결할 컴퓨터나 휴대폰에서 등록을 완료하세요.',
    'settings.tailscale.sipeed.claimExpiry': '남은 시간 {{time}} · 1회만 사용 가능',
    'settings.tailscale.sipeed.claimExpired': '인증 코드가 만료되었습니다',
    'settings.tailscale.sipeed.claimCopy': '인증 코드 복사',
    'settings.tailscale.sipeed.claimCopied': '복사했습니다',
    'settings.tailscale.sipeed.claimRenew': '코드 다시 받기',
    'settings.tailscale.sipeed.claimHint':
      '추가할 기기에 Tailscale 클라이언트를 설치해 열고, 사용자 지정 서버에 {{server}} 값을 입력하세요. 브라우저에 접속 페이지가 열리면 이 숫자를 입력합니다.'
  },
  zh: {
    'tailscale.continue': '继续',
    'tailscale.start': '启用',
    'tailscale.stop': '停用',
    'tailscale.restart': '重启',
    'tailscale.disconnect': '此操作将断开 Tailscale 连接，并关闭开机自动启动。',
    'tailscale.confirm': '确认',
    'tailscale.cancel': '取消',
    'tailscale.stopped': '已停用',
    'tailscale.connecting': '正在连接',
    'tailscale.error': '连接异常',
    'tailscale.connected': '已连接',
    'tailscale.offline': '离线',
    'tailscale.notLogin': '未登录',
    ...sipeedManagedFallbackMessages,
    ...sipeedRuntimeTranslations.zh,
    ...sipeedManagementTranslations.zh,
    'settings.tailscale.title': 'Tailscale',
    'settings.tailscale.restart': '确定要重启 Tailscale 吗？',
    'settings.tailscale.stop': '确定要停止 Tailscale 吗？',
    'settings.tailscale.stopDesc': '该操作会退出登录，并停止开机自动启动。',
    'settings.tailscale.loading': '加载中...',
    'settings.tailscale.notInstall': '未检测到 Tailscale，请先安装',
    'settings.tailscale.install': '安装',
    'settings.tailscale.installing': '安装中',
    'settings.tailscale.failed': '安装失败',
    'settings.tailscale.retry': '请刷新后重试，或尝试手动安装',
    'settings.tailscale.download': '下载',
    'settings.tailscale.package': '安装包',
    'settings.tailscale.unzip': '并解压',
    'settings.tailscale.upTailscale': '将 tailscale 文件上传到 /usr/bin/ 目录',
    'settings.tailscale.upTailscaled': '将 tailscaled 文件上传到 /usr/sbin/ 目录',
    'settings.tailscale.refresh': '刷新页面',
    'settings.tailscale.notRunning': 'Tailscale 尚未运行，请先执行启动操作',
    'settings.tailscale.run': '启动',
    'settings.tailscale.notLogin': '该设备尚未绑定，请点击登录并将这台设备绑定到您的账号。',
    'settings.tailscale.urlPeriod': '该链接10分钟内有效',
    'settings.tailscale.login': '登录',
    'settings.tailscale.loginSuccess': '登录完成',
    'settings.tailscale.enable': '启用 Tailscale',
    'settings.tailscale.exitNode': '出口节点',
    'settings.tailscale.exitNodeDesc': '将此设备发布为 tailnet 的互联网出口。',
    'settings.tailscale.subnetRoute': '子网路由',
    'settings.tailscale.subnetRouteDesc': '向 tailnet 发布本机直连局域网网段 {{subnet}}。',
    'settings.tailscale.subnetUnavailable': '未检测到可发布的直连局域网网段。',
    'settings.tailscale.routingUnavailable': '此设备当前无法配置 Tailscale 路由。',
    'settings.tailscale.pendingApproval': '启用后，请在 Tailscale 管理后台批准发布的路由。',
    'settings.tailscale.routingUpdateFailed': '更新 Tailscale 路由设置失败。',
    'settings.tailscale.deviceName': '设备名称',
    'settings.tailscale.deviceIP': '设备地址',
    'settings.tailscale.account': '账号',
    'settings.tailscale.logout': '退出',
    'settings.tailscale.logoutDesc': '确定要退出吗？',
    'settings.tailscale.logoutFailed': '退出失败。',
    'settings.tailscale.uninstall': '卸载 Tailscale',
    'settings.tailscale.uninstallDesc': '确定要卸载 Tailscale 吗？',
    'settings.tailscale.okBtn': '确认',
    'settings.tailscale.cancelBtn': '取消',
    'settings.tailscale.mode.official': '官方 Tailscale',
    'settings.tailscale.mode.officialDesc': '使用 Tailscale 官方服务。',
    'settings.tailscale.mode.sipeed': 'Sipeed 远程网络',
    'settings.tailscale.mode.sipeedDesc': '设备之间点对点直连，不提供中继。',
    'settings.tailscale.sipeed.disclaimerTitle': '免责声明',
    'settings.tailscale.sipeed.enableDesc':
      '启用 Sipeed 远程网络后，本设备会向 Sipeed 服务器发送请求，并提供设备标识、设备名称、虚拟网络 IP 地址等连接信息，用于设备注册、身份验证、连接协调和网络管理。\n\n远程访问依赖网络环境、设备配置及相关服务，可能出现延迟、中断或暂时不可用。请自行评估使用风险，并保留必要的备用访问方式。\n\n请仅在您拥有或已获得授权的设备与网络上使用本服务，并妥善保管登录凭据和配对码，谨慎授予他人访问权限。\n\n点击“同意并启用”即表示您已阅读并理解本声明，同意为提供远程连接而进行上述通信和信息处理。同意状态将保存在本设备上，后续启用无需重复确认。\n\n您可在已连接页面右上角菜单中撤回同意。撤回后会退出 Sipeed 远程网络，并请求删除对应的云端网络及用户；删除后，该网络中的其他设备也将无法继续通过此网络访问。再次启用时需重新同意。',
    'settings.tailscale.sipeed.acceptAndEnable': '同意并启用',
    'settings.tailscale.sipeed.revokeDisclaimer': '撤回免责声明同意',
    'settings.tailscale.sipeed.revokeDisclaimerDesc':
      '撤回同意将退出 Sipeed 远程网络、删除云端网络及用户，并清除已保存的同意状态。该网络中的所有设备将断开连接，是否继续？',
    'settings.tailscale.sipeed.revokeDisclaimerFailed': '撤回未完成，请重试。',
    'settings.tailscale.sipeed.enabling': '正在启用远程网络…',
    'settings.tailscale.sipeed.enableFailed': '远程网络启用失败。',
    'settings.tailscale.sipeed.officialActive':
      '当前已有官方 Tailscale 身份，本版本暂不支持直接切换身份。',
    'settings.tailscale.sipeed.invalidIdentity': '远程网络身份已失效，请重试以创建新身份。',
    'settings.tailscale.sipeed.rateLimited': '请求过于频繁，请在 {{seconds}} 秒后重试。',
    'settings.tailscale.sipeed.retry': '重试',
    'settings.tailscale.sipeed.localStatus': '本机连接',
    'settings.tailscale.sipeed.connected': '已连接',
    'settings.tailscale.sipeed.copyIP': '复制设备地址',
    'settings.tailscale.sipeed.ipCopied': '设备地址已复制',
    'settings.tailscale.sipeed.devices': '设备',
    'settings.tailscale.sipeed.emptyDevices': '尚未添加其他设备。',
    'settings.tailscale.sipeed.addDevice': '添加设备',
    'settings.tailscale.sipeed.addDeviceDesc':
      '该设备尚未添加。点击“添加设备”，在要加入的电脑或手机上完成接入。',
    'settings.tailscale.sipeed.claimExpiry': '剩余 {{time}} · 仅可使用一次',
    'settings.tailscale.sipeed.claimExpired': '认领码已失效',
    'settings.tailscale.sipeed.claimCopy': '复制',
    'settings.tailscale.sipeed.claimCopied': '已复制',
    'settings.tailscale.sipeed.claimRenew': '重新获取',
    'settings.tailscale.sipeed.claimHint':
      '在要添加的设备上安装并打开 Tailscale 客户端，选择“自定义服务器”并填入 {{server}}。浏览器会弹出接入页，把这串数字输进去。'
  },
  zh_tw: {
    'tailscale.continue': '繼續',
    'tailscale.start': '啟用',
    'tailscale.stop': '停用',
    'tailscale.restart': '重新啟動',
    'tailscale.disconnect': '此操作將中斷 Tailscale 連線，並關閉開機自動啟動。',
    'tailscale.confirm': '確認',
    'tailscale.cancel': '取消',
    'tailscale.stopped': '已停用',
    'tailscale.connecting': '正在連線',
    'tailscale.error': '連線異常',
    'tailscale.connected': '已連線',
    'tailscale.offline': '離線',
    'tailscale.notLogin': '未登入',
    ...sipeedManagedFallbackMessages,
    ...sipeedRuntimeTranslations.zh_tw,
    ...sipeedManagementTranslations.zh_tw,
    'settings.tailscale.title': 'Tailscale',
    'settings.tailscale.restart': '確定要重啟 Tailscale 嗎？',
    'settings.tailscale.stop': '確定要停止 Tailscale 嗎？',
    'settings.tailscale.stopDesc': '此操作將會登出帳號，並停止開機自動啟動。',
    'settings.tailscale.loading': '載入中...',
    'settings.tailscale.notInstall': '未找到 Tailscale ！請先安裝。',
    'settings.tailscale.install': '安裝',
    'settings.tailscale.installing': '安裝中',
    'settings.tailscale.failed': '安裝失敗',
    'settings.tailscale.retry': '請重新整理並重試。或嘗試手動安裝',
    'settings.tailscale.download': '下載',
    'settings.tailscale.package': '安裝包',
    'settings.tailscale.unzip': '並解壓縮它',
    'settings.tailscale.upTailscale': '將 Tailscale 上傳到 NanoKVM 目錄 /usr/bin/',
    'settings.tailscale.upTailscaled': '將 tailscaled 上傳到 NanoKVM 目錄 /usr/sbin/',
    'settings.tailscale.refresh': '重新整理頁面',
    'settings.tailscale.notRunning': 'Tailscale 尚未執行',
    'settings.tailscale.run': '啟動',
    'settings.tailscale.notLogin': '設備尚未綁定。請登入並將該裝置綁定到您的帳戶。',
    'settings.tailscale.urlPeriod': '此網址有效期限為 10 分鐘',
    'settings.tailscale.login': '登入',
    'settings.tailscale.loginSuccess': '登入成功',
    'settings.tailscale.enable': '啟用 Tailscale',
    'settings.tailscale.exitNode': '出口節點',
    'settings.tailscale.exitNodeDesc': '將此裝置發佈為 tailnet 的網際網路出口。',
    'settings.tailscale.subnetRoute': '子網路由',
    'settings.tailscale.subnetRouteDesc': '向 tailnet 發佈本機直接連線的區域網路網段 {{subnet}}。',
    'settings.tailscale.subnetUnavailable': '未偵測到可發佈的直接連線區域網路網段。',
    'settings.tailscale.routingUnavailable': '此裝置目前無法設定 Tailscale 路由。',
    'settings.tailscale.pendingApproval': '啟用後，請在 Tailscale 管理後台核准發佈的路由。',
    'settings.tailscale.routingUpdateFailed': '更新 Tailscale 路由設定失敗。',
    'settings.tailscale.deviceName': '裝置名稱',
    'settings.tailscale.deviceIP': '裝置 IP',
    'settings.tailscale.account': '帳號',
    'settings.tailscale.logout': '登出',
    'settings.tailscale.logoutDesc': '確認要登出嗎？',
    'settings.tailscale.logoutFailed': '登出失敗。',
    'settings.tailscale.uninstall': '移除 Tailscale',
    'settings.tailscale.uninstallDesc': '確定要解除安裝 Tailscale 嗎？',
    'settings.tailscale.okBtn': '確認',
    'settings.tailscale.cancelBtn': '取消',
    'settings.tailscale.mode.official': '官方 Tailscale',
    'settings.tailscale.mode.officialDesc': '使用 Tailscale 官方服務。',
    'settings.tailscale.mode.sipeed': 'Sipeed 遠端網路',
    'settings.tailscale.mode.sipeedDesc': '裝置之間點對點直連，不提供中繼。',
    'settings.tailscale.sipeed.disclaimerTitle': '免責聲明',
    'settings.tailscale.sipeed.enableDesc':
      '啟用 Sipeed 遠端網路後，本裝置會向 Sipeed 伺服器傳送請求，並提供裝置識別碼、裝置名稱、虛擬網路 IP 位址等連線資訊，用於裝置註冊、身分驗證、連線協調和網路管理。\n\n遠端存取依賴網路環境、裝置設定及相關服務，可能出現延遲、中斷或暫時無法使用。請自行評估使用風險，並保留必要的備用存取方式。\n\n請僅在您擁有或已獲授權的裝置與網路上使用本服務，並妥善保管登入憑證和配對碼，謹慎授予他人存取權限。\n\n點擊「同意並啟用」即表示您已閱讀並理解本聲明，同意為提供遠端連線而進行上述通訊和資訊處理。同意狀態將儲存在本裝置上，後續啟用無需重複確認。\n\n您可在已連線頁面右上角選單中撤回同意。撤回後會退出 Sipeed 遠端網路，並請求刪除對應的雲端網路及使用者；刪除後，該網路中的其他裝置也將無法繼續透過此網路存取。再次啟用時需重新同意。',
    'settings.tailscale.sipeed.acceptAndEnable': '同意並啟用',
    'settings.tailscale.sipeed.revokeDisclaimer': '撤回免責聲明同意',
    'settings.tailscale.sipeed.revokeDisclaimerDesc':
      '撤回同意將退出 Sipeed 遠端網路、刪除雲端網路及使用者，並清除已儲存的同意狀態。此網路中的所有裝置將中斷連線，是否繼續？',
    'settings.tailscale.sipeed.revokeDisclaimerFailed': '撤回未完成，請重試。',
    'settings.tailscale.sipeed.addDevice': '新增裝置',
    'settings.tailscale.sipeed.addDeviceDesc':
      '尚未新增任何裝置。點選「新增裝置」，在要加入的電腦或手機上完成接入。',
    'settings.tailscale.sipeed.claimExpiry': '剩餘 {{time}} · 僅可使用一次',
    'settings.tailscale.sipeed.claimExpired': '認領碼已失效',
    'settings.tailscale.sipeed.claimCopy': '複製',
    'settings.tailscale.sipeed.claimCopied': '已複製',
    'settings.tailscale.sipeed.claimRenew': '重新取得',
    'settings.tailscale.sipeed.claimHint':
      '在要新增的裝置上安裝並開啟 Tailscale 用戶端，選擇「自訂伺服器」並填入 {{server}}。瀏覽器會彈出接入頁，把這串數字輸入進去。'
  },
  hu: {
    'tailscale.continue': 'Folytatás',
    'tailscale.start': 'Engedélyezés',
    'tailscale.stop': 'Letiltás',
    'tailscale.restart': 'Újraindítás',
    'tailscale.disconnect':
      'Ez megszakítja a Tailscale kapcsolatát, és letiltja az automatikus indítást rendszerindításkor.',
    'tailscale.confirm': 'Megerősítés',
    'tailscale.cancel': 'Mégse',
    'tailscale.stopped': 'Letiltva',
    'tailscale.connecting': 'Csatlakozás',
    'tailscale.error': 'Kapcsolati hiba',
    'tailscale.connected': 'Csatlakoztatva',
    'tailscale.offline': 'Nem elérhető',
    'tailscale.notLogin': 'Nincs bejelentkezve',
    ...sipeedManagedFallbackMessages,
    ...sipeedRuntimeTranslations.hu,
    ...sipeedManagementTranslations.hu,
    'settings.tailscale.title': 'Tailscale',
    'settings.tailscale.restart': 'Újraindítja a Tailscale-t?',
    'settings.tailscale.stop': 'Leállítja a Tailscale-t?',
    'settings.tailscale.stopDesc':
      'Kijelentkezés a Tailscale-ből, és az automatikus indítás letiltása rendszerindításkor.',
    'settings.tailscale.loading': 'Betöltés...',
    'settings.tailscale.notInstall': 'Tailscale nem található! Kérem, telepítse.',
    'settings.tailscale.install': 'Telepítés',
    'settings.tailscale.installing': 'Telepítés folyamatban',
    'settings.tailscale.failed': 'Telepítés sikertelen',
    'settings.tailscale.retry':
      'Frissítse az oldalt, majd próbálja újra. Vagy próbálja meg manuálisan telepíteni.',
    'settings.tailscale.download': 'Letöltés a',
    'settings.tailscale.package': 'telepítési csomag',
    'settings.tailscale.unzip': 'és kicsomagolás',
    'settings.tailscale.upTailscale': 'Töltsön fel tailscale-t a NanoKVM /usr/bin/ könyvtárába',
    'settings.tailscale.upTailscaled': 'Töltsön fel tailscaled-t a NanoKVM /usr/sbin/ könyvtárába',
    'settings.tailscale.refresh': 'Frissítse az aktuális oldalt',
    'settings.tailscale.notRunning': 'Tailscale nem fut. Kérjük, indítsa el a folytatáshoz.',
    'settings.tailscale.run': 'Indítás',
    'settings.tailscale.notLogin':
      'Az eszköz még nincs kötve. Kérem, jelentkezzen be és kösse az eszközt a fiókjához.',
    'settings.tailscale.urlPeriod': 'Ez az url 10 percig érvényes',
    'settings.tailscale.login': 'Bejelentkezés',
    'settings.tailscale.loginSuccess': 'Sikeres bejelentkezés',
    'settings.tailscale.enable': 'Tailscale engedélyezése',
    'settings.tailscale.exitNode': 'Kilépési csomópont',
    'settings.tailscale.exitNodeDesc':
      'Az eszköz hirdetése internetes kijáratként a tailnet számára.',
    'settings.tailscale.subnetRoute': 'Alhálózati útvonal',
    'settings.tailscale.subnetRouteDesc':
      'A közvetlenül csatlakozó {{subnet}} LAN-alhálózat hirdetése a tailneten.',
    'settings.tailscale.subnetUnavailable': 'Nem érhető el közvetlenül csatlakozó LAN-alhálózat.',
    'settings.tailscale.routingUnavailable':
      'A Tailscale útválasztási beállításai nem érhetők el ezen az eszközön.',
    'settings.tailscale.pendingApproval':
      'Engedélyezés után hagyja jóvá a hirdetett útvonalat a Tailscale felügyeleti konzolján.',
    'settings.tailscale.routingUpdateFailed':
      'Nem sikerült frissíteni a Tailscale útválasztási beállításait.',
    'settings.tailscale.deviceName': 'Eszköz neve',
    'settings.tailscale.deviceIP': 'Eszköz IP',
    'settings.tailscale.account': 'Fiók',
    'settings.tailscale.logout': 'Kijelentkezés',
    'settings.tailscale.logoutDesc': 'Biztos, hogy ki szeretne jelentkezni?',
    'settings.tailscale.logoutFailed': 'Sikertelen kijelentkezés.',
    'settings.tailscale.uninstall': 'Eltávolítás Tailscale',
    'settings.tailscale.uninstallDesc': 'Biztosan eltávolítja a Tailscale alkalmazást?',
    'settings.tailscale.okBtn': 'Igen',
    'settings.tailscale.cancelBtn': 'Nem',
    'settings.tailscale.mode.official': 'Hivatalos Tailscale',
    'settings.tailscale.mode.officialDesc': 'A hivatalos Tailscale-szolgáltatás használata.',
    'settings.tailscale.mode.sipeed': 'Sipeed távoli hálózat',
    'settings.tailscale.mode.sipeedDesc': 'Közvetlen eszközök közötti kapcsolat; relé nélkül.',
    'settings.tailscale.sipeed.disclaimerTitle': 'Felelősségkizárás',
    'settings.tailscale.sipeed.enableDesc':
      'A Sipeed távoli hálózat engedélyezése kéréseket küld a Sipeed szervereinek olyan kapcsolati adatokkal, mint az eszközazonosító, az eszköz neve és a virtuális hálózati IP-cím. Ezek az adatok az eszközregisztrációt, a hitelesítést, a kapcsolatok összehangolását és a hálózat kezelését szolgálják.\n\nA távoli hozzáférés a hálózati körülményektől, az eszköz beállításaitól és a szolgáltatások elérhetőségétől függ. Késések, megszakadások vagy átmeneti elérhetetlenség fordulhat elő. Mérlegelje a használat kockázatait, és szükség esetén tartson fenn más hozzáférési lehetőséget.\n\nA szolgáltatást csak saját vagy engedéllyel elérhető eszközökön és hálózatokon használja. Óvja bejelentkezési adatait és párosítási kódjait, és körültekintően adjon hozzáférést másoknak.\n\nAz „Elfogadás és indítás” gombra kattintva megerősíti, hogy elolvasta és megértette ezt a nyilatkozatot, és hozzájárul a távoli kapcsolatok biztosításához leírt kommunikációhoz és adatkezeléshez. A hozzájárulás ezen az eszközön tárolódik, így nem kell minden engedélyezéskor újra megerősíteni.\n\nA hozzájárulást az aktív kapcsolat oldalának jobb felső menüjében vonhatja vissza. Ekkor kilép a Sipeed távoli hálózatból, és a rendszer kéri a kapcsolódó felhőhálózat és felhasználó törlését. A törlés után más eszközök sem használhatják ezt a hálózatot távoli hozzáférésre. Az újbóli engedélyezéshez új hozzájárulás szükséges.',
    'settings.tailscale.sipeed.acceptAndEnable': 'Elfogadás és indítás',
    'settings.tailscale.sipeed.revokeDisclaimer': 'Hozzájárulás visszavonása',
    'settings.tailscale.sipeed.revokeDisclaimerDesc':
      'Ezzel kilép a Sipeed távoli hálózatból, törli a felhőhálózatot és annak felhasználóját, valamint a mentett hozzájárulást. A hálózat összes eszköze lecsatlakozik. Folytatja?',
    'settings.tailscale.sipeed.revokeDisclaimerFailed':
      'A visszavonás nem fejeződött be. Próbálja újra.',
    'settings.tailscale.sipeed.addDevice': 'Eszköz hozzáadása',
    'settings.tailscale.sipeed.addDeviceDesc':
      'Még nincs hozzáadott eszköz. Válassza az „Eszköz hozzáadása” lehetőséget, és fejezze be a csatlakozást a számítógépen vagy telefonon.',
    'settings.tailscale.sipeed.claimExpiry': 'Hátralévő idő: {{time}} · egyszeri használat',
    'settings.tailscale.sipeed.claimExpired': 'A kód lejárt',
    'settings.tailscale.sipeed.claimCopy': 'Kód másolása',
    'settings.tailscale.sipeed.claimCopied': 'Másolva',
    'settings.tailscale.sipeed.claimRenew': 'Új kód kérése',
    'settings.tailscale.sipeed.claimHint':
      'Telepítsd és nyisd meg a Tailscale klienst a hozzáadni kívánt eszközön, válaszd az egyéni szervert, és add meg a {{server}} címet. A böngésző megnyitja a csatlakozási oldalt, ahol beírod ezeket a számjegyeket.'
  },
  vi: {
    'tailscale.continue': 'Tiếp tục',
    'tailscale.start': 'Bật',
    'tailscale.stop': 'Tắt',
    'tailscale.restart': 'Khởi động lại',
    'tailscale.disconnect':
      'Thao tác này sẽ ngắt kết nối Tailscale và tắt tự động khởi động khi bật máy.',
    'tailscale.confirm': 'Xác nhận',
    'tailscale.cancel': 'Hủy',
    'tailscale.stopped': 'Đã tắt',
    'tailscale.connecting': 'Đang kết nối',
    'tailscale.error': 'Lỗi kết nối',
    'tailscale.connected': 'Đã kết nối',
    'tailscale.offline': 'Ngoại tuyến',
    'tailscale.notLogin': 'Chưa đăng nhập',
    ...sipeedManagedFallbackMessages,
    ...sipeedRuntimeTranslations.vi,
    ...sipeedManagementTranslations.vi,
    'settings.tailscale.title': 'Tailscale',
    'settings.tailscale.restart': 'Khởi động lại Tailscale?',
    'settings.tailscale.stop': 'Dừng Tailscale?',
    'settings.tailscale.stopDesc': 'Đăng xuất khỏi Tailscale và tắt tự động khởi động khi boot.',
    'settings.tailscale.loading': 'Đang tải...',
    'settings.tailscale.notInstall': 'Không tìm thấy Tailscale! Vui lòng cài đặt.',
    'settings.tailscale.install': 'Cài đặt',
    'settings.tailscale.installing': 'Đang cài đặt',
    'settings.tailscale.failed': 'Cài đặt thất bại',
    'settings.tailscale.retry': 'Vui lòng làm mới và thử lại. Hoặc thử cài đặt thủ công',
    'settings.tailscale.download': 'Tải xuống',
    'settings.tailscale.package': 'gói cài đặt',
    'settings.tailscale.unzip': 'và giải nén nó',
    'settings.tailscale.upTailscale': 'Tải tailscale lên thư mục /usr/bin/ của NanoKVM',
    'settings.tailscale.upTailscaled': 'Tải tailscaled lên thư mục /usr/sbin/ của NanoKVM',
    'settings.tailscale.refresh': 'Làm mới trang hiện tại',
    'settings.tailscale.notRunning': 'Tailscale không chạy. Hãy bắt đầu nó để tiếp tục.',
    'settings.tailscale.run': 'Khởi động',
    'settings.tailscale.notLogin':
      'Thiết bị chưa được liên kết. Vui lòng đăng nhập và liên kết thiết bị này với tài khoản của bạn.',
    'settings.tailscale.urlPeriod': 'URL này có hiệu lực trong 10 phút',
    'settings.tailscale.login': 'Đăng nhập',
    'settings.tailscale.loginSuccess': 'Đăng nhập thành công',
    'settings.tailscale.enable': 'Kích hoạt Tailscale',
    'settings.tailscale.exitNode': 'Nút thoát',
    'settings.tailscale.exitNodeDesc':
      'Quảng bá thiết bị này làm lối ra Internet cho tailnet của bạn.',
    'settings.tailscale.subnetRoute': 'Tuyến mạng con',
    'settings.tailscale.subnetRouteDesc':
      'Quảng bá mạng LAN kết nối trực tiếp {{subnet}} vào tailnet của bạn.',
    'settings.tailscale.subnetUnavailable': 'Không có mạng LAN kết nối trực tiếp nào khả dụng.',
    'settings.tailscale.routingUnavailable':
      'Thiết bị này không hỗ trợ cài đặt định tuyến Tailscale.',
    'settings.tailscale.pendingApproval':
      'Sau khi bật, hãy phê duyệt tuyến được quảng bá trong bảng điều khiển quản trị Tailscale.',
    'settings.tailscale.routingUpdateFailed': 'Không thể cập nhật cài đặt định tuyến Tailscale.',
    'settings.tailscale.deviceName': 'Tên Thiết bị',
    'settings.tailscale.deviceIP': 'IP Thiết bị',
    'settings.tailscale.account': 'Tài khoản',
    'settings.tailscale.logout': 'Đăng xuất',
    'settings.tailscale.logoutDesc': 'Bạn có chắc chắn muốn đăng xuất không?',
    'settings.tailscale.logoutFailed': 'Đăng xuất không thành công.',
    'settings.tailscale.uninstall': 'Gỡ cài đặt Tailscale',
    'settings.tailscale.uninstallDesc': 'Bạn có chắc chắn muốn gỡ cài đặt Tailscale không?',
    'settings.tailscale.okBtn': 'Có',
    'settings.tailscale.cancelBtn': 'Không',
    'settings.tailscale.mode.official': 'Tailscale chính thức',
    'settings.tailscale.mode.officialDesc': 'Dùng dịch vụ Tailscale chính thức.',
    'settings.tailscale.mode.sipeed': 'Mạng từ xa Sipeed',
    'settings.tailscale.mode.sipeedDesc': 'Kết nối trực tiếp giữa các thiết bị; không chuyển tiếp.',
    'settings.tailscale.sipeed.disclaimerTitle': 'Tuyên bố miễn trừ trách nhiệm',
    'settings.tailscale.sipeed.enableDesc':
      'Bật mạng từ xa Sipeed sẽ gửi yêu cầu đến máy chủ Sipeed cùng thông tin kết nối, như mã định danh thiết bị, tên thiết bị và địa chỉ IP mạng ảo. Thông tin này được dùng để đăng ký thiết bị, xác thực, điều phối kết nối và quản lý mạng.\n\nTruy cập từ xa phụ thuộc vào điều kiện mạng, cấu hình thiết bị và tình trạng hoạt động của dịch vụ. Kết nối có thể bị chậm, gián đoạn hoặc tạm thời không khả dụng. Hãy cân nhắc rủi ro và duy trì phương thức truy cập thay thế khi cần.\n\nChỉ sử dụng dịch vụ với thiết bị và mạng thuộc sở hữu của bạn hoặc được phép truy cập. Bảo vệ thông tin đăng nhập và mã ghép nối, đồng thời cân nhắc kỹ khi cấp quyền truy cập cho người khác.\n\nNhấp vào “Đồng ý và khởi động” có nghĩa là bạn đã đọc, hiểu thông báo này và đồng ý với việc liên lạc và xử lý thông tin nêu trên để cung cấp kết nối từ xa. Sự đồng ý được lưu trên thiết bị này nên không cần xác nhận lại mỗi lần bật dịch vụ.\n\nBạn có thể rút lại sự đồng ý trong menu phía trên bên phải của trang kết nối đang hoạt động. Thao tác này sẽ thoát mạng từ xa Sipeed và yêu cầu xóa mạng đám mây liên quan cùng người dùng của mạng. Sau khi xóa, các thiết bị khác cũng không thể sử dụng mạng này để truy cập từ xa. Khi bật lại, bạn cần đồng ý lần nữa.',
    'settings.tailscale.sipeed.acceptAndEnable': 'Đồng ý và khởi động',
    'settings.tailscale.sipeed.revokeDisclaimer': 'Rút lại sự đồng ý',
    'settings.tailscale.sipeed.revokeDisclaimerDesc':
      'Thao tác này sẽ thoát mạng từ xa Sipeed, xóa mạng đám mây cùng người dùng của mạng và xóa trạng thái đồng ý đã lưu. Tất cả thiết bị trong mạng này sẽ bị ngắt kết nối. Tiếp tục?',
    'settings.tailscale.sipeed.revokeDisclaimerFailed':
      'Chưa thể hoàn tất việc rút lại sự đồng ý. Vui lòng thử lại.',
    'settings.tailscale.sipeed.addDevice': 'Thêm thiết bị',
    'settings.tailscale.sipeed.addDeviceDesc':
      'Chưa có thiết bị nào được thêm. Chọn “Thêm thiết bị” và hoàn tất kết nối trên máy tính hoặc điện thoại bạn muốn kết nối.',
    'settings.tailscale.sipeed.claimExpiry': 'Còn {{time}} · dùng một lần',
    'settings.tailscale.sipeed.claimExpired': 'Mã đã hết hạn',
    'settings.tailscale.sipeed.claimCopy': 'Sao chép mã',
    'settings.tailscale.sipeed.claimCopied': 'Đã sao chép',
    'settings.tailscale.sipeed.claimRenew': 'Lấy mã mới',
    'settings.tailscale.sipeed.claimHint':
      'Cài đặt và mở ứng dụng Tailscale trên thiết bị bạn muốn thêm, chọn máy chủ tùy chỉnh và nhập {{server}}. Trình duyệt sẽ mở trang tham gia, hãy nhập các chữ số này tại đó.'
  },
  ja: {
    'tailscale.continue': '続行',
    'tailscale.start': '有効にする',
    'tailscale.stop': '無効にする',
    'tailscale.restart': '再起動',
    'tailscale.disconnect': 'Tailscale の接続を切断し、起動時の自動開始を無効にします。',
    'tailscale.confirm': '確認',
    'tailscale.cancel': 'キャンセル',
    'tailscale.stopped': '無効',
    'tailscale.connecting': '接続中',
    'tailscale.error': '接続エラー',
    'tailscale.connected': '接続済み',
    'tailscale.offline': 'オフライン',
    'tailscale.notLogin': '未ログイン',
    ...sipeedManagedFallbackMessages,
    ...sipeedRuntimeTranslations.ja,
    ...sipeedManagementTranslations.ja,
    'settings.tailscale.title': 'Tailscale',
    'settings.tailscale.restart': 'Tailscale を再起動しますか？',
    'settings.tailscale.stop': 'Tailscale を停止しますか？',
    'settings.tailscale.stopDesc': 'Tailscale からログアウトし、起動時の自動開始を無効にします。',
    'settings.tailscale.loading': '読み込み中...',
    'settings.tailscale.notInstall': 'Tailscaleが見つかりません！インストールしてください。',
    'settings.tailscale.install': 'インストール',
    'settings.tailscale.installing': 'インストール中',
    'settings.tailscale.failed': 'インストールに失敗しました',
    'settings.tailscale.retry':
      'ページをリフレッシュして再試行してください。または手動でインストールしてください',
    'settings.tailscale.download': 'インストールパッケージをダウンロードして',
    'settings.tailscale.package': '解凍してください',
    'settings.tailscale.unzip': '解凍してください',
    'settings.tailscale.upTailscale':
      'tailscaleをNanoKVMのディレクトリ/usr/bin/にアップロードしてください',
    'settings.tailscale.upTailscaled':
      'tailscaledをNanoKVMのディレクトリ/usr/sbin/にアップロードしてください',
    'settings.tailscale.refresh': '現在のページをリフレッシュします',
    'settings.tailscale.notRunning':
      'Tailscale はまだ実行されていません。起動操作を実行してください',
    'settings.tailscale.run': '開始',
    'settings.tailscale.notLogin':
      'デバイスはまだバインドされていません。ログインしてこのデバイスをアカウントにバインドしてください。',
    'settings.tailscale.urlPeriod': 'このURLは10分間有効です',
    'settings.tailscale.login': 'ログイン',
    'settings.tailscale.loginSuccess': 'ログイン成功',
    'settings.tailscale.enable': 'Tailscaleを有効化',
    'settings.tailscale.exitNode': '出口ノード',
    'settings.tailscale.exitNodeDesc':
      'このデバイスを tailnet のインターネット出口として公開します。',
    'settings.tailscale.subnetRoute': 'サブネットルート',
    'settings.tailscale.subnetRouteDesc':
      '直接接続された LAN サブネット {{subnet}} を tailnet に公開します。',
    'settings.tailscale.subnetUnavailable': '公開できる直接接続 LAN サブネットがありません。',
    'settings.tailscale.routingUnavailable':
      'このデバイスでは Tailscale のルーティング設定を利用できません。',
    'settings.tailscale.pendingApproval':
      '有効化後、Tailscale 管理コンソールで公開されたルートを承認してください。',
    'settings.tailscale.routingUpdateFailed':
      'Tailscale のルーティング設定を更新できませんでした。',
    'settings.tailscale.deviceName': 'デバイス名',
    'settings.tailscale.deviceIP': 'デバイスIP',
    'settings.tailscale.account': 'アカウント',
    'settings.tailscale.logout': 'ログアウト',
    'settings.tailscale.logoutDesc': 'ログアウトしてもよろしいですか？',
    'settings.tailscale.logoutFailed': 'ログアウトに失敗しました。',
    'settings.tailscale.uninstall': 'Tailscale をアンインストール',
    'settings.tailscale.uninstallDesc': 'Tailscale をアンインストールしてもよろしいですか？',
    'settings.tailscale.okBtn': 'はい',
    'settings.tailscale.cancelBtn': 'いいえ',
    'settings.tailscale.mode.official': '公式 Tailscale',
    'settings.tailscale.mode.officialDesc': 'Tailscale 公式サービスを使用します。',
    'settings.tailscale.mode.sipeed': 'Sipeed リモートネットワーク',
    'settings.tailscale.mode.sipeedDesc': 'デバイス間を P2P で直接接続します。中継は行いません。',
    'settings.tailscale.sipeed.disclaimerTitle': '免責事項',
    'settings.tailscale.sipeed.enableDesc':
      'Sipeed リモートネットワークを有効にすると、デバイス識別子、デバイス名、仮想ネットワークの IP アドレスなどの接続情報を含むリクエストが Sipeed サーバーへ送信されます。これらの情報は、デバイス登録、認証、接続の調整、ネットワーク管理に使用されます。\n\nリモートアクセスはネットワーク環境、デバイス設定、サービスの稼働状況に依存し、遅延、中断、一時的な利用不能が発生する場合があります。利用上のリスクを確認し、必要に応じて代替のアクセス手段を確保してください。\n\nご自身が所有するか、アクセス権限を持つデバイスとネットワークでのみご利用ください。ログイン認証情報とペアリングコードを適切に管理し、他者へのアクセス許可は慎重に行ってください。\n\n「同意して開始」をクリックすると、本事項を読み理解したうえで、リモート接続の提供に必要な上記の通信と情報処理に同意したものとみなされます。同意状態はこのデバイスに保存されるため、有効にするたびに再確認する必要はありません。\n\n接続済み画面の右上メニューから同意を撤回できます。撤回すると Sipeed リモートネットワークからログアウトし、関連するクラウドネットワークとユーザーの削除を要求します。削除後は、他のデバイスもこのネットワークを使ったリモートアクセスができなくなります。再度有効にする場合は改めて同意が必要です。',
    'settings.tailscale.sipeed.acceptAndEnable': '同意して開始',
    'settings.tailscale.sipeed.revokeDisclaimer': '免責事項への同意を撤回',
    'settings.tailscale.sipeed.revokeDisclaimerDesc':
      '同意を撤回すると、Sipeed リモートネットワークからログアウトし、クラウドネットワークとそのユーザー、および保存された同意状態を削除します。このネットワーク内のすべてのデバイスが切断されます。続行しますか？',
    'settings.tailscale.sipeed.revokeDisclaimerFailed':
      '同意の撤回を完了できませんでした。もう一度お試しください。',
    'settings.tailscale.sipeed.addDevice': 'デバイスを追加',
    'settings.tailscale.sipeed.addDeviceDesc':
      'まだデバイスが追加されていません。「デバイスを追加」を選び、追加するパソコンまたはスマートフォンで接続を完了してください。',
    'settings.tailscale.sipeed.claimExpiry': '残り {{time}} · 1 回のみ有効',
    'settings.tailscale.sipeed.claimExpired': '認証コードは無効になりました',
    'settings.tailscale.sipeed.claimCopy': '認証コードをコピー',
    'settings.tailscale.sipeed.claimCopied': 'コピーしました',
    'settings.tailscale.sipeed.claimRenew': 'コードを再取得',
    'settings.tailscale.sipeed.claimHint':
      '追加するデバイスに Tailscale クライアントをインストールして開き、「カスタムサーバー」に {{server}} を入力します。ブラウザーで接続ページが開くので、この数字を入力してください。'
  },
  cz: {
    'tailscale.continue': 'Pokračovat',
    'tailscale.start': 'Povolit',
    'tailscale.stop': 'Zakázat',
    'tailscale.restart': 'Restartovat',
    'tailscale.disconnect':
      'Tím se odpojí Tailscale a zakáže automatické spouštění při startu systému.',
    'tailscale.confirm': 'Potvrdit',
    'tailscale.cancel': 'Zrušit',
    'tailscale.stopped': 'Zakázáno',
    'tailscale.connecting': 'Připojování',
    'tailscale.error': 'Chyba připojení',
    'tailscale.connected': 'Připojeno',
    'tailscale.offline': 'Offline',
    'tailscale.notLogin': 'Nepřihlášeno',
    ...sipeedManagedFallbackMessages,
    ...sipeedRuntimeTranslations.cz,
    ...sipeedManagementTranslations.cz,
    'settings.tailscale.title': 'Tailscale',
    'settings.tailscale.restart': 'Restartovat Tailscale?',
    'settings.tailscale.stop': 'Zastavit Tailscale?',
    'settings.tailscale.stopDesc': 'Odhlásit Tailscale a zakázat automatické spouštění při startu.',
    'settings.tailscale.loading': 'Načítání...',
    'settings.tailscale.notInstall': 'Tailscale nebyl nalezen! Prosím nainstalujte.',
    'settings.tailscale.install': 'Nainstalovat',
    'settings.tailscale.installing': 'Instalace probíhá',
    'settings.tailscale.failed': 'Instalace se nezdařila',
    'settings.tailscale.retry': 'Obnovte stránku a zkuste to znovu. Nebo zkuste instalaci manuálně',
    'settings.tailscale.download': 'Stáhnout',
    'settings.tailscale.package': 'instalační balíček',
    'settings.tailscale.unzip': 'a rozbalit ho',
    'settings.tailscale.upTailscale': 'Nahrajte Tailscale do adresáře NanoKVM /usr/bin/',
    'settings.tailscale.upTailscaled': 'Nahrajte Tailscaled do adresáře NanoKVM /usr/sbin/',
    'settings.tailscale.refresh': 'Obnovit stránku',
    'settings.tailscale.notRunning': 'Tailscale neběží. Chcete-li pokračovat, spusťte jej.',
    'settings.tailscale.run': 'Spustit',
    'settings.tailscale.notLogin':
      'Zařízení nebylo dosud spárováno. Přihlaste se prosím a spárujte toto zařízení s vaším účtem.',
    'settings.tailscale.urlPeriod': 'Tento odkaz je platný po dobu 10 minut',
    'settings.tailscale.login': 'Přihlášení',
    'settings.tailscale.loginSuccess': 'Přihlášení úspěšné',
    'settings.tailscale.enable': 'Povolit Tailscale',
    'settings.tailscale.exitNode': 'Výstupní uzel',
    'settings.tailscale.exitNodeDesc':
      'Inzerovat toto zařízení jako internetový výstup pro váš tailnet.',
    'settings.tailscale.subnetRoute': 'Trasa podsítě',
    'settings.tailscale.subnetRouteDesc':
      'Inzerovat přímo připojenou síť LAN {{subnet}} ve vašem tailnetu.',
    'settings.tailscale.subnetUnavailable': 'Není k dispozici žádná přímo připojená síť LAN.',
    'settings.tailscale.routingUnavailable':
      'Nastavení směrování Tailscale nejsou na tomto zařízení dostupná.',
    'settings.tailscale.pendingApproval':
      'Po povolení schvalte inzerovanou trasu v konzoli pro správu Tailscale.',
    'settings.tailscale.routingUpdateFailed':
      'Nastavení směrování Tailscale se nepodařilo aktualizovat.',
    'settings.tailscale.deviceName': 'Název zařízení',
    'settings.tailscale.deviceIP': 'IP zařízení',
    'settings.tailscale.account': 'Účet',
    'settings.tailscale.logout': 'Odhlásit se',
    'settings.tailscale.logoutDesc': 'Opravdu se chcete odhlásit?',
    'settings.tailscale.logoutFailed': 'Odhlášení se nezdařilo.',
    'settings.tailscale.uninstall': 'Odinstalovat Tailscale',
    'settings.tailscale.uninstallDesc': 'Opravdu chcete odinstalovat Tailscale?',
    'settings.tailscale.okBtn': 'Ano',
    'settings.tailscale.cancelBtn': 'Ne',
    'settings.tailscale.mode.official': 'Oficiální Tailscale',
    'settings.tailscale.mode.officialDesc': 'Použít oficiální službu Tailscale.',
    'settings.tailscale.mode.sipeed': 'Vzdálená síť Sipeed',
    'settings.tailscale.mode.sipeedDesc': 'Přímá spojení mezi zařízeními; bez přenosu přes server.',
    'settings.tailscale.sipeed.disclaimerTitle': 'Prohlášení o vyloučení odpovědnosti',
    'settings.tailscale.sipeed.enableDesc':
      'Zapnutí vzdálené sítě Sipeed odesílá požadavky na servery Sipeed spolu s údaji o připojení, například identifikátorem zařízení, jeho názvem a IP adresou virtuální sítě. Údaje se používají k registraci zařízení, ověřování, koordinaci připojení a správě sítě.\n\nVzdálený přístup závisí na stavu sítě, nastavení zařízení a dostupnosti služeb. Může docházet ke zpoždění, přerušení nebo dočasné nedostupnosti. Zvažte rizika používání a v případě potřeby si ponechte alternativní způsob přístupu.\n\nSlužbu používejte pouze se zařízeními a sítěmi, které vlastníte nebo k nim máte oprávnění přistupovat. Chraňte přihlašovací údaje a párovací kódy a přístup ostatním udělujte uvážlivě.\n\nKliknutím na „Souhlasit a spustit“ potvrzujete, že jste si toto prohlášení přečetli a porozuměli mu a souhlasíte s popsanou komunikací a zpracováním údajů pro zajištění vzdálených připojení. Souhlas se uloží na tomto zařízení, takže jej nemusíte potvrzovat při každém zapnutí.\n\nSouhlas můžete odvolat v nabídce vpravo nahoře na stránce aktivního připojení. Tím opustíte vzdálenou síť Sipeed a odešlete žádost o odstranění související cloudové sítě a jejího uživatele. Po odstranění nebudou moci tuto síť pro vzdálený přístup používat ani ostatní zařízení. Opětovné zapnutí vyžaduje nový souhlas.',
    'settings.tailscale.sipeed.acceptAndEnable': 'Souhlasit a spustit',
    'settings.tailscale.sipeed.revokeDisclaimer': 'Odvolat souhlas',
    'settings.tailscale.sipeed.revokeDisclaimerDesc':
      'Tímto opustíte vzdálenou síť Sipeed, odstraníte cloudovou síť i jejího uživatele a vymažete uložený souhlas. Všechna zařízení v této síti budou odpojena. Pokračovat?',
    'settings.tailscale.sipeed.revokeDisclaimerFailed':
      'Odvolání souhlasu se nepodařilo dokončit. Zkuste to znovu.',
    'settings.tailscale.sipeed.addDevice': 'Přidat zařízení',
    'settings.tailscale.sipeed.addDeviceDesc':
      'Zatím není přidáno žádné zařízení. Zvolte „Přidat zařízení“ a dokončete připojení na počítači nebo telefonu.',
    'settings.tailscale.sipeed.claimExpiry': 'Zbývá {{time}} · jednorázově',
    'settings.tailscale.sipeed.claimExpired': 'Platnost kódu vypršela',
    'settings.tailscale.sipeed.claimCopy': 'Kopírovat kód',
    'settings.tailscale.sipeed.claimCopied': 'Zkopírováno',
    'settings.tailscale.sipeed.claimRenew': 'Získat nový kód',
    'settings.tailscale.sipeed.claimHint':
      'Nainstaluj a otevři klienta Tailscale na zařízení, které chceš přidat, zvol vlastní server a zadej {{server}}. Prohlížeč otevře stránku pro připojení, kde tyto číslice zadáš.'
  },
  uk: {
    'tailscale.continue': 'Продовжити',
    'tailscale.start': 'Увімкнути',
    'tailscale.stop': 'Вимкнути',
    'tailscale.restart': 'Перезапустити',
    'tailscale.disconnect':
      'Це відключить Tailscale та його автоматичний запуск під час завантаження системи.',
    'tailscale.confirm': 'Підтвердити',
    'tailscale.cancel': 'Скасувати',
    'tailscale.stopped': 'Вимкнено',
    'tailscale.connecting': 'Підключення',
    'tailscale.error': 'Помилка підключення',
    'tailscale.connected': 'Підключено',
    'tailscale.offline': 'Поза мережею',
    'tailscale.notLogin': 'Вхід не виконано',
    ...sipeedManagedFallbackMessages,
    ...sipeedRuntimeTranslations.uk,
    ...sipeedManagementTranslations.uk,
    'settings.tailscale.title': 'Tailscale',
    'settings.tailscale.restart': 'Ви впевнені, що бажаєте перезавантажити Tailscale?',
    'settings.tailscale.stop': 'Ви впевнені, що бажаєте зупинити Tailscale?',
    'settings.tailscale.stopDesc':
      'Вийдіть з Tailscale і вимкніть автоматичний запуск при завантаженні.',
    'settings.tailscale.loading': 'Завантаження...',
    'settings.tailscale.notInstall': 'Tailscale не знайдено! Будь ласка, встановіть його.',
    'settings.tailscale.install': 'Встановити',
    'settings.tailscale.installing': 'Встановлення',
    'settings.tailscale.failed': 'Не вдалося встановити',
    'settings.tailscale.retry':
      'Будь ласка, оновіть сторінку та спробуйте ще раз. Або спробуйте встановити вручну',
    'settings.tailscale.download': 'Завантажте',
    'settings.tailscale.package': 'пакет встановлення',
    'settings.tailscale.unzip': 'та розпакуйте його',
    'settings.tailscale.upTailscale': 'Завантажте Tailscale до каталогу /usr/bin/ на NanoKVM',
    'settings.tailscale.upTailscaled': 'Завантажте Tailscaled до каталогу /usr/sbin/ на NanoKVM',
    'settings.tailscale.refresh': 'Оновіть поточну сторінку',
    'settings.tailscale.notRunning': 'Tailscale не працює. Щоб продовжити, запустіть його.',
    'settings.tailscale.run': 'Запустити',
    'settings.tailscale.notLogin':
      "Пристрій ще не прив'язаний. Будь ласка, увійдіть і прив'яжіть цей пристрій до вашого облікового запису.",
    'settings.tailscale.urlPeriod': 'Ця URL-адреса дійсна протягом 10 хвилин',
    'settings.tailscale.login': 'Увійти',
    'settings.tailscale.loginSuccess': 'Успішний вхід',
    'settings.tailscale.enable': 'Увімкнути Tailscale',
    'settings.tailscale.exitNode': 'Вихідний вузол',
    'settings.tailscale.exitNodeDesc':
      'Оголосити цей пристрій інтернет-шлюзом для вашої мережі tailnet.',
    'settings.tailscale.subnetRoute': 'Маршрут підмережі',
    'settings.tailscale.subnetRouteDesc':
      'Оголосити безпосередньо підключену LAN-підмережу {{subnet}} у вашій мережі tailnet.',
    'settings.tailscale.subnetUnavailable':
      'Немає доступної безпосередньо підключеної LAN-підмережі.',
    'settings.tailscale.routingUnavailable':
      'Налаштування маршрутизації Tailscale недоступні на цьому пристрої.',
    'settings.tailscale.pendingApproval':
      'Після ввімкнення схваліть оголошений маршрут у консолі адміністрування Tailscale.',
    'settings.tailscale.routingUpdateFailed':
      'Не вдалося оновити налаштування маршрутизації Tailscale.',
    'settings.tailscale.deviceName': 'Назва пристрою',
    'settings.tailscale.deviceIP': 'IP пристрою',
    'settings.tailscale.account': 'Обліковий запис',
    'settings.tailscale.logout': 'Вийти',
    'settings.tailscale.logoutDesc': 'Ви впевнені, що хочете вийти?',
    'settings.tailscale.logoutFailed': 'Не вдалося вийти.',
    'settings.tailscale.uninstall': 'Видалити Tailscale',
    'settings.tailscale.uninstallDesc': 'Ви впевнені, що хочете видалити Tailscale?',
    'settings.tailscale.okBtn': 'Так',
    'settings.tailscale.cancelBtn': 'Ні',
    'settings.tailscale.mode.official': 'Офіційний Tailscale',
    'settings.tailscale.mode.officialDesc': 'Використовувати офіційний сервіс Tailscale.',
    'settings.tailscale.mode.sipeed': 'Віддалена мережа Sipeed',
    'settings.tailscale.mode.sipeedDesc': 'Прямі з’єднання між пристроями; без ретрансляції.',
    'settings.tailscale.sipeed.disclaimerTitle': 'Відмова від відповідальності',
    'settings.tailscale.sipeed.enableDesc':
      'Увімкнення віддаленої мережі Sipeed надсилає запити до серверів Sipeed разом із даними підключення, зокрема ідентифікатором пристрою, його назвою та IP-адресою віртуальної мережі. Ці дані використовуються для реєстрації пристроїв, автентифікації, координації з’єднань і керування мережею.\n\nВіддалений доступ залежить від стану мережі, налаштувань пристрою та доступності сервісів. Можливі затримки, переривання або тимчасова недоступність. Оцініть ризики використання та за потреби збережіть альтернативний спосіб доступу.\n\nВикористовуйте сервіс лише з власними пристроями й мережами або за наявності дозволу на доступ. Захищайте облікові дані та коди сполучення й обережно надавайте доступ іншим людям.\n\nНатискаючи «Погодитися й запустити», ви підтверджуєте, що прочитали й зрозуміли це повідомлення та погоджуєтеся на описаний обмін даними й їх обробку для забезпечення віддалених з’єднань. Згода зберігається на цьому пристрої, тому не потрібно підтверджувати її щоразу під час увімкнення.\n\nВи можете відкликати згоду в меню у верхньому правому куті сторінки активного підключення. Це завершить сеанс у віддаленій мережі Sipeed і надішле запит на видалення пов’язаної хмарної мережі та її користувача. Після видалення інші пристрої також не зможуть використовувати цю мережу для віддаленого доступу. Повторне ввімкнення потребує нової згоди.',
    'settings.tailscale.sipeed.acceptAndEnable': 'Погодитися й запустити',
    'settings.tailscale.sipeed.revokeDisclaimer': 'Відкликати згоду',
    'settings.tailscale.sipeed.revokeDisclaimerDesc':
      'Ця дія завершить сеанс у віддаленій мережі Sipeed, видалить хмарну мережу та її користувача, а також збережену згоду. Усі пристрої цієї мережі буде відключено. Продовжити?',
    'settings.tailscale.sipeed.revokeDisclaimerFailed':
      'Не вдалося завершити відкликання згоди. Спробуйте ще раз.',
    'settings.tailscale.sipeed.addDevice': 'Додати пристрій',
    'settings.tailscale.sipeed.addDeviceDesc':
      'Пристроїв ще не додано. Виберіть «Додати пристрій» і завершіть підключення на комп’ютері чи телефоні.',
    'settings.tailscale.sipeed.claimExpiry': 'Залишилось {{time}} · одноразовий код',
    'settings.tailscale.sipeed.claimExpired': 'Термін дії коду минув',
    'settings.tailscale.sipeed.claimCopy': 'Копіювати код',
    'settings.tailscale.sipeed.claimCopied': 'Скопійовано',
    'settings.tailscale.sipeed.claimRenew': 'Отримати новий код',
    'settings.tailscale.sipeed.claimHint':
      'Встанови та відкрий клієнт Tailscale на пристрої, який хочеш додати, вибери власний сервер та введи {{server}}. Браузер відкриє сторінку приєднання, де треба ввести ці цифри.'
  },
  nb: {
    'tailscale.continue': 'Fortsett',
    'tailscale.start': 'Aktiver',
    'tailscale.stop': 'Deaktiver',
    'tailscale.restart': 'Start på nytt',
    'tailscale.disconnect':
      'Dette kobler fra Tailscale og deaktiverer automatisk oppstart ved oppstart av systemet.',
    'tailscale.confirm': 'Bekreft',
    'tailscale.cancel': 'Avbryt',
    'tailscale.stopped': 'Deaktivert',
    'tailscale.connecting': 'Kobler til',
    'tailscale.error': 'Tilkoblingsfeil',
    'tailscale.connected': 'Tilkoblet',
    'tailscale.offline': 'Frakoblet',
    'tailscale.notLogin': 'Ikke innlogget',
    ...sipeedManagedFallbackMessages,
    ...sipeedRuntimeTranslations.nb,
    ...sipeedManagementTranslations.nb,
    'settings.tailscale.title': 'Tailscale',
    'settings.tailscale.restart': 'Starte Tailscale på nytt?',
    'settings.tailscale.stop': 'Stoppe Tailscale?',
    'settings.tailscale.stopDesc':
      'Logg ut av Tailscale og deaktiver automatisk oppstart ved boot.',
    'settings.tailscale.loading': 'Laster...',
    'settings.tailscale.notInstall': 'Tailscale er ikke funnet! Vennligst installer.',
    'settings.tailscale.install': 'Installér',
    'settings.tailscale.installing': 'Installerer',
    'settings.tailscale.failed': 'Installering feilet',
    'settings.tailscale.retry':
      'Vennligst last inn siden på nytt og forsøk igjen eller installer manuelt',
    'settings.tailscale.download': 'Last ned',
    'settings.tailscale.package': 'installasjonspakken',
    'settings.tailscale.unzip': 'og pakk den ut',
    'settings.tailscale.upTailscale': 'Last opp Tailscale til NanoKVM-enhetens mappe /usr/bin/',
    'settings.tailscale.upTailscaled': 'Last opp tailscaled til NanoKVM-enhetens mappe /usr/sbin/',
    'settings.tailscale.refresh': 'Last inn denne siden på nytt',
    'settings.tailscale.notRunning': 'Tailscale kjører ikke. Start den for å fortsette.',
    'settings.tailscale.run': 'Start',
    'settings.tailscale.notLogin':
      'Denne enheten er ikke knyttet til din konto enda. Vennligst logg inn og knytt den til kontoen din..',
    'settings.tailscale.urlPeriod': 'Denne lenken er gyldig i 10 minutter',
    'settings.tailscale.login': 'Logg inn',
    'settings.tailscale.loginSuccess': 'Logget inn',
    'settings.tailscale.enable': 'Skru på Tailscale',
    'settings.tailscale.exitNode': 'Utgangsnode',
    'settings.tailscale.exitNodeDesc':
      'Annonser denne enheten som internettutgang for tailnettet ditt.',
    'settings.tailscale.subnetRoute': 'Delnettrute',
    'settings.tailscale.subnetRouteDesc':
      'Annonser det direkte tilkoblede LAN-delnettet {{subnet}} til tailnettet ditt.',
    'settings.tailscale.subnetUnavailable': 'Ingen direkte tilkoblet LAN-delnett er tilgjengelig.',
    'settings.tailscale.routingUnavailable':
      'Tailscale-ruting er ikke tilgjengelig på denne enheten.',
    'settings.tailscale.pendingApproval':
      'Etter aktivering må den annonserte ruten godkjennes i Tailscale-administrasjonskonsollen.',
    'settings.tailscale.routingUpdateFailed':
      'Kunne ikke oppdatere innstillingene for Tailscale-ruting.',
    'settings.tailscale.deviceName': 'Enhetens navn',
    'settings.tailscale.deviceIP': 'Enhetens IP',
    'settings.tailscale.account': 'Konto',
    'settings.tailscale.logout': 'Logg ut',
    'settings.tailscale.logoutDesc': 'Er du sikker på at du vil logge ut?',
    'settings.tailscale.logoutFailed': 'Kunne ikke logge ut.',
    'settings.tailscale.uninstall': 'Avinstaller Tailscale',
    'settings.tailscale.uninstallDesc': 'Er du sikker på at du vil avinstallere Tailscale?',
    'settings.tailscale.okBtn': 'Ja',
    'settings.tailscale.cancelBtn': 'Nei',
    'settings.tailscale.mode.official': 'Offisiell Tailscale',
    'settings.tailscale.mode.officialDesc': 'Bruk Tailscales offisielle tjeneste.',
    'settings.tailscale.mode.sipeed': 'Sipeed-fjernnettverk',
    'settings.tailscale.mode.sipeedDesc':
      'Direkte forbindelser mellom enheter; ingen videresending.',
    'settings.tailscale.sipeed.disclaimerTitle': 'Ansvarsfraskrivelse',
    'settings.tailscale.sipeed.enableDesc':
      'Når du aktiverer Sipeed Remote Network, sendes forespørsler til Sipeeds servere med tilkoblingsopplysninger som enhetsidentifikator, enhetsnavn og virtuell IP-adresse. Opplysningene brukes til registrering av enheter, autentisering, koordinering av tilkoblinger og nettverksadministrasjon.\n\nEkstern tilgang avhenger av nettverksforhold, enhetsinnstillinger og tilgjengelige tjenester. Tilkoblinger kan bli forsinket, avbrutt eller midlertidig utilgjengelige. Vurder risikoen ved bruk og sørg ved behov for en alternativ tilgangsmåte.\n\nBruk tjenesten bare med enheter og nettverk du eier eller har tillatelse til å bruke. Oppbevar påloggingsopplysninger og paringskoder sikkert, og vær varsom med å gi andre tilgang.\n\nVed å klikke på «Godta og start» bekrefter du at du har lest og forstått denne erklæringen og samtykker til kommunikasjonen og behandlingen av opplysninger som er beskrevet for å tilby eksterne tilkoblinger. Samtykket lagres på enheten, så du slipper å bekrefte det hver gang tjenesten aktiveres.\n\nDu kan trekke tilbake samtykket i menyen øverst til høyre på siden for den aktive tilkoblingen. Da forlater du Sipeed Remote Network, og sletting av det tilknyttede skynettverket og brukeren blir forespurt. Etter sletting kan heller ikke andre enheter bruke dette nettverket til ekstern tilgang. Ny aktivering krever nytt samtykke.',
    'settings.tailscale.sipeed.acceptAndEnable': 'Godta og start',
    'settings.tailscale.sipeed.revokeDisclaimer': 'Trekk tilbake samtykke',
    'settings.tailscale.sipeed.revokeDisclaimerDesc':
      'Dette forlater Sipeed Remote Network, sletter skynettverket og brukeren og fjerner det lagrede samtykket. Alle enheter i nettverket blir koblet fra. Vil du fortsette?',
    'settings.tailscale.sipeed.revokeDisclaimerFailed':
      'Tilbaketrekkingen kunne ikke fullføres. Prøv igjen.',
    'settings.tailscale.sipeed.addDevice': 'Legg til enhet',
    'settings.tailscale.sipeed.addDeviceDesc':
      'Ingen enhet er lagt til ennå. Velg «Legg til enhet» og fullfør tilkoblingen på datamaskinen eller telefonen.',
    'settings.tailscale.sipeed.claimExpiry': '{{time}} igjen · kun én gang',
    'settings.tailscale.sipeed.claimExpired': 'Koden er utløpt',
    'settings.tailscale.sipeed.claimCopy': 'Kopier kode',
    'settings.tailscale.sipeed.claimCopied': 'Kopiert',
    'settings.tailscale.sipeed.claimRenew': 'Hent en ny kode',
    'settings.tailscale.sipeed.claimHint':
      'Installer og åpne Tailscale-klienten på enheten du vil legge til, velg den egendefinerte serveren og skriv inn {{server}}. Nettleseren åpner tilkoblingssiden, der du skriver inn disse sifrene.'
  },
  th: {
    'tailscale.continue': 'ดำเนินการต่อ',
    'tailscale.start': 'เปิดใช้งาน',
    'tailscale.stop': 'ปิดใช้งาน',
    'tailscale.restart': 'เริ่มใหม่',
    'tailscale.disconnect':
      'การดำเนินการนี้จะตัดการเชื่อมต่อ Tailscale และปิดการเริ่มต้นอัตโนมัติเมื่อบูตเครื่อง',
    'tailscale.confirm': 'ยืนยัน',
    'tailscale.cancel': 'ยกเลิก',
    'tailscale.stopped': 'ปิดใช้งานแล้ว',
    'tailscale.connecting': 'กำลังเชื่อมต่อ',
    'tailscale.error': 'การเชื่อมต่อผิดพลาด',
    'tailscale.connected': 'เชื่อมต่อแล้ว',
    'tailscale.offline': 'ออฟไลน์',
    'tailscale.notLogin': 'ยังไม่ได้เข้าสู่ระบบ',
    ...sipeedManagedFallbackMessages,
    ...sipeedRuntimeTranslations.th,
    ...sipeedManagementTranslations.th,
    'settings.tailscale.title': 'Tailscale',
    'settings.tailscale.restart': 'คุณแน่ใจที่จะรีสตาร์ท Tailscale หรือไม่?',
    'settings.tailscale.stop': 'คุณแน่ใจที่จะปิด Tailscale หรือไม่?',
    'settings.tailscale.stopDesc':
      'ออกจากระบบ Tailscale และปิดใช้งานการเริ่มต้นอัตโนมัติเมื่อบูตเครื่อง',
    'settings.tailscale.loading': 'กำลังโหลด...',
    'settings.tailscale.notInstall': 'ไม่พบ Tailscale! กรุณาติดตั้ง',
    'settings.tailscale.install': 'ติดตั้ง',
    'settings.tailscale.installing': 'กำลังติดตั้ง',
    'settings.tailscale.failed': 'การติดตั้งล้มเหลว',
    'settings.tailscale.retry': 'กรุณารีเฟรชและลองใหม่ หรือพยายามติดตั้งด้วยตนเอง',
    'settings.tailscale.download': 'ดาวน์โหลด',
    'settings.tailscale.package': 'แพ็กเกจการติดตั้ง',
    'settings.tailscale.unzip': 'และแตกไฟล์',
    'settings.tailscale.upTailscale': 'อัปโหลด tailscale ไปยังไดเรกทอรี /usr/bin/ ของ NanoKVM',
    'settings.tailscale.upTailscaled': 'อัปโหลด tailscaled ไปยังไดเรกทอรี /usr/sbin/ ของ NanoKVM',
    'settings.tailscale.refresh': 'รีเฟรชหน้าปัจจุบัน',
    'settings.tailscale.notRunning': 'Tailscale ไม่ทำงาน กรุณาเริ่มต้นเพื่อดำเนินการต่อ',
    'settings.tailscale.run': 'เริ่ม',
    'settings.tailscale.notLogin':
      'อุปกรณ์ยังไม่ได้ผูกบัญชี กรุณาล็อกอินและผูกอุปกรณ์นี้กับบัญชีของคุณ',
    'settings.tailscale.urlPeriod': 'ลิงก์นี้ใช้ได้ 10 นาที',
    'settings.tailscale.login': 'ล็อกอิน',
    'settings.tailscale.loginSuccess': 'ล็อกอินสำเร็จ',
    'settings.tailscale.enable': 'เปิดใช้งาน Tailscale',
    'settings.tailscale.exitNode': 'โหนดทางออก',
    'settings.tailscale.exitNodeDesc':
      'ประกาศอุปกรณ์นี้เป็นทางออกอินเทอร์เน็ตสำหรับ tailnet ของคุณ',
    'settings.tailscale.subnetRoute': 'เส้นทางซับเน็ต',
    'settings.tailscale.subnetRouteDesc':
      'ประกาศซับเน็ต LAN ที่เชื่อมต่อโดยตรง {{subnet}} ไปยัง tailnet ของคุณ',
    'settings.tailscale.subnetUnavailable': 'ไม่มีซับเน็ต LAN ที่เชื่อมต่อโดยตรงให้ใช้งาน',
    'settings.tailscale.routingUnavailable':
      'อุปกรณ์นี้ไม่รองรับการตั้งค่าการกำหนดเส้นทาง Tailscale',
    'settings.tailscale.pendingApproval':
      'หลังจากเปิดใช้งาน โปรดอนุมัติเส้นทางที่ประกาศในคอนโซลผู้ดูแลระบบ Tailscale',
    'settings.tailscale.routingUpdateFailed': 'อัปเดตการตั้งค่าการกำหนดเส้นทาง Tailscale ไม่สำเร็จ',
    'settings.tailscale.deviceName': 'ชื่ออุปกรณ์',
    'settings.tailscale.deviceIP': 'IP อุปกรณ์',
    'settings.tailscale.account': 'บัญชี',
    'settings.tailscale.logout': 'ออกจากระบบ',
    'settings.tailscale.logoutDesc': 'คุณแน่ใจที่จะออกจากระบบหรือไม่?',
    'settings.tailscale.logoutFailed': 'ออกจากระบบไม่สำเร็จ',
    'settings.tailscale.uninstall': 'ถอนการติดตั้ง Tailscale',
    'settings.tailscale.uninstallDesc': 'คุณแน่ใจหรือไม่ว่าต้องการถอนการติดตั้ง Tailscale',
    'settings.tailscale.okBtn': 'ใช่',
    'settings.tailscale.cancelBtn': 'ไม่ใช่',
    'settings.tailscale.mode.official': 'Tailscale อย่างเป็นทางการ',
    'settings.tailscale.mode.officialDesc': 'ใช้บริการอย่างเป็นทางการของ Tailscale',
    'settings.tailscale.mode.sipeed': 'เครือข่ายระยะไกล Sipeed',
    'settings.tailscale.mode.sipeedDesc': 'เชื่อมต่อโดยตรงระหว่างอุปกรณ์ ไม่มีการส่งต่อ',
    'settings.tailscale.sipeed.disclaimerTitle': 'ข้อจำกัดความรับผิดชอบ',
    'settings.tailscale.sipeed.enableDesc':
      'การเปิดใช้งานเครือข่ายระยะไกล Sipeed จะส่งคำขอไปยังเซิร์ฟเวอร์ Sipeed พร้อมข้อมูลการเชื่อมต่อ เช่น ตัวระบุอุปกรณ์ ชื่ออุปกรณ์ และที่อยู่ IP ของเครือข่ายเสมือน ข้อมูลนี้ใช้สำหรับลงทะเบียนอุปกรณ์ ยืนยันตัวตน ประสานการเชื่อมต่อ และจัดการเครือข่าย\n\nการเข้าถึงระยะไกลขึ้นอยู่กับสภาพเครือข่าย การตั้งค่าอุปกรณ์ และความพร้อมของบริการ การเชื่อมต่ออาจล่าช้า ขัดข้อง หรือใช้งานไม่ได้ชั่วคราว โปรดประเมินความเสี่ยงและเตรียมวิธีเข้าถึงสำรองเมื่อจำเป็น\n\nใช้บริการเฉพาะกับอุปกรณ์และเครือข่ายที่คุณเป็นเจ้าของหรือได้รับอนุญาตให้เข้าถึง เก็บข้อมูลเข้าสู่ระบบและรหัสจับคู่ให้ปลอดภัย และพิจารณาอย่างรอบคอบก่อนให้สิทธิ์ผู้อื่นเข้าถึง\n\nการคลิก “ยอมรับและเริ่ม” หมายถึงคุณได้อ่านและเข้าใจข้อความนี้ และยินยอมให้มีการสื่อสารและประมวลผลข้อมูลตามที่อธิบายเพื่อให้บริการเชื่อมต่อระยะไกล ความยินยอมจะบันทึกไว้ในอุปกรณ์นี้ จึงไม่ต้องยืนยันใหม่ทุกครั้งที่เปิดใช้งาน\n\nคุณสามารถถอนความยินยอมได้จากเมนูมุมขวาบนของหน้าที่เชื่อมต่อแล้ว การถอนจะออกจากเครือข่ายระยะไกล Sipeed และส่งคำขอลบเครือข่ายคลาวด์ที่เกี่ยวข้องพร้อมผู้ใช้ของเครือข่าย หลังลบแล้ว อุปกรณ์อื่นจะไม่สามารถใช้เครือข่ายนี้เพื่อเข้าถึงระยะไกลได้อีก การเปิดใช้งานใหม่ต้องให้ความยินยอมอีกครั้ง',
    'settings.tailscale.sipeed.acceptAndEnable': 'ยอมรับและเริ่ม',
    'settings.tailscale.sipeed.revokeDisclaimer': 'ถอนความยินยอม',
    'settings.tailscale.sipeed.revokeDisclaimerDesc':
      'การดำเนินการนี้จะออกจากเครือข่ายระยะไกล Sipeed ลบเครือข่ายคลาวด์และผู้ใช้ของเครือข่าย รวมถึงล้างความยินยอมที่บันทึกไว้ อุปกรณ์ทั้งหมดในเครือข่ายนี้จะถูกตัดการเชื่อมต่อ ต้องการดำเนินการต่อหรือไม่',
    'settings.tailscale.sipeed.revokeDisclaimerFailed':
      'ไม่สามารถถอนความยินยอมได้สำเร็จ โปรดลองอีกครั้ง',
    'settings.tailscale.sipeed.addDevice': 'เพิ่มอุปกรณ์',
    'settings.tailscale.sipeed.addDeviceDesc':
      'ยังไม่มีอุปกรณ์ที่เพิ่มไว้ เลือก "เพิ่มอุปกรณ์" แล้วทำการเชื่อมต่อให้เสร็จบนคอมพิวเตอร์หรือโทรศัพท์ที่ต้องการเชื่อมต่อ',
    'settings.tailscale.sipeed.claimExpiry': 'เหลือ {{time}} · ใช้ได้ครั้งเดียว',
    'settings.tailscale.sipeed.claimExpired': 'รหัสหมดอายุแล้ว',
    'settings.tailscale.sipeed.claimCopy': 'คัดลอกรหัส',
    'settings.tailscale.sipeed.claimCopied': 'คัดลอกแล้ว',
    'settings.tailscale.sipeed.claimRenew': 'ขอรหัสใหม่',
    'settings.tailscale.sipeed.claimHint':
      'ติดตั้งและเปิดไคลเอนต์ Tailscale บนอุปกรณ์ที่ต้องการเพิ่ม เลือกเซิร์ฟเวอร์แบบกำหนดเอง แล้วกรอก {{server}} เบราว์เซอร์จะเปิดหน้าลงทะเบียน ให้กรอกตัวเลขนี้'
  }
};
