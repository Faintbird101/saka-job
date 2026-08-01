import 'package:flutter/material.dart';
import 'package:material_design_icons_flutter/material_design_icons_flutter.dart';

import '../../core/theme/app_colors.dart';
import '../../core/theme/app_theme.dart';
import '../../data/models/cv_edit.dart';
import '../../l10n/app_localizations.dart';

/// The Changes tab: every edit, with its reason.
///
/// The deck's second argument — no CV edit is silent. Struck-through before,
/// highlighted after, and a pill saying why. Everything rendered here has been
/// verified server-side against the actual documents, so each quotation can be
/// found in the CV the candidate is about to send.
class CvDiff extends StatelessWidget {
  const CvDiff({super.key, required this.edits});

  final List<CvEdit> edits;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);

    if (edits.isEmpty) {
      // Says nothing rather than something reassuring. An older job genuinely
      // has no recorded edits, and claiming "no changes were made" would be a
      // different — and false — statement.
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(AppSpacing.section),
          child: Text(
            l.noDiffYet,
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodyMedium,
          ),
        ),
      );
    }

    return ListView.separated(
      padding: const EdgeInsets.fromLTRB(
          AppSpacing.xl, 0, AppSpacing.xl, AppSpacing.xl),
      itemCount: edits.length,
      separatorBuilder: (_, _) => const SizedBox(height: AppSpacing.md),
      itemBuilder: (context, i) => _EditCard(edit: edits[i]),
    );
  }
}

class _EditCard extends StatelessWidget {
  const _EditCard({required this.edit});

  final CvEdit edit;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final dark = Theme.of(context).brightness == Brightness.dark;

    return Container(
      decoration: BoxDecoration(
        color: dark ? AppColors.darkSurface : AppColors.surface,
        borderRadius: AppRadii.cardShape,
        boxShadow: dark ? AppShadows.cardDark : AppShadows.card,
      ),
      clipBehavior: Clip.antiAlias,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Section on the left, reason as a pill on the right — the deck's
          // layout, and the reason is the part that earns trust.
          Container(
            padding: const EdgeInsets.symmetric(
                horizontal: AppSpacing.lg, vertical: AppSpacing.md),
            color: dark ? AppColors.darkBg : AppColors.bg,
            child: Row(
              children: [
                Expanded(
                  child: Text(
                    edit.section.toUpperCase(),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: text.labelSmall,
                  ),
                ),
                if (edit.reason.isNotEmpty)
                  Flexible(
                    child: Container(
                      padding: const EdgeInsets.symmetric(
                          horizontal: AppSpacing.md, vertical: 4),
                      decoration: BoxDecoration(
                        color: AppColors.primaryTint,
                        borderRadius: AppRadii.pillShape,
                      ),
                      child: Text(
                        edit.reason,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(
                          fontSize: 11,
                          fontWeight: FontWeight.w700,
                          color: AppColors.primary,
                        ),
                      ),
                    ),
                  ),
              ],
            ),
          ),

          Padding(
            padding: const EdgeInsets.all(AppSpacing.lg),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                if (!edit.isAddition)
                  _Side(
                    body: edit.before,
                    removed: true,
                    // Only label it when there is no counterpart, so an
                    // ordinary rewrite stays uncluttered.
                    caption: edit.isDeletion ? 'Removed' : null,
                  ),
                if (!edit.isAddition && !edit.isDeletion)
                  const SizedBox(height: AppSpacing.sm),
                if (!edit.isDeletion)
                  _Side(
                    body: edit.after,
                    removed: false,
                    caption: edit.isAddition ? 'Added' : null,
                  ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _Side extends StatelessWidget {
  const _Side({required this.body, required this.removed, this.caption});

  final String body;
  final bool removed;
  final String? caption;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final dark = Theme.of(context).brightness == Brightness.dark;
    final tint = removed ? AppColors.danger : AppColors.success;

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(AppSpacing.md),
      decoration: BoxDecoration(
        color: tint.withValues(alpha: dark ? 0.12 : 0.08),
        borderRadius: AppRadii.innerShape,
        border: Border(left: BorderSide(color: tint, width: 3)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (caption != null) ...[
            Row(
              children: [
                Icon(removed ? MdiIcons.minusCircleOutline : MdiIcons.plusCircleOutline,
                    size: 13, color: tint),
                const SizedBox(width: 4),
                Text(
                  caption!,
                  style: TextStyle(
                    fontSize: 10.5,
                    fontWeight: FontWeight.w700,
                    color: tint,
                    letterSpacing: 0.4,
                  ),
                ),
              ],
            ),
            const SizedBox(height: AppSpacing.sm),
          ],
          SelectableText(
            body,
            style: text.bodyMedium?.copyWith(
              color: dark ? AppColors.darkInk : AppColors.ink,
              // Struck through for the "before", per the design — the visual
              // that makes a diff readable at a glance.
              decoration: removed ? TextDecoration.lineThrough : null,
              decorationColor: AppColors.danger,
              decorationThickness: 1.5,
              height: 1.45,
            ),
          ),
        ],
      ),
    );
  }
}
