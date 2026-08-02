import 'package:get/get.dart';

import '../features/auth/controllers/auth_controller.dart';
import '../features/auth/views/sign_in_screen.dart';
import '../features/approval/controllers/triage_controller.dart';
import '../features/approval/views/triage_screen.dart';
import '../features/home/controllers/home_controller.dart';
import '../features/job_detail/controllers/job_detail_controller.dart';
import '../features/notifications/controllers/replies_controller.dart';
import '../features/notifications/views/replies_screen.dart';
import '../features/job_detail/controllers/documents_controller.dart';
import '../features/job_detail/views/documents_screen.dart';
import '../features/job_detail/views/job_detail_screen.dart';
import '../features/dashboard/controllers/pipeline_controller.dart';
import '../features/jobs/controllers/jobs_controller.dart';
import '../features/profile/controllers/cv_controller.dart';
import '../features/profile/controllers/profile_controller.dart';
import '../features/onboarding/views/onboarding_screen.dart';
import '../features/profile/views/cv_screen.dart';
import '../features/shell/shell_screen.dart';
import '../features/splash/splash_screen.dart';
import '../data/models/job.dart';
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
      name: Routes.shell,
      page: () => const ShellScreen(),
      binding: BindingsBuilder(() {
        Get.put<ShellController>(ShellController());
        Get.lazyPut<HomeController>(HomeController.new);
        // fenix so a tab controller disposed under memory pressure is rebuilt
        // on next access rather than throwing. Lazy, so opening the app does
        // not fetch for three tabs the user may never visit — an IndexedStack
        // builds every child, but the controller need not run.
        Get.lazyPut<JobsController>(JobsController.new, fenix: true);
        Get.lazyPut<PipelineController>(PipelineController.new, fenix: true);
        Get.lazyPut<ProfileController>(ProfileController.new, fenix: true);
      }),
    ),
    GetPage(
      name: Routes.jobDetail,
      page: () => const JobDetailScreen(),
      binding: BindingsBuilder(() {
        // The list already holds most of the job, so it is passed through as
        // an argument and rendered immediately while the full record loads.
        final args = Get.arguments as Map<String, dynamic>?;
        Get.lazyPut<JobDetailController>(
          () => JobDetailController(
            jobId: args?['id'] as String? ?? '',
            initial: args?['job'] as Job?,
          ),
        );
      }),
    ),
    GetPage(
      name: Routes.onboarding,
      page: () => const OnboardingScreen(),
    ),
    GetPage(
      name: Routes.masterCv,
      page: () => const CvScreen(),
      binding: BindingsBuilder(() {
        Get.lazyPut<CvController>(CvController.new);
      }),
    ),
    GetPage(
      name: Routes.documents,
      page: () => const DocumentsScreen(),
      binding: BindingsBuilder(() {
        final args = Get.arguments as Map<String, dynamic>?;
        Get.lazyPut<DocumentsController>(
          () => DocumentsController(
            jobId: args?['id'] as String? ?? '',
            jobTitle: args?['title'] as String? ?? '',
          ),
        );
      }),
    ),
    GetPage(
      name: Routes.replies,
      page: () => const RepliesScreen(),
      binding: BindingsBuilder(() {
        Get.lazyPut<RepliesController>(RepliesController.new);
      }),
    ),
    GetPage(
      name: Routes.triage,
      page: () => const TriageScreen(),
      binding: BindingsBuilder(() {
        Get.lazyPut<TriageController>(TriageController.new);
      }),
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
