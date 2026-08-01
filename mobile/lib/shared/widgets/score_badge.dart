import 'package:flutter/material.dart';

import '../../core/theme/app_colors.dart';
import '../../core/theme/app_theme.dart';

/// The circular score chip from the design.
///
/// Colour comes from [AppColors.forScore], whose thresholds match the wording
/// the backend already puts in its summaries — so a green badge never sits next
/// to the words "weak match".
class ScoreBadge extends StatelessWidget {
  const ScoreBadge({
    super.key,
    required this.score,
    this.size = 44,
    this.showLabel = false,
  });

  /// Null renders a muted dash: an unscored job is a real state (it is queued),
  /// not a zero.
  final int? score;
  final double size;

  /// Adds the tiny "MATCH" caption used on the large variant.
  final bool showLabel;

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final value = score;

    if (value == null) {
      return Container(
        width: size,
        height: size,
        alignment: Alignment.center,
        decoration: BoxDecoration(
          color: dark ? AppColors.darkSurfaceRaised : const Color(0xFFEDF3F8),
          borderRadius: AppRadii.pillShape,
        ),
        child: Text(
          '–',
          style: TextStyle(
            color: dark ? AppColors.darkMuted : AppColors.muted,
            fontWeight: FontWeight.w700,
            fontSize: size * 0.36,
          ),
        ),
      );
    }

    final colour = AppColors.forScore(value);
    return Container(
      width: size,
      height: size,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: AppColors.forScoreTint(value, dark: dark),
        borderRadius: AppRadii.pillShape,
      ),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Text(
            '$value',
            style: TextStyle(
              color: colour,
              fontWeight: FontWeight.w800,
              fontSize: size * 0.38,
              height: 1,
            ),
          ),
          if (showLabel)
            Text(
              'MATCH',
              style: TextStyle(
                color: colour.withValues(alpha: 0.8),
                fontWeight: FontWeight.w700,
                fontSize: size * 0.13,
                letterSpacing: 0.6,
              ),
            ),
        ],
      ),
    );
  }
}
