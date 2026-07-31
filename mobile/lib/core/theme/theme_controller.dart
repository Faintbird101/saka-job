import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../storage/secure_store.dart';

/// Theme and locale, both persisted.
///
/// Kept together because they are the two things the user changes that have to
/// survive a restart and rebuild the whole tree.
class ThemeController extends GetxController {
  SecureStore get _store => Get.find<SecureStore>();

  /// Defaults to system, per the brief.
  final Rx<ThemeMode> themeMode = ThemeMode.system.obs;

  /// Null means "follow the device", which is what a fresh install does.
  final Rxn<Locale> locale = Rxn<Locale>();

  static const supportedLocales = [Locale('en'), Locale('sw')];

  @override
  void onInit() {
    super.onInit();
    themeMode.value = switch (_store.themeMode) {
      'light' => ThemeMode.light,
      'dark' => ThemeMode.dark,
      _ => ThemeMode.system,
    };
    final code = _store.localeCode;
    if (code != null) locale.value = Locale(code);
  }

  Future<void> setThemeMode(ThemeMode mode) async {
    themeMode.value = mode;
    await _store.setThemeMode(switch (mode) {
      ThemeMode.light => 'light',
      ThemeMode.dark => 'dark',
      ThemeMode.system => 'system',
    });
    Get.changeThemeMode(mode);
  }

  /// Pass null to follow the device again.
  Future<void> setLocale(Locale? value) async {
    locale.value = value;
    await _store.setLocaleCode(value?.languageCode);
    Get.updateLocale(value ?? Get.deviceLocale ?? const Locale('en'));
  }

  bool get isDark => themeMode.value == ThemeMode.dark ||
      (themeMode.value == ThemeMode.system &&
          Get.mediaQuery.platformBrightness == Brightness.dark);
}
