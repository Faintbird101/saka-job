import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:material_design_icons_flutter/material_design_icons_flutter.dart';

import '../../../core/theme/app_colors.dart';
import '../../../core/theme/app_theme.dart';
import '../../../data/models/job_event.dart';
import '../../../l10n/app_localizations.dart';
import '../../../shared/widgets/failure_view.dart';
import '../../../shared/widgets/skeletons.dart';
import '../../../shared/widgets/status_label.dart';
import '../controllers/replies_controller.dart';

/// Screen 11 — confirm what an employer's reply actually meant.
class RepliesScreen extends GetView<RepliesController> {
  const RepliesScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);

    return Scaffold(
      appBar: AppBar(
        title: Text(l.repliesNeedingYou),
        leading: IconButton(
          icon: Icon(MdiIcons.chevronLeft),
          onPressed: Get.back<void>,
        ),
      ),
      body: SafeArea(
        child: Obx(() {
          if (controller.loading.value) {
            return const Padding(
              padding: EdgeInsets.all(AppSpacing.xl),
              child: JobListSkeleton(count: 2),
            );
          }

          final failure = controller.error.value;
          if (failure != null) {
            return FailureView(failure: failure, onRetry: controller.load);
          }

          if (controller.pending.isEmpty) {
            return _Empty(message: l.nothingToReview);
          }

          return RefreshIndicator(
            onRefresh: controller.refresh,
            color: AppColors.primary,
            child: ListView.separated(
              physics: const AlwaysScrollableScrollPhysics(),
              padding: const EdgeInsets.all(AppSpacing.xl),
              itemCount: controller.pending.length,
              separatorBuilder: (_, _) => const SizedBox(height: AppSpacing.md),
              itemBuilder: (context, i) =>
                  _ReplyCard(event: controller.pending[i]),
            ),
          );
        }),
      ),
    );
  }
}

class _ReplyCard extends StatelessWidget {
  const _ReplyCard({required this.event});

  final JobEvent event;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final text = Theme.of(context).textTheme;
    final dark = Theme.of(context).brightness == Brightness.dark;
    final controller = Get.find<RepliesController>();
    final colour = statusColour(event.suggestedStatus ?? '');

    return Container(
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
              Icon(_iconFor(event.kind), size: 18, color: colour),
              const SizedBox(width: AppSpacing.sm),
              Expanded(
                child: Text(
                  event.senderDomain ?? event.sender ?? '',
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: text.titleMedium,
                ),
              ),
              if (event.suggestedStatus != null)
                StatusChip(status: event.suggestedStatus!, compact: true),
            ],
          ),

          if (event.subject != null) ...[
            const SizedBox(height: AppSpacing.sm),
            Text(event.subject!, style: text.bodyLarge, maxLines: 2,
                overflow: TextOverflow.ellipsis),
          ],

          if (event.excerpt != null) ...[
            const SizedBox(height: AppSpacing.sm),
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(AppSpacing.md),
              decoration: BoxDecoration(
                color: dark ? AppColors.darkBg : AppColors.bg,
                borderRadius: AppRadii.innerShape,
              ),
              // The phrase that decided the classification. Showing it is what
              // makes the suggestion checkable rather than something to trust.
              child: Text(
                '“${event.excerpt!}”',
                style: text.bodySmall?.copyWith(fontStyle: FontStyle.italic),
              ),
            ),
          ],

          const SizedBox(height: AppSpacing.md),
          Row(
            children: [
              Icon(MdiIcons.gauge, size: 13,
                  color: event.isLowConfidence ? AppColors.warn : AppColors.muted),
              const SizedBox(width: 4),
              Text(
                '${event.confidence}%',
                style: text.labelSmall?.copyWith(
                  color: event.isLowConfidence ? AppColors.warn : null,
                ),
              ),
              if (event.classifier != null) ...[
                const SizedBox(width: AppSpacing.sm),
                Text('· ${event.classifier}', style: text.labelSmall),
              ],
              if (event.matchReason != null) ...[
                const SizedBox(width: AppSpacing.sm),
                Expanded(
                  child: Text(
                    '· ${event.matchReason}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: text.labelSmall,
                  ),
                ),
              ],
            ],
          ),

          const SizedBox(height: AppSpacing.lg),
          Obx(() {
            final busy = controller.acting.contains(event.id);
            return Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: busy
                        ? null
                        : () => controller.decide(event, accept: false),
                    style: OutlinedButton.styleFrom(
                      minimumSize: const Size.fromHeight(44),
                      foregroundColor: AppColors.muted,
                    ),
                    child: Text(l.noDismiss),
                  ),
                ),
                const SizedBox(width: AppSpacing.md),
                Expanded(
                  child: FilledButton(
                    onPressed: busy
                        ? null
                        : () => controller.decide(event, accept: true),
                    style: FilledButton.styleFrom(
                      minimumSize: const Size.fromHeight(44),
                      backgroundColor: colour,
                    ),
                    child: busy
                        ? const SizedBox(
                            height: 18,
                            width: 18,
                            child: CircularProgressIndicator(
                                strokeWidth: 2.2, color: Colors.white),
                          )
                        : Text(l.yesConfirm),
                  ),
                ),
              ],
            );
          }),
        ],
      ),
    );
  }

  static IconData _iconFor(String kind) => switch (kind) {
        'interview' => MdiIcons.calendarCheck,
        'offer' => MdiIcons.partyPopper,
        'rejection' => MdiIcons.emailRemoveOutline,
        'acknowledgement' => MdiIcons.emailCheckOutline,
        _ => MdiIcons.emailOutline,
      };
}

class _Empty extends StatelessWidget {
  const _Empty({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(AppSpacing.section),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(MdiIcons.checkAll, size: 40, color: AppColors.success),
            const SizedBox(height: AppSpacing.lg),
            Text(
              message,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          ],
        ),
      ),
    );
  }
}
