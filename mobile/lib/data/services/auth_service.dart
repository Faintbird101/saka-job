import 'package:get/get.dart';

import '../../core/error/failures.dart';
import '../../core/network/api_client.dart';
import '../../core/storage/secure_store.dart';
import '../models/user.dart';

/// Everything about being signed in.
///
/// Services own the endpoints; controllers own screen state. Nothing above
/// this file knows that `/auth/login` exists, which is what lets the API move
/// without touching a widget.
class AuthService extends GetxService {
  ApiClient get _api => Get.find<ApiClient>();
  SecureStore get _store => Get.find<SecureStore>();

  /// Null while signed out. Reactive so the router and the profile screen can
  /// both watch it without either owning it.
  final Rxn<AppUser> user = Rxn<AppUser>();

  bool get isSignedIn => user.value != null;

  Future<AuthService> init() async {
    // A token on disk is not proof of a live session — it may have expired or
    // been revoked from another device — so it is verified against the server
    // before the app treats the user as signed in.
    final token = await _store.readToken();
    if (token == null || token.isEmpty) return this;

    try {
      user.value = await me();
    } on Failure {
      // Expired, revoked, or the server is unreachable. Either way, start at
      // sign-in rather than showing a shell that will 401 on every request.
      await _store.clearSession();
    }
    return this;
  }

  /// Whether an account exists at all. Decides between the sign-in screen and
  /// first-run setup, and is the one call made before authenticating.
  Future<bool> needsSetup() => _api.get<bool>(
        '/auth/status',
        parse: (data) => (data as Map)['needs_setup'] == true,
      );

  Future<AppUser> signIn({required String email, required String password}) async {
    final result = await _api.post<Map<String, dynamic>>(
      '/auth/login',
      body: {'email': email.trim(), 'password': password},
      parse: (data) => Map<String, dynamic>.from(data as Map),
    );

    final token = result['token'] as String;
    final parsed = AppUser.fromJson(Map<String, dynamic>.from(result['user'] as Map));

    // The token is returned exactly once — the server stores only a hash — so
    // it is persisted before anything else can fail.
    await _store.writeSession(token: token, email: parsed.email);
    user.value = parsed;
    return parsed;
  }

  /// Creates the first and only account. The server closes this permanently
  /// once one exists, so it is offered only when [needsSetup] is true.
  Future<AppUser> createFirstAccount({
    required String email,
    required String password,
    String? displayName,
  }) async {
    await _api.post<void>(
      '/auth/bootstrap',
      body: {
        'email': email.trim(),
        'password': password,
        if (displayName != null && displayName.trim().isNotEmpty)
          'display_name': displayName.trim(),
      },
      parse: (_) {},
    );
    // Bootstrap returns the user but not a session, so sign in to get one
    // rather than asking the user to type the password twice.
    return signIn(email: email, password: password);
  }

  Future<AppUser> me() => _api.get<AppUser>(
        '/auth/me',
        parse: (data) => AppUser.fromJson(Map<String, dynamic>.from(data as Map)),
      );

  /// Revokes this session server-side, then clears it locally.
  ///
  /// The local clear happens even if the call fails: a user who taps sign out
  /// on a shared device must end up signed out regardless of the network.
  Future<void> signOut() async {
    try {
      await _api.post<void>('/auth/logout', parse: (_) {});
    } on Failure {
      // Ignored deliberately — see above.
    }
    await forgetSession();
  }

  /// Drops the local session without calling the server. Used when a 401 says
  /// the session is already gone.
  Future<void> forgetSession() async {
    await _store.clearSession();
    user.value = null;
  }
}
