import 'package:flutter_test/flutter_test.dart';
import 'package:saka_job/data/models/job.dart';

/// Parsing is where a mismatch between backend and app goes unnoticed: a
/// mistyped key produces a null, the UI renders a dash, and nobody notices for
/// a week. These pin the fields whose absence is meaningful.
void main() {
  test('an unknown axis stays null and is never coerced to zero', () {
    final job = Job.fromJson({
      'id': 'j',
      'title': 't',
      'organization': 'o',
      'status': 'Scored',
      'score': 82,
      'score_axes': {
        'skills': 63,
        'seniority': 100,
        'domain': 100,
        'location': 100,
        // pay omitted: the posting did not state a salary.
      },
    });

    final axes = job.axes!;
    expect(axes.pay, isNull,
        reason: 'a job with no stated salary must not render a red pay bar it '
            'never earned');
    expect(axes.skills, 63);
    expect(axes.hasAny, isTrue);
  });

  test('the weakest axis ignores unknown ones', () {
    final job = Job.fromJson({
      'id': 'j', 'title': 't', 'organization': 'o', 'status': 'Scored',
      'score_axes': {'skills': 40, 'seniority': 90},
    });

    // Unknown axes cannot be "weakest" — they are simply not known, and
    // highlighting one would tell the user to fix something unmeasured.
    expect(job.axes!.weakest!.key, 'skills');
    expect(job.axes!.weakest!.value, 40);
  });

  test('a job scored before axes existed has no breakdown at all', () {
    final job = Job.fromJson({
      'id': 'j', 'title': 't', 'organization': 'o', 'status': 'Scored',
      'score': 70,
    });
    expect(job.axes, isNull,
        reason: 'the detail screen shows an explanation rather than five '
            'invented bars');
  });

  test('JSONB arrays parse whether they arrive as a list or a string', () {
    final asList = Job.fromJson({
      'id': 'j', 'title': 't', 'organization': 'o', 'status': 'New',
      'matched_skills': ['Flutter', 'Dart'],
    });
    final asString = Job.fromJson({
      'id': 'j', 'title': 't', 'organization': 'o', 'status': 'New',
      'matched_skills': '["Flutter","Dart"]',
    });

    expect(asList.matchedSkills, ['Flutter', 'Dart']);
    expect(asString.matchedSkills, ['Flutter', 'Dart']);
  });

  test('malformed JSONB degrades to empty rather than throwing', () {
    final job = Job.fromJson({
      'id': 'j', 'title': 't', 'organization': 'o', 'status': 'New',
      'matched_skills': '{not json',
      'score_axes': 'also not json',
    });
    expect(job.matchedSkills, isEmpty);
    expect(job.axes, isNull);
  });

  test('salary shows nothing rather than an empty range', () {
    final none = Job.fromJson({
      'id': 'j', 'title': 't', 'organization': 'o', 'status': 'New',
    });
    expect(none.salaryLabel, isNull,
        reason: 'most postings state no salary; a blank "KES —" is worse than '
            'no line');

    final range = Job.fromJson({
      'id': 'j', 'title': 't', 'organization': 'o', 'status': 'New',
      'salary_min': 480000, 'salary_max': 620000, 'salary_currency': 'KES',
    });
    expect(range.salaryLabel, contains('480k'));
    expect(range.salaryLabel, contains('620k'));

    final open = Job.fromJson({
      'id': 'j', 'title': 't', 'organization': 'o', 'status': 'New',
      'salary_min': 1500000, 'salary_currency': 'KES',
    });
    expect(open.salaryLabel, contains('1.5M'));
  });

  test('cv_edits parse, and an addition keeps its empty before', () {
    final job = Job.fromJson({
      'id': 'j', 'title': 't', 'organization': 'o', 'status': 'AwaitingApproval',
      'cv_edits': [
        {'section': 'Summary', 'before': '', 'after': 'New summary',
         'reason': 'leads with the match'},
        {'section': 'Trimmed', 'before': 'An old role', 'after': '',
         'reason': 'one page'},
      ],
    });

    expect(job.cvEdits.length, 2);
    expect(job.cvEdits[0].isAddition, isTrue);
    expect(job.cvEdits[1].isDeletion, isTrue);
  });

  test('hasDocuments follows generated_at, not the presence of a url', () {
    final notGenerated = Job.fromJson({
      'id': 'j', 'title': 't', 'organization': 'o', 'status': 'Scored',
      'cv_url': '/jobs/j/cv',
    });
    expect(notGenerated.hasDocuments, isFalse,
        reason: 'a url without documents behind it is a dead link');

    final generated = Job.fromJson({
      'id': 'j', 'title': 't', 'organization': 'o', 'status': 'AwaitingApproval',
      'generated_at': '2026-08-01T10:00:00Z',
    });
    expect(generated.hasDocuments, isTrue);
  });

  test('JobPage tolerates a response with no jobs key', () {
    expect(JobPage.fromJson({'total': 0}).jobs, isEmpty);
    expect(JobPage.fromJson({}).total, 0);
  });
}
