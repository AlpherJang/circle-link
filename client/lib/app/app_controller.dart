import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';

import '../core/network/api_client.dart';
import '../core/storage/in_memory_token_store.dart';
import '../core/storage/token_store.dart';
import '../features/auth/data/auth_api_repository.dart';
import '../features/auth/data/auth_repository.dart';
import '../features/auth/domain/auth_models.dart';
import '../features/chat/data/chat_api_repository.dart';
import '../features/chat/data/chat_repository.dart';
import '../features/chat/domain/message_models.dart';
import '../features/devices/data/device_api_repository.dart';
import '../features/devices/data/device_repository.dart';
import '../features/devices/domain/device_models.dart';

enum AppStage {
  auth,
  verifyEmail,
  home,
}

class AppController extends ChangeNotifier {
  factory AppController({
    ApiClient? apiClient,
    TokenStore? tokenStore,
    AuthRepository? authRepository,
    DeviceRepository? deviceRepository,
    ChatRepository? chatRepository,
  }) {
    final resolvedApiClient =
        apiClient ?? const ApiClient(baseUrl: 'http://127.0.0.1:8080');
    final resolvedTokenStore = tokenStore ?? InMemoryTokenStore();

    return AppController._(
      apiClient: resolvedApiClient,
      tokenStore: resolvedTokenStore,
      authRepository: authRepository ??
          AuthApiRepository(
            apiClient: resolvedApiClient,
            tokenStore: resolvedTokenStore,
          ),
      deviceRepository: deviceRepository ??
          DeviceApiRepository(
            apiClient: resolvedApiClient,
            tokenStore: resolvedTokenStore,
          ),
      chatRepository: chatRepository ??
          ChatApiRepository(
            apiClient: resolvedApiClient,
            tokenStore: resolvedTokenStore,
          ),
    );
  }

  AppController._({
    required ApiClient apiClient,
    required TokenStore tokenStore,
    required AuthRepository authRepository,
    required DeviceRepository deviceRepository,
    required ChatRepository chatRepository,
  })  : _tokenStore = tokenStore,
        _apiClient = apiClient,
        _authRepository = authRepository,
        _deviceRepository = deviceRepository,
        _chatRepository = chatRepository {
    _chatSubscription = _chatRepository.events().listen(_handleChatEvent);
  }

  final ApiClient _apiClient;
  final TokenStore _tokenStore;
  final AuthRepository _authRepository;
  final DeviceRepository _deviceRepository;
  final ChatRepository _chatRepository;

  late final StreamSubscription<ChatEvent> _chatSubscription;

  AppStage stage = AppStage.auth;
  bool isBusy = false;
  bool isRealtimeConnected = false;
  String? errorMessage;
  String? infoMessage;
  String? pendingEmail;
  String? verificationTokenHint;
  String? currentDeviceId;
  String? selectedConversationId;
  String? selectedPeerEmail;
  AuthSession? session;
  List<DeviceRecord> devices = const <DeviceRecord>[];
  List<ContactRecord> contactRecords = const <ContactRecord>[];
  List<ConversationRecord> conversationRecords = const <ConversationRecord>[];
  List<MessageRecord> inboxMessages = const <MessageRecord>[];
  List<MessageRecord> sentMessages = const <MessageRecord>[];
  int _messageSequence = 0;

  String get baseUrl => _apiClient.baseUrl;

  DeviceRecord? get currentDevice {
    final selectedId = currentDeviceId;
    if (selectedId == null) {
      return null;
    }

    for (final device in devices) {
      if (device.id == selectedId) {
        return device;
      }
    }

    return null;
  }

  List<MessageRecord> get allMessages =>
      _sortMessages(<MessageRecord>[...sentMessages, ...inboxMessages]);

  List<ContactSummary> get contacts {
    final ownEmail = session?.user.email ?? '';
    final byEmail = <String, ContactSummary>{};

    for (final contact in contactRecords) {
      byEmail[contact.email] = ContactSummary(
        peerUserId: contact.peerUserId,
        email: contact.email,
        displayName: contact.displayName.isEmpty
            ? _displayNameFromEmail(contact.email)
            : contact.displayName,
        lastMessageAt: contact.createdAt,
        messageCount: 0,
        state: contact.state,
        direction: contact.direction,
        canAccept: contact.canAccept,
        canReject: contact.canReject,
        isServerBacked: true,
      );
    }

    for (final message in allMessages) {
      final peerEmail = _peerEmailForMessage(message, ownEmail);
      if (peerEmail.isEmpty) {
        continue;
      }

      final previous = byEmail[peerEmail];
      byEmail[peerEmail] = ContactSummary(
        peerUserId: previous?.peerUserId ?? '',
        email: peerEmail,
        displayName: previous?.displayName ?? _displayNameFromEmail(peerEmail),
        lastMessageAt: previous == null || message.sentAt.isAfter(previous.lastMessageAt)
            ? message.sentAt
            : previous.lastMessageAt,
        messageCount: (previous?.messageCount ?? 0) + 1,
        state: previous?.state ?? 'discovered',
        direction: previous?.direction ?? 'discovered',
        canAccept: previous?.canAccept ?? false,
        canReject: previous?.canReject ?? false,
        isServerBacked: previous?.isServerBacked ?? false,
      );
    }

    final result = byEmail.values.toList(growable: false);
    result.sort((left, right) => right.lastMessageAt.compareTo(left.lastMessageAt));
    return result;
  }

  List<ConversationSummary> get conversationSummaries {
    if (conversationRecords.isNotEmpty) {
      final result = conversationRecords
          .map(
            (conversation) => ConversationSummary(
              id: conversation.id,
              peerUserId: conversation.peerUserId,
              peerEmail: conversation.peerEmail,
              peerDisplayName: conversation.peerDisplayName.isEmpty
                  ? _displayNameFromEmail(conversation.peerEmail)
                  : conversation.peerDisplayName,
              lastMessagePreview: conversation.lastMessagePreview,
              lastMessageAt: conversation.lastMessageAt,
              unreadCount: conversation.unreadCount,
              messageCount: conversation.messageCount,
              latestDeliveryStatus: conversation.latestDeliveryStatus,
              isServerBacked: true,
            ),
          )
          .toList(growable: false);
      result.sort((left, right) => right.lastMessageAt.compareTo(left.lastMessageAt));
      return result;
    }

    return _deriveConversationSummariesFromMessages();
  }

  List<MessageRecord> get visibleInboxMessages => _filterMessages(inboxMessages);

  List<MessageRecord> get visibleSentMessages => _filterMessages(sentMessages);

  Future<void> signUp(SignUpCommand command) async {
    await _run(() async {
      final result = await _authRepository.signUp(command);
      pendingEmail = command.email;
      verificationTokenHint = result.verificationToken;
      infoMessage = result.emailVerificationRequired
          ? 'Account created. Verify your email before logging in.'
          : 'Account created.';
      stage = result.emailVerificationRequired ? AppStage.verifyEmail : AppStage.auth;
    });
  }

  Future<void> verifyEmail(String token) async {
    final email = pendingEmail;
    if (email == null || email.isEmpty) {
      errorMessage = 'Missing email address for verification.';
      notifyListeners();
      return;
    }

    await _run(() async {
      await _authRepository.verifyEmail(
        VerifyEmailCommand(
          email: email,
          verificationToken: token,
        ),
      );
      verificationTokenHint = null;
      infoMessage = 'Email verified. You can now log in.';
      stage = AppStage.auth;
    });
  }

  Future<void> login(LoginCommand command) async {
    await _run(() async {
      session = await _authRepository.login(command);
      pendingEmail = command.email;
      stage = AppStage.home;
      await _refreshDevicesAndMaybeConnect();
    });
  }

  Future<void> logout() async {
    final refreshToken = session?.tokens.refreshToken;
    if (refreshToken == null) {
      return;
    }

    await _run(() async {
      await _chatRepository.disconnect();
      await _authRepository.logout(refreshToken);
      await _tokenStore.clear();
      session = null;
      devices = const <DeviceRecord>[];
      contactRecords = const <ContactRecord>[];
      conversationRecords = const <ConversationRecord>[];
      inboxMessages = const <MessageRecord>[];
      sentMessages = const <MessageRecord>[];
      currentDeviceId = null;
      selectedConversationId = null;
      selectedPeerEmail = null;
      isRealtimeConnected = false;
      stage = AppStage.auth;
      infoMessage = 'Logged out.';
    });
  }

  Future<void> loadDevices() async {
    await _run(() async {
      await _refreshDevicesAndMaybeConnect();
    }, preserveMessages: true);
  }

  Future<void> loadContacts({bool preserveMessages = false}) async {
    await _run(() async {
      contactRecords = await _chatRepository.loadContacts();
      infoMessage = 'Loaded ${contactRecords.length} contacts.';
    }, preserveMessages: preserveMessages);
  }

  Future<void> loadConversations({bool preserveMessages = false}) async {
    await _run(() async {
      conversationRecords = await _chatRepository.loadConversations();
      infoMessage = 'Loaded ${conversationRecords.length} conversations.';
    }, preserveMessages: preserveMessages);
  }

  Future<void> registerCurrentDevice() async {
    await _run(() async {
      final now = DateTime.now().millisecondsSinceEpoch;
      final created = await _deviceRepository.registerDevice(
        RegisterDeviceCommand(
          deviceName: _defaultDeviceName(),
          platform: _currentPlatform(),
          pushToken: '',
          identityKeyPublic: 'identity-$now',
          signedPrekeyPublic: 'signed-$now',
          signedPrekeySignature: 'signature-$now',
          signedPrekeyVersion: 1,
          oneTimePrekeys: <String>[
            'prekey-${now}a',
            'prekey-${now}b',
            'prekey-${now}c',
          ],
        ),
      );
      devices = <DeviceRecord>[
        ...devices.where((device) => device.id != created.id),
        created,
      ];
      currentDeviceId = created.id;
      infoMessage = 'Device registered with placeholder key material.';
      await _connectRealtime();
      inboxMessages = _sortMessages(await _chatRepository.loadInbox(
        deviceId: currentDeviceId,
      ));
    });
  }

  Future<void> revokeDevice(String deviceId) async {
    await _run(() async {
      await _deviceRepository.revokeDevice(deviceId);
      if (currentDeviceId == deviceId) {
        currentDeviceId = null;
        isRealtimeConnected = false;
        await _chatRepository.disconnect();
      }
      await _refreshDevicesAndMaybeConnect();
    });
  }

  Future<void> selectDevice(String? deviceId) async {
    if (deviceId == null || deviceId == currentDeviceId) {
      return;
    }

    await _run(() async {
      currentDeviceId = deviceId;
      await _connectRealtime();
      inboxMessages = _sortMessages(await _chatRepository.loadInbox(
        deviceId: currentDeviceId,
      ));
    }, preserveMessages: true);
  }

  Future<void> inviteContact(String peerEmail) async {
    final normalizedEmail = peerEmail.trim().toLowerCase();
    if (normalizedEmail.isEmpty) {
      errorMessage = 'Enter an email address before adding a contact.';
      notifyListeners();
      return;
    }

    await _run(() async {
      final created = await _chatRepository.inviteContact(normalizedEmail);
      _upsertContactRecord(created);
      selectedPeerEmail = created.email;
      infoMessage = created.state == 'pending'
          ? 'Invite sent to ${created.email}.'
          : 'Contact added: ${created.email}';
    }, preserveMessages: true);
  }

  Future<void> acceptContact(String peerUserId) async {
    await _run(() async {
      final updated = await _chatRepository.acceptContact(peerUserId);
      _upsertContactRecord(updated);
      selectedPeerEmail = updated.email;
      infoMessage = 'Accepted contact invite from ${updated.email}.';
    }, preserveMessages: true);
  }

  Future<void> rejectContact(String peerUserId) async {
    await _run(() async {
      await _chatRepository.rejectContact(peerUserId);
      contactRecords = contactRecords
          .where((contact) => contact.peerUserId != peerUserId)
          .toList(growable: false);
      if (selectedPeerEmail != null &&
          !contactRecords.any((contact) => contact.email == selectedPeerEmail)) {
        selectedPeerEmail = null;
      }
      infoMessage = 'Invite declined.';
    }, preserveMessages: true);
  }

  void selectConversation(ConversationSummary summary) {
    selectedConversationId = summary.id;
    selectedPeerEmail = summary.peerEmail;
    infoMessage = 'Conversation selected: ${summary.peerEmail}';
    notifyListeners();
  }

  void selectContact(ContactSummary summary) {
    selectedPeerEmail = summary.email;
    final matchingConversation = conversationSummaries.where(
      (conversation) => conversation.peerEmail == summary.email,
    );
    selectedConversationId =
        matchingConversation.isEmpty ? null : matchingConversation.first.id;
    infoMessage = 'Contact selected: ${summary.email}';
    notifyListeners();
  }

  void clearSelection() {
    selectedConversationId = null;
    selectedPeerEmail = null;
    infoMessage = 'Showing all conversations.';
    notifyListeners();
  }

  Future<void> connectRealtime() async {
    await _run(() async {
      await _connectRealtime();
    }, preserveMessages: true);
  }

  Future<void> loadInbox({bool preserveMessages = false}) async {
    await _run(() async {
      inboxMessages = await _chatRepository.loadInbox(
        deviceId: currentDeviceId,
      );
      inboxMessages = _sortMessages(inboxMessages);
      infoMessage = 'Loaded ${inboxMessages.length} inbox messages.';
    }, preserveMessages: preserveMessages);
  }

  Future<void> sendMessage({
    required String recipientEmail,
    required String body,
  }) async {
    final authSession = session;
    final deviceId = currentDeviceId;
    if (authSession == null) {
      errorMessage = 'Log in before sending messages.';
      notifyListeners();
      return;
    }
    if (deviceId == null || deviceId.isEmpty) {
      errorMessage = 'Register or select a device before sending messages.';
      notifyListeners();
      return;
    }

    await _run(() async {
      _messageSequence += 1;
      final now = DateTime.now();
      final resolvedConversationId =
          selectedConversationId ?? 'conv_$recipientEmail';
      final draft = DebugOutgoingMessageDraft(
        messageId: 'msg_${now.millisecondsSinceEpoch}_$_messageSequence',
        conversationId: resolvedConversationId,
        recipientEmail: recipientEmail,
        contentType: 'text/plain',
        clientMessageSeq: _messageSequence,
        body: body,
      );

      final optimistic = MessageRecord(
        id: draft.messageId,
        conversationId: draft.conversationId,
        senderUserId: authSession.user.id,
        senderDeviceId: deviceId,
        senderEmail: authSession.user.email,
        recipientUserId: '',
        recipientDeviceId: draft.recipientDeviceId,
        recipientEmail: recipientEmail,
        contentType: draft.contentType,
        ciphertext: base64Encode(utf8.encode(body)),
        body: body,
        status: DeliveryStatus.pending,
        clientMessageSeq: draft.clientMessageSeq,
        sentAt: now,
      );
      _upsertSentMessage(optimistic);
      selectedConversationId = resolvedConversationId;
      selectedPeerEmail = recipientEmail;

      final response = await _chatRepository.sendDebugMessage(draft);
      if (response != null) {
        _upsertSentMessage(response);
        infoMessage = 'Message stored through HTTP fallback.';
      } else {
        infoMessage = 'Message queued for realtime relay.';
      }
    }, preserveMessages: true);
  }

  Future<void> markMessageRead(String messageId) async {
    final item = _messageById(inboxMessages, messageId);
    if (item == null || item.status == DeliveryStatus.read) {
      return;
    }

    await _run(() async {
      await _chatRepository.acknowledgeMessage(
        messageId: messageId,
        status: DeliveryStatus.read,
      );
      _upsertInboxMessage(item.copyWith(
        status: DeliveryStatus.read,
        readAt: DateTime.now(),
        deliveredAt: item.deliveredAt ?? DateTime.now(),
      ));
      infoMessage = 'Read receipt sent.';
    }, preserveMessages: true);
  }

  @override
  void dispose() {
    unawaited(_chatSubscription.cancel());
    unawaited(_chatRepository.disconnect());
    super.dispose();
  }

  Future<void> _refreshDevicesAndMaybeConnect() async {
    devices = await _deviceRepository.listDevices();
    contactRecords = await _chatRepository.loadContacts();
    conversationRecords = await _chatRepository.loadConversations();
    currentDeviceId = _reconcileCurrentDeviceId(currentDeviceId, devices);
    if (currentDeviceId != null) {
      await _connectRealtime();
      inboxMessages = _sortMessages(await _chatRepository.loadInbox(
        deviceId: currentDeviceId,
      ));
    } else {
      await _chatRepository.disconnect();
      isRealtimeConnected = false;
      inboxMessages = const <MessageRecord>[];
      infoMessage = 'Register a device to enable realtime messaging.';
    }
  }

  Future<void> _connectRealtime() async {
    final authSession = session;
    final deviceId = currentDeviceId;
    if (authSession == null || deviceId == null || deviceId.isEmpty) {
      isRealtimeConnected = false;
      return;
    }

    isRealtimeConnected = false;
    notifyListeners();
    await _chatRepository.connectSession(
      userId: authSession.user.id,
      deviceId: deviceId,
    );
  }

  void _handleChatEvent(ChatEvent event) {
    switch (event.kind) {
      case ChatEventKind.sessionBound:
        isRealtimeConnected = true;
        infoMessage = 'Realtime bound to ${event.deviceId ?? currentDeviceId ?? ''}.';
        break;
      case ChatEventKind.messageMailbox:
        if (event.message != null) {
          _upsertInboxMessage(event.message!);
        }
        break;
      case ChatEventKind.messageDelivered:
        if (event.message != null) {
          final delivered = event.message!.copyWith(
            status: DeliveryStatus.delivered,
            deliveredAt: event.message!.deliveredAt ?? DateTime.now(),
          );
          _upsertInboxMessage(delivered);
          unawaited(_chatRepository.acknowledgeMessage(
            messageId: delivered.id,
            status: DeliveryStatus.delivered,
          ));
        }
        break;
      case ChatEventKind.deliveryAck:
        final receipt = event.receipt;
        if (receipt != null) {
          final current = _messageById(sentMessages, receipt.messageId);
          if (current != null) {
            _upsertSentMessage(current.copyWith(
              status: receipt.status,
              deliveredAt: receipt.status == DeliveryStatus.delivered ||
                      receipt.status == DeliveryStatus.read
                  ? receipt.ackedAt
                  : current.deliveredAt,
              readAt: receipt.status == DeliveryStatus.read
                  ? receipt.ackedAt
                  : current.readAt,
              recipientDeviceId: receipt.recipientDeviceId,
            ));
          }
          infoMessage =
              'Message ${receipt.messageId} is ${receipt.status.label}.';
        }
        break;
      case ChatEventKind.systemError:
        errorMessage = event.errorMessage ?? 'Unknown realtime error.';
        break;
      case ChatEventKind.disconnected:
        isRealtimeConnected = false;
        infoMessage = 'Realtime disconnected.';
        break;
    }
    notifyListeners();
  }

  Future<void> _run(
    Future<void> Function() action, {
    bool preserveMessages = false,
  }) async {
    isBusy = true;
    if (!preserveMessages) {
      errorMessage = null;
      infoMessage = null;
    }
    notifyListeners();

    try {
      await action();
    } on ApiException catch (error) {
      errorMessage = error.message;
    } catch (error) {
      errorMessage = error.toString();
    } finally {
      isBusy = false;
      notifyListeners();
    }
  }

  void _upsertInboxMessage(MessageRecord message) {
    final previous = _messageById(inboxMessages, message.id);
    inboxMessages = _upsertMessageList(inboxMessages, message);
    _syncConversationFromMessage(message, previous: previous);
  }

  void _upsertSentMessage(MessageRecord message) {
    final previous = _messageById(sentMessages, message.id);
    sentMessages = _upsertMessageList(sentMessages, message);
    _syncConversationFromMessage(message, previous: previous);
  }

  List<MessageRecord> _upsertMessageList(
    List<MessageRecord> source,
    MessageRecord message,
  ) {
    final items = <MessageRecord>[...source];
    final index = items.indexWhere((item) => item.id == message.id);
    if (index == -1) {
      items.insert(0, message);
    } else {
      items[index] = message;
    }
    return _sortMessages(items);
  }

  List<MessageRecord> _sortMessages(List<MessageRecord> items) {
    final sorted = <MessageRecord>[...items];
    sorted.sort((left, right) => right.sentAt.compareTo(left.sentAt));
    return sorted;
  }

  void _upsertContactRecord(ContactRecord contact) {
    final items = <ContactRecord>[...contactRecords];
    final index = items.indexWhere((item) => item.email == contact.email);
    if (index == -1) {
      items.insert(0, contact);
    } else {
      items[index] = contact;
    }
    contactRecords = items;
  }

  List<ConversationSummary> _deriveConversationSummariesFromMessages() {
    final ownEmail = session?.user.email ?? '';
    final byConversation = <String, ConversationSummary>{};

    for (final message in allMessages) {
      final peerEmail = _peerEmailForMessage(message, ownEmail);
      if (peerEmail.isEmpty) {
        continue;
      }

      final preview = message.body.isEmpty ? '[encrypted payload]' : message.body;
      final previous = byConversation[message.conversationId];
      final unreadIncrement = _isUnreadIncomingMessage(message, ownEmail) ? 1 : 0;

      if (previous == null) {
        byConversation[message.conversationId] = ConversationSummary(
          id: message.conversationId,
          peerUserId: '',
          peerEmail: peerEmail,
          peerDisplayName: _displayNameFromEmail(peerEmail),
          lastMessagePreview: preview,
          lastMessageAt: message.sentAt,
          unreadCount: unreadIncrement,
          messageCount: 1,
          latestDeliveryStatus: message.status,
          isServerBacked: false,
        );
        continue;
      }

      final latestPreview = message.sentAt.isAfter(previous.lastMessageAt)
          ? preview
          : previous.lastMessagePreview;
      final latestTime = message.sentAt.isAfter(previous.lastMessageAt)
          ? message.sentAt
          : previous.lastMessageAt;

      byConversation[message.conversationId] = ConversationSummary(
        id: previous.id,
        peerUserId: previous.peerUserId,
        peerEmail: previous.peerEmail,
        peerDisplayName: previous.peerDisplayName,
        lastMessagePreview: latestPreview,
        lastMessageAt: latestTime,
        unreadCount: previous.unreadCount + unreadIncrement,
        messageCount: previous.messageCount + 1,
        latestDeliveryStatus: message.sentAt.isAfter(previous.lastMessageAt)
            ? message.status
            : previous.latestDeliveryStatus,
        isServerBacked: previous.isServerBacked,
      );
    }

    final result = byConversation.values.toList(growable: false);
    result.sort((left, right) => right.lastMessageAt.compareTo(left.lastMessageAt));
    return result;
  }

  void _syncConversationFromMessage(
    MessageRecord message, {
    required MessageRecord? previous,
  }) {
    final ownEmail = session?.user.email ?? '';
    final peerEmail = _peerEmailForMessage(message, ownEmail);
    if (peerEmail.isEmpty) {
      return;
    }

    final items = <ConversationRecord>[...conversationRecords];
    final index = items.indexWhere((item) => item.id == message.conversationId);
    final unreadDelta =
        _unreadFlagForMessage(message, ownEmail) - _unreadFlagForMessage(previous, ownEmail);

    if (index == -1) {
      items.insert(
        0,
        ConversationRecord(
          id: message.conversationId,
          lastMessageId: message.id,
          peerUserId: '',
          peerEmail: peerEmail,
          peerDisplayName: _displayNameFromEmail(peerEmail),
          lastMessagePreview: message.body.isEmpty ? '[encrypted payload]' : message.body,
          lastMessageAt: message.sentAt,
          unreadCount: _unreadFlagForMessage(message, ownEmail),
          messageCount: 1,
          latestDeliveryStatus: message.status,
        ),
      );
      conversationRecords = items;
      return;
    }

    final current = items[index];
    final isLatestMessage = current.lastMessageId == message.id ||
        message.sentAt.isAfter(current.lastMessageAt) ||
        message.sentAt.isAtSameMomentAs(current.lastMessageAt);

    items[index] = ConversationRecord(
      id: current.id,
      lastMessageId: isLatestMessage ? message.id : current.lastMessageId,
      peerUserId: current.peerUserId,
      peerEmail: current.peerEmail.isEmpty ? peerEmail : current.peerEmail,
      peerDisplayName: current.peerDisplayName.isEmpty
          ? _displayNameFromEmail(peerEmail)
          : current.peerDisplayName,
      lastMessagePreview: isLatestMessage
          ? (message.body.isEmpty ? '[encrypted payload]' : message.body)
          : current.lastMessagePreview,
      lastMessageAt: isLatestMessage ? message.sentAt : current.lastMessageAt,
      unreadCount:
          ((current.unreadCount + unreadDelta).clamp(0, 1 << 20)) as int,
      messageCount: previous == null ? current.messageCount + 1 : current.messageCount,
      latestDeliveryStatus:
          isLatestMessage ? message.status : current.latestDeliveryStatus,
    );

    items.sort((left, right) => right.lastMessageAt.compareTo(left.lastMessageAt));
    conversationRecords = items;
  }

  bool _isUnreadIncomingMessage(MessageRecord message, String ownEmail) {
    return _unreadFlagForMessage(message, ownEmail) == 1;
  }

  int _unreadFlagForMessage(MessageRecord? message, String ownEmail) {
    if (message == null) {
      return 0;
    }
    if (message.senderEmail.isEmpty || message.senderEmail == ownEmail) {
      return 0;
    }
    if (message.status == DeliveryStatus.read) {
      return 0;
    }
    return 1;
  }

  MessageRecord? _messageById(List<MessageRecord> items, String messageId) {
    for (final item in items) {
      if (item.id == messageId) {
        return item;
      }
    }
    return null;
  }

  List<MessageRecord> _filterMessages(List<MessageRecord> source) {
    final conversationId = selectedConversationId;
    final peerEmail = selectedPeerEmail;
    if ((conversationId == null || conversationId.isEmpty) &&
        (peerEmail == null || peerEmail.isEmpty)) {
      return source;
    }

    return source.where((message) {
      final matchesConversation =
          conversationId == null || conversationId.isEmpty || message.conversationId == conversationId;
      final matchesPeer = peerEmail == null ||
          peerEmail.isEmpty ||
          message.senderEmail == peerEmail ||
          message.recipientEmail == peerEmail;
      return matchesConversation && matchesPeer;
    }).toList(growable: false);
  }

  String _peerEmailForMessage(MessageRecord message, String ownEmail) {
    if (message.senderEmail.isNotEmpty && message.senderEmail != ownEmail) {
      return message.senderEmail;
    }
    if (message.recipientEmail.isNotEmpty && message.recipientEmail != ownEmail) {
      return message.recipientEmail;
    }
    return '';
  }

  String _displayNameFromEmail(String email) {
    final atIndex = email.indexOf('@');
    if (atIndex <= 0) {
      return email;
    }
    return email.substring(0, atIndex);
  }

  String? _reconcileCurrentDeviceId(
    String? current,
    List<DeviceRecord> availableDevices,
  ) {
    if (current != null) {
      for (final device in availableDevices) {
        if (device.id == current && device.revokedAt == null) {
          return current;
        }
      }
    }

    for (final device in availableDevices) {
      if (device.revokedAt == null) {
        return device.id;
      }
    }

    return null;
  }

  DevicePlatform _currentPlatform() {
    switch (defaultTargetPlatform) {
      case TargetPlatform.iOS:
        return DevicePlatform.ios;
      case TargetPlatform.android:
        return DevicePlatform.android;
      case TargetPlatform.macOS:
      default:
        return DevicePlatform.macos;
    }
  }

  String _defaultDeviceName() {
    switch (_currentPlatform()) {
      case DevicePlatform.ios:
        return 'circle-link iPhone';
      case DevicePlatform.android:
        return 'circle-link Android';
      case DevicePlatform.macos:
        return 'circle-link Mac';
    }
  }
}
