import '../domain/message_models.dart';

abstract interface class ChatRepository {
  Future<List<ContactRecord>> loadContacts();
  Future<List<ConversationRecord>> loadConversations();
  Future<ContactRecord> inviteContact(String peerEmail);
  Future<ContactRecord> acceptContact(String peerUserId);
  Future<void> rejectContact(String peerUserId);
  Future<List<MessageRecord>> loadInbox({String? deviceId});
  Future<void> connectSession({
    required String userId,
    required String deviceId,
  });
  Future<void> disconnect();
  Future<MessageRecord?> sendDebugMessage(DebugOutgoingMessageDraft draft);
  Future<void> acknowledgeMessage({
    required String messageId,
    required DeliveryStatus status,
  });
  Stream<ChatEvent> events();
}
