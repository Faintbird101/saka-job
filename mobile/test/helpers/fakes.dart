import 'package:get/get.dart';
import 'package:saka_job/core/error/failures.dart';
import 'package:saka_job/data/models/job.dart';
import 'package:saka_job/data/models/job_event.dart';
import 'package:saka_job/data/services/events_service.dart';
import 'package:saka_job/data/services/jobs_service.dart';

/// Stand-ins for the services, so controller logic can be tested without a
/// network, a database, or a running backend.
///
/// Hand-written rather than generated: the surface is small, and a fake that
/// records what it was asked is more useful here than one that only returns
/// canned values — several of these tests are about *what* got called.

Job job({
  String id = 'j1',
  String title = 'Flutter Engineer',
  String org = 'Acme',
  String status = 'AwaitingApproval',
  int? score = 80,
}) =>
    Job(
      id: id,
      title: title,
      organization: org,
      status: status,
      score: score,
    );

class FakeJobsService extends JobsService {
  FakeJobsService({this.pages = const [], this.failWith, this.listFailWith});

  /// Successive responses for list()/awaitingApproval(), consumed in order.
  final List<JobPage> pages;

  /// When set, every MUTATING call throws it — for testing rollback.
  final Failure? failWith;

  /// When set, every READ throws it. Separate from [failWith] on purpose: a
  /// rollback test needs the deck to load and then the decision to fail, which
  /// a single flag could not express.
  final Failure? listFailWith;

  int listCalls = 0;
  final approved = <String>[];
  final rejected = <String>[];

  JobPage _next() {
    if (listFailWith != null) throw listFailWith!;
    if (pages.isEmpty) return JobPage.empty;
    final p = pages[listCalls.clamp(0, pages.length - 1)];
    listCalls++;
    return p;
  }

  @override
  Future<JobPage> list({
    String? status,
    int? minScore,
    String? search,
    int limit = 20,
    int offset = 0,
  }) async =>
      _next();

  @override
  Future<JobPage> awaitingApproval({int limit = 20, int offset = 0}) async =>
      _next();

  @override
  Future<Job> approve(String id) async {
    if (failWith != null) throw failWith!;
    approved.add(id);
    return job(id: id, status: 'Approved');
  }

  @override
  Future<Job> reject(String id) async {
    if (failWith != null) throw failWith!;
    rejected.add(id);
    return job(id: id, status: 'Rejected');
  }
}

class FakeEventsService extends EventsService {
  FakeEventsService({this.events = const [], this.failWith});

  final List<JobEvent> events;
  final Failure? failWith;

  final confirmed = <String, bool>{};

  @override
  Future<List<JobEvent>> pending({int limit = 50}) async {
    if (failWith != null) throw failWith!;
    return events;
  }

  @override
  Future<JobEvent> confirm(String eventId, {required bool accept}) async {
    if (failWith != null) throw failWith!;
    confirmed[eventId] = accept;
    return events.firstWhere((e) => e.id == eventId);
  }
}

/// Registers fakes and returns them. Call [Get.reset] between tests.
({FakeJobsService jobs, FakeEventsService events}) installFakes({
  List<JobPage> pages = const [],
  Failure? jobsFailWith,
  Failure? listFailWith,
  List<JobEvent> events = const [],
  Failure? eventsFailWith,
}) {
  final j = FakeJobsService(
    pages: pages,
    failWith: jobsFailWith,
    listFailWith: listFailWith,
  );
  final e = FakeEventsService(events: events, failWith: eventsFailWith);
  Get.put<JobsService>(j);
  Get.put<EventsService>(e);
  return (jobs: j, events: e);
}
