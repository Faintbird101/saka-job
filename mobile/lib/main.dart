import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:get/get.dart';
import 'package:toastification/toastification.dart';

import 'core/bindings/initial_binding.dart';
import 'core/theme/app_theme.dart';
import 'core/theme/theme_controller.dart';
import 'l10n/app_localizations.dart';
import 'routes/app_pages.dart';
import 'routes/app_routes.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // Portrait only: every screen in the design is a single column, and a
  // landscape layout would be a second design rather than a stretched one.
  await SystemChrome.setPreferredOrientations([
    DeviceOrientation.portraitUp,
    DeviceOrientation.portraitDown,
  ]);

  // Services first, so the very first frame already knows whether there is a
  // session — no flash of the sign-in screen for a signed-in user.
  await InitialBinding.initServices();

  runApp(const SakaJobApp());
}

class SakaJobApp extends StatelessWidget {
  const SakaJobApp({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Get.put<ThemeController>(ThemeController(), permanent: true);

    // ToastificationWrapper sits above GetMaterialApp so a toast can be raised
    // from a controller mid-transition without needing a BuildContext.
    return ToastificationWrapper(
      child: Obx(
        () => GetMaterialApp(
          title: 'Saka Job',
          debugShowCheckedModeBanner: false,

          theme: AppTheme.light(),
          darkTheme: AppTheme.dark(),
          themeMode: theme.themeMode.value,

          locale: theme.locale.value,
          fallbackLocale: const Locale('en'),
          supportedLocales: L10n.supportedLocales,
          localizationsDelegates: const [
            L10n.delegate,
            GlobalMaterialLocalizations.delegate,
            GlobalWidgetsLocalizations.delegate,
            GlobalCupertinoLocalizations.delegate,
          ],

          initialRoute: Routes.splash,
          getPages: AppPages.routes,

          // The design is a stack of cards, so a lateral slide reads better
          // than a fade.
          defaultTransition: Transition.cupertino,
          transitionDuration: const Duration(milliseconds: 260),

          builder: (context, child) {
            // Clamp the system font scale. The design relies on a type ramp,
            // and a 2x scale turns every card into its own scroll view —
            // clamping rather than locking keeps it accessible.
            final scale = MediaQuery.textScalerOf(context).clamp(
              minScaleFactor: 0.9,
              maxScaleFactor: 1.3,
            );
            return MediaQuery(
              data: MediaQuery.of(context).copyWith(textScaler: scale),
              child: child ?? const SizedBox.shrink(),
            );
          },
        ),
      ),
    );
  }
}
