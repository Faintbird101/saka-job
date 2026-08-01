import 'package:flutter/widgets.dart';
import 'package:get/get.dart';

import '../../data/services/auth_service.dart';
import '../../routes/app_routes.dart';
import '../storage/secure_store.dart';

/// Enforces the inactivity rule while the app is running.
///
/// Checking only at startup would miss the common case: the app sits in the
/// background for two days, is resumed, and the process was never killed — so
/// init() never runs again and a stale session stays live.
///
/// Resume is also where the clock is restarted, because time spent backgrounded
/// is exactly what the window is measuring.
class SessionWatcher with WidgetsBindingObserver {
  SessionWatcher._();
  static final instance = SessionWatcher._();

  void start() => WidgetsBinding.instance.addObserver(this);
  void stop() => WidgetsBinding.instance.removeObserver(this);

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state != AppLifecycleState.resumed) return;

    final auth = Get.find<AuthService>();
    if (!auth.isSignedIn) return;

    final store = Get.find<SecureStore>();
    if (store.isInactiveBeyond(AuthService.inactivityLimit)) {
      auth.forgetSession();
      Get.offAllNamed<void>(Routes.signIn);
      return;
    }
    store.touchActivity();
  }
}
