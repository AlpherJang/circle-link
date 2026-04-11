import '../domain/auth_models.dart';

abstract interface class AuthRepository {
  Future<SignUpResult> signUp(SignUpCommand command);
  Future<void> verifyEmail(VerifyEmailCommand command);
  Future<AuthSession> login(LoginCommand command);
  Future<AuthTokens> refresh(String refreshToken);
  Future<void> logout(String refreshToken);
  Future<void> changePassword(ChangePasswordCommand command);
}
