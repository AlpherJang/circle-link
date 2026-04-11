enum DeliveryStatus {
  pending,
  sent,
  storedOffline,
  delivered,
  read,
  expired,
  failed,
}

enum RetentionMode {
  persistent,
  disappearing,
}

class DisappearingPolicy {
  const DisappearingPolicy({
    required this.mode,
    this.disappearAfterSeconds,
  });

  final RetentionMode mode;
  final int? disappearAfterSeconds;
}

class ConversationSummary {
  const ConversationSummary({
    required this.id,
    required this.ownerUserId,
    required this.peerUserId,
    required this.unreadCount,
    required this.retentionDefault,
    this.lastMessageAt,
    this.lastMessagePreview,
  });

  final String id;
  final String ownerUserId;
  final String peerUserId;
  final int unreadCount;
  final RetentionMode retentionDefault;
  final DateTime? lastMessageAt;
  final String? lastMessagePreview;
}

class MessageRecord {
  const MessageRecord({
    required this.id,
    required this.conversationId,
    required this.senderUserId,
    required this.senderDeviceId,
    required this.recipientUserId,
    required this.recipientDeviceId,
    required this.contentType,
    required this.ciphertext,
    required this.status,
    required this.retentionMode,
    required this.sentAt,
    this.disappearAfterSeconds,
    this.expiresAt,
    this.deliveredAt,
    this.readAt,
  });

  final String id;
  final String conversationId;
  final String senderUserId;
  final String senderDeviceId;
  final String recipientUserId;
  final String recipientDeviceId;
  final String contentType;
  final List<int> ciphertext;
  final DeliveryStatus status;
  final RetentionMode retentionMode;
  final DateTime sentAt;
  final int? disappearAfterSeconds;
  final DateTime? expiresAt;
  final DateTime? deliveredAt;
  final DateTime? readAt;
}

class OutgoingMessageDraft {
  const OutgoingMessageDraft({
    required this.conversationId,
    required this.recipientUserId,
    required this.recipientDeviceId,
    required this.contentType,
    required this.plaintext,
    required this.offlineEligible,
    required this.policy,
  });

  final String conversationId;
  final String recipientUserId;
  final String recipientDeviceId;
  final String contentType;
  final String plaintext;
  final bool offlineEligible;
  final DisappearingPolicy policy;
}

class DeliveryReceipt {
  const DeliveryReceipt({
    required this.messageId,
    required this.recipientDeviceId,
    required this.status,
    required this.ackedAt,
    required this.fromMailbox,
  });

  final String messageId;
  final String recipientDeviceId;
  final DeliveryStatus status;
  final DateTime ackedAt;
  final bool fromMailbox;
}
