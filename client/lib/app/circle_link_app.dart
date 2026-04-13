import 'package:flutter/material.dart';

import '../features/auth/domain/auth_models.dart';
import '../features/chat/domain/message_models.dart';
import '../features/devices/domain/device_models.dart';
import 'app_controller.dart';

class CircleLinkApp extends StatefulWidget {
  const CircleLinkApp({super.key});

  @override
  State<CircleLinkApp> createState() => _CircleLinkAppState();
}

class _CircleLinkAppState extends State<CircleLinkApp> {
  late final AppController _controller;

  @override
  void initState() {
    super.initState();
    _controller = AppController();
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'circle-link',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFF1E5B4F)),
        inputDecorationTheme: const InputDecorationTheme(
          border: OutlineInputBorder(),
        ),
        cardTheme: const CardTheme(
          elevation: 0,
          margin: EdgeInsets.zero,
        ),
      ),
      home: _AppShell(controller: _controller),
    );
  }
}

class _AppShell extends StatelessWidget {
  const _AppShell({
    required this.controller,
  });

  final AppController controller;

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: controller,
      builder: (context, _) {
        switch (controller.stage) {
          case AppStage.auth:
            return _AuthScreen(controller: controller);
          case AppStage.verifyEmail:
            return _VerifyEmailScreen(controller: controller);
          case AppStage.home:
            return _HomeScreen(controller: controller);
        }
      },
    );
  }
}

class _AuthScreen extends StatefulWidget {
  const _AuthScreen({
    required this.controller,
  });

  final AppController controller;

  @override
  State<_AuthScreen> createState() => _AuthScreenState();
}

class _AuthScreenState extends State<_AuthScreen> {
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  final _displayNameController = TextEditingController();
  bool _isLoginMode = true;

  @override
  void dispose() {
    _emailController.dispose();
    _passwordController.dispose();
    _displayNameController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final controller = widget.controller;
    final theme = Theme.of(context);

    return Scaffold(
      body: Container(
        decoration: const BoxDecoration(
          gradient: LinearGradient(
            colors: <Color>[Color(0xFFF2EFE8), Color(0xFFD7E5E1)],
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
          ),
        ),
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 460),
            child: Card(
              margin: const EdgeInsets.all(24),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(28),
              ),
              child: Padding(
                padding: const EdgeInsets.all(28),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: <Widget>[
                    Text(
                      'circle-link',
                      style: theme.textTheme.headlineMedium,
                    ),
                    const SizedBox(height: 8),
                    Text(
                      _isLoginMode
                          ? 'Sign in to your private relay.'
                          : 'Create a local test account.',
                    ),
                    const SizedBox(height: 12),
                    Text(
                      'Server: ${controller.baseUrl}',
                      style: theme.textTheme.bodySmall,
                    ),
                    const SizedBox(height: 24),
                    if (!_isLoginMode) ...<Widget>[
                      TextField(
                        controller: _displayNameController,
                        decoration: const InputDecoration(
                          labelText: 'Display name',
                        ),
                      ),
                      const SizedBox(height: 12),
                    ],
                    TextField(
                      controller: _emailController,
                      decoration: const InputDecoration(labelText: 'Email'),
                      keyboardType: TextInputType.emailAddress,
                    ),
                    const SizedBox(height: 12),
                    TextField(
                      controller: _passwordController,
                      decoration: const InputDecoration(labelText: 'Password'),
                      obscureText: true,
                    ),
                    const SizedBox(height: 16),
                    _FeedbackBlock(controller: controller),
                    const SizedBox(height: 16),
                    FilledButton(
                      onPressed: controller.isBusy ? null : _submit,
                      child: Text(
                        controller.isBusy
                            ? 'Please wait...'
                            : _isLoginMode
                                ? 'Login'
                                : 'Create account',
                      ),
                    ),
                    const SizedBox(height: 12),
                    TextButton(
                      onPressed: controller.isBusy
                          ? null
                          : () {
                              setState(() {
                                _isLoginMode = !_isLoginMode;
                              });
                            },
                      child: Text(
                        _isLoginMode
                            ? 'Need an account? Sign up'
                            : 'Already have an account? Login',
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }

  void _submit() {
    if (_isLoginMode) {
      widget.controller.login(
        LoginCommand(
          email: _emailController.text.trim(),
          password: _passwordController.text,
        ),
      );
      return;
    }

    widget.controller.signUp(
      SignUpCommand(
        email: _emailController.text.trim(),
        password: _passwordController.text,
        displayName: _displayNameController.text.trim(),
      ),
    );
  }
}

class _VerifyEmailScreen extends StatefulWidget {
  const _VerifyEmailScreen({
    required this.controller,
  });

  final AppController controller;

  @override
  State<_VerifyEmailScreen> createState() => _VerifyEmailScreenState();
}

class _VerifyEmailScreenState extends State<_VerifyEmailScreen> {
  final _tokenController = TextEditingController();

  @override
  void initState() {
    super.initState();
    final hint = widget.controller.verificationTokenHint;
    if (hint != null) {
      _tokenController.text = hint;
    }
  }

  @override
  void dispose() {
    _tokenController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final controller = widget.controller;

    return Scaffold(
      appBar: AppBar(title: const Text('Verify email')),
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 460),
          child: Card(
            margin: const EdgeInsets.all(24),
            child: Padding(
              padding: const EdgeInsets.all(28),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: <Widget>[
                  Text(
                    'Verify ${controller.pendingEmail ?? ''}',
                    style: Theme.of(context).textTheme.headlineSmall,
                  ),
                  const SizedBox(height: 12),
                  const Text(
                    'The current dev server returns a verification token directly until email sending is implemented.',
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: _tokenController,
                    decoration: const InputDecoration(
                      labelText: 'Verification token',
                    ),
                  ),
                  const SizedBox(height: 16),
                  _FeedbackBlock(controller: controller),
                  const SizedBox(height: 16),
                  FilledButton(
                    onPressed: controller.isBusy
                        ? null
                        : () => controller.verifyEmail(_tokenController.text),
                    child: Text(controller.isBusy ? 'Please wait...' : 'Verify'),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _HomeScreen extends StatefulWidget {
  const _HomeScreen({
    required this.controller,
  });

  final AppController controller;

  @override
  State<_HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<_HomeScreen> {
  final _recipientController = TextEditingController();
  final _bodyController = TextEditingController();

  @override
  void didUpdateWidget(covariant _HomeScreen oldWidget) {
    super.didUpdateWidget(oldWidget);
    final selectedPeerEmail = widget.controller.selectedPeerEmail;
    if (selectedPeerEmail != null &&
        selectedPeerEmail.isNotEmpty &&
        _recipientController.text != selectedPeerEmail) {
      _recipientController.text = selectedPeerEmail;
    }
  }

  @override
  void dispose() {
    _recipientController.dispose();
    _bodyController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final controller = widget.controller;
    final session = controller.session;
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: const Text('circle-link client'),
        actions: <Widget>[
          IconButton(
            onPressed: controller.isBusy
                ? null
                : () {
                    controller.logout();
                  },
            icon: const Icon(Icons.logout),
          ),
        ],
      ),
      body: Container(
        color: const Color(0xFFF5F3EE),
        child: ListView(
          padding: const EdgeInsets.all(20),
          children: <Widget>[
            _SectionCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  Text(
                    session == null
                        ? 'No session'
                        : 'Signed in as ${session.user.email}',
                    style: theme.textTheme.headlineSmall,
                  ),
                  const SizedBox(height: 8),
                  if (session != null)
                    Text(
                      'Token expires at ${session.tokens.expiresAt.toLocal()}',
                    ),
                  const SizedBox(height: 8),
                  Text('Server: ${controller.baseUrl}'),
                  const SizedBox(height: 12),
                  _FeedbackBlock(controller: controller),
                ],
              ),
            ),
            const SizedBox(height: 16),
            _SectionCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  Row(
                    children: <Widget>[
                      Expanded(
                        child: Text(
                          'Device & Relay',
                          style: theme.textTheme.titleLarge,
                        ),
                      ),
                      Chip(
                        label: Text(
                          controller.isRealtimeConnected
                              ? 'Realtime connected'
                              : 'Realtime offline',
                        ),
                        backgroundColor: controller.isRealtimeConnected
                            ? const Color(0xFFCFE8D8)
                            : const Color(0xFFE9DDD3),
                      ),
                    ],
                  ),
                  const SizedBox(height: 12),
                  DropdownButtonFormField<String>(
                    value: controller.currentDeviceId,
                    items: controller.devices
                        .where((device) => device.revokedAt == null)
                        .map((device) => DropdownMenuItem<String>(
                              value: device.id,
                              child: Text(
                                '${device.deviceName} · ${device.platform.name}',
                              ),
                            ))
                        .toList(growable: false),
                    onChanged: controller.isBusy
                        ? null
                        : (value) {
                            controller.selectDevice(value);
                          },
                    decoration: const InputDecoration(
                      labelText: 'Active device',
                    ),
                  ),
                  const SizedBox(height: 12),
                  Wrap(
                    spacing: 12,
                    runSpacing: 12,
                    children: <Widget>[
                      FilledButton(
                        onPressed: controller.isBusy
                            ? null
                            : () {
                                controller.registerCurrentDevice();
                              },
                        child: const Text('Register this device'),
                      ),
                      OutlinedButton(
                        onPressed: controller.isBusy
                            ? null
                            : () {
                                controller.loadDevices();
                              },
                        child: const Text('Refresh devices'),
                      ),
                      OutlinedButton(
                        onPressed: controller.isBusy
                            ? null
                            : () {
                                controller.connectRealtime();
                              },
                        child: const Text('Reconnect relay'),
                      ),
                    ],
                  ),
                  const SizedBox(height: 16),
                  ...controller.devices.map(
                    (device) => Padding(
                      padding: const EdgeInsets.only(bottom: 10),
                      child: ListTile(
                        shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(18),
                        ),
                        tileColor: Colors.white,
                        title: Text(device.deviceName),
                        subtitle: Text(
                          '${device.platform.name} · last seen ${device.lastSeenAt?.toLocal() ?? 'unknown'}',
                        ),
                        trailing: IconButton(
                          onPressed: controller.isBusy
                              ? null
                              : () => controller.revokeDevice(device.id),
                          icon: const Icon(Icons.delete_outline),
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 16),
            _SectionCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  Row(
                    children: <Widget>[
                      Expanded(
                        child: Text(
                          'Conversations',
                          style: theme.textTheme.titleLarge,
                        ),
                      ),
                      if (controller.selectedConversationId != null ||
                          controller.selectedPeerEmail != null)
                        TextButton(
                          onPressed: controller.clearSelection,
                          child: const Text('Show all'),
                        ),
                    ],
                  ),
                  const SizedBox(height: 12),
                  if (controller.conversationSummaries.isEmpty)
                    const Text('No conversations yet.')
                  else
                    ...controller.conversationSummaries.map(
                      (conversation) => Padding(
                        padding: const EdgeInsets.only(bottom: 10),
                        child: _ConversationTile(
                          summary: conversation,
                          selected:
                              controller.selectedConversationId == conversation.id,
                          onTap: () {
                            widget.controller.selectConversation(conversation);
                            _recipientController.text = conversation.peerEmail;
                          },
                        ),
                      ),
                    ),
                ],
              ),
            ),
            const SizedBox(height: 16),
            _SectionCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  Text('Contacts', style: theme.textTheme.titleLarge),
                  const SizedBox(height: 12),
                  if (controller.contacts.isEmpty)
                    const Text('No saved contacts yet. Add one from the composer below.')
                  else
                    ...controller.contacts.map(
                      (contact) => Padding(
                        padding: const EdgeInsets.only(bottom: 10),
                        child: _ContactTile(
                          summary: contact,
                          onSelect: () {
                            widget.controller.selectContact(contact);
                            _recipientController.text = contact.email;
                          },
                          onAccept: contact.canAccept
                              ? () => widget.controller.acceptContact(
                                    contact.peerUserId,
                                  )
                              : null,
                          onReject: contact.canReject
                              ? () => widget.controller.rejectContact(
                                    contact.peerUserId,
                                  )
                              : null,
                        ),
                      ),
                    ),
                ],
              ),
            ),
            const SizedBox(height: 16),
            _SectionCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  Text('Send message', style: theme.textTheme.titleLarge),
                  if (controller.selectedPeerEmail != null &&
                      controller.selectedPeerEmail!.isNotEmpty)
                    Padding(
                      padding: const EdgeInsets.only(top: 8),
                      child: Text(
                        'Composing in ${controller.selectedConversationId ?? 'new conversation'}',
                        style: theme.textTheme.bodySmall,
                      ),
                    ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: _recipientController,
                    decoration: const InputDecoration(
                      labelText: 'Recipient email',
                    ),
                  ),
                  const SizedBox(height: 12),
                  Align(
                    alignment: Alignment.centerLeft,
                    child: OutlinedButton(
                      onPressed: controller.isBusy
                          ? null
                          : () => controller.inviteContact(
                                _recipientController.text.trim(),
                              ),
                      child: const Text('Add to contacts'),
                    ),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: _bodyController,
                    minLines: 4,
                    maxLines: 6,
                    decoration: const InputDecoration(
                      labelText: 'Message',
                    ),
                  ),
                  const SizedBox(height: 12),
                  FilledButton(
                    onPressed: controller.isBusy
                        ? null
                        : () async {
                            await controller.sendMessage(
                              recipientEmail: _recipientController.text.trim(),
                              body: _bodyController.text,
                            );
                            _bodyController.clear();
                          },
                    child: const Text('Send'),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 16),
            _MessageSection(
              title: 'Inbox',
              hint: 'Tap a delivered message to emit a read receipt.',
              messages: controller.visibleInboxMessages,
              emptyLabel: 'No inbox messages yet.',
              onTapMessage: controller.markMessageRead,
            ),
            const SizedBox(height: 16),
            _MessageSection(
              title: 'Sent',
              hint: 'Statuses update from websocket delivery acknowledgements.',
              messages: controller.visibleSentMessages,
              emptyLabel: 'No sent messages yet.',
              onTapMessage: null,
            ),
          ],
        ),
      ),
    );
  }
}

class _ConversationTile extends StatelessWidget {
  const _ConversationTile({
    required this.summary,
    required this.selected,
    required this.onTap,
  });

  final ConversationSummary summary;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: selected ? const Color(0xFFE0EDE8) : Colors.white,
      borderRadius: BorderRadius.circular(18),
      child: InkWell(
        borderRadius: BorderRadius.circular(18),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: <Widget>[
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: <Widget>[
                    Text(
                      summary.peerEmail,
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                    const SizedBox(height: 4),
                    Text(
                      summary.lastMessagePreview,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    const SizedBox(height: 6),
                    Text(
                      '${summary.messageCount} messages · ${summary.lastMessageAt.toLocal()}',
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                  ],
                ),
              ),
              Column(
                crossAxisAlignment: CrossAxisAlignment.end,
                children: <Widget>[
                  Chip(label: Text(summary.latestDeliveryStatus.label)),
                  if (summary.unreadCount > 0)
                    Padding(
                      padding: const EdgeInsets.only(top: 8),
                      child: Chip(label: Text('${summary.unreadCount} unread')),
                    ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _ContactTile extends StatelessWidget {
  const _ContactTile({
    required this.summary,
    required this.onSelect,
    required this.onAccept,
    required this.onReject,
  });

  final ContactSummary summary;
  final VoidCallback onSelect;
  final VoidCallback? onAccept;
  final VoidCallback? onReject;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: Colors.white,
      borderRadius: BorderRadius.circular(18),
      child: InkWell(
        borderRadius: BorderRadius.circular(18),
        onTap: onSelect,
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              Row(
                children: <Widget>[
                  Expanded(
                    child: Text(
                      summary.email,
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                  ),
                  Chip(
                    label: Text(
                      summary.direction == 'accepted'
                          ? 'accepted'
                          : '${summary.direction} · ${summary.state}',
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 6),
              Text(
                summary.displayName,
                style: Theme.of(context).textTheme.bodyMedium,
              ),
              const SizedBox(height: 6),
              Text(
                summary.messageCount > 0
                    ? '${summary.messageCount} messages · last activity ${summary.lastMessageAt.toLocal()}'
                    : 'No local messages yet.',
                style: Theme.of(context).textTheme.bodySmall,
              ),
              if (onAccept != null || onReject != null) ...<Widget>[
                const SizedBox(height: 12),
                Wrap(
                  spacing: 10,
                  runSpacing: 10,
                  children: <Widget>[
                    if (onAccept != null)
                      FilledButton(
                        onPressed: onAccept,
                        child: const Text('Accept'),
                      ),
                    if (onReject != null)
                      OutlinedButton(
                        onPressed: onReject,
                        child: const Text('Decline'),
                      ),
                  ],
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

class _MessageSection extends StatelessWidget {
  const _MessageSection({
    required this.title,
    required this.hint,
    required this.messages,
    required this.emptyLabel,
    required this.onTapMessage,
  });

  final String title;
  final String hint;
  final List<MessageRecord> messages;
  final String emptyLabel;
  final ValueChanged<String>? onTapMessage;

  @override
  Widget build(BuildContext context) {
    return _SectionCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Text(title, style: Theme.of(context).textTheme.titleLarge),
          const SizedBox(height: 6),
          Text(hint, style: Theme.of(context).textTheme.bodySmall),
          const SizedBox(height: 12),
          if (messages.isEmpty)
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 16),
              child: Text(emptyLabel),
            )
          else
            ...messages.map(
              (message) => Padding(
                padding: const EdgeInsets.only(bottom: 12),
                child: _MessageTile(
                  message: message,
                  onTap: onTapMessage == null
                      ? null
                      : () => onTapMessage!(message.id),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _MessageTile extends StatelessWidget {
  const _MessageTile({
    required this.message,
    required this.onTap,
  });

  final MessageRecord message;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: Colors.white,
      borderRadius: BorderRadius.circular(18),
      child: InkWell(
        borderRadius: BorderRadius.circular(18),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              Row(
                children: <Widget>[
                  Expanded(
                    child: Text(
                      message.senderEmail.isEmpty
                          ? 'To ${message.recipientEmail}'
                          : '${message.senderEmail} -> ${message.recipientEmail.isEmpty ? message.recipientUserId : message.recipientEmail}',
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                  ),
                  Chip(label: Text(message.status.label)),
                ],
              ),
              const SizedBox(height: 8),
              Text(message.body),
              const SizedBox(height: 10),
              Text(
                'sent ${message.sentAt.toLocal()}'
                '${message.recipientDeviceId.isEmpty ? '' : ' · ${message.recipientDeviceId}'}'
                '${message.readAt != null ? ' · read ${message.readAt!.toLocal()}' : message.deliveredAt != null ? ' · delivered ${message.deliveredAt!.toLocal()}' : ''}',
                style: Theme.of(context).textTheme.bodySmall,
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _SectionCard extends StatelessWidget {
  const _SectionCard({
    required this.child,
  });

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Card(
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(26),
      ),
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: child,
      ),
    );
  }
}

class _FeedbackBlock extends StatelessWidget {
  const _FeedbackBlock({
    required this.controller,
  });

  final AppController controller;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    if (controller.errorMessage == null && controller.infoMessage == null) {
      return const SizedBox.shrink();
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        if (controller.errorMessage != null)
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: Text(
              controller.errorMessage!,
              style: TextStyle(color: theme.colorScheme.error),
            ),
          ),
        if (controller.infoMessage != null)
          Text(
            controller.infoMessage!,
            style: theme.textTheme.bodyMedium,
          ),
      ],
    );
  }
}
