import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:material_design_icons_flutter/material_design_icons_flutter.dart';

import '../../../core/theme/app_colors.dart';
import '../../../core/theme/app_theme.dart';
import '../../../l10n/app_localizations.dart';
import '../../../routes/app_routes.dart';
import '../../../shared/widgets/failure_view.dart';
import '../../../shared/widgets/job_card.dart';
import '../../../shared/widgets/skeletons.dart';
import '../../../shared/widgets/status_label.dart';
import '../controllers/jobs_controller.dart';

/// Screen 04 — jobs by status.
class JobsScreen extends GetView<JobsController> {
  const JobsScreen({super.key});

  /// The filters worth surfacing, in pipeline order.
  ///
  /// Deliberately not all sixteen statuses: a chip row you have to scroll
  /// through twice is a worse filter than no filter. The rest remain reachable
  /// by search and from the pipeline screen.
  static const _filters = <String>[
    '',
    'AwaitingApproval',
    'Scored',
    'Applied',
    'Interviewing',
    'LowMatch',
  ];

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);

    return Scaffold(
      body: SafeArea(
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(
                  AppSpacing.xl, AppSpacing.lg, AppSpacing.xl, AppSpacing.md),
              child: TextField(
                onChanged: (v) => controller.search.value = v,
                textInputAction: TextInputAction.search,
                decoration: InputDecoration(
                  hintText: l.searchHint,
                  prefixIcon: Icon(MdiIcons.magnify, size: 20),
                  isDense: true,
                ),
              ),
            ),

            SizedBox(
              height: 40,
              child: Obx(() {
                // Read the observable HERE, in the Obx builder itself.
                //
                // Reading it inside itemBuilder instead looks equivalent but is
                // not: ListView builds items lazily, after Obx has already
                // returned, so Obx observes nothing and GetX throws "improper
                // use of GetX". Hoisting it into a local both fixes the
                // subscription and gives every chip the same value.
                final active = controller.status.value;

                return ListView.separated(
                  scrollDirection: Axis.horizontal,
                  padding: const EdgeInsets.symmetric(horizontal: AppSpacing.xl),
                  itemCount: _filters.length,
                  separatorBuilder: (_, _) => const SizedBox(width: AppSpacing.sm),
                  itemBuilder: (context, i) {
                    final value = _filters[i];
                    return _FilterChip(
                      label: value.isEmpty ? l.seeAll : statusLabel(l, value),
                      selected: active == value,
                      onTap: () => controller.setStatus(value),
                    );
                  },
                );
              }),
            ),
            const SizedBox(height: AppSpacing.md),

            Expanded(
              child: Obx(() {
                if (controller.loading.value) {
                  return const Padding(
                    padding: EdgeInsets.symmetric(horizontal: AppSpacing.xl),
                    child: JobListSkeleton(count: 4),
                  );
                }

                final failure = controller.error.value;
                if (failure != null) {
                  return FailureView(failure: failure, onRetry: controller.load);
                }

                if (controller.jobs.isEmpty) {
                  return _Empty(message: l.nothingToReview);
                }

                return RefreshIndicator(
                  onRefresh: controller.refresh,
                  color: AppColors.primary,
                  child: NotificationListener<ScrollNotification>(
                    // Prefetch a page before hitting the bottom, so scrolling
                    // does not stall waiting for the next request.
                    onNotification: (n) {
                      if (n.metrics.pixels >= n.metrics.maxScrollExtent - 400) {
                        controller.loadMore();
                      }
                      return false;
                    },
                    child: ListView.separated(
                      physics: const AlwaysScrollableScrollPhysics(),
                      padding: const EdgeInsets.fromLTRB(
                          AppSpacing.xl, 0, AppSpacing.xl, AppSpacing.section),
                      itemCount: controller.jobs.length + 1,
                      separatorBuilder: (_, _) =>
                          const SizedBox(height: AppSpacing.md),
                      itemBuilder: (context, i) {
                        if (i == controller.jobs.length) {
                          return Obx(
                            () => controller.loadingMore.value
                                ? const Padding(
                                    padding: EdgeInsets.only(top: AppSpacing.md),
                                    child: JobCardSkeleton(),
                                  )
                                : const SizedBox(height: AppSpacing.lg),
                          );
                        }

                        final job = controller.jobs[i];
                        return JobCard(
                          job: job,
                          onTap: () async {
                            await Get.toNamed<void>(
                              Routes.jobDetail,
                              arguments: {'id': job.id, 'job': job},
                            );
                            await controller.refresh();
                          },
                        );
                      },
                    ),
                  ),
                );
              }),
            ),
          ],
        ),
      ),
    );
  }
}

class _FilterChip extends StatelessWidget {
  const _FilterChip({
    required this.label,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    return Material(
      color: selected
          ? AppColors.primary
          : (dark ? AppColors.darkSurface : AppColors.surface),
      borderRadius: AppRadii.pillShape,
      child: InkWell(
        onTap: onTap,
        borderRadius: AppRadii.pillShape,
        child: Padding(
          padding: const EdgeInsets.symmetric(
              horizontal: AppSpacing.lg, vertical: AppSpacing.sm),
          child: Center(
            child: Text(
              label,
              style: TextStyle(
                fontSize: 12.5,
                fontWeight: FontWeight.w700,
                color: selected
                    ? Colors.white
                    : (dark ? AppColors.darkMuted : AppColors.muted),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _Empty extends StatelessWidget {
  const _Empty({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    return ListView(
      // A ListView rather than a Center, so pull-to-refresh still works on an
      // empty result.
      physics: const AlwaysScrollableScrollPhysics(),
      children: [
        const SizedBox(height: 120),
        Icon(MdiIcons.inboxOutline,
            size: 40, color: Theme.of(context).textTheme.bodySmall?.color),
        const SizedBox(height: AppSpacing.lg),
        Text(
          message,
          textAlign: TextAlign.center,
          style: Theme.of(context).textTheme.bodyMedium,
        ),
      ],
    );
  }
}
