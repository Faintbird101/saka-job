import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:get/get.dart';
import 'package:material_design_icons_flutter/material_design_icons_flutter.dart';

import '../../../core/theme/app_colors.dart';
import '../../../core/theme/app_theme.dart';
import '../../../data/models/job.dart';
import '../../../l10n/app_localizations.dart';
import '../../../shared/widgets/failure_view.dart';
import '../../../shared/widgets/skeletons.dart';
import '../controllers/triage_controller.dart';

/// Screen 05 — triage.
///
/// Drag right to approve, left to pass. The buttons underneath do the same
/// thing, because a swipe-only interface is unusable one-handed on a large
/// phone and invisible to screen readers.
class TriageScreen extends GetView<TriageController> {
  const TriageScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final text = Theme.of(context).textTheme;

    return Scaffold(
      appBar: AppBar(
        title: Text(l.approve),
        leading: IconButton(
          icon: Icon(MdiIcons.close),
          onPressed: Get.back<void>,
        ),
      ),
      body: SafeArea(
        child: Obx(() {
          if (controller.loading.value) {
            return const Padding(
              padding: EdgeInsets.all(AppSpacing.xl),
              child: JobCardSkeleton(),
            );
          }

          final failure = controller.error.value;
          if (failure != null) {
            return FailureView(failure: failure, onRetry: controller.load);
          }

          final job = controller.current;
          if (job == null) {
            return _AllDone(message: l.nothingToReview);
          }

          return Column(
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(
                    AppSpacing.xl, 0, AppSpacing.xl, AppSpacing.md),
                child: Row(
                  children: [
                    Text(
                      l.jobsLeft(controller.remaining),
                      style: text.bodyMedium,
                    ),
                    const SizedBox(width: AppSpacing.sm),
                    Text('·', style: text.bodyMedium),
                    const SizedBox(width: AppSpacing.sm),
                    Text(
                      l.aboutMinutes(controller.minutesLeft),
                      style: text.bodyMedium,
                    ),
                  ],
                ),
              ),

              Expanded(
                child: Padding(
                  padding: const EdgeInsets.symmetric(horizontal: AppSpacing.xl),
                  // Keyed by job id so Flutter treats each card as a new
                  // widget and the dismiss animation restarts cleanly.
                  child: _SwipeCard(
                    key: ValueKey(job.id),
                    job: job,
                    onApprove: () => controller.approve(job),
                    onPass: () => controller.pass(job),
                  ),
                ),
              ),

              Padding(
                padding: const EdgeInsets.all(AppSpacing.xl),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    _CircleAction(
                      icon: MdiIcons.close,
                      colour: AppColors.danger,
                      onTap: () => controller.pass(job),
                      semanticLabel: l.pass,
                    ),
                    const SizedBox(width: AppSpacing.section),
                    _CircleAction(
                      icon: MdiIcons.check,
                      colour: AppColors.success,
                      filled: true,
                      onTap: () => controller.approve(job),
                      semanticLabel: l.approve,
                    ),
                  ],
                ),
              ),
            ],
          );
        }),
      ),
    );
  }
}

/// The draggable card.
///
/// Dismissible rather than a hand-rolled gesture: it gives the drag threshold,
/// the fling velocity and the resistance curve for free, and those are exactly
/// the details that make a swipe feel right or wrong.
class _SwipeCard extends StatelessWidget {
  const _SwipeCard({
    super.key,
    required this.job,
    required this.onApprove,
    required this.onPass,
  });

  final Job job;
  final VoidCallback onApprove;
  final VoidCallback onPass;

  @override
  Widget build(BuildContext context) {
    return Dismissible(
      key: ValueKey('swipe-${job.id}'),
      // Both directions need a deliberate drag: an accidental brush must not
      // send an application.
      dismissThresholds: const {
        DismissDirection.startToEnd: 0.35,
        DismissDirection.endToStart: 0.35,
      },
      background: const _SwipeHint(
        alignment: Alignment.centerLeft,
        colour: AppColors.success,
        icon: Icons.check_rounded,
      ),
      secondaryBackground: const _SwipeHint(
        alignment: Alignment.centerRight,
        colour: AppColors.danger,
        icon: Icons.close_rounded,
      ),
      onDismissed: (direction) {
        // A confirmed decision deserves physical feedback — it is the one
        // moment in the app with a real consequence.
        HapticFeedback.mediumImpact();
        if (direction == DismissDirection.startToEnd) {
          onApprove();
        } else {
          onPass();
        }
      },
      child: _TriageCardBody(job: job),
    );
  }
}

class _SwipeHint extends StatelessWidget {
  const _SwipeHint({
    required this.alignment,
    required this.colour,
    required this.icon,
  });

  final Alignment alignment;
  final Color colour;
  final IconData icon;

  @override
  Widget build(BuildContext context) {
    return Container(
      alignment: alignment,
      padding: const EdgeInsets.symmetric(horizontal: AppSpacing.section),
      decoration: BoxDecoration(
        color: colour.withValues(alpha: 0.12),
        borderRadius: AppRadii.cardShape,
      ),
      child: Icon(icon, color: colour, size: 34),
    );
  }
}

class _TriageCardBody extends StatelessWidget {
  const _TriageCardBody({required this.job});

  final Job job;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final text = Theme.of(context).textTheme;

    return Container(
      width: double.infinity,
      decoration: BoxDecoration(
        gradient: AppColors.primaryGradient,
        borderRadius: AppRadii.cardShape,
        boxShadow: AppShadows.gradient,
      ),
      padding: const EdgeInsets.all(AppSpacing.xxl),
      child: SingleChildScrollView(
        child: Column(
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
                        job.organization,
                        style: text.titleMedium?.copyWith(color: Colors.white),
                      ),
                      if (job.locationLabel.isNotEmpty)
                        Text(
                          job.locationLabel,
                          style: text.bodySmall?.copyWith(
                            color: Colors.white.withValues(alpha: 0.8),
                          ),
                        ),
                    ],
                  ),
                ),
                if (job.score != null)
                  Column(
                    crossAxisAlignment: CrossAxisAlignment.end,
                    children: [
                      Text(
                        '${job.score}',
                        style: const TextStyle(
                          color: Colors.white,
                          fontSize: 40,
                          fontWeight: FontWeight.w800,
                          height: 1,
                        ),
                      ),
                      Text(
                        'MATCH',
                        style: TextStyle(
                          color: Colors.white.withValues(alpha: 0.75),
                          fontSize: 10,
                          fontWeight: FontWeight.w700,
                          letterSpacing: 0.8,
                        ),
                      ),
                    ],
                  ),
              ],
            ),
            const SizedBox(height: AppSpacing.lg),

            Text(
              job.title,
              style: text.headlineSmall?.copyWith(color: Colors.white),
            ),
            if (job.salaryLabel != null) ...[
              const SizedBox(height: AppSpacing.xs),
              Text(
                job.salaryLabel!,
                style: text.bodyLarge?.copyWith(
                  color: Colors.white.withValues(alpha: 0.9),
                ),
              ),
            ],
            const SizedBox(height: AppSpacing.xl),

            // The reasoning, straight from the scorer. Matched skills read as
            // ticks; the gaps are listed honestly rather than hidden, because
            // hiding them is what makes an agent feel like a black box.
            ...job.matchedSkills.take(5).map(
                  (s) => _Reason(text: s, positive: true),
                ),
            ...job.missingSkills.take(2).map(
                  (s) => _Reason(text: s, positive: false),
                ),

            if (job.aiSummary != null) ...[
              const SizedBox(height: AppSpacing.lg),
              Text(
                job.aiSummary!,
                style: text.bodyMedium?.copyWith(
                  color: Colors.white.withValues(alpha: 0.9),
                ),
              ),
            ],

            if (job.hasDocuments) ...[
              const SizedBox(height: AppSpacing.xl),
              Container(
                padding: const EdgeInsets.all(AppSpacing.md),
                decoration: BoxDecoration(
                  color: Colors.white.withValues(alpha: 0.16),
                  borderRadius: AppRadii.innerShape,
                ),
                child: Row(
                  children: [
                    Icon(MdiIcons.fileDocumentCheckOutline,
                        color: Colors.white, size: 18),
                    const SizedBox(width: AppSpacing.sm),
                    Expanded(
                      child: Text(
                        l.tailoredCv,
                        style: text.bodyMedium?.copyWith(color: Colors.white),
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class _Reason extends StatelessWidget {
  const _Reason({required this.text, required this.positive});

  final String text;
  final bool positive;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: AppSpacing.sm),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            positive ? MdiIcons.checkCircle : MdiIcons.alertCircleOutline,
            size: 16,
            color: Colors.white.withValues(alpha: positive ? 0.95 : 0.7),
          ),
          const SizedBox(width: AppSpacing.sm),
          Expanded(
            child: Text(
              text,
              style: TextStyle(
                color: Colors.white.withValues(alpha: positive ? 0.95 : 0.75),
                fontSize: 13.5,
                height: 1.35,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _CircleAction extends StatelessWidget {
  const _CircleAction({
    required this.icon,
    required this.colour,
    required this.onTap,
    required this.semanticLabel,
    this.filled = false,
  });

  final IconData icon;
  final Color colour;
  final VoidCallback onTap;
  final String semanticLabel;
  final bool filled;

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    return Semantics(
      button: true,
      label: semanticLabel,
      child: Material(
        color: filled
            ? colour
            : (dark ? AppColors.darkSurface : AppColors.surface),
        shape: const CircleBorder(),
        elevation: 0,
        child: InkWell(
          onTap: onTap,
          customBorder: const CircleBorder(),
          child: Container(
            width: filled ? 68 : 58,
            height: filled ? 68 : 58,
            alignment: Alignment.center,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              boxShadow: dark ? AppShadows.cardDark : AppShadows.card,
              color: Colors.transparent,
            ),
            child: Icon(
              icon,
              color: filled ? Colors.white : colour,
              size: filled ? 30 : 24,
            ),
          ),
        ),
      ),
    );
  }
}

class _AllDone extends StatelessWidget {
  const _AllDone({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(AppSpacing.section),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 64,
              height: 64,
              alignment: Alignment.center,
              decoration: BoxDecoration(
                color: AppColors.success.withValues(alpha: 0.12),
                borderRadius: AppRadii.pillShape,
              ),
              child: Icon(MdiIcons.checkAll,
                  color: AppColors.success, size: 30),
            ),
            const SizedBox(height: AppSpacing.xl),
            Text(message, style: text.headlineSmall, textAlign: TextAlign.center),
          ],
        ),
      ),
    );
  }
}
