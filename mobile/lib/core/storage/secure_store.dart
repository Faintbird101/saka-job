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
///   - theme and locale go in [SharedPreferences], because encrypting a
///     preference for "dark mode" costs a platform channel round trip on every
///     read and protects nothing.
class SecureStore extends GetxService {
  static const _kToken = 'session_token';
  static const _kEmail = 'session_email';
  static const _kThemeMode = 'theme_mode';
  static const _kLocale = 'locale';

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

  Future<void> writeSession({required String token, required String email}) async {
    await _secure.write(key: _kToken, value: token);
    await _secure.write(key: _kEmail, value: email);
  }

  Future<String?> readEmail() async {
    try {
      return await _secure.read(key: _kEmail);
    } catch (_) {
      return null;
    }
  }

  Future<void> clearSession() async {
    try {
      await _secure.delete(key: _kToken);
      await _secure.delete(key: _kEmail);
    } catch (_) {
      // Best effort: if the keystore is unreadable there is nothing usable in
      // it either, so failing to delete is not a security problem.
    }
  }

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
