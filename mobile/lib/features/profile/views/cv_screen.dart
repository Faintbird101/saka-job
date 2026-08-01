import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:material_design_icons_flutter/material_design_icons_flutter.dart';

import '../../../core/theme/app_colors.dart';
import '../../../core/theme/app_theme.dart';
import '../../../l10n/app_localizations.dart';
import '../../../shared/widgets/app_toast.dart';
import '../controllers/cv_controller.dart';

/// The master CV editor — upload a file, or edit the text directly.
class CvScreen extends GetView<CvController> {
  const CvScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final text = Theme.of(context).textTheme;
    final dark = Theme.of(context).brightness == Brightness.dark;

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: Icon(MdiIcons.chevronLeft),
          onPressed: Get.back<void>,
        ),
        title: Text(l.masterCv),
      ),
      body: SafeArea(
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(
                  AppSpacing.xl, 0, AppSpacing.xl, AppSpacing.md),
              child: Obx(
                () => OutlinedButton.icon(
                  onPressed:
                      controller.extracting.value ? null : controller.pickFile,
                  icon: controller.extracting.value
                      ? const SizedBox(
                          height: 16,
                          width: 16,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : Icon(MdiIcons.fileUploadOutline, size: 18),
                  label: Text(
                    controller.sourceFile.value.isEmpty
                        ? l.upload
                        : controller.sourceFile.value,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              ),
            ),

            // The extracted text is shown for review rather than saved straight
            // away: PDF extraction mangles designed CVs often enough that
            // silently overwriting the profile would break scoring invisibly.
            Expanded(
              child: Padding(
                padding: const EdgeInsets.symmetric(horizontal: AppSpacing.xl),
                child: Container(
                  decoration: BoxDecoration(
                    color: dark ? AppColors.darkSurface : AppColors.surface,
                    borderRadius: AppRadii.cardShape,
                    boxShadow: dark ? AppShadows.cardDark : AppShadows.card,
                  ),
                  padding: const EdgeInsets.all(AppSpacing.lg),
                  child: TextField(
                    controller: controller.text,
                    maxLines: null,
                    expands: true,
                    textAlignVertical: TextAlignVertical.top,
                    keyboardType: TextInputType.multiline,
                    style: text.bodyMedium?.copyWith(
                      color: dark ? AppColors.darkInk : AppColors.ink,
                      height: 1.5,
                    ),
                    decoration: const InputDecoration(
                      border: InputBorder.none,
                      enabledBorder: InputBorder.none,
                      focusedBorder: InputBorder.none,
                      filled: false,
                      contentPadding: EdgeInsets.zero,
                    ),
                  ),
                ),
              ),
            ),

            Padding(
              padding: const EdgeInsets.all(AppSpacing.xl),
              child: Obx(
                () => Column(
                  children: [
                    Row(
                      children: [
                        Icon(MdiIcons.informationOutline,
                            size: 13, color: AppColors.muted),
                        const SizedBox(width: 5),
                        Expanded(
                          child: Text(
                            // The scorer matches literal strings, so naming the
                            // technologies matters more than prose quality.
                            '${controller.charCount} characters',
                            style: text.labelSmall,
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: AppSpacing.md),
                    FilledButton(
                      onPressed:
                          (controller.busy.value || !controller.dirty.value)
                              ? null
                              : () async {
                                  if (await controller.save()) {
                                    AppToast.success(l.saved);
                                    Get.back<void>();
                                  }
                                },
                      child: controller.busy.value
                          ? const SizedBox(
                              height: 20,
                              width: 20,
                              child: CircularProgressIndicator(
                                  strokeWidth: 2.2, color: Colors.white),
                            )
                          : Text(l.save),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
