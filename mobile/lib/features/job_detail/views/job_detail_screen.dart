import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:material_design_icons_flutter/material_design_icons_flutter.dart';

import '../../../core/theme/app_colors.dart';
import '../../../core/theme/app_theme.dart';
import '../../../data/models/job.dart';
import '../../../l10n/app_localizations.dart';
import '../../../routes/app_routes.dart';
import '../../../shared/widgets/app_toast.dart';
import '../../../shared/widgets/axis_bars.dart';
import '../../../shared/widgets/failure_view.dart';
import '../../../shared/widgets/skeletons.dart';
import '../controllers/job_detail_controller.dart';

/// Screen 06 — the job, and why it scored what it scored.
class JobDetailScreen extends GetView<JobDetailController> {
  const JobDetailScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);

    return Scaffold(
      body: SafeArea(
        child: Obx(() {
          final failure = controller.error.value;
          if (failure != null && controller.job.value == null) {
            return FailureView(failure: failure, onRetry: controller.load);
          }

          final job = controller.job.value;
          if (job == null || controller.loading.value) {
            return const Padding(
              padding: EdgeInsets.all(AppSpacing.xl),
              child: JobCardSkeleton(),
            );
          }

          return Column(
            children: [
              _Header(job: job),
              Expanded(
                child: ListView(
                  padding: const EdgeInsets.fromLTRB(
                      AppSpacing.xl, 0, AppSpacing.xl, AppSpacing.xl),
                  children: [
                    _ScoreCard(job: job),
                    const SizedBox(height: AppSpacing.lg),
                    if (job.hasDocuments) ...[
                      _DocumentsRow(job: job),
                      const SizedBox(height: AppSpacing.md),
                    ],
                    if (job.matchedSkills.isNotEmpty)
                      _SkillGroup(
                        title: l.matchedSkills,
                        skills: job.matchedSkills,
                        positive: true,
                      ),
                    if (job.missingSkills.isNotEmpty) ...[
                      const SizedBox(height: AppSpacing.md),
                      _SkillGroup(
                        title: l.missingSkills,
                        skills: job.missingSkills,
                        positive: false,
                      ),
                    ],
                    const SizedBox(height: AppSpacing.section),
                  ],
                ),
              ),
              if (job.isAwaitingApproval) _Actions(job: job),
            ],
          );
        }),
      ),
    );
  }
}

class _Header extends StatelessWidget {
  const _Header({required this.job});

  final Job job;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;

    return Padding(
      padding: const EdgeInsets.fromLTRB(
          AppSpacing.sm, AppSpacing.sm, AppSpacing.xl, AppSpacing.lg),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          IconButton(
            icon: Icon(MdiIcons.chevronLeft),
            onPressed: Get.back<void>,
          ),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(job.organization, style: text.bodyMedium),
                Text(job.title, style: text.headlineSmall),
                if (job.locationLabel.isNotEmpty || job.workArrangement != null) ...[
                  const SizedBox(height: AppSpacing.sm),
                  Wrap(
                    spacing: AppSpacing.sm,
                    runSpacing: AppSpacing.xs,
                    children: [
                      if (job.workArrangement != null) _Pill(job.workArrangement!),
                      if (job.locationLabel.isNotEmpty) _Pill(job.locationLabel),
                      if (job.employmentType != null) _Pill(job.employmentType!),
                      if (job.salaryLabel != null) _Pill(job.salaryLabel!),
                    ],
                  ),
                ],
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _Pill extends StatelessWidget {
  const _Pill(this.label);

  final String label;

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    return Container(
      padding: const EdgeInsets.symmetric(
          horizontal: AppSpacing.md, vertical: 5),
      decoration: BoxDecoration(
        color: dark ? AppColors.darkSurfaceRaised : AppColors.primaryTint,
        borderRadius: AppRadii.pillShape,
      ),
      child: Text(
        label,
        style: TextStyle(
          fontSize: 11.5,
          fontWeight: FontWeight.w600,
          color: dark ? AppColors.darkInk : AppColors.ink,
        ),
      ),
    );
  }
}

/// The gradient score card with the five axes.
class _ScoreCard extends StatelessWidget {
  const _ScoreCard({required this.job});

  final Job job;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final text = Theme.of(context).textTheme;
    final score = job.score;
    final axes = job.axes;

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(AppSpacing.xl),
      decoration: BoxDecoration(
        gradient: AppColors.primaryGradient,
        borderRadius: AppRadii.cardShape,
        boxShadow: AppShadows.gradient,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                score?.toString() ?? '–',
                style: const TextStyle(
                  color: Colors.white,
                  fontSize: 46,
                  fontWeight: FontWeight.w800,
                  height: 1,
                ),
              ),
              const SizedBox(width: AppSpacing.md),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      _verdict(l, score),
                      style: text.titleLarge?.copyWith(color: Colors.white),
                    ),
                    if (job.aiSummary != null)
                      Text(
                        job.aiSummary!,
                        maxLines: 3,
                        overflow: TextOverflow.ellipsis,
                        style: text.bodySmall?.copyWith(
                          color: Colors.white.withValues(alpha: 0.85),
                        ),
                      ),
                  ],
                ),
              ),
            ],
          ),

          if (axes != null && axes.hasAny) ...[
            const SizedBox(height: AppSpacing.xl),
            AxisBars(axes: axes, onGradient: true),
          ] else ...[
            const SizedBox(height: AppSpacing.lg),
            // Honest about the gap: jobs scored before the axes existed have
            // no breakdown, and inventing five bars would defeat the point.
            Container(
              padding: const EdgeInsets.all(AppSpacing.md),
              decoration: BoxDecoration(
                color: Colors.white.withValues(alpha: 0.16),
                borderRadius: AppRadii.innerShape,
              ),
              child: Text(
                l.noDiffYet,
                style: text.bodySmall?.copyWith(color: Colors.white),
              ),
            ),
          ],
        ],
      ),
    );
  }

  String _verdict(L10n l, int? score) {
    if (score == null) return l.statusNew;
    if (score >= 85) return l.strongMatch;
    if (score >= 70) return l.goodMatch;
    if (score >= 50) return l.partialMatch;
    return l.weakMatch;
  }
}

/// Opens the generated CV and cover letter. Only shown once they exist —
/// offering a link to documents that were never generated would be a dead end.
class _DocumentsRow extends StatelessWidget {
  const _DocumentsRow({required this.job});

  final Job job;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final text = Theme.of(context).textTheme;
    final dark = Theme.of(context).brightness == Brightness.dark;

    return Material(
      color: dark ? AppColors.darkSurface : AppColors.surface,
      borderRadius: AppRadii.cardShape,
      child: InkWell(
        borderRadius: AppRadii.cardShape,
        onTap: () => Get.toNamed<void>(
          Routes.documents,
          arguments: {'id': job.id, 'title': job.title},
        ),
        child: Padding(
          padding: const EdgeInsets.all(AppSpacing.lg),
          child: Row(
            children: [
              Icon(MdiIcons.fileDocumentCheckOutline,
                  size: 20, color: AppColors.primary),
              const SizedBox(width: AppSpacing.md),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(l.tailoredCv, style: text.titleMedium),
                    Text(l.coverLetter, style: text.bodySmall),
                  ],
                ),
              ),
              Icon(Icons.chevron_right_rounded,
                  size: 20, color: AppColors.primary),
            ],
          ),
        ),
      ),
    );
  }
}

class _SkillGroup extends StatelessWidget {
  const _SkillGroup({
    required this.title,
    required this.skills,
    required this.positive,
  });

  final String title;
  final List<String> skills;
  final bool positive;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final dark = Theme.of(context).brightness == Brightness.dark;
    final tint = positive ? AppColors.success : AppColors.warn;

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(AppSpacing.lg),
      decoration: BoxDecoration(
        color: dark ? AppColors.darkSurface : AppColors.surface,
        borderRadius: AppRadii.cardShape,
        boxShadow: dark ? AppShadows.cardDark : AppShadows.card,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(
                positive ? MdiIcons.checkCircle : MdiIcons.alertCircleOutline,
                size: 16,
                color: tint,
              ),
              const SizedBox(width: AppSpacing.sm),
              Text(title, style: text.labelLarge),
            ],
          ),
          const SizedBox(height: AppSpacing.md),
          Wrap(
            spacing: AppSpacing.sm,
            runSpacing: AppSpacing.sm,
            children: [
              for (final s in skills)
                Container(
                  padding: const EdgeInsets.symmetric(
                      horizontal: AppSpacing.md, vertical: 6),
                  decoration: BoxDecoration(
                    color: tint.withValues(alpha: dark ? 0.18 : 0.1),
                    borderRadius: AppRadii.pillShape,
                  ),
                  child: Text(
                    s,
                    style: TextStyle(
                      fontSize: 12,
                      fontWeight: FontWeight.w600,
                      color: dark ? AppColors.darkInk : AppColors.ink,
                    ),
                  ),
                ),
            ],
          ),
        ],
      ),
    );
  }
}

class _Actions extends StatelessWidget {
  const _Actions({required this.job});

  final Job job;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final controller = Get.find<JobDetailController>();
    final dark = Theme.of(context).brightness == Brightness.dark;

    return Container(
      padding: const EdgeInsets.fromLTRB(
          AppSpacing.xl, AppSpacing.md, AppSpacing.xl, AppSpacing.lg),
      decoration: BoxDecoration(
        color: dark ? AppColors.darkSurface : AppColors.surface,
        boxShadow: dark ? AppShadows.cardDark : AppShadows.card,
      ),
      child: Obx(
        () => Row(
          children: [
            SizedBox(
              width: 64,
              child: OutlinedButton(
                onPressed: controller.acting.value
                    ? null
                    : () async {
                        if (await controller.pass()) {
                          AppToast.info(l.statusRejected);
                          Get.back<bool>(result: true);
                        }
                      },
                style: OutlinedButton.styleFrom(
                  minimumSize: const Size(64, 52),
                  padding: EdgeInsets.zero,
                  foregroundColor: AppColors.danger,
                  side: BorderSide(color: AppColors.danger.withValues(alpha: 0.4)),
                ),
                child: Icon(MdiIcons.close, size: 22),
              ),
            ),
            const SizedBox(width: AppSpacing.md),
            Expanded(
              child: FilledButton(
                onPressed: controller.acting.value
                    ? null
                    : () async {
                        if (await controller.approve()) {
                          AppToast.success(l.statusApproved);
                          Get.back<bool>(result: true);
                        }
                      },
                child: controller.acting.value
                    ? const SizedBox(
                        height: 20,
                        width: 20,
                        child: CircularProgressIndicator(
                            strokeWidth: 2.2, color: Colors.white),
                      )
                    : Text(l.approveAndApply),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
