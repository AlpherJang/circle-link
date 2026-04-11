enum DevicePlatform {
  ios,
  macos,
  android,
}

class DeviceRecord {
  const DeviceRecord({
    required this.id,
    required this.deviceName,
    required this.platform,
    this.lastSeenAt,
    this.revokedAt,
  });

  final String id;
  final String deviceName;
  final DevicePlatform platform;
  final DateTime? lastSeenAt;
  final DateTime? revokedAt;
}

class RegisterDeviceCommand {
  const RegisterDeviceCommand({
    required this.deviceName,
    required this.platform,
    required this.pushToken,
    required this.identityKeyPublic,
    required this.signedPrekeyPublic,
    required this.signedPrekeySignature,
    required this.signedPrekeyVersion,
    required this.oneTimePrekeys,
  });

  final String deviceName;
  final DevicePlatform platform;
  final String pushToken;
  final String identityKeyPublic;
  final String signedPrekeyPublic;
  final String signedPrekeySignature;
  final int signedPrekeyVersion;
  final List<String> oneTimePrekeys;
}
