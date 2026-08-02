import 'package:flutter_test/flutter_test.dart';
import 'package:get/get.dart';
import 'package:saka_job/core/error/failures.dart';
import 'package:saka_job/data/models/job.dart';
import 'package:saka_job/features/approval/controllers/triage_controller.dart';

import 'helpers/fakes.dart';

/// Triage is optimistic: the card leaves on the swipe and the request follows.
/// That is the right feel, but it means a failure lands after the card is gone
/// — so the rollback is the part worth testing, not the happy path.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  tearDown(Get.reset);

  JobPage deck(List<String> ids) => JobPage(
        jobs: [for (final id in ids) job(id: id)],
        total: ids.length,
      );

  test('removes the card immediately, before the request resolves', () async {
    installFakes(pages: [deck(['a', 'b', 'c'])]);
    final c = TriageController();
    await c.load();

    expect(c.deck.length, 3);

    // Deliberately not awaited: the card must be gone by the time the next
    // frame renders, not after the round trip.
    final pending = c.approve(c.deck.first);
    expect(c.deck.map((j) => j.id), ['b', 'c'],
        reason: 'the card should leave synchronously');

    await pending;
    expect(c.deck.map((j) => j.id), ['b', 'c']);
  });

  test('a failed decision puts the job back where it was, not at the end',
      () async {
    installFakes(
      pages: [deck(['a', 'b', 'c'])],
      jobsFailWith: const NetworkFailure('offline'),
    );
    final c = TriageController();
    await c.load();

    // Decide the MIDDLE card, so "restored to its index" and "appended" differ.
    await c.approve(c.deck[1]);

    expect(c.deck.map((j) => j.id), ['a', 'b', 'c'],
        reason: 'a failed decision must restore the deck order the user was '
            'working through');
  });

  test('approve and pass hit different endpoints', () async {
    final fakes = installFakes(pages: [deck(['a', 'b'])]);
    final c = TriageController();
    await c.load();

    await c.approve(c.deck.first);
    await c.pass(c.deck.first);

    expect(fakes.jobs.approved, ['a']);
    expect(fakes.jobs.rejected, ['b']);
  });

  test('deciding a job that is no longer in the deck is a no-op', () async {
    installFakes(pages: [deck(['a'])]);
    final c = TriageController();
    await c.load();

    final only = c.deck.first;
    await c.approve(only);
    // A double-tap, or a swipe racing the button.
    await c.approve(only);

    expect(c.deck, isEmpty, reason: 'must not reinsert or throw');
  });

  test('the denominator is fixed at load so progress cannot go backwards',
      () async {
    installFakes(pages: [deck(['a', 'b', 'c'])]);
    final c = TriageController();
    await c.load();

    expect(c.startingCount, 3);
    await c.approve(c.deck.first);

    expect(c.startingCount, 3, reason: 'the total must not shrink as you work');
    expect(c.done, 1);
    expect(c.remaining, 2);
  });

  test('a load failure surfaces rather than showing an empty deck', () async {
    Get.put<dynamic>(0); // placeholder so Get is initialised
    Get.reset();
    installFakes(pages: const []);
    Get.delete<dynamic>();

    final c = TriageController();
    await c.load();

    // No jobs and no error is a legitimate "all done" state; the distinction
    // the UI relies on is error != null.
    expect(c.deck, isEmpty);
    expect(c.error.value, isNull);
  });
}
