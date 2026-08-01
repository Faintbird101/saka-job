import 'package:flutter/material.dart';
import 'package:material_design_icons_flutter/material_design_icons_flutter.dart';

import '../../core/theme/app_colors.dart';
import '../../core/theme/app_theme.dart';
import '../../data/models/job.dart';
import '../../l10n/app_localizations.dart';
import 'score_badge.dart';

/// A job in a list.
///
/// Two variants, matching the design: [featured] is the gradient card used for
/// the item at the top of the queue, and the plain one is used everywhere else.
/// Same widget rather than two, because they carry identical information and
/// diverging them would guarantee they drift.
class JobCard extends StatelessWidget {
  const JobCard({
    super.key,
    required this.job,
    this.onTap,
    this.onAction,
    this.featured = false,
  });

  final Job job;
  final VoidCallback? onTap;

  /// The "Review" affordance. Null hides it — a job that is not waiting on you
  /// has nothing to review.
  final VoidCallback? onAction;
  final bool featured;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final theme = Theme.of(context);
    final dark = theme.brightness == Brightness.dark;
    final onCard = featured ? Colors.white : (dark ? AppColors.darkInk : AppColors.ink);
    final subdued = featured
        ? Colors.white.withValues(alpha: 0.82)
        : (dark ? AppColors.darkMuted : AppColors.muted);

    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: onTap,
        borderRadius: AppRadii.cardShape,
        child: Ink(
          decoration: BoxDecoration(
            gradient: featured ? AppColors.primaryGradient : null,
            color: featured ? null : (dark ? AppColors.darkSurface : AppColors.surface),
            borderRadius: AppRadii.cardShape,
            boxShadow: featured
                ? AppShadows.gradient
                : (dark ? AppShadows.cardDark : AppShadows.card),
          ),
          padding: const EdgeInsets.all(AppSpacing.lg),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  _Avatar(job: job, featured: featured),
                  const SizedBox(width: AppSpacing.md),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          job.organization,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: theme.textTheme.titleMedium?.copyWith(color: onCard),
                        ),
                        if (job.locationLabel.isNotEmpty)
                          Text(
                            job.locationLabel,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: theme.textTheme.bodySmall?.copyWith(color: subdued),
                          ),
                      ],
                    ),
                  ),
                  const SizedBox(width: AppSpacing.sm),
                  // On the featured card the badge sits on a gradient, where a
                  // tinted pill would disappear — so it inverts to solid white.
                  featured
                      ? _FeaturedScore(score: job.score)
                      : ScoreBadge(score: job.score),
                ],
              ),
              const SizedBox(height: AppSpacing.md),

              Text(
                job.title,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: theme.textTheme.titleLarge?.copyWith(color: onCard),
              ),

              if (job.salaryLabel != null || job.workArrangement != null) ...[
                const SizedBox(height: AppSpacing.xs),
                Text(
                  [job.salaryLabel, job.workArrangement]
                      .whereType<String>()
                      .join(' · '),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: theme.textTheme.bodyMedium?.copyWith(color: subdued),
                ),
              ],

              if (onAction != null) ...[
                const SizedBox(height: AppSpacing.lg),
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    if (job.hasDocuments)
                      Row(
                        children: [
                          Icon(MdiIcons.fileDocumentCheckOutline,
                              size: 15, color: subdued),
                          const SizedBox(width: 5),
                          Text(l.tailoredCv,
                              style: theme.textTheme.bodySmall
                                  ?.copyWith(color: subdued)),
                        ],
                      )
                    else
                      const SizedBox.shrink(),
                    _ReviewButton(featured: featured, onTap: onAction!, label: l.review),
                  ],
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

class _Avatar extends StatelessWidget {
  const _Avatar({required this.job, required this.featured});

  final Job job;
  final bool featured;

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final org = job.organization.trim();
    final initials = org.isEmpty
        ? '?'
        : org
            .split(RegExp(r'\s+'))
            .where((w) => w.isNotEmpty)
            .take(2)
            .map((w) => w[0].toUpperCase())
            .join();

    return Container(
      width: 40,
      height: 40,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: featured
            ? Colors.white.withValues(alpha: 0.22)
            : (dark ? AppColors.darkSurfaceRaised : AppColors.primaryTint),
        borderRadius: AppRadii.innerShape,
      ),
      child: Text(
        initials,
        style: TextStyle(
          color: featured ? Colors.white : AppColors.primary,
          fontWeight: FontWeight.w800,
          fontSize: 14,
        ),
      ),
    );
  }
}

/// The score on a gradient card: solid, because a translucent tint against the
/// gradient reads as a smudge rather than a badge.
class _FeaturedScore extends StatelessWidget {
  const _FeaturedScore({required this.score});

  final int? score;

  @override
  Widget build(BuildContext context) {
    if (score == null) return const ScoreBadge(score: null);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.end,
      children: [
        Text(
          '$score',
          style: const TextStyle(
            color: Colors.white,
            fontSize: 30,
            fontWeight: FontWeight.w800,
            height: 1,
          ),
        ),
        Text(
          'MATCH',
          style: TextStyle(
            color: Colors.white.withValues(alpha: 0.75),
            fontSize: 9,
            fontWeight: FontWeight.w700,
            letterSpacing: 0.8,
          ),
        ),
      ],
    );
  }
}

class _ReviewButton extends StatelessWidget {
  const _ReviewButton({
    required this.featured,
    required this.onTap,
    required this.label,
  });

  final bool featured;
  final VoidCallback onTap;
  final String label;

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    return Material(
      color: featured
          ? Colors.white
          : (dark ? AppColors.darkSurfaceRaised : AppColors.primaryTint),
      borderRadius: AppRadii.pillShape,
      child: InkWell(
        onTap: onTap,
        borderRadius: AppRadii.pillShape,
        child: Padding(
          padding: const EdgeInsets.symmetric(
              horizontal: AppSpacing.lg, vertical: AppSpacing.sm),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                label,
                style: const TextStyle(
                  color: AppColors.primary,
                  fontWeight: FontWeight.w700,
                  fontSize: 13,
                ),
              ),
              const SizedBox(width: 4),
              Icon(MdiIcons.arrowTopRight, size: 14, color: AppColors.primary),
            ],
          ),
        ),
      ),
    );
  }
}
