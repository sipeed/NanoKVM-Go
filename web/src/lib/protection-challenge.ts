import * as authApi from '@/api/auth.ts';

export type ProtectionGateKey =
  | 'ssh'
  | 'tailscale'
  | 'frp'
  | 'cloudflare'
  | 'zerotier'
  | 'netbird'
  | 'mcp'
  | 'targetPower'
  | 'nanoKVMRestart'
  | 'deleteImage'
  | 'screenTimelapse';

export type ProtectionSensitiveAction = Extract<
  authApi.SensitiveActionName,
  | 'netbird_install'
  | 'netbird_uninstall'
  | 'netbird_start'
  | 'netbird_stop'
  | 'netbird_restart'
  | 'netbird_login'
  | 'netbird_login_cancel'
  | 'netbird_logout'
  | 'zerotier_install'
  | 'zerotier_uninstall'
  | 'zerotier_start'
  | 'zerotier_stop'
  | 'zerotier_restart'
  | 'zerotier_join'
  | 'zerotier_leave'
  | 'cloudflare_install'
  | 'cloudflare_uninstall'
  | 'cloudflare_configure'
  | 'cloudflare_start'
  | 'cloudflare_stop'
  | 'cloudflare_restart'
  | 'frp_install'
  | 'frp_uninstall'
  | 'frp_configure'
  | 'frp_start'
  | 'frp_stop'
  | 'frp_restart'
  | 'enable_ssh'
  | 'disable_ssh'
  | 'tailscale_install'
  | 'tailscale_login'
  | 'tailscale_up'
  | 'tailscale_down'
  | 'tailscale_start'
  | 'tailscale_stop'
  | 'tailscale_restart'
  | 'tailscale_logout'
  | 'tailscale_uninstall'
  | 'tailscale_set_exit_node'
  | 'tailscale_set_subnet_route'
  | 'mcp_show_key'
  | 'mcp_copy_key'
  | 'mcp_regenerate_key'
  | 'mcp_enable'
  | 'mcp_disable'
  | 'target_power'
  | 'nanokvm_restart'
  | 'delete_image'
  | 'screen_timelapse_enable'
  | 'screen_timelapse_playback'
  | 'set_screen_timelapse_retention'
  | 'trigger_screen_timelapse_cleanup'
>;

function isTwoFactorStatus(value: unknown): value is authApi.TwoFactorStatus {
  if (!value || typeof value !== 'object') return false;
  return typeof (value as authApi.TwoFactorStatus).enabled === 'boolean';
}

function isTwoFactorProtectionStatus(value: unknown): value is authApi.TwoFactorProtectionStatus {
  if (!value || typeof value !== 'object') return false;
  const policy = (value as authApi.TwoFactorProtectionStatus).policy;
  if (!policy || typeof policy !== 'object') return false;
  return (
    typeof policy.ssh === 'boolean' &&
    typeof policy.tailscale === 'boolean' &&
    typeof policy.mcp === 'boolean' &&
    typeof policy.targetPower === 'boolean' &&
    typeof policy.nanoKVMRestart === 'boolean' &&
    typeof policy.deleteImage === 'boolean' &&
    typeof policy.screenTimelapse === 'boolean' &&
    typeof (value as authApi.TwoFactorProtectionStatus).authorizationExpiresIn === 'number' &&
    typeof (value as authApi.TwoFactorProtectionStatus).mcpAuthorizationExpiresIn === 'number' &&
    typeof (value as authApi.TwoFactorProtectionStatus).screenTimelapseAuthorizationExpiresIn ===
      'number'
  );
}

/**
 * Resolve whether a protection-gated op needs a sensitive-action challenge.
 * 2FA off → false (same as historical). 2FA on + switch off → false.
 * 2FA on + switch on / load failure → true (fail closed).
 */
export async function resolveProtectionRequiresChallenge(
  key: ProtectionGateKey
): Promise<{ twoFactorEnabled: boolean; requireChallenge: boolean }> {
  try {
    const statusRsp = await authApi.getTwoFactorStatus();
    // Unknown/failed status: do not claim "2FA off" (would skip challenge while
    // the backend may still require it). Fail closed; callers also retry on a
    // sensitive-action response code.
    if (statusRsp.code !== 0) {
      return { twoFactorEnabled: true, requireChallenge: true };
    }

    const status = isTwoFactorStatus(statusRsp.data) ? statusRsp.data : null;
    if (!status) {
      return { twoFactorEnabled: true, requireChallenge: true };
    }
    if (!status.enabled) {
      return { twoFactorEnabled: false, requireChallenge: false };
    }

    try {
      const protectionRsp = await authApi.getTwoFactorProtection();
      const protection = isTwoFactorProtectionStatus(protectionRsp.data)
        ? protectionRsp.data
        : null;
      if (protectionRsp.code === 0 && protection && typeof protection.policy[key] === 'boolean') {
        return {
          twoFactorEnabled: true,
          requireChallenge:
            protection.policy[key] &&
            (key !== 'mcp' || protection.mcpAuthorizationExpiresIn <= 0) &&
            (key !== 'screenTimelapse' || protection.screenTimelapseAuthorizationExpiresIn <= 0)
        };
      }
    } catch {
      // fail closed when protection cannot be loaded
    }

    return { twoFactorEnabled: true, requireChallenge: true };
  } catch {
    return { twoFactorEnabled: true, requireChallenge: true };
  }
}
