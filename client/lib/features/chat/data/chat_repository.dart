import '../domain/message_models.dart';

abstract interface class ChatRepository {
  Future<void> sendMessage(OutgoingMessageDraft draft);
  Future<void> acknowledgeDelivery(DeliveryReceipt receipt);
  Future<void> expireMessage(String messageId);
  Stream<List<MessageRecord>> watchConversation(String conversationId);
}
