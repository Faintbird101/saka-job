import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter/foundation.dart';
import 'package:get/get.dart';

import '../../core/error/failures.dart';
import '../../core/network/api_client.dart';
import '../../firebase_options.dart';
import '../../shared/widgets/app_toast.dart';

/// Handles a message that arrives while the app is terminated or backgrounded.
///
/// Must be a top-level function: Android spins up a separate isolate for it,
/// and a closure or method would not survive that boundary.
@pragma('vm:entry-point')
Future<void> _backgroundHandler(RemoteMessage message) async {
  // Deliberately does almost nothing. The system tray notification is drawn by
  // FCM itself from the `notification` payload, and doing real work here means
  // doing it in an isolate with no services registered.
  await Firebase.initializeApp(options: DefaultFirebaseOptions.currentPlatform);
}

/// Push notifications.
///
/// Registration is best-effort throughout: a phone with notifications denied,
/// no Play Services, or no network must still get a working app. Nothing here
/// is allowed to block startup or throw into the UI.
class PushService extends GetxService {
  ApiClient get _api => Get.find<ApiClient>();

  final Rxn<String> token = Rxn<String>();
  final enabled = false.obs;

  /// True once Firebase itself is up. Kept so the profile screen can explain
  /// why push is unavailable rather than showing a toggle that does nothing.
  final available = false.obs;

  Future<PushService> init() async {
    try {
      await Firebase.initializeApp(
        options: DefaultFirebaseOptions.currentPlatform,
      );
      FirebaseMessaging.onBackgroundMessage(_backgroundHandler);
      available.value = true;
    } catch (e) {
      // A missing or malformed google-services.json, or a device without Play
      // Services. The app is fully usable without push, so this is logged and
      // dropped rather than rethrown.
      debugPrint('Firebase unavailable, push disabled: $e');
      return this;
    }

    // Foreground messages do not raise a system notification, so they are
    // surfaced as a toast — otherwise a reply arriving while you are looking
    // at the app is silently invisible.
    FirebaseMessaging.onMessage.listen((message) {
      final title = message.notification?.title;
      if (title != null && title.isNotEmpty) {
        AppToast.info(title);
      }
    });

    // FCM rotates tokens on reinstall, restore, and occasionally on its own.
    // Without this the backend keeps sending to a dead token and the phone
    // quietly stops receiving anything.
    FirebaseMessaging.instance.onTokenRefresh.listen(_register);

    return this;
  }

  /// Asks for permission and registers the device.
  ///
  /// Called after sign-in rather than at launch: a permission prompt on the
  /// very first screen, before the app has shown what it does, is the one most
  /// people decline out of hand.
  Future<void> enable() async {
    if (!available.value) return;

    try {
      final settings = await FirebaseMessaging.instance.requestPermission(
        alert: true,
        badge: true,
        sound: true,
      );
      final granted =
          settings.authorizationStatus == AuthorizationStatus.authorized ||
              settings.authorizationStatus == AuthorizationStatus.provisional;
      enabled.value = granted;
      if (!granted) return;

      final fcmToken = await FirebaseMessaging.instance.getToken();
      if (fcmToken != null) await _register(fcmToken);
    } catch (e) {
      debugPrint('Push registration failed: $e');
    }
  }

  Future<void> _register(String fcmToken) async {
    token.value = fcmToken;
    try {
      await _api.post<void>(
        '/devices',
        body: {
          'token': fcmToken,
          'platform': defaultTargetPlatform == TargetPlatform.iOS ? 'ios' : 'android',
        },
        parse: (_) {},
      );
    } on Failure catch (f) {
      // Not surfaced: failing to register for notifications is not a reason to
      // interrupt someone who has just signed in successfully.
      debugPrint('Could not register device for push: ${f.message}');
    }
  }

  /// Drops the device on sign-out, so a shared or sold phone stops receiving
  /// notifications about someone else's job search.
  Future<void> unregister() async {
    final current = token.value;
    if (current == null) return;
    try {
      await _api.delete<void>('/devices', body: {'token': current}, parse: (_) {});
      await FirebaseMessaging.instance.deleteToken();
    } catch (e) {
      debugPrint('Could not unregister device: $e');
    }
    token.value = null;
    enabled.value = false;
  }
}
