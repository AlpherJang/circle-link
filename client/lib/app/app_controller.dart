import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

import '../core/network/api_client.dart';
import '../core/storage/in_memory_token_store.dart';
import '../core/storage/token_store.dart';
import '../features/auth/data/auth_api_repository.dart';
import '../features/auth/data/auth_repository.dart';
import '../features/auth/domain/auth_models.dart';
import '../features/devices/data/device_api_repository.dart';
import '../features/devices/data/device_repository.dart';
import '../features/devices/domain/device_models.dart';

enum AppStage {
  auth,
  verifyEmail,
  home,
}

class AppController extends ChangeNotifier {
  factory AppController({
    ApiClient? apiClient,
    TokenStore? tokenStore,
    AuthRepository? authRepository,
    DeviceRepository? deviceRepository,
  }) {
    final resolvedApiClient =
        apiClient ?? const ApiClient(baseUrl: 'http://127.0.0.1:8080');
    final resolvedTokenStore = tokenStore ?? InMemoryTokenStore();

    return AppController._(
      apiClient: resolvedApiClient,
      tokenStore: resolvedTokenStore,
      authRepository: authRepository ??
          AuthApiRepository(
            apiClient: resolvedApiClient,
            tokenStore: resolvedTokenStore,
          ),
      deviceRepository: deviceRepository ??
          DeviceApiRepository(
            apiClient: resolvedApiClient,
            tokenStore: resolvedTokenStore,
          ),
    );
  }

  AppController._({
    required ApiClient apiClient,
    required TokenStore tokenStore,
    required AuthRepository authRepository,
    required DeviceRepository deviceRepository,
  })  : _tokenStore = tokenStore,
        _apiClient = apiClient,
        _authRepository = authRepository,
        _deviceRepository = deviceRepository;

  final ApiClient _apiClient;
  final TokenStore _tokenStore;
  final AuthRepository _authRepository;
  final DeviceRepository _deviceRepository;

  AppStage stage = AppStage.auth;
  bool isBusy = false;
  String? errorMessage;
  String? infoMessage;
  String? pendingEmail;
  String? verificationTokenHint;
  AuthSession? session;
  List<DeviceRecord> devices = const <DeviceRecord>[];

  String get baseUrl => _apiClient.baseUrl;

  Future<void> signUp(SignUpCommand command) async {
    await _run(() async {
      final result = await _authRepository.signUp(command);
      pendingEmail = command.email;
      verificationTokenHint = result.verificationToken;
      infoMessage = result.emailVerificationRequired
          ? 'Account created. Verify your email before logging in.'
          : 'Account created.';
      stage = result.emailVerificationRequired ? AppStage.verifyEmail : AppStage.auth;
    });
  }

  Future<void> verifyEmail(String token) async {
    final email = pendingEmail;
    if (email == null || email.isEmpty) {
      errorMessage = 'Missing email address for verification.';
      notifyListeners();
      return;
    }

    await _run(() async {
      await _authRepository.verifyEmail(
        VerifyEmailCommand(
          email: email,
          verificationToken: token,
        ),
      );
      verificationTokenHint = null;
      infoMessage = 'Email verified. You can now log in.';
      stage = AppStage.auth;
    });
  }

  Future<void> login(LoginCommand command) async {
    await _run(() async {
      session = await _authRepository.login(command);
      pendingEmail = command.email;
      stage = AppStage.home;
      devices = await _deviceRepository.listDevices();
    });
  }

  Future<void> logout() async {
    final refreshToken = session?.tokens.refreshToken;
    if (refreshToken == null) {
      return;
    }

    await _run(() async {
      await _authRepository.logout(refreshToken);
      await _tokenStore.clear();
      session = null;
      devices = const <DeviceRecord>[];
      stage = AppStage.auth;
      infoMessage = 'Logged out.';
    });
  }

  Future<void> loadDevices() async {
    await _run(() async {
      devices = await _deviceRepository.listDevices();
    }, preserveMessages: true);
  }

  Future<void> registerCurrentDevice() async {
    await _run(() async {
      final now = DateTime.now().millisecondsSinceEpoch;
      final created = await _deviceRepository.registerDevice(
        RegisterDeviceCommand(
          deviceName: _defaultDeviceName(),
          platform: _currentPlatform(),
          pushToken: '',
          identityKeyPublic: 'identity-$now',
          signedPrekeyPublic: 'signed-$now',
          signedPrekeySignature: 'signature-$now',
          signedPrekeyVersion: 1,
          oneTimePrekeys: <String>[
            'prekey-${now}a',
            'prekey-${now}b',
            'prekey-${now}c',
          ],
        ),
      );
      devices = <DeviceRecord>[...devices, created];
      infoMessage = 'Device registered with placeholder key material.';
    });
  }

  Future<void> revokeDevice(String deviceId) async {
    await _run(() async {
      await _deviceRepository.revokeDevice(deviceId);
      devices = await _deviceRepository.listDevices();
    });
  }

  Future<void> _run(
    Future<void> Function() action, {
    bool preserveMessages = false,
  }) async {
    isBusy = true;
    if (!preserveMessages) {
      errorMessage = null;
      infoMessage = null;
    }
    notifyListeners();

    try {
      await action();
    } on ApiException catch (error) {
      errorMessage = error.message;
    } catch (error) {
      errorMessage = error.toString();
    } finally {
      isBusy = false;
      notifyListeners();
    }
  }

  DevicePlatform _currentPlatform() {
    switch (defaultTargetPlatform) {
      case TargetPlatform.iOS:
        return DevicePlatform.ios;
      case TargetPlatform.android:
        return DevicePlatform.android;
      case TargetPlatform.macOS:
      default:
        return DevicePlatform.macos;
    }
  }

  String _defaultDeviceName() {
    switch (_currentPlatform()) {
      case DevicePlatform.ios:
        return 'circle-link iPhone';
      case DevicePlatform.android:
        return 'circle-link Android';
      case DevicePlatform.macos:
        return 'circle-link Mac';
    }
  }
}
