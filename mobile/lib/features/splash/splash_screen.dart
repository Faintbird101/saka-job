import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../core/theme/app_colors.dart';
import '../../data/services/auth_service.dart';
import '../../routes/app_routes.dart';

/// Decides where to start.
///
/// AuthService has already verified any stored token against the server before
/// the first frame, so this is a branch rather than a wait — the brief moment
/// it is visible is the platform's own launch, not a spinner we added.
class SplashScreen extends StatefulWidget {
  const SplashScreen({super.key});

  @override
  State<SplashScreen> createState() => _SplashScreenState();
}

class _SplashScreenState extends State<SplashScreen> {
  @override
  void initState() {
    super.initState();
    // After the first frame, so a route change is never issued mid-build.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final signedIn = Get.find<AuthService>().isSignedIn;
      Get.offAllNamed(signedIn ? Routes.shell : Routes.signIn);
    });
  }

  @override
  Widget build(BuildContext context) {
    return const Scaffold(
      body: Center(
        child: _Mark(),
      ),
    );
  }
}

class _Mark extends StatelessWidget {
  const _Mark();

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 84,
      height: 84,
      decoration: BoxDecoration(
        gradient: AppColors.primaryGradient,
        borderRadius: BorderRadius.circular(24),
      ),
      alignment: Alignment.center,
      child: const Text(
        'S',
        style: TextStyle(
          color: Colors.white,
          fontSize: 42,
          fontWeight: FontWeight.w800,
        ),
      ),
    );
  }
}
