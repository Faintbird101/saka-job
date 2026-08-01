import 'package:get/get.dart';

import '../../data/services/auth_service.dart';
import '../../data/services/events_service.dart';
import '../../data/services/jobs_service.dart';
import '../../data/services/profile_service.dart';
import '../../data/services/push_service.dart';
import '../network/api_client.dart';
import '../storage/secure_store.dart';
import '../theme/theme_controller.dart';

/// Wires the long-lived singletons.
///
/// Order is not incidental: SecureStore is read by ApiClient's request
/// interceptor, and AuthService needs both to verify a stored token. Creating
/// them in dependency order rather than lazily avoids a first-request race
/// where the interceptor asks for a store that does not exist yet.
class InitialBinding extends Bindings {
  @override
  void dependencies() {
    // Permanent: these outlive every route, and letting GetX dispose them on
    // navigation would drop the session mid-session.
    Get.put<ThemeController>(ThemeController(), permanent: true);
  }

  /// The services that must exist before the first frame, awaited in main().
  static Future<void> initServices() async {
    await Get.putAsync<SecureStore>(() => SecureStore().init(), permanent: true);
    await Get.putAsync<ApiClient>(() => ApiClient().init(), permanent: true);
    await Get.putAsync<AuthService>(() => AuthService().init(), permanent: true);
    Get.put<JobsService>(JobsService(), permanent: true);
    Get.put<ProfileService>(ProfileService(), permanent: true);
    Get.put<EventsService>(EventsService(), permanent: true);
    // Push is initialised but not permission-prompted here — see
    // PushService.enable, which runs after sign-in.
    await Get.putAsync<PushService>(() => PushService().init(), permanent: true);
  }
}
