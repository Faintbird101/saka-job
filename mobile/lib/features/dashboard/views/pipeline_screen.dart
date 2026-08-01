import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../../core/theme/app_colors.dart';
import '../../../core/theme/app_theme.dart';
import '../../../l10n/app_localizations.dart';
import '../../../shared/widgets/failure_view.dart';
import '../../../shared/widgets/skeletons.dart';
import '../controllers/pipeline_controller.dart';

/// Screen 08 — the funnel.
class PipelineScreen extends GetView<PipelineController> {
  const PipelineScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final text = Theme.of(context).textTheme;

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

          final stages = controller.stages;
          final labels = <String, ({String title, String sub})>{
            'queue': (title: l.inYourQueue, sub: l.waitingOnYou),
            'applied': (title: l.applied, sub: l.sentNoReply),
            'screening': (title: l.statusAcknowledged, sub: l.repliesNeedingYou),
            'interview': (title: l.interview, sub: l.statusInterviewing),
            'offer': (title: l.offer, sub: l.statusOfferReceived),
          };

          return RefreshIndicator(
            onRefresh: controller.refresh,
            color: AppColors.primary,
            child: ListView(
              physics: const AlwaysScrollableScrollPhysics(),
              padding: const EdgeInsets.fromLTRB(
                  AppSpacing.xl, AppSpacing.lg, AppSpacing.xl, AppSpacing.section),
              children: [
                Text(l.pipeline, style: text.headlineMedium),
                Text(
                  l.liveApplications(controller.live),
                  style: text.bodyMedium,
                ),
                const SizedBox(height: AppSpacing.lg),

                _FunnelBar(stages: stages),
                const SizedBox(height: AppSpacing.xl),

                for (final stage in stages)
                  Padding(
                    padding: const EdgeInsets.only(bottom: AppSpacing.sm),
                    child: _StageRow(
                      title: labels[stage.key]?.title ?? stage.key,
                      subtitle: labels[stage.key]?.sub ?? '',
                      count: stage.count,
                      colour: _stageColour(stage.key),
                    ),
                  ),

                const SizedBox(height: AppSpacing.lg),
                _ProofCard(
                  replyRate: controller.replyRate,
                  live: controller.live,
                ),
              ],
            ),
          );
        }),
      ),
    );
  }

  static Color _stageColour(String key) => switch (key) {
        'queue' => AppColors.warn,
        'applied' => AppColors.primaryLight,
        'screening' => AppColors.primary,
        'interview' => AppColors.success,
        'offer' => AppColors.danger,
        _ => AppColors.muted,
      };
}

/// The single bar across the top: one segment per stage, sized by share.
class _FunnelBar extends StatelessWidget {
  const _FunnelBar({required this.stages});

  final List<Stage> stages;

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final total = stages.fold<int>(0, (sum, s) => sum + s.count);

    if (total == 0) {
      return Container(
        height: 10,
        decoration: BoxDecoration(
          color: dark ? AppColors.darkSurfaceRaised : const Color(0xFFE3EDF4),
          borderRadius: AppRadii.pillShape,
        ),
      );
    }

    return ClipRRect(
      borderRadius: AppRadii.pillShape,
      child: SizedBox(
        height: 10,
        child: Row(
          children: [
            for (final s in stages)
              if (s.count > 0)
                Expanded(
                  flex: s.count,
                  child: Container(
                    color: PipelineScreen._stageColour(s.key),
                    // A hairline between segments so adjacent similar colours
                    // stay distinguishable.
                    margin: const EdgeInsets.only(right: 1),
                  ),
                ),
          ],
        ),
      ),
    );
  }
}

class _StageRow extends StatelessWidget {
  const _StageRow({
    required this.title,
    required this.subtitle,
    required this.count,
    required this.colour,
  });

  final String title;
  final String subtitle;
  final int count;
  final Color colour;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final dark = Theme.of(context).brightness == Brightness.dark;

    return Container(
      padding: const EdgeInsets.symmetric(
          horizontal: AppSpacing.lg, vertical: AppSpacing.md),
      decoration: BoxDecoration(
        color: dark ? AppColors.darkSurface : AppColors.surface,
        borderRadius: AppRadii.innerShape,
        boxShadow: dark ? AppShadows.cardDark : AppShadows.card,
      ),
      child: Row(
        children: [
          Container(
            width: 8,
            height: 8,
            decoration: BoxDecoration(color: colour, shape: BoxShape.circle),
          ),
          const SizedBox(width: AppSpacing.md),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title, style: text.titleMedium),
                if (subtitle.isNotEmpty)
                  Text(subtitle, style: text.bodySmall),
              ],
            ),
          ),
          Text('$count', style: text.headlineSmall),
        ],
      ),
    );
  }
}

/// The headline number. Says nothing rather than something flattering when
/// there is not yet enough evidence to say anything true.
class _ProofCard extends StatelessWidget {
  const _ProofCard({required this.replyRate, required this.live});

  final double? replyRate;
  final int live;

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
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l.pipeline.toUpperCase(),
            style: text.labelSmall?.copyWith(
              color: Colors.white.withValues(alpha: 0.75),
            ),
          ),
          const SizedBox(height: AppSpacing.sm),
          if (replyRate == null)
            Text(
              l.nothingToReviewBody,
              style: text.bodyLarge?.copyWith(color: Colors.white),
            )
          else
            Row(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                Text(
                  '${(replyRate! * 100).round()}%',
                  style: const TextStyle(
                    color: Colors.white,
                    fontSize: 40,
                    fontWeight: FontWeight.w800,
                    height: 1,
                  ),
                ),
                const SizedBox(width: AppSpacing.md),
                Expanded(
                  child: Padding(
                    padding: const EdgeInsets.only(bottom: 4),
                    child: Text(
                      l.liveApplications(live),
                      style: text.bodyMedium?.copyWith(
                        color: Colors.white.withValues(alpha: 0.9),
                      ),
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
