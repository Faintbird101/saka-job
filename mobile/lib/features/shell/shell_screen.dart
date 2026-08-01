import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:material_design_icons_flutter/material_design_icons_flutter.dart';

import '../../core/theme/app_colors.dart';
import '../../l10n/app_localizations.dart';
import '../home/views/home_screen.dart';

/// The tabbed shell.
///
/// IndexedStack rather than swapping the body: each tab keeps its scroll
/// position and its loaded data, so moving between them does not re-fetch or
/// jump back to the top — which is the difference between an app that feels
/// instant and one that feels like a website.
class ShellController extends GetxController {
  final index = 0.obs;
}

class ShellScreen extends StatelessWidget {
  const ShellScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final controller = Get.put(ShellController());

    return Scaffold(
      body: Obx(
        () => IndexedStack(
          index: controller.index.value,
          children: const [
            HomeScreen(),
            _Placeholder(label: 'Jobs'),
            _Placeholder(label: 'Pipeline'),
            _Placeholder(label: 'Profile'),
          ],
        ),
      ),
      bottomNavigationBar: Obx(
        () => NavigationBar(
          selectedIndex: controller.index.value,
          onDestinationSelected: (i) => controller.index.value = i,
          destinations: [
            NavigationDestination(
              icon: Icon(MdiIcons.homeOutline),
              selectedIcon: Icon(MdiIcons.home, color: AppColors.primary),
              label: l.home,
            ),
            NavigationDestination(
              icon: Icon(MdiIcons.viewListOutline),
              selectedIcon: Icon(MdiIcons.viewList, color: AppColors.primary),
              label: l.jobs,
            ),
            NavigationDestination(
              icon: Icon(MdiIcons.chartTimelineVariant),
              selectedIcon:
                  Icon(MdiIcons.chartTimelineVariantShimmer, color: AppColors.primary),
              label: l.pipeline,
            ),
            NavigationDestination(
              icon: Icon(MdiIcons.accountCircleOutline),
              selectedIcon: Icon(MdiIcons.accountCircle, color: AppColors.primary),
              label: l.profile,
            ),
          ],
        ),
      ),
    );
  }
}

/// Stands in for the tabs not built yet. Deliberately says so rather than
/// showing an empty screen that looks broken.
class _Placeholder extends StatelessWidget {
  const _Placeholder({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Text(
        '$label — coming next',
        style: Theme.of(context).textTheme.bodyMedium,
      ),
    );
  }
}
