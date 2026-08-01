import 'package:flutter/material.dart';
import 'package:shimmer/shimmer.dart';

import '../../core/theme/app_colors.dart';
import '../../core/theme/app_theme.dart';

/// Loading placeholders.
///
/// Shimmering shapes rather than a spinner, per the brief — a block the size of
/// the card that is coming reads as "nearly there", where a spinner reads as
/// "something is happening, possibly forever". The shapes deliberately match
/// the real layout so nothing jumps when the data lands.
class Skeleton extends StatelessWidget {
  const Skeleton({
    super.key,
    required this.width,
    required this.height,
    this.radius = 8,
  });

  final double width;
  final double height;
  final double radius;

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    return Shimmer.fromColors(
      baseColor: dark ? AppColors.darkSurfaceRaised : const Color(0xFFE4EDF4),
      highlightColor: dark ? const Color(0xFF24405A) : const Color(0xFFF3F8FC),
      period: const Duration(milliseconds: 1400),
      child: Container(
        width: width,
        height: height,
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.circular(radius),
        ),
      ),
    );
  }
}

/// A placeholder shaped like a job card.
class JobCardSkeleton extends StatelessWidget {
  const JobCardSkeleton({super.key});

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
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
              const Skeleton(width: 40, height: 40, radius: 12),
              const SizedBox(width: AppSpacing.md),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: const [
                    Skeleton(width: 120, height: 12),
                    SizedBox(height: 6),
                    Skeleton(width: 80, height: 10),
                  ],
                ),
              ),
              const Skeleton(width: 42, height: 42, radius: 99),
            ],
          ),
          const SizedBox(height: AppSpacing.lg),
          const Skeleton(width: double.infinity, height: 14),
          const SizedBox(height: AppSpacing.sm),
          const Skeleton(width: 160, height: 12),
        ],
      ),
    );
  }
}

/// A column of card skeletons, for a list that has not loaded yet.
class JobListSkeleton extends StatelessWidget {
  const JobListSkeleton({super.key, this.count = 3});

  final int count;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: List.generate(
        count,
        (i) => Padding(
          padding: EdgeInsets.only(bottom: i == count - 1 ? 0 : AppSpacing.md),
          child: const JobCardSkeleton(),
        ),
      ),
    );
  }
}
