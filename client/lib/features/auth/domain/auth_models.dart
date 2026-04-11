enum UserStatus {
  pendingVerification,
  active,
  disabled,
}

class AuthUser {
  const AuthUser({
    required this.id,
    required this.email,
    required this.displayName,
    required this.status,
    this.emailVerifiedAt,
  });

  final String id;
  final String email;
  final String displayName;
  final UserStatus status;
  final DateTime? emailVerifiedAt;
}

class AuthTokens {
  const AuthTokens({
    required this.accessToken,
    required this.refreshToken,
    required this.expiresAt,
  });

  final String accessToken;
  final String refreshToken;
  final DateTime expiresAt;
}

class SignUpCommand {
  const SignUpCommand({
    required this.email,
    required this.password,
    required this.displayName,
  });

  final String email;
  final String password;
  final String displayName;
}

class SignUpResult {
  const SignUpResult({
    required this.userId,
    required this.emailVerificationRequired,
    this.verificationToken,
  });

  final String userId;
  final bool emailVerificationRequired;
  final String? verificationToken;
}

class VerifyEmailCommand {
  const VerifyEmailCommand({
    required this.email,
    required this.verificationToken,
  });

  final String email;
  final String verificationToken;
}

class LoginCommand {
  const LoginCommand({
    required this.email,
    required this.password,
  });

  final String email;
  final String password;
}

class ChangePasswordCommand {
  const ChangePasswordCommand({
    required this.currentPassword,
    required this.newPassword,
  });

  final String currentPassword;
  final String newPassword;
}

class AuthSession {
  const AuthSession({
    required this.user,
    required this.tokens,
  });

  final AuthUser user;
  final AuthTokens tokens;
}
