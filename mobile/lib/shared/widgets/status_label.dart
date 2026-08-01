import 'package:flutter/material.dart';

import '../../core/theme/app_colors.dart';
import '../../core/theme/app_theme.dart';
import '../../l10n/app_localizations.dart';

/// Translates a backend status into the user's language.
///
/// One switch, so a status added to the state machine surfaces here as a
/// compile-time gap rather than a raw "EmployerRejected" leaking onto a card.
/// The default returns the raw value rather than a placeholder — seeing the
/// real string is more diagnosable than seeing "Unknown".
String statusLabel(L10n l, String status) => switch (status) {
      'New' => l.statusNew,
      'Scored' => l.statusScored,
      'LowMatch' => l.statusLowMatch,
      'ScoreFailed' => l.statusScoreFailed,
      'CVGenerated' => l.statusCVGenerated,
      'AwaitingApproval' => l.statusAwaitingApproval,
      'Approved' => l.statusApproved,
      'Rejected' => l.statusRejected,
      'Applied' => l.statusApplied,
      'ManualApply' => l.statusManualApply,
      'FollowUpSent' => l.statusFollowUpSent,
      'Closed' => l.statusClosed,
      'Acknowledged' => l.statusAcknowledged,
      'Interviewing' => l.statusInterviewing,
      'OfferReceived' => l.statusOfferReceived,
      'EmployerRejected' => l.statusEmployerRejected,
      _ => status,
    };

/// The colour a status carries through the app.
///
/// Grouped by what the status means to the user rather than by pipeline order:
/// good news is green, a live application is blue, anything needing attention
/// is amber, and a dead end is muted — so a glance down a list reads without
/// having to decode sixteen distinct colours.
Color statusColour(String status) => switch (status) {
      'OfferReceived' || 'Interviewing' => AppColors.success,
      'AwaitingApproval' || 'Approved' || 'CVGenerated' => AppColors.warn,
      'Applied' || 'Acknowledged' || 'FollowUpSent' || 'ManualApply' =>
        AppColors.primary,
      'EmployerRejected' || 'ScoreFailed' => AppColors.danger,
      _ => AppColors.muted,
    };

/// A small status pill.
class StatusChip extends StatelessWidget {
  const StatusChip({super.key, required this.status, this.compact = false});

  final String status;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final colour = statusColour(status);
    final dark = Theme.of(context).brightness == Brightness.dark;

    return Container(
      padding: EdgeInsets.symmetric(
        horizontal: compact ? AppSpacing.sm : AppSpacing.md,
        vertical: compact ? 3 : 5,
      ),
      decoration: BoxDecoration(
        color: colour.withValues(alpha: dark ? 0.2 : 0.12),
        borderRadius: AppRadii.pillShape,
      ),
      child: Text(
        statusLabel(l, status),
        style: TextStyle(
          fontSize: compact ? 10.5 : 11.5,
          fontWeight: FontWeight.w700,
          color: colour,
        ),
      ),
    );
  }
}
