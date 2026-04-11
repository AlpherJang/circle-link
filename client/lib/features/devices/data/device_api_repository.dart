import '../../../core/network/api_client.dart';
import '../../../core/storage/token_store.dart';
import '../domain/device_models.dart';
import 'device_repository.dart';

class DeviceApiRepository implements DeviceRepository {
  const DeviceApiRepository({
    required this.apiClient,
    required this.tokenStore,
  });

  final ApiClient apiClient;
  final TokenStore tokenStore;

  @override
  Future<List<DeviceRecord>> listDevices() async {
    final accessToken = await tokenStore.readAccessToken();
    final response = await apiClient.getJson(
      '/v1/devices',
      accessToken: accessToken,
    );
    final data = _requireData(response);
    final items = data['items'];
    if (items is! List) {
      return const <DeviceRecord>[];
    }

    return items
        .whereType<Map<String, dynamic>>()
        .map(_deviceFromJson)
        .toList(growable: false);
  }

  @override
  Future<DeviceRecord> registerDevice(RegisterDeviceCommand command) async {
    final accessToken = await tokenStore.readAccessToken();
    final response = await apiClient.postJson(
      '/v1/devices',
      accessToken: accessToken,
      body: <String, dynamic>{
        'deviceName': command.deviceName,
        'platform': command.platform.name,
        'pushToken': command.pushToken,
        'keyBundle': <String, dynamic>{
          'identityKeyPublic': command.identityKeyPublic,
          'signedPrekeyPublic': command.signedPrekeyPublic,
          'signedPrekeySignature': command.signedPrekeySignature,
          'signedPrekeyVersion': command.signedPrekeyVersion,
          'oneTimePrekeys': command.oneTimePrekeys,
        },
      },
    );
    final data = _requireData(response);

    return DeviceRecord(
      id: data['deviceId'] as String,
      deviceName: command.deviceName,
      platform: command.platform,
      lastSeenAt: DateTime.tryParse(data['registeredAt'] as String? ?? ''),
      revokedAt: null,
    );
  }

  @override
  Future<void> revokeDevice(String deviceId) async {
    final accessToken = await tokenStore.readAccessToken();
    final response = await apiClient.deleteJson(
      '/v1/devices/$deviceId',
      accessToken: accessToken,
    );
    _throwIfError(response);
  }

  DeviceRecord _deviceFromJson(Map<String, dynamic> json) {
    return DeviceRecord(
      id: json['deviceId'] as String,
      deviceName: json['deviceName'] as String? ?? 'Unnamed device',
      platform: _platformFromString(json['platform'] as String? ?? ''),
      lastSeenAt: DateTime.tryParse(json['lastSeenAt'] as String? ?? ''),
      revokedAt: DateTime.tryParse(json['revokedAt'] as String? ?? ''),
    );
  }

  DevicePlatform _platformFromString(String value) {
    switch (value) {
      case 'ios':
        return DevicePlatform.ios;
      case 'macos':
        return DevicePlatform.macos;
      case 'android':
        return DevicePlatform.android;
      default:
        return DevicePlatform.macos;
    }
  }

  Map<String, dynamic> _requireData(Map<String, dynamic> response) {
    _throwIfError(response);
    final data = response['data'];
    if (data is! Map<String, dynamic>) {
      throw const ApiException(
        code: 'INVALID_RESPONSE',
        message: 'Server response did not include a valid data object.',
      );
    }

    return data;
  }

  void _throwIfError(Map<String, dynamic> response) {
    final error = response['error'];
    if (error is Map<String, dynamic>) {
      throw ApiException(
        code: error['code'] as String? ?? 'UNKNOWN_ERROR',
        message: error['message'] as String? ?? 'Unknown server error.',
      );
    }
  }
}
