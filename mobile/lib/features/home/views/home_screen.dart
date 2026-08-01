import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../../core/theme/app_colors.dart';
import '../../../core/theme/app_theme.dart';
import '../../../data/models/user.dart';
import '../../../data/services/auth_service.dart';
import '../../../l10n/app_localizations.dart';
import '../../../shared/widgets/failure_view.dart';
import '../../../shared/widgets/job_card.dart';
import '../../../routes/app_routes.dart';
import '../../../shared/widgets/skeletons.dart';
import '../controllers/home_controller.dart';

/// Screen 03 — the home feed.
///
/// Ordered by what the user has to do: the queue first, because the pipeline is
/// literally blocked on it, then everything else.
class HomeScreen extends GetView<HomeController> {
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final user = Get.find<AuthService>().user.value;

    return Scaffold(
      body: SafeArea(
        child: Obx(() {
          // Only a hard failure with nothing cached takes over the screen; a
          // refresh that fails with data on screen just raises a toast.
          final failure = controller.error.value;
          if (failure != null && controller.queue.isEmpty && controller.recent.isEmpty) {
            return FailureView(failure: failure, onRetry: controller.load);
          }

          return RefreshIndicator(
            onRefresh: controller.refresh,
            color: AppColors.primary,
            child: CustomScrollView(
              // Always scrollable so pull-to-refresh works even when the list
              // is short enough not to overflow.
              physics: const AlwaysScrollableScrollPhysics(),
              slivers: [
                SliverToBoxAdapter(child: _Greeting(user: user)),

                if (controller.replies.isNotEmpty)
                  SliverToBoxAdapter(
                    child: _RepliesBanner(count: controller.replies.length),
                  ),

                if (controller.loading.value)
                  const SliverPadding(
                    padding: EdgeInsets.fromLTRB(
                        AppSpacing.xl, AppSpacing.sm, AppSpacing.xl, 0),
                    sliver: SliverToBoxAdapter(child: JobListSkeleton()),
                  )
                else ...[
                  if (controller.queue.isNotEmpty)
                    _Section(
                      title: l.needsYourYes,
                      count: controller.queue.length,
                      jobs: controller.queue,
                      // The first card is the featured gradient one, per the
                      // design — it is the single most actionable thing.
                      featuredFirst: true,
                    )
                  else
                    SliverToBoxAdapter(
                      child: _EmptyQueue(title: l.nothingToReview, body: l.nothingToReviewBody),
                    ),

                  if (controller.recent.isNotEmpty)
                    _Section(
                      title: l.latestMatches,
                      count: controller.recent.length,
                      jobs: controller.recent,
                    ),
                ],

                const SliverToBoxAdapter(child: SizedBox(height: AppSpacing.section)),
              ],
            ),
          );
        }),
      ),
    );
  }
}

/// Surfaced above everything else: an interview invitation waiting unread is
/// more costly than an unapproved job, which simply waits.
class _RepliesBanner extends StatelessWidget {
  const _RepliesBanner({required this.count});

  final int count;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final text = Theme.of(context).textTheme;

    return Padding(
      padding: const EdgeInsets.fromLTRB(
          AppSpacing.xl, 0, AppSpacing.xl, AppSpacing.md),
      child: Material(
        color: AppColors.success.withValues(alpha: 0.12),
        borderRadius: AppRadii.innerShape,
        child: InkWell(
          borderRadius: AppRadii.innerShape,
          onTap: () async {
            await Get.toNamed<void>(Routes.replies);
            await Get.find<HomeController>().refresh();
          },
          child: Padding(
            padding: const EdgeInsets.all(AppSpacing.lg),
            child: Row(
              children: [
                Container(
                  width: 26,
                  height: 26,
                  alignment: Alignment.center,
                  decoration: const BoxDecoration(
                    color: AppColors.success,
                    shape: BoxShape.circle,
                  ),
                  child: Text(
                    '$count',
                    style: const TextStyle(
                      color: Colors.white,
                      fontSize: 12,
                      fontWeight: FontWeight.w800,
                    ),
                  ),
                ),
                const SizedBox(width: AppSpacing.md),
                Expanded(
                  child: Text(l.repliesNeedingYou, style: text.titleMedium),
                ),
                Icon(Icons.chevron_right_rounded,
                    color: AppColors.success, size: 20),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _Greeting extends StatelessWidget {
  const _Greeting({this.user});

  final AppUser? user;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final text = Theme.of(context).textTheme;
    final hour = DateTime.now().hour;
    final greeting = hour < 12
        ? l.goodMorning
        : (hour < 18 ? l.goodAfternoon : l.goodEvening);

    return Padding(
      padding: const EdgeInsets.fromLTRB(
          AppSpacing.xl, AppSpacing.lg, AppSpacing.xl, AppSpacing.md),
      child: Row(
        children: [
          Container(
            width: 44,
            height: 44,
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
                fontSize: 15,
              ),
            ),
          ),
          const SizedBox(width: AppSpacing.md),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(greeting, style: text.bodySmall),
                Text(
                  user?.firstName ?? '',
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: text.headlineSmall,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _Section extends StatelessWidget {
  const _Section({
    required this.title,
    required this.count,
    required this.jobs,
    this.featuredFirst = false,
  });

  final String title;
  final int count;
  final List jobs;
  final bool featuredFirst;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;

    return SliverPadding(
      padding: const EdgeInsets.fromLTRB(
          AppSpacing.xl, AppSpacing.md, AppSpacing.xl, 0),
      sliver: SliverList.separated(
        itemCount: jobs.length + 1,
        separatorBuilder: (_, _) => const SizedBox(height: AppSpacing.md),
        itemBuilder: (context, index) {
          if (index == 0) {
            return Padding(
              padding: const EdgeInsets.only(bottom: AppSpacing.xs),
              child: Text(title, style: text.headlineSmall),
            );
          }
          final job = jobs[index - 1];
          return JobCard(
            job: job,
            onTap: () async {
              await Get.toNamed<void>(
                Routes.jobDetail,
                arguments: {'id': job.id, 'job': job},
              );
              await Get.find<HomeController>().refresh();
            },
            featured: featuredFirst && index == 1,
            onAction: job.isAwaitingApproval
                ? () async {
                    await Get.toNamed<void>(Routes.triage);
                    // Whatever was decided in triage changed the queue, so the
                    // feed is stale the moment it returns.
                    await Get.find<HomeController>().refresh();
                  }
                : null,
          );
        },
      ),
    );
  }
}

class _EmptyQueue extends StatelessWidget {
  const _EmptyQueue({required this.title, required this.body});

  final String title;
  final String body;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final dark = Theme.of(context).brightness == Brightness.dark;

    return Container(
      margin: const EdgeInsets.symmetric(
          horizontal: AppSpacing.xl, vertical: AppSpacing.md),
      padding: const EdgeInsets.all(AppSpacing.xxl),
      decoration: BoxDecoration(
        color: dark ? AppColors.darkSurface : AppColors.surface,
        borderRadius: AppRadii.cardShape,
        boxShadow: dark ? AppShadows.cardDark : AppShadows.card,
      ),
      child: Column(
        children: [
          Text(title, textAlign: TextAlign.center, style: text.titleLarge),
          const SizedBox(height: AppSpacing.sm),
          Text(body, textAlign: TextAlign.center, style: text.bodyMedium),
        ],
      ),
    );
  }
}
