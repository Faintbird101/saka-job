import 'package:flutter/material.dart';

import '../../core/theme/app_colors.dart';
import '../../core/theme/app_theme.dart';
import '../../data/models/job.dart';
import '../../l10n/app_localizations.dart';

/// The five-axis breakdown.
///
/// The deck's argument is that nobody hands over their job search to a black
/// box, so the score is never just a number. All five rows are always shown —
/// including the ones the posting did not state, which render as "Not stated"
/// rather than an empty or zero bar. A missing axis is information; a zero bar
/// would be a lie about it.
class AxisBars extends StatelessWidget {
  const AxisBars({
    super.key,
    required this.axes,
    this.onGradient = false,
  });

  final ScoreAxes axes;

  /// True when drawn on the primary gradient card, where the bars invert to
  /// white so they stay legible.
  final bool onGradient;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final labels = <String, String>{
      'skills': l.axisSkills,
      'seniority': l.axisSeniority,
      'domain': l.axisDomain,
      'location': l.axisLocation,
      'pay': l.axisPay,
    };
    final weakest = axes.weakest;

    return Column(
      children: [
        for (final a in axes.all)
          Padding(
            padding: const EdgeInsets.only(bottom: AppSpacing.md),
            child: _AxisRow(
              label: labels[a.key] ?? a.key,
              value: a.value,
              unknownLabel: l.axisUnknown,
              onGradient: onGradient,
              // The weak one is called out, per the deck — that is the row
              // that tells you what to fix.
              highlight: weakest != null && weakest.key == a.key && axes.all.length > 1,
            ),
          ),
      ],
    );
  }
}

class _AxisRow extends StatelessWidget {
  const _AxisRow({
    required this.label,
    required this.value,
    required this.unknownLabel,
    required this.onGradient,
    required this.highlight,
  });

  final String label;
  final int? value;
  final String unknownLabel;
  final bool onGradient;
  final bool highlight;

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final known = value != null;

    final labelColour = onGradient
        ? Colors.white.withValues(alpha: known ? 0.95 : 0.6)
        : (known
            ? (dark ? AppColors.darkInk : AppColors.ink)
            : (dark ? AppColors.darkMuted : AppColors.muted));

    final track = onGradient
        ? Colors.white.withValues(alpha: 0.22)
        : (dark ? AppColors.darkSurfaceRaised : const Color(0xFFE3EDF4));

    final fill = onGradient
        ? Colors.white
        : AppColors.forScore(value ?? 0);

    return Row(
      children: [
        SizedBox(
          width: 116,
          child: Row(
            children: [
              Flexible(
                child: Text(
                  label,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    fontSize: 12.5,
                    fontWeight: highlight ? FontWeight.w800 : FontWeight.w600,
                    color: labelColour,
                  ),
                ),
              ),
              if (highlight) ...[
                const SizedBox(width: 3),
                Icon(
                  Icons.priority_high_rounded,
                  size: 12,
                  color: onGradient ? Colors.white : AppColors.warn,
                ),
              ],
            ],
          ),
        ),
        const SizedBox(width: AppSpacing.sm),

        Expanded(
          child: ClipRRect(
            borderRadius: BorderRadius.circular(99),
            child: known
                ? TweenAnimationBuilder<double>(
                    // Animating from zero makes the bars read as being
                    // measured rather than simply drawn.
                    tween: Tween(begin: 0, end: value! / 100),
                    duration: const Duration(milliseconds: 620),
                    curve: Curves.easeOutCubic,
                    builder: (context, v, _) => LinearProgressIndicator(
                      value: v,
                      minHeight: 6,
                      backgroundColor: track,
                      valueColor: AlwaysStoppedAnimation(fill),
                    ),
                  )
                // A dashed-looking flat track: visibly not a score, so it can
                // never be misread as a very low one.
                : Container(height: 6, color: track),
          ),
        ),
        const SizedBox(width: AppSpacing.md),

        SizedBox(
          width: 62,
          child: Text(
            known ? '${value!}%' : unknownLabel,
            textAlign: TextAlign.right,
            style: TextStyle(
              fontSize: known ? 12.5 : 10.5,
              fontWeight: FontWeight.w700,
              color: labelColour,
            ),
          ),
        ),
      ],
    );
  }
}
