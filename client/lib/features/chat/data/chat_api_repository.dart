import 'dart:async';
import 'dart:convert';

import '../../../core/network/api_client.dart';
import '../../../core/network/ws_client.dart';
import '../../../core/storage/token_store.dart';
import '../domain/message_models.dart';
import 'chat_repository.dart';

class ChatApiRepository implements ChatRepository {
  ChatApiRepository({
    required this.apiClient,
    required this.tokenStore,
    WsClient? wsClient,
  }) : _wsClient = wsClient ?? IoWsClient() {
    _subscription = _wsClient.events().listen(_handleEnvelope);
  }

  final ApiClient apiClient;
  final TokenStore tokenStore;
  final WsClient _wsClient;
  final StreamController<ChatEvent> _events =
      StreamController<ChatEvent>.broadcast();
  late final StreamSubscription<WsEnvelope> _subscription;

  @override
  Future<void> acknowledgeMessage({
    required String messageId,
    required DeliveryStatus status,
  }) async {
    await _wsClient.send(WsEnvelope(
      type: 'message.ack',
      payload: <String, dynamic>{
        'messageId': messageId,
        'status': deliveryStatusToWire(status),
      },
    ));
  }

  @override
  Future<ContactRecord> acceptContact(String peerUserId) async {
    final accessToken = await tokenStore.readAccessToken();
    final response = await apiClient.postJson(
      '/v1/contacts/$peerUserId/accept',
      accessToken: accessToken,
    );
    return _contactFromJson(_requireData(response));
  }

  @override
  Future<ContactRecord> inviteContact(String peerEmail) async {
    final accessToken = await tokenStore.readAccessToken();
    final response = await apiClient.postJson(
      '/v1/contacts/invite',
      accessToken: accessToken,
      body: <String, dynamic>{
        'peerEmail': peerEmail,
      },
    );
    return _contactFromJson(_requireData(response));
  }

  @override
  Future<void> connectSession({
    required String userId,
    required String deviceId,
  }) async {
    final accessToken = await tokenStore.readAccessToken();
    if (accessToken == null || accessToken.isEmpty) {
      throw const ApiException(
        code: 'AUTH_UNAUTHORIZED',
        message: 'Missing access token for websocket session.',
      );
    }

    await _wsClient.connect(_webSocketUri());
    await _wsClient.send(WsEnvelope(
      type: 'session.bind',
      payload: <String, dynamic>{
        'accessToken': accessToken,
        'userId': userId,
        'deviceId': deviceId,
      },
    ));
  }

  @override
  Future<void> disconnect() => _wsClient.disconnect();

  @override
  Stream<ChatEvent> events() => _events.stream;

  @override
  Future<void> rejectContact(String peerUserId) async {
    final accessToken = await tokenStore.readAccessToken();
    final response = await apiClient.postJson(
      '/v1/contacts/$peerUserId/reject',
      accessToken: accessToken,
    );
    _throwIfError(response);
  }

  @override
  Future<List<ConversationRecord>> loadConversations() async {
    final accessToken = await tokenStore.readAccessToken();
    final response = await apiClient.getJson(
      '/v1/conversations',
      accessToken: accessToken,
    );
    final data = _requireData(response);
    final items = data['items'];
    if (items is! List) {
      return const <ConversationRecord>[];
    }

    return items
        .whereType<Map<String, dynamic>>()
        .map(_conversationFromJson)
        .toList(growable: false);
  }

  @override
  Future<List<ContactRecord>> loadContacts() async {
    final accessToken = await tokenStore.readAccessToken();
    final response = await apiClient.getJson(
      '/v1/contacts',
      accessToken: accessToken,
    );
    final data = _requireData(response);
    final items = data['items'];
    if (items is! List) {
      return const <ContactRecord>[];
    }

    return items
        .whereType<Map<String, dynamic>>()
        .map(_contactFromJson)
        .toList(growable: false);
  }

  @override
  Future<List<MessageRecord>> loadInbox({String? deviceId}) async {
    final accessToken = await tokenStore.readAccessToken();
    final path = deviceId == null || deviceId.isEmpty
        ? '/v1/messages'
        : '/v1/messages?deviceId=${Uri.encodeQueryComponent(deviceId)}';
    final response = await apiClient.getJson(
      path,
      accessToken: accessToken,
    );
    final data = _requireData(response);
    final items = data['items'];
    if (items is! List) {
      return const <MessageRecord>[];
    }

    return items
        .whereType<Map<String, dynamic>>()
        .map(_messageFromJson)
        .toList(growable: false);
  }

  @override
  Future<MessageRecord?> sendDebugMessage(DebugOutgoingMessageDraft draft) async {
    final payload = <String, dynamic>{
      'messageId': draft.messageId,
      'conversationId': draft.conversationId,
      'recipientEmail': draft.recipientEmail,
      'recipientDeviceId': draft.recipientDeviceId,
      'contentType': draft.contentType,
      'clientMessageSeq': draft.clientMessageSeq,
      'header': <String, dynamic>{
        'scheme': 'debug-placeholder',
        'encoding': 'debug-base64-utf8',
        'version': 1,
      },
      'ratchetPublicKey':
          'debug-rpk-${draft.recipientEmail}-${DateTime.now().millisecondsSinceEpoch}',
      'ciphertext': base64Encode(utf8.encode(draft.body)),
    };

    if (_wsClient.isConnected) {
      await _wsClient.send(WsEnvelope(
        type: 'message.send',
        payload: payload,
      ));
      return null;
    }

    final accessToken = await tokenStore.readAccessToken();
    final response = await apiClient.postJson(
      '/v1/messages',
      accessToken: accessToken,
      body: payload,
    );
    return _messageFromJson(_requireData(response));
  }

  void _handleEnvelope(WsEnvelope envelope) {
    switch (envelope.type) {
      case 'session.bound':
        _events.add(ChatEvent(
          kind: ChatEventKind.sessionBound,
          deviceId: envelope.payload['deviceId'] as String?,
        ));
        return;
      case 'message.mailbox':
        _events.add(ChatEvent(
          kind: ChatEventKind.messageMailbox,
          message: _messageFromJson(envelope.payload),
        ));
        return;
      case 'message.deliver':
        _events.add(ChatEvent(
          kind: ChatEventKind.messageDelivered,
          message: _messageFromJson(envelope.payload),
        ));
        return;
      case 'delivery.ack':
        _events.add(ChatEvent(
          kind: ChatEventKind.deliveryAck,
          receipt: _receiptFromJson(envelope.payload),
        ));
        return;
      case 'system.error':
        _events.add(ChatEvent(
          kind: ChatEventKind.systemError,
          errorMessage: envelope.payload['message'] as String? ??
              'Unknown websocket error.',
        ));
        return;
      case '__disconnected__':
        _events.add(const ChatEvent(kind: ChatEventKind.disconnected));
        return;
      default:
        return;
    }
  }

  MessageRecord _messageFromJson(Map<String, dynamic> json) {
    return MessageRecord(
      id: json['messageId'] as String? ?? '',
      conversationId: json['conversationId'] as String? ?? '',
      senderUserId: json['senderUserId'] as String? ?? '',
      senderDeviceId: json['senderDeviceId'] as String? ?? '',
      senderEmail: json['senderEmail'] as String? ?? '',
      recipientUserId: json['recipientUserId'] as String? ?? '',
      recipientDeviceId: json['recipientDeviceId'] as String? ?? '',
      recipientEmail: json['recipientEmail'] as String? ?? '',
      contentType: json['contentType'] as String? ?? 'text/plain',
      ciphertext: json['ciphertext'] as String? ?? '',
      body: json['body'] as String? ?? '',
      status: deliveryStatusFromWire(
        json['deliveryStatus'] as String? ?? 'pending',
      ),
      clientMessageSeq: (json['clientMessageSeq'] as num?)?.toInt() ?? 0,
      sentAt: DateTime.tryParse(json['sentAt'] as String? ?? '') ??
          DateTime.now(),
      storedAt: DateTime.tryParse(json['storedAt'] as String? ?? ''),
      deliveredAt: DateTime.tryParse(json['deliveredAt'] as String? ?? ''),
      readAt: DateTime.tryParse(json['readAt'] as String? ?? ''),
    );
  }

  DeliveryReceipt _receiptFromJson(Map<String, dynamic> json) {
    return DeliveryReceipt(
      messageId: json['messageId'] as String? ?? '',
      conversationId: json['conversationId'] as String? ?? '',
      senderUserId: json['senderUserId'] as String? ?? '',
      senderDeviceId: json['senderDeviceId'] as String? ?? '',
      recipientUserId: json['recipientUserId'] as String? ?? '',
      recipientDeviceId: json['recipientDeviceId'] as String? ?? '',
      clientMessageSeq: (json['clientMessageSeq'] as num?)?.toInt() ?? 0,
      status: deliveryStatusFromWire(
        json['status'] as String? ?? 'pending',
      ),
      ackedAt: DateTime.tryParse(json['ackedAt'] as String? ?? '') ??
          DateTime.now(),
      fromMailbox: json['fromMailbox'] as bool? ?? false,
    );
  }

  ContactRecord _contactFromJson(Map<String, dynamic> json) {
    return ContactRecord(
      peerUserId: json['peerUserId'] as String? ?? '',
      email: json['peerEmail'] as String? ?? '',
      displayName: json['displayName'] as String? ?? '',
      state: json['state'] as String? ?? 'accepted',
      direction: json['direction'] as String? ?? 'accepted',
      canAccept: json['canAccept'] as bool? ?? false,
      canReject: json['canReject'] as bool? ?? false,
      createdAt: DateTime.tryParse(json['createdAt'] as String? ?? '') ??
          DateTime.now(),
      updatedAt: DateTime.tryParse(json['updatedAt'] as String? ?? '') ??
          DateTime.now(),
    );
  }

  ConversationRecord _conversationFromJson(Map<String, dynamic> json) {
    return ConversationRecord(
      id: json['conversationId'] as String? ?? '',
      lastMessageId: json['lastMessageId'] as String? ?? '',
      peerUserId: json['peerUserId'] as String? ?? '',
      peerEmail: json['peerEmail'] as String? ?? '',
      peerDisplayName: json['peerDisplayName'] as String? ?? '',
      lastMessagePreview: json['lastMessagePreview'] as String? ?? '',
      lastMessageAt: DateTime.tryParse(json['lastMessageAt'] as String? ?? '') ??
          DateTime.now(),
      unreadCount: (json['unreadCount'] as num?)?.toInt() ?? 0,
      messageCount: (json['messageCount'] as num?)?.toInt() ?? 0,
      latestDeliveryStatus: deliveryStatusFromWire(
        json['latestDeliveryStatus'] as String? ?? 'pending',
      ),
    );
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

  Uri _webSocketUri() {
    final base = Uri.parse(apiClient.baseUrl);
    return Uri(
      scheme: base.scheme == 'https' ? 'wss' : 'ws',
      host: base.host,
      port: base.hasPort ? base.port : null,
      path: '/v1/ws',
    );
  }
}
