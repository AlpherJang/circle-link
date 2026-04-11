import 'dart:convert';
import 'dart:io';

class ApiClient {
  const ApiClient({
    required this.baseUrl,
  });

  final String baseUrl;

  Uri uri(String path) => Uri.parse('$baseUrl$path');

  Future<Map<String, dynamic>> getJson(
    String path, {
    String? accessToken,
  }) {
    return _sendJson(
      method: 'GET',
      path: path,
      accessToken: accessToken,
    );
  }

  Future<Map<String, dynamic>> postJson(
    String path, {
    Map<String, dynamic>? body,
    String? accessToken,
  }) {
    return _sendJson(
      method: 'POST',
      path: path,
      body: body,
      accessToken: accessToken,
    );
  }

  Future<Map<String, dynamic>> deleteJson(
    String path, {
    String? accessToken,
  }) {
    return _sendJson(
      method: 'DELETE',
      path: path,
      accessToken: accessToken,
    );
  }

  Future<Map<String, dynamic>> _sendJson({
    required String method,
    required String path,
    Map<String, dynamic>? body,
    String? accessToken,
  }) async {
    final client = HttpClient();
    try {
      final request = await client.openUrl(method, uri(path));
      headers(accessToken: accessToken).forEach(request.headers.set);

      if (body != null) {
        request.write(jsonEncode(body));
      }

      final response = await request.close();
      final payload = await response.transform(utf8.decoder).join();
      if (payload.isEmpty) {
        return <String, dynamic>{};
      }

      final decoded = jsonDecode(payload);
      if (decoded is Map<String, dynamic>) {
        return decoded;
      }

      throw const ApiException(
        code: 'INVALID_RESPONSE',
        message: 'Server returned a non-object JSON response.',
      );
    } on SocketException {
      throw const ApiException(
        code: 'NETWORK_UNAVAILABLE',
        message: 'Unable to connect to the circle-link server.',
      );
    } finally {
      client.close(force: true);
    }
  }

  Map<String, String> headers({String? accessToken}) {
    return <String, String>{
      'Content-Type': 'application/json',
      if (accessToken != null && accessToken.isNotEmpty)
        'Authorization': 'Bearer $accessToken',
    };
  }
}

class ApiException implements Exception {
  const ApiException({
    required this.code,
    required this.message,
  });

  final String code;
  final String message;

  @override
  String toString() => 'ApiException($code, $message)';
}
