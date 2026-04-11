import '../domain/device_models.dart';

abstract interface class DeviceRepository {
  Future<DeviceRecord> registerDevice(RegisterDeviceCommand command);
  Future<List<DeviceRecord>> listDevices();
  Future<void> revokeDevice(String deviceId);
}
