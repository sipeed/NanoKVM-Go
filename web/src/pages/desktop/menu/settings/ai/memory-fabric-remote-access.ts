export type MemoryFabricRemoteAccess = 'disabled' | 'requiresMcp' | 'available';

export function getMemoryFabricRemoteAccess(
  mcpEnabled: boolean,
  memoryFabricEnabled: boolean
): MemoryFabricRemoteAccess {
  if (!memoryFabricEnabled) return 'disabled';
  return mcpEnabled ? 'available' : 'requiresMcp';
}
