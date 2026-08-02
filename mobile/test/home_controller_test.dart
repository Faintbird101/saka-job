import 'package:flutter_test/flutter_test.dart';
import 'package:get/get.dart';
import 'package:saka_job/core/error/failures.dart';
import 'package:saka_job/data/models/job.dart';
import 'package:saka_job/features/home/controllers/home_controller.dart';

import 'helpers/fakes.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  tearDown(Get.reset);

  JobPage page(List<String> ids) => JobPage(
        jobs: [for (final id in ids) job(id: id)],
        total: ids.length,
      );

  test('a job in the queue is not repeated under latest matches', () async {
    // awaitingApproval() is consumed first, then list().
    installFakes(pages: [
      page(['a', 'b']),
      page(['a', 'b', 'c', 'd']),
    ]);

    final c = HomeController();
    await c.load();

    expect(c.queue.map((j) => j.id), ['a', 'b']);
    expect(c.recent.map((j) => j.id), ['c', 'd'],
        reason: 'repeating the queue cards below makes the feed look like it '
            'is stuttering');
  });

  test('a failure with nothing cached surfaces; with data it does not',
      () async {
    installFakes(pages: const []);
    final c = HomeController();

    // Nothing loaded yet: the screen has nothing to show, so the failure must
    // take over.
    Get.reset();
    installFakes(pages: const [], listFailWith: const NetworkFailure('down'));
    final empty = HomeController();
    await empty.load();
    expect(empty.error.value, isNotNull);

    // With data on screen, a failed refresh must not blank the feed — that
    // loses more than it explains.
    Get.reset();
    installFakes(pages: [page(['a']), page(['a', 'b'])]);
    await c.load();
    expect(c.queue, isNotEmpty);

    Get.reset();
    installFakes(pages: const [], listFailWith: const NetworkFailure('down'));
    await c.refresh();
    expect(c.error.value, isNull,
        reason: 'a refresh failure with cached data should toast, not take '
            'over the screen');
    expect(c.queue, isNotEmpty, reason: 'cached data must survive');
  });

  test('replies failing does not cost the feed', () async {
    installFakes(
      pages: [page(['a']), page(['a', 'b'])],
      eventsFailWith: const NetworkFailure('events endpoint missing'),
    );

    final c = HomeController();
    await c.load();

    // An older backend without /events/pending should still render jobs.
    expect(c.queue, isNotEmpty);
    expect(c.replies, isEmpty);
    expect(c.error.value, isNull);
  });

  test('removeLocally drops the card from both lists', () async {
    installFakes(pages: [page(['a']), page(['a', 'b'])]);
    final c = HomeController();
    await c.load();

    c.removeLocally('b');
    expect(c.recent.map((j) => j.id), isNot(contains('b')));

    c.removeLocally('a');
    expect(c.queue, isEmpty);
  });
}
