import 'package:flutter/material.dart';

import '../features/auth/domain/auth_models.dart';
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
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'circle-link',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFF1E5B4F)),
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

    return Scaffold(
      appBar: AppBar(
        title: const Text('circle-link'),
      ),
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 420),
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  _isLoginMode ? 'Login' : 'Create account',
                  style: Theme.of(context).textTheme.headlineMedium,
                ),
                const SizedBox(height: 12),
                Text(
                  'Server: ${controller.baseUrl}',
                  style: Theme.of(context).textTheme.bodySmall,
                ),
                const SizedBox(height: 24),
                if (!_isLoginMode) ...[
                  TextField(
                    controller: _displayNameController,
                    decoration: const InputDecoration(labelText: 'Display name'),
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
                if (controller.errorMessage != null)
                  Text(
                    controller.errorMessage!,
                    style: TextStyle(color: Theme.of(context).colorScheme.error),
                  ),
                if (controller.infoMessage != null)
                  Padding(
                    padding: const EdgeInsets.only(top: 8),
                    child: Text(controller.infoMessage!),
                  ),
                const SizedBox(height: 16),
                FilledButton(
                  onPressed: controller.isBusy ? null : _submit,
                  child: Text(controller.isBusy
                      ? 'Please wait...'
                      : _isLoginMode
                          ? 'Login'
                          : 'Create account'),
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
                  child: Text(_isLoginMode
                      ? 'Need an account? Sign up'
                      : 'Already have an account? Login'),
                ),
              ],
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
          email: _emailController.text,
          password: _passwordController.text,
        ),
      );
      return;
    }

    widget.controller.signUp(
      SignUpCommand(
        email: _emailController.text,
        password: _passwordController.text,
        displayName: _displayNameController.text,
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
          constraints: const BoxConstraints(maxWidth: 420),
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
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
                  decoration: const InputDecoration(labelText: 'Verification token'),
                ),
                const SizedBox(height: 16),
                if (controller.errorMessage != null)
                  Text(
                    controller.errorMessage!,
                    style: TextStyle(color: Theme.of(context).colorScheme.error),
                  ),
                if (controller.infoMessage != null)
                  Padding(
                    padding: const EdgeInsets.only(top: 8),
                    child: Text(controller.infoMessage!),
                  ),
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
    );
  }
}

class _HomeScreen extends StatelessWidget {
  const _HomeScreen({
    required this.controller,
  });

  final AppController controller;

  @override
  Widget build(BuildContext context) {
    final session = controller.session;

    return Scaffold(
      appBar: AppBar(
        title: const Text('circle-link home'),
        actions: [
          IconButton(
            onPressed: controller.isBusy ? null : controller.logout,
            icon: const Icon(Icons.logout),
          ),
        ],
      ),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              session == null ? 'No session' : 'Signed in as ${session.user.email}',
              style: Theme.of(context).textTheme.headlineSmall,
            ),
            const SizedBox(height: 8),
            if (session != null)
              Text('Access token expires at ${session.tokens.expiresAt.toLocal()}'),
            const SizedBox(height: 12),
            if (controller.errorMessage != null)
              Text(
                controller.errorMessage!,
                style: TextStyle(color: Theme.of(context).colorScheme.error),
              ),
            if (controller.infoMessage != null)
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: Text(controller.infoMessage!),
              ),
            const SizedBox(height: 16),
            Wrap(
              spacing: 12,
              runSpacing: 12,
              children: [
                FilledButton(
                  onPressed: controller.isBusy ? null : controller.registerCurrentDevice,
                  child: const Text('Register this device'),
                ),
                OutlinedButton(
                  onPressed: controller.isBusy ? null : controller.loadDevices,
                  child: const Text('Refresh devices'),
                ),
              ],
            ),
            const SizedBox(height: 24),
            Text(
              'Devices',
              style: Theme.of(context).textTheme.titleLarge,
            ),
            const SizedBox(height: 12),
            Expanded(
              child: controller.devices.isEmpty
                  ? const Center(
                      child: Text('No devices registered yet.'),
                    )
                  : ListView.separated(
                      itemCount: controller.devices.length,
                      separatorBuilder: (_, __) => const SizedBox(height: 12),
                      itemBuilder: (context, index) {
                        final device = controller.devices[index];
                        return Card(
                          child: ListTile(
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
                        );
                      },
                    ),
            ),
          ],
        ),
      ),
    );
  }
}
