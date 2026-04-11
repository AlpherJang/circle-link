class ApiError {
  const ApiError({
    required this.code,
    required this.message,
  });

  final String code;
  final String message;
}

class ApiResponse<T> {
  const ApiResponse({
    required this.data,
    required this.error,
    required this.requestId,
  });

  final T? data;
  final ApiError? error;
  final String requestId;

  bool get isSuccess => error == null;
}
