import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:material_design_icons_flutter/material_design_icons_flutter.dart';

import '../../../core/storage/secure_store.dart';
import '../../../core/theme/app_colors.dart';
import '../../../core/theme/app_theme.dart';
import '../../../l10n/app_localizations.dart';
import '../../../routes/app_routes.dart';

/// Screen 02 — what the agent does, shown once after the account is created.
///
/// Functional rather than decorative. A new account has no CV, and scoring
/// refuses to run without one, so the last step sends you to the CV editor
/// instead of dropping you on an empty feed that will never fill.
class OnboardingScreen extends StatelessWidget {
  const OnboardingScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final text = Theme.of(context).textTheme;

    return Scaffold(
      body: SafeArea(
        child: Column(
          children: [
            Expanded(
              child: ListView(
                padding: const EdgeInsets.symmetric(
                    horizontal: AppSpacing.xxl, vertical: AppSpacing.section),
                children: [
                  Row(
                    children: [
                      Container(
                        width: 34,
                        height: 34,
                        alignment: Alignment.center,
                        decoration: BoxDecoration(
                          gradient: AppColors.primaryGradient,
                          borderRadius: BorderRadius.circular(10),
                        ),
                        child: const Text(
                          'S',
                          style: TextStyle(
                            color: Colors.white,
                            fontWeight: FontWeight.w800,
                            fontSize: 16,
                          ),
                        ),
                      ),
                      const SizedBox(width: AppSpacing.md),
                      Text(l.appName, style: text.titleMedium),
                    ],
                  ),
                  const SizedBox(height: AppSpacing.section),

                  Text(l.tagline, style: text.displayMedium),
                  const SizedBox(height: AppSpacing.md),
                  Text(l.welcomeBody, style: text.bodyLarge),
                  const SizedBox(height: AppSpacing.section),

                  const _Step(
                    number: 1,
                    titleKey: _StepKey.cv,
                  ),
                  const SizedBox(height: AppSpacing.md),
                  const _Step(
                    number: 2,
                    titleKey: _StepKey.bar,
                  ),
                  const SizedBox(height: AppSpacing.md),
                  const _Step(
                    number: 3,
                    titleKey: _StepKey.yes,
                  ),
                ],
              ),
            ),

            Padding(
              padding: const EdgeInsets.all(AppSpacing.xxl),
              child: Column(
                children: [
                  FilledButton(
                    onPressed: () async {
                      await Get.find<SecureStore>().markOnboardingSeen();
                      // Straight to the CV rather than the feed: without one,
                      // scoring refuses to run and the feed would stay empty
                      // with nothing explaining why.
                      await Get.toNamed<void>(Routes.masterCv);
                      Get.offAllNamed<void>(Routes.shell);
                    },
                    child: Text(l.upload),
                  ),
                  const SizedBox(height: AppSpacing.sm),
                  TextButton(
                    onPressed: () async {
                      await Get.find<SecureStore>().markOnboardingSeen();
                      Get.offAllNamed<void>(Routes.shell);
                    },
                    child: Text(l.close),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

enum _StepKey { cv, bar, yes }

class _Step extends StatelessWidget {
  const _Step({required this.number, required this.titleKey});

  final int number;
  final _StepKey titleKey;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final text = Theme.of(context).textTheme;
    final dark = Theme.of(context).brightness == Brightness.dark;

    final (title, body, icon) = switch (titleKey) {
      _StepKey.cv => (l.onboardCvTitle, l.onboardCvBody, MdiIcons.fileUploadOutline),
      _StepKey.bar => (l.onboardBarTitle, l.onboardBarBody, MdiIcons.tuneVertical),
      _StepKey.yes => (l.onboardYesTitle, l.onboardYesBody, MdiIcons.checkCircleOutline),
    };

    return Container(
      padding: const EdgeInsets.all(AppSpacing.lg),
      decoration: BoxDecoration(
        color: dark ? AppColors.darkSurface : AppColors.surface,
        borderRadius: AppRadii.cardShape,
        boxShadow: dark ? AppShadows.cardDark : AppShadows.card,
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 30,
            height: 30,
            alignment: Alignment.center,
            decoration: BoxDecoration(
              color: dark ? AppColors.darkSurfaceRaised : AppColors.primaryTint,
              borderRadius: AppRadii.pillShape,
            ),
            child: Text(
              '$number',
              style: const TextStyle(
                color: AppColors.primary,
                fontWeight: FontWeight.w800,
                fontSize: 13,
              ),
            ),
          ),
          const SizedBox(width: AppSpacing.md),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title, style: text.titleMedium),
                const SizedBox(height: 2),
                Text(body, style: text.bodyMedium),
              ],
            ),
          ),
          Icon(icon, size: 18, color: AppColors.primary),
        ],
      ),
    );
  }
}
