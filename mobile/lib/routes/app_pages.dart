import 'package:get/get.dart';

import '../features/auth/controllers/auth_controller.dart';
import '../features/auth/views/sign_in_screen.dart';
import '../features/splash/splash_screen.dart';
import 'app_routes.dart';

/// The route table.
///
/// Controllers are bound per-route with [BindingsBuilder] rather than up front,
/// so a screen's state is created when it opens and disposed when it closes —
/// the sign-in controller should not outlive the sign-in screen.
abstract final class AppPages {
  static final routes = <GetPage<dynamic>>[
    GetPage(
      name: Routes.splash,
      page: () => const SplashScreen(),
    ),
    GetPage(
      name: Routes.signIn,
      page: () => const SignInScreen(),
      binding: BindingsBuilder(() {
        Get.lazyPut<AuthController>(AuthController.new);
      }),
    ),
  ];
}
