import 'package:flutter/material.dart';
import 'package:material_design_icons_flutter/material_design_icons_flutter.dart';

import '../../core/error/failures.dart';
import '../../core/theme/app_colors.dart';
import '../../core/theme/app_theme.dart';
import '../../l10n/app_localizations.dart';

/// Screen 14 — the failure state.
///
/// Takes over only when there is nothing cached to show. The design's wording
/// matters here: it says the queue is saved and nothing is lost, because the
/// most common failure is the laptop being asleep, and that is temporary rather
/// than data loss.
class FailureView extends StatelessWidget {
  const FailureView({
    super.key,
    required this.failure,
    this.onRetry,
    this.onWorkOffline,
  });

  final Failure failure;
  final VoidCallback? onRetry;
  final VoidCallback? onWorkOffline;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final text = Theme.of(context).textTheme;
    final dark = Theme.of(context).brightness == Brightness.dark;

    final isNetwork = failure is NetworkFailure;
    final title = isNetwork ? l.couldNotReach : l.somethingWentWrong;
    final body = isNetwork ? l.couldNotReachBody : failure.message;

    return Center(
      child: SingleChildScrollView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.all(AppSpacing.section),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 64,
              height: 64,
              alignment: Alignment.center,
              decoration: BoxDecoration(
                color: AppColors.warn.withValues(alpha: dark ? 0.2 : 0.12),
                borderRadius: AppRadii.pillShape,
              ),
              child: Icon(
                isNetwork ? MdiIcons.flashOutline : MdiIcons.alertOutline,
                color: AppColors.warn,
                size: 28,
              ),
            ),
            const SizedBox(height: AppSpacing.xl),

            Text(title, textAlign: TextAlign.center, style: text.headlineSmall),
            const SizedBox(height: AppSpacing.sm),
            Text(body, textAlign: TextAlign.center, style: text.bodyMedium),

            if (failure.technical != null) ...[
              const SizedBox(height: AppSpacing.lg),
              // Monospace and quiet: this line exists for reporting a problem,
              // and is meant to be ignorable the rest of the time.
              Container(
                padding: const EdgeInsets.symmetric(
                    horizontal: AppSpacing.md, vertical: AppSpacing.sm),
                decoration: BoxDecoration(
                  color: dark ? AppColors.darkSurface : const Color(0xFFE9F0F6),
                  borderRadius: AppRadii.innerShape,
                ),
                child: Text(
                  '${failure.runtimeType} · ${failure.technical}',
                  textAlign: TextAlign.center,
                  style: text.bodySmall?.copyWith(
                    fontFamily: 'monospace',
                    fontSize: 11.5,
                  ),
                ),
              ),
            ],

            const SizedBox(height: AppSpacing.xxl),
            if (onRetry != null)
              SizedBox(
                width: 220,
                child: FilledButton(onPressed: onRetry, child: Text(l.retry)),
              ),
            if (onWorkOffline != null) ...[
              const SizedBox(height: AppSpacing.md),
              SizedBox(
                width: 220,
                child: OutlinedButton(
                  onPressed: onWorkOffline,
                  child: Text(l.workOffline),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
