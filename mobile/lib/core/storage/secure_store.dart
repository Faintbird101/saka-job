import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:get/get.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// Where credentials and preferences live.
///
/// Deliberately two stores, split on whether losing the value would matter:
///
///   - the session token goes in [FlutterSecureStorage], backed by the Android
///     Keystore and the iOS Keychain, so a dump of the app's files does not
///     hand someone a working login;
///   - theme, locale and the activity clock go in [SharedPreferences], because
///     encrypting a preference for "dark mode" costs a platform channel round
///     trip on every read and protects nothing.
class SecureStore extends GetxService {
  static const _kToken = 'session_token';
  static const _kEmail = 'session_email';
  static const _kUserId = 'session_user_id';
  static const _kName = 'session_name';
  static const _kLastActive = 'last_active_at';
  static const _kThemeMode = 'theme_mode';
  static const _kLocale = 'locale';
  static const _kOnboarded = 'onboarding_seen';

  final _secure = const FlutterSecureStorage(
    aOptions: AndroidOptions(encryptedSharedPreferences: true),
    iOptions: IOSOptions(accessibility: KeychainAccessibility.first_unlock),
  );

  late final SharedPreferences _prefs;

  Future<SecureStore> init() async {
    _prefs = await SharedPreferences.getInstance();
    return this;
  }

  // ---- session ----

  Future<String?> readToken() async {
    try {
      return await _secure.read(key: _kToken);
    } catch (_) {
      // A corrupt keystore entry (app reinstall on some Android builds, or a
      // restored backup) throws rather than returning null. Treat it as "no
      // session" — the user signs in again — instead of crashing at startup.
      await clearSession();
      return null;
    }
  }

  /// Stores the session, and the identity alongside it.
  ///
  /// Caching the user is what lets the app open straight into the shell
  /// offline: without it, every launch would have to reach the server before it
  /// could even decide which screen to show.
  Future<void> writeSession({
    required String token,
    required String email,
    String? userId,
    String? displayName,
  }) async {
    await _secure.write(key: _kToken, value: token);
    await _secure.write(key: _kEmail, value: email);
    if (userId != null) await _secure.write(key: _kUserId, value: userId);
    if (displayName != null) await _secure.write(key: _kName, value: displayName);
    await touchActivity();
  }

  Future<({String? email, String? userId, String? name})> readCachedUser() async {
    try {
      return (
        email: await _secure.read(key: _kEmail),
        userId: await _secure.read(key: _kUserId),
        name: await _secure.read(key: _kName),
      );
    } catch (_) {
      return (email: null, userId: null, name: null);
    }
  }

  Future<void> clearSession() async {
    try {
      for (final k in [_kToken, _kEmail, _kUserId, _kName]) {
        await _secure.delete(key: k);
      }
    } catch (_) {
      // Best effort: if the keystore is unreadable there is nothing usable in
      // it either, so failing to delete is not a security problem.
    }
    await _prefs.remove(_kLastActive);
  }

  // ---- inactivity clock ----

  /// Marks the app as used now. Called on launch, on resume, and after any
  /// successful authenticated request.
  Future<void> touchActivity() =>
      _prefs.setInt(_kLastActive, DateTime.now().millisecondsSinceEpoch);

  DateTime? get lastActiveAt {
    final ms = _prefs.getInt(_kLastActive);
    return ms == null ? null : DateTime.fromMillisecondsSinceEpoch(ms);
  }

  /// Whether the session has gone stale through disuse.
  ///
  /// Separate from the server's own 30-day expiry: this is a local rule so an
  /// unattended phone stops showing someone's job search, and it applies even
  /// while the server would still accept the token.
  bool isInactiveBeyond(Duration limit) {
    final last = lastActiveAt;
    // No timestamp means a session written before this existed. Treating that
    // as expired would sign everyone out on upgrade, so it counts as active
    // and the clock starts now.
    if (last == null) return false;
    return DateTime.now().difference(last) > limit;
  }

  /// Whether the three-step intro has been shown.
  ///
  /// Not part of the session: it survives sign-out, because the explanation is
  /// about the product rather than the account, and re-showing it to someone
  /// signing back in would be noise.
  bool get hasOnboarded => _prefs.getBool(_kOnboarded) ?? false;
  Future<void> markOnboardingSeen() => _prefs.setBool(_kOnboarded, true);

  // ---- preferences ----

  /// 'system' | 'light' | 'dark'. System is the default, per the brief.
  String get themeMode => _prefs.getString(_kThemeMode) ?? 'system';
  Future<void> setThemeMode(String mode) => _prefs.setString(_kThemeMode, mode);

  /// Null means "follow the device", which is what a fresh install should do.
  String? get localeCode => _prefs.getString(_kLocale);
  Future<void> setLocaleCode(String? code) async {
    if (code == null) {
      await _prefs.remove(_kLocale);
    } else {
      await _prefs.setString(_kLocale, code);
    }
  }
}
