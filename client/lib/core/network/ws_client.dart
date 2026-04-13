import 'dart:async';
import 'dart:convert';
import 'dart:io';

class WsEnvelope {
  const WsEnvelope({
    required this.type,
    required this.payload,
    this.traceId,
  });

  final String type;
  final Map<String, dynamic> payload;
  final String? traceId;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'type': type,
      'payload': payload,
      if (traceId != null) 'traceId': traceId,
    };
  }

  factory WsEnvelope.fromJson(Map<String, dynamic> json) {
    final payload = json['payload'];
    return WsEnvelope(
      type: json['type'] as String? ?? 'unknown',
      payload: payload is Map<String, dynamic>
          ? payload
          : <String, dynamic>{},
      traceId: json['traceId'] as String?,
    );
  }
}

abstract interface class WsClient {
  bool get isConnected;

  Future<void> connect(Uri uri);
  Future<void> disconnect();
  Future<void> send(WsEnvelope event);
  Stream<WsEnvelope> events();
}

class IoWsClient implements WsClient {
  IoWsClient();

  final StreamController<WsEnvelope> _events =
      StreamController<WsEnvelope>.broadcast();

  WebSocket? _socket;

  @override
  bool get isConnected => _socket?.readyState == WebSocket.open;

  @override
  Future<void> connect(Uri uri) async {
    await disconnect();
    final socket = await WebSocket.connect(uri.toString());
    _socket = socket;

    socket.listen(
      _handleData,
      onDone: () {
        _events.add(const WsEnvelope(
          type: '__disconnected__',
          payload: <String, dynamic>{},
        ));
      },
      onError: (Object error) {
        _events.add(WsEnvelope(
          type: 'system.error',
          payload: <String, dynamic>{
            'message': error.toString(),
          },
        ));
      },
      cancelOnError: false,
    );
  }

  @override
  Future<void> disconnect() async {
    final socket = _socket;
    _socket = null;
    if (socket != null) {
      await socket.close();
    }
  }

  @override
  Stream<WsEnvelope> events() => _events.stream;

  @override
  Future<void> send(WsEnvelope event) async {
    final socket = _socket;
    if (socket == null || socket.readyState != WebSocket.open) {
      throw WebSocketException('WebSocket is not connected.');
    }

    socket.add(jsonEncode(event.toJson()));
  }

  void _handleData(dynamic data) {
    try {
      final rawText = switch (data) {
        String value => value,
        List<int> value => utf8.decode(value),
        _ => '',
      };
      if (rawText.isEmpty) {
        return;
      }

      final decoded = jsonDecode(rawText);
      if (decoded is Map<String, dynamic>) {
        _events.add(WsEnvelope.fromJson(decoded));
      }
    } catch (error) {
      _events.add(WsEnvelope(
        type: 'system.error',
        payload: <String, dynamic>{
          'message': error.toString(),
        },
      ));
    }
  }
}
