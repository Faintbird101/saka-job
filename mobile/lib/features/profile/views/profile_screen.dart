import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:material_design_icons_flutter/material_design_icons_flutter.dart';

import '../../../core/theme/app_colors.dart';
import '../../../core/theme/app_theme.dart';
import '../../../core/theme/theme_controller.dart';
import '../../../data/services/auth_service.dart';
import '../../../l10n/app_localizations.dart';
import '../../../routes/app_routes.dart';
import '../../../shared/widgets/failure_view.dart';
import '../../../shared/widgets/skeletons.dart';
import '../controllers/profile_controller.dart';

/// Screens 09 + 10 — profile, and the settings that steer the agent.
class ProfileScreen extends GetView<ProfileController> {
  const ProfileScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final text = Theme.of(context).textTheme;
    final auth = Get.find<AuthService>();

    return Scaffold(
      body: SafeArea(
        child: Obx(() {
          if (controller.loading.value) {
            return const Padding(
              padding: EdgeInsets.all(AppSpacing.xl),
              child: JobListSkeleton(count: 3),
            );
          }

          final failure = controller.error.value;
          if (failure != null) {
            return FailureView(failure: failure, onRetry: controller.load);
          }

          final profile = controller.profile.value;
          final user = auth.user.value;

          return RefreshIndicator(
            onRefresh: controller.load,
            color: AppColors.primary,
            child: ListView(
              physics: const AlwaysScrollableScrollPhysics(),
              padding: const EdgeInsets.fromLTRB(
                  AppSpacing.xl, AppSpacing.lg, AppSpacing.xl, AppSpacing.section),
              children: [
                Row(
                  children: [
                    Container(
                      width: 52,
                      height: 52,
                      alignment: Alignment.center,
                      decoration: const BoxDecoration(
                        gradient: AppColors.primaryGradient,
                        borderRadius: AppRadii.pillShape,
                      ),
                      child: Text(
                        user?.initials ?? '?',
                        style: const TextStyle(
                          color: Colors.white,
                          fontWeight: FontWeight.w800,
                          fontSize: 17,
                        ),
                      ),
                    ),
                    const SizedBox(width: AppSpacing.lg),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(user?.displayName ?? user?.email ?? '',
                              style: text.headlineSmall),
                          if (user?.displayName != null)
                            Text(user!.email, style: text.bodySmall),
                        ],
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: AppSpacing.xl),

                // ---- master CV ----
                _Card(
                  onTap: () async {
                    await Get.toNamed<void>(Routes.masterCv);
                    await controller.load();
                  },
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          Icon(MdiIcons.fileDocumentOutline,
                              size: 18, color: AppColors.primary),
                          const SizedBox(width: AppSpacing.sm),
                          Expanded(
                              child: Text(l.masterCv, style: text.titleMedium)),
                          Text(
                            (profile?.hasCv ?? false) ? l.replace : l.upload,
                            style: text.labelMedium
                                ?.copyWith(color: AppColors.primary),
                          ),
                          Icon(Icons.chevron_right_rounded,
                              size: 18, color: AppColors.primary),
                        ],
                      ),
                      const SizedBox(height: AppSpacing.md),

                      if (profile?.hasCv ?? false) ...[
                        Text(
                          '${profile!.masterCv.length} characters',
                          style: text.bodySmall,
                        ),
                        const SizedBox(height: AppSpacing.md),
                        _StrengthMeter(value: profile.cvStrength, label: l.cvStrength),
                      ] else
                        // Direct about the consequence: scoring refuses to run
                        // without a CV, so this is not a cosmetic gap.
                        Container(
                          padding: const EdgeInsets.all(AppSpacing.md),
                          decoration: BoxDecoration(
                            color: AppColors.warn.withValues(alpha: 0.12),
                            borderRadius: AppRadii.innerShape,
                          ),
                          child: Row(
                            children: [
                              Icon(MdiIcons.alertCircleOutline,
                                  size: 16, color: AppColors.warn),
                              const SizedBox(width: AppSpacing.sm),
                              Expanded(
                                child: Text(l.nothingToReviewBody,
                                    style: text.bodySmall),
                              ),
                            ],
                          ),
                        ),
                    ],
                  ),
                ),
                const SizedBox(height: AppSpacing.md),

                // ---- threshold ----
                _ThresholdCard(controller: controller),
                const SizedBox(height: AppSpacing.md),

                // ---- keywords ----
                if ((profile?.preferredSkills ?? []).isNotEmpty)
                  _Card(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(l.targetKeywords, style: text.titleMedium),
                        const SizedBox(height: AppSpacing.md),
                        Wrap(
                          spacing: AppSpacing.sm,
                          runSpacing: AppSpacing.sm,
                          children: [
                            for (final s in profile!.preferredSkills)
                              Chip(label: Text(s)),
                          ],
                        ),
                      ],
                    ),
                  ),
                const SizedBox(height: AppSpacing.md),

                // ---- appearance ----
                _Card(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(l.theme, style: text.titleMedium),
                      const SizedBox(height: AppSpacing.md),
                      const _ThemePicker(),
                      const SizedBox(height: AppSpacing.xl),
                      Text(l.language, style: text.titleMedium),
                      const SizedBox(height: AppSpacing.md),
                      const _LanguagePicker(),
                    ],
                  ),
                ),
                const SizedBox(height: AppSpacing.xl),

                OutlinedButton.icon(
                  onPressed: () async {
                    await auth.signOut();
                    Get.offAllNamed<void>(Routes.signIn);
                  },
                  icon: Icon(MdiIcons.logoutVariant, size: 18),
                  style: OutlinedButton.styleFrom(
                    foregroundColor: AppColors.danger,
                    side: BorderSide(
                        color: AppColors.danger.withValues(alpha: 0.35)),
                  ),
                  label: Text(l.signOut),
                ),
              ],
            ),
          );
        }),
      ),
    );
  }
}

class _Card extends StatelessWidget {
  const _Card({required this.child, this.onTap});

  final Widget child;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final body = Container(
      width: double.infinity,
      padding: const EdgeInsets.all(AppSpacing.lg),
      decoration: BoxDecoration(
        color: dark ? AppColors.darkSurface : AppColors.surface,
        borderRadius: AppRadii.cardShape,
        boxShadow: dark ? AppShadows.cardDark : AppShadows.card,
      ),
      child: child,
    );

    if (onTap == null) return body;
    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: onTap,
        borderRadius: AppRadii.cardShape,
        child: body,
      ),
    );
  }
}

class _StrengthMeter extends StatelessWidget {
  const _StrengthMeter({required this.value, required this.label});

  final int value;
  final String label;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final dark = Theme.of(context).brightness == Brightness.dark;
    final colour = AppColors.forScore(value);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Expanded(child: Text(label, style: text.bodySmall)),
            Text('$value',
                style: text.titleMedium?.copyWith(color: colour)),
          ],
        ),
        const SizedBox(height: AppSpacing.sm),
        ClipRRect(
          borderRadius: BorderRadius.circular(99),
          child: LinearProgressIndicator(
            value: value / 100,
            minHeight: 6,
            backgroundColor:
                dark ? AppColors.darkSurfaceRaised : const Color(0xFFE3EDF4),
            valueColor: AlwaysStoppedAnimation(colour),
          ),
        ),
      ],
    );
  }
}

/// The dial that "predicts its own consequence", per the design.
class _ThresholdCard extends StatelessWidget {
  const _ThresholdCard({required this.controller});

  final ProfileController controller;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final text = Theme.of(context).textTheme;

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(AppSpacing.xl),
      decoration: BoxDecoration(
        gradient: AppColors.primaryGradient,
        borderRadius: AppRadii.cardShape,
        boxShadow: AppShadows.gradient,
      ),
      child: Obx(
        () => Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        l.scoreThreshold.toUpperCase(),
                        style: text.labelSmall?.copyWith(
                          color: Colors.white.withValues(alpha: 0.75),
                        ),
                      ),
                      const SizedBox(height: AppSpacing.xs),
                      Text(
                        l.thresholdHelp,
                        style: text.bodySmall?.copyWith(
                          color: Colors.white.withValues(alpha: 0.9),
                        ),
                      ),
                    ],
                  ),
                ),
                Text(
                  '${controller.threshold.value}',
                  style: const TextStyle(
                    color: Colors.white,
                    fontSize: 38,
                    fontWeight: FontWeight.w800,
                    height: 1,
                  ),
                ),
              ],
            ),
            const SizedBox(height: AppSpacing.md),

            SliderTheme(
              data: SliderThemeData(
                activeTrackColor: Colors.white,
                inactiveTrackColor: Colors.white.withValues(alpha: 0.3),
                thumbColor: Colors.white,
                overlayColor: Colors.white.withValues(alpha: 0.15),
                trackHeight: 4,
              ),
              child: Slider(
                value: controller.threshold.value.toDouble(),
                min: 40,
                max: 95,
                divisions: 11,
                // Commit on release, not on every tick: dragging would
                // otherwise fire a PATCH per pixel.
                onChanged: (v) => controller.threshold.value = v.round(),
                onChangeEnd: (_) => controller.saveThreshold(),
              ),
            ),

            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(l.widerNet,
                    style: text.labelSmall
                        ?.copyWith(color: Colors.white.withValues(alpha: 0.75))),
                Text(l.onlyTheBest,
                    style: text.labelSmall
                        ?.copyWith(color: Colors.white.withValues(alpha: 0.75))),
              ],
            ),
            const SizedBox(height: AppSpacing.md),

            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(AppSpacing.md),
              decoration: BoxDecoration(
                color: Colors.white.withValues(alpha: 0.16),
                borderRadius: AppRadii.innerShape,
              ),
              child: Text(
                l.aboutMinutes(controller.estimatedPerDay),
                style: text.bodyMedium?.copyWith(color: Colors.white),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _ThemePicker extends StatelessWidget {
  const _ThemePicker();

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final theme = Get.find<ThemeController>();

    return Obx(
      () => SegmentedButton<String>(
        segments: [
          ButtonSegment(value: 'system', label: Text(l.themeSystem)),
          ButtonSegment(value: 'light', label: Text(l.themeLight)),
          ButtonSegment(value: 'dark', label: Text(l.themeDark)),
        ],
        selected: {
          switch (theme.themeMode.value) {
            ThemeMode.light => 'light',
            ThemeMode.dark => 'dark',
            ThemeMode.system => 'system',
          }
        },
        showSelectedIcon: false,
        onSelectionChanged: (s) => theme.setThemeMode(switch (s.first) {
          'light' => ThemeMode.light,
          'dark' => ThemeMode.dark,
          _ => ThemeMode.system,
        }),
      ),
    );
  }
}

class _LanguagePicker extends StatelessWidget {
  const _LanguagePicker();

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final theme = Get.find<ThemeController>();

    return Obx(
      () => SegmentedButton<String>(
        segments: [
          ButtonSegment(value: 'en', label: Text(l.english)),
          ButtonSegment(value: 'sw', label: Text(l.swahili)),
        ],
        selected: {theme.locale.value?.languageCode ?? 'en'},
        showSelectedIcon: false,
        onSelectionChanged: (s) => theme.setLocale(Locale(s.first)),
      ),
    );
  }
}
