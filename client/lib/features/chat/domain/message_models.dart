enum DeliveryStatus {
  pending,
  accepted,
  storedOffline,
  delivered,
  read,
  expired,
  failed,
}

DeliveryStatus deliveryStatusFromWire(String value) {
  switch (value) {
    case 'accepted':
      return DeliveryStatus.accepted;
    case 'stored_offline':
      return DeliveryStatus.storedOffline;
    case 'delivered':
      return DeliveryStatus.delivered;
    case 'read':
      return DeliveryStatus.read;
    case 'expired':
      return DeliveryStatus.expired;
    case 'failed':
      return DeliveryStatus.failed;
    case 'pending':
    default:
      return DeliveryStatus.pending;
  }
}

String deliveryStatusToWire(DeliveryStatus value) {
  switch (value) {
    case DeliveryStatus.accepted:
      return 'accepted';
    case DeliveryStatus.storedOffline:
      return 'stored_offline';
    case DeliveryStatus.delivered:
      return 'delivered';
    case DeliveryStatus.read:
      return 'read';
    case DeliveryStatus.expired:
      return 'expired';
    case DeliveryStatus.failed:
      return 'failed';
    case DeliveryStatus.pending:
      return 'pending';
  }
}

extension DeliveryStatusLabel on DeliveryStatus {
  String get label {
    switch (this) {
      case DeliveryStatus.pending:
        return 'pending';
      case DeliveryStatus.accepted:
        return 'accepted';
      case DeliveryStatus.storedOffline:
        return 'stored offline';
      case DeliveryStatus.delivered:
        return 'delivered';
      case DeliveryStatus.read:
        return 'read';
      case DeliveryStatus.expired:
        return 'expired';
      case DeliveryStatus.failed:
        return 'failed';
    }
  }
}

class MessageRecord {
  const MessageRecord({
    required this.id,
    required this.conversationId,
    required this.senderUserId,
    required this.senderDeviceId,
    required this.senderEmail,
    required this.recipientUserId,
    required this.recipientDeviceId,
    required this.recipientEmail,
    required this.contentType,
    required this.ciphertext,
    required this.body,
    required this.status,
    required this.clientMessageSeq,
    required this.sentAt,
    this.storedAt,
    this.deliveredAt,
    this.readAt,
  });

  final String id;
  final String conversationId;
  final String senderUserId;
  final String senderDeviceId;
  final String senderEmail;
  final String recipientUserId;
  final String recipientDeviceId;
  final String recipientEmail;
  final String contentType;
  final String ciphertext;
  final String body;
  final DeliveryStatus status;
  final int clientMessageSeq;
  final DateTime sentAt;
  final DateTime? storedAt;
  final DateTime? deliveredAt;
  final DateTime? readAt;

  MessageRecord copyWith({
    String? id,
    String? conversationId,
    String? senderUserId,
    String? senderDeviceId,
    String? senderEmail,
    String? recipientUserId,
    String? recipientDeviceId,
    String? recipientEmail,
    String? contentType,
    String? ciphertext,
    String? body,
    DeliveryStatus? status,
    int? clientMessageSeq,
    DateTime? sentAt,
    DateTime? storedAt,
    DateTime? deliveredAt,
    DateTime? readAt,
  }) {
    return MessageRecord(
      id: id ?? this.id,
      conversationId: conversationId ?? this.conversationId,
      senderUserId: senderUserId ?? this.senderUserId,
      senderDeviceId: senderDeviceId ?? this.senderDeviceId,
      senderEmail: senderEmail ?? this.senderEmail,
      recipientUserId: recipientUserId ?? this.recipientUserId,
      recipientDeviceId: recipientDeviceId ?? this.recipientDeviceId,
      recipientEmail: recipientEmail ?? this.recipientEmail,
      contentType: contentType ?? this.contentType,
      ciphertext: ciphertext ?? this.ciphertext,
      body: body ?? this.body,
      status: status ?? this.status,
      clientMessageSeq: clientMessageSeq ?? this.clientMessageSeq,
      sentAt: sentAt ?? this.sentAt,
      storedAt: storedAt ?? this.storedAt,
      deliveredAt: deliveredAt ?? this.deliveredAt,
      readAt: readAt ?? this.readAt,
    );
  }
}

class ContactRecord {
  const ContactRecord({
    required this.peerUserId,
    required this.email,
    required this.displayName,
    required this.state,
    required this.direction,
    required this.canAccept,
    required this.canReject,
    required this.createdAt,
    required this.updatedAt,
  });

  final String peerUserId;
  final String email;
  final String displayName;
  final String state;
  final String direction;
  final bool canAccept;
  final bool canReject;
  final DateTime createdAt;
  final DateTime updatedAt;
}

class ContactSummary {
  const ContactSummary({
    required this.peerUserId,
    required this.email,
    required this.displayName,
    required this.lastMessageAt,
    required this.messageCount,
    required this.state,
    required this.direction,
    required this.canAccept,
    required this.canReject,
    required this.isServerBacked,
  });

  final String peerUserId;
  final String email;
  final String displayName;
  final DateTime lastMessageAt;
  final int messageCount;
  final String state;
  final String direction;
  final bool canAccept;
  final bool canReject;
  final bool isServerBacked;
}

class ConversationSummary {
  const ConversationSummary({
    required this.id,
    required this.peerUserId,
    required this.peerEmail,
    required this.peerDisplayName,
    required this.lastMessagePreview,
    required this.lastMessageAt,
    required this.unreadCount,
    required this.messageCount,
    required this.latestDeliveryStatus,
    required this.isServerBacked,
  });

  final String id;
  final String peerUserId;
  final String peerEmail;
  final String peerDisplayName;
  final String lastMessagePreview;
  final DateTime lastMessageAt;
  final int unreadCount;
  final int messageCount;
  final DeliveryStatus latestDeliveryStatus;
  final bool isServerBacked;
}

class ConversationRecord {
  const ConversationRecord({
    required this.id,
    required this.lastMessageId,
    required this.peerUserId,
    required this.peerEmail,
    required this.peerDisplayName,
    required this.lastMessagePreview,
    required this.lastMessageAt,
    required this.unreadCount,
    required this.messageCount,
    required this.latestDeliveryStatus,
  });

  final String id;
  final String lastMessageId;
  final String peerUserId;
  final String peerEmail;
  final String peerDisplayName;
  final String lastMessagePreview;
  final DateTime lastMessageAt;
  final int unreadCount;
  final int messageCount;
  final DeliveryStatus latestDeliveryStatus;
}

class DebugOutgoingMessageDraft {
  const DebugOutgoingMessageDraft({
    required this.messageId,
    required this.conversationId,
    required this.recipientEmail,
    required this.contentType,
    required this.clientMessageSeq,
    required this.body,
    this.recipientDeviceId = '',
  });

  final String messageId;
  final String conversationId;
  final String recipientEmail;
  final String recipientDeviceId;
  final String contentType;
  final int clientMessageSeq;
  final String body;
}

class DeliveryReceipt {
  const DeliveryReceipt({
    required this.messageId,
    required this.conversationId,
    required this.senderUserId,
    required this.senderDeviceId,
    required this.recipientUserId,
    required this.recipientDeviceId,
    required this.clientMessageSeq,
    required this.status,
    required this.ackedAt,
    required this.fromMailbox,
  });

  final String messageId;
  final String conversationId;
  final String senderUserId;
  final String senderDeviceId;
  final String recipientUserId;
  final String recipientDeviceId;
  final int clientMessageSeq;
  final DeliveryStatus status;
  final DateTime ackedAt;
  final bool fromMailbox;
}

enum ChatEventKind {
  sessionBound,
  messageMailbox,
  messageDelivered,
  deliveryAck,
  systemError,
  disconnected,
}

class ChatEvent {
  const ChatEvent({
    required this.kind,
    this.deviceId,
    this.message,
    this.receipt,
    this.errorMessage,
  });

  final ChatEventKind kind;
  final String? deviceId;
  final MessageRecord? message;
  final DeliveryReceipt? receipt;
  final String? errorMessage;
}
