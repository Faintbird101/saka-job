import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import 'package:get/get.dart';
import 'package:material_design_icons_flutter/material_design_icons_flutter.dart';

import '../../../core/theme/app_colors.dart';
import '../../../core/theme/app_theme.dart';
import '../../../l10n/app_localizations.dart';
import '../../../shared/widgets/failure_view.dart';
import '../../../shared/widgets/skeletons.dart';
import '../controllers/documents_controller.dart';

/// Screen 07 — the tailored CV and cover letter.
class DocumentsScreen extends GetView<DocumentsController> {
  const DocumentsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final text = Theme.of(context).textTheme;

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: Icon(MdiIcons.chevronLeft),
          onPressed: Get.back<void>,
        ),
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(l.tailoredCv, style: text.titleLarge),
            Text(
              controller.jobTitle,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: text.bodySmall,
            ),
          ],
        ),
      ),
      body: SafeArea(
        child: Obx(() {
          if (controller.loading.value) {
            return const Padding(
              padding: EdgeInsets.all(AppSpacing.xl),
              child: JobCardSkeleton(),
            );
          }

          final failure = controller.error.value;
          if (failure != null) {
            return FailureView(failure: failure, onRetry: controller.load);
          }

          return Column(
            children: [
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: AppSpacing.xl),
                child: SegmentedButton<int>(
                  segments: [
                    ButtonSegment(value: 0, label: Text(l.fullCv)),
                    ButtonSegment(value: 1, label: Text(l.coverLetter)),
                  ],
                  selected: {controller.tab.value},
                  showSelectedIcon: false,
                  onSelectionChanged: (s) => controller.tab.value = s.first,
                ),
              ),
              const SizedBox(height: AppSpacing.md),

              Expanded(
                child: _Document(
                  markdown: controller.tab.value == 0
                      ? controller.cv.value
                      : controller.coverLetter.value,
                ),
              ),
            ],
          );
        }),
      ),
    );
  }
}

class _Document extends StatelessWidget {
  const _Document({required this.markdown});

  final String markdown;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final dark = theme.brightness == Brightness.dark;

    if (markdown.trim().isEmpty) {
      return Center(
        child: Text(
          L10n.of(context).noDiffYet,
          textAlign: TextAlign.center,
          style: theme.textTheme.bodyMedium,
        ),
      );
    }

    return Container(
      margin: const EdgeInsets.fromLTRB(
          AppSpacing.xl, 0, AppSpacing.xl, AppSpacing.xl),
      padding: const EdgeInsets.all(AppSpacing.xl),
      decoration: BoxDecoration(
        color: dark ? AppColors.darkSurface : AppColors.surface,
        borderRadius: AppRadii.cardShape,
        boxShadow: dark ? AppShadows.cardDark : AppShadows.card,
      ),
      child: Markdown(
        data: markdown,
        padding: EdgeInsets.zero,
        // Selectable because the realistic next step is pasting this into an
        // application form — the pipeline cannot submit it for you.
        selectable: true,
        styleSheet: MarkdownStyleSheet(
          h1: theme.textTheme.headlineSmall,
          h2: theme.textTheme.titleLarge,
          h3: theme.textTheme.titleMedium,
          p: theme.textTheme.bodyLarge?.copyWith(height: 1.5),
          listBullet: theme.textTheme.bodyLarge,
          strong: theme.textTheme.bodyLarge?.copyWith(fontWeight: FontWeight.w800),
          blockquoteDecoration: BoxDecoration(
            color: dark ? AppColors.darkBg : AppColors.bg,
            borderRadius: AppRadii.innerShape,
          ),
          horizontalRuleDecoration: BoxDecoration(
            border: Border(
              top: BorderSide(
                color: dark ? AppColors.darkSurfaceRaised : const Color(0xFFE2ECF3),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
