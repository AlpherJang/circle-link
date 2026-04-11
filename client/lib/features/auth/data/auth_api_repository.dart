import '../../../core/network/api_client.dart';
import '../../../core/storage/token_store.dart';
import '../domain/auth_models.dart';
import 'auth_repository.dart';

class AuthApiRepository implements AuthRepository {
  const AuthApiRepository({
    required this.apiClient,
    required this.tokenStore,
  });

  final ApiClient apiClient;
  final TokenStore tokenStore;

  @override
  Future<void> changePassword(ChangePasswordCommand command) async {
    final accessToken = await tokenStore.readAccessToken();
    final response = await apiClient.postJson(
      '/v1/auth/change-password',
      accessToken: accessToken,
      body: <String, dynamic>{
        'currentPassword': command.currentPassword,
        'newPassword': command.newPassword,
      },
    );
    _throwIfError(response);
  }

  @override
  Future<AuthSession> login(LoginCommand command) async {
    final response = await apiClient.postJson(
      '/v1/auth/login',
      body: <String, dynamic>{
        'email': command.email,
        'password': command.password,
      },
    );
    final data = _requireData(response);

    final tokens = AuthTokens(
      accessToken: data['accessToken'] as String,
      refreshToken: data['refreshToken'] as String,
      expiresAt: DateTime.parse(data['expiresAt'] as String),
    );
    await tokenStore.save(
      accessToken: tokens.accessToken,
      refreshToken: tokens.refreshToken,
    );

    return AuthSession(
      user: AuthUser(
        id: data['userId'] as String,
        email: command.email,
        displayName: command.email.split('@').first,
        status: UserStatus.active,
      ),
      tokens: tokens,
    );
  }

  @override
  Future<void> logout(String refreshToken) async {
    await apiClient.postJson(
      '/v1/auth/logout',
      body: <String, dynamic>{
        'refreshToken': refreshToken,
      },
    );
    await tokenStore.clear();
  }

  @override
  Future<AuthTokens> refresh(String refreshToken) async {
    final response = await apiClient.postJson(
      '/v1/auth/refresh',
      body: <String, dynamic>{
        'refreshToken': refreshToken,
      },
    );
    final data = _requireData(response);

    final tokens = AuthTokens(
      accessToken: data['accessToken'] as String,
      refreshToken: data['refreshToken'] as String,
      expiresAt: DateTime.parse(data['expiresAt'] as String),
    );
    await tokenStore.save(
      accessToken: tokens.accessToken,
      refreshToken: tokens.refreshToken,
    );

    return tokens;
  }

  @override
  Future<SignUpResult> signUp(SignUpCommand command) async {
    final response = await apiClient.postJson(
      '/v1/auth/signup',
      body: <String, dynamic>{
        'email': command.email,
        'password': command.password,
        'displayName': command.displayName,
      },
    );
    final data = _requireData(response);

    return SignUpResult(
      userId: data['userId'] as String,
      emailVerificationRequired: data['emailVerificationRequired'] as bool? ?? true,
      verificationToken: data['verificationToken'] as String?,
    );
  }

  @override
  Future<void> verifyEmail(VerifyEmailCommand command) async {
    final response = await apiClient.postJson(
      '/v1/auth/verify-email',
      body: <String, dynamic>{
        'email': command.email,
        'verificationToken': command.verificationToken,
      },
    );
    _throwIfError(response);
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
