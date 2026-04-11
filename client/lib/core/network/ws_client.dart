enum WsEventType {
  sessionBind('session.bind'),
  presenceSnapshot('presence.snapshot'),
  presenceUpdate('presence.update'),
  messageSend('message.send'),
  messageDeliver('message.deliver'),
  messageMailbox('message.mailbox'),
  messageAck('message.ack'),
  messageNack('message.nack'),
  mailboxDrained('mailbox.drained'),
  systemError('system.error');

  const WsEventType(this.value);

  final String value;
}

class WsEnvelope<T> {
  const WsEnvelope({
    required this.type,
    required this.payload,
    this.traceId,
  });

  final WsEventType type;
  final T payload;
  final String? traceId;
}

abstract interface class WsClient {
  Future<void> connect();
  Future<void> disconnect();
  Future<void> send<T>(WsEnvelope<T> event);
  Stream<WsEnvelope<Object?>> events();
}
