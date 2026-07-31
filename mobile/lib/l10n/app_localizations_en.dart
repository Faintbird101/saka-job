// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class L10nEn extends L10n {
  L10nEn([String locale = 'en']) : super(locale);

  @override
  String get appName => 'Saka Job';

  @override
  String get tagline => 'Your agent applies. You just say yes.';

  @override
  String get signIn => 'Sign in';

  @override
  String get signOut => 'Sign out';

  @override
  String get email => 'Email';

  @override
  String get password => 'Password';

  @override
  String get show => 'Show';

  @override
  String get hide => 'Hide';

  @override
  String get welcomeTitle => 'Welcome to Saka Job';

  @override
  String get welcomeBody =>
      'Sign in and your agent starts reading the boards tonight.';

  @override
  String get createAccount => 'Create your account';

  @override
  String get displayName => 'Your name';

  @override
  String get emailRequired => 'Enter your email';

  @override
  String get emailInvalid => 'That does not look like an email address';

  @override
  String get passwordRequired => 'Enter your password';

  @override
  String passwordTooShort(int count) {
    return 'At least $count characters';
  }

  @override
  String get home => 'Home';

  @override
  String get jobs => 'Jobs';

  @override
  String get pipeline => 'Pipeline';

  @override
  String get profile => 'Profile';

  @override
  String get goodMorning => 'Good morning';

  @override
  String get goodAfternoon => 'Good afternoon';

  @override
  String get goodEvening => 'Good evening';

  @override
  String get needsYourYes => 'Needs your yes';

  @override
  String get latestMatches => 'Latest matches';

  @override
  String get seeAll => 'See all';

  @override
  String get review => 'Review';

  @override
  String get nothingToReview => 'Nothing waiting on you';

  @override
  String get nothingToReviewBody =>
      'Your agent is still reading. New matches land after the next scan.';

  @override
  String get searchHint => 'Search roles, companies...';

  @override
  String get approve => 'Approve';

  @override
  String get approveAndApply => 'Approve & apply';

  @override
  String get pass => 'Pass';

  @override
  String get askForEdits => 'Ask for edits';

  @override
  String jobsLeft(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count left',
      one: '1 left',
      zero: 'All done',
    );
    return '$_temp0';
  }

  @override
  String aboutMinutes(int count) {
    return 'about $count min';
  }

  @override
  String get aiScore => 'AI score';

  @override
  String get theRole => 'The role';

  @override
  String get strongMatch => 'Strong match';

  @override
  String get goodMatch => 'Good match';

  @override
  String get partialMatch => 'Partial match';

  @override
  String get weakMatch => 'Weak match';

  @override
  String aboveThreshold(int threshold) {
    return 'Above your $threshold threshold';
  }

  @override
  String belowThreshold(int threshold) {
    return 'Below your $threshold threshold';
  }

  @override
  String get axisSkills => 'Skills & stack';

  @override
  String get axisSeniority => 'Seniority fit';

  @override
  String get axisDomain => 'Domain';

  @override
  String get axisLocation => 'Location & mode';

  @override
  String get axisPay => 'Pay vs your floor';

  @override
  String get axisUnknown => 'Not stated';

  @override
  String weakestAxis(String axis) {
    return 'Weakest: $axis';
  }

  @override
  String get matchedSkills => 'Matched';

  @override
  String get missingSkills => 'Not evidenced';

  @override
  String get tailoredCv => 'Tailored CV';

  @override
  String get coverLetter => 'Cover letter';

  @override
  String get changes => 'Changes';

  @override
  String get fullCv => 'Full CV';

  @override
  String get noDiffYet =>
      'This CV was generated before edit tracking. Regenerate to see the changes.';

  @override
  String get looksGoodSendIt => 'Looks good — send it';

  @override
  String get viewPosting => 'View posting';

  @override
  String get statusNew => 'New';

  @override
  String get statusScored => 'Scored';

  @override
  String get statusLowMatch => 'Low match';

  @override
  String get statusScoreFailed => 'Scoring failed';

  @override
  String get statusCVGenerated => 'CV ready';

  @override
  String get statusAwaitingApproval => 'Needs your yes';

  @override
  String get statusApproved => 'Approved';

  @override
  String get statusRejected => 'Passed';

  @override
  String get statusApplied => 'Applied';

  @override
  String get statusManualApply => 'Apply by hand';

  @override
  String get statusFollowUpSent => 'Followed up';

  @override
  String get statusClosed => 'Closed';

  @override
  String get statusAcknowledged => 'Acknowledged';

  @override
  String get statusInterviewing => 'Interviewing';

  @override
  String get statusOfferReceived => 'Offer';

  @override
  String get statusEmployerRejected => 'Not selected';

  @override
  String get inYourQueue => 'In your queue';

  @override
  String get waitingOnYou => 'waiting on your yes';

  @override
  String get applied => 'Applied';

  @override
  String get sentNoReply => 'sent, no reply yet';

  @override
  String get interview => 'Interview';

  @override
  String get offer => 'Offer';

  @override
  String liveApplications(int count) {
    return '$count live applications';
  }

  @override
  String get masterCv => 'Master CV';

  @override
  String get replace => 'Replace';

  @override
  String get upload => 'Upload';

  @override
  String get cvStrength => 'CV strength';

  @override
  String get targetKeywords => 'Target keywords';

  @override
  String get agentSettings => 'Agent settings';

  @override
  String get scoreThreshold => 'Score threshold';

  @override
  String get thresholdHelp =>
      'Only surface roles scoring at least this. Higher means fewer, better.';

  @override
  String get widerNet => 'Wider net';

  @override
  String get onlyTheBest => 'Only the best';

  @override
  String get notifications => 'Notifications';

  @override
  String get theme => 'Theme';

  @override
  String get themeSystem => 'System';

  @override
  String get themeLight => 'Light';

  @override
  String get themeDark => 'Dark';

  @override
  String get language => 'Language';

  @override
  String get english => 'English';

  @override
  String get swahili => 'Kiswahili';

  @override
  String get preferredLocations => 'Preferred locations';

  @override
  String get remotePreference => 'Work arrangement';

  @override
  String get salaryFloor => 'Salary floor';

  @override
  String get notifyOnApproval => 'Jobs awaiting your approval';

  @override
  String get notifyOnReply => 'Employer replies';

  @override
  String get notifyOnFollowUp => 'Follow-ups due';

  @override
  String get notifyOnFailure => 'Pipeline problems';

  @override
  String get repliesNeedingYou => 'Replies needing you';

  @override
  String get confirmSuggestion => 'Is this right?';

  @override
  String get yesConfirm => 'Yes, that\'s right';

  @override
  String get noDismiss => 'No, dismiss';

  @override
  String get retry => 'Try again';

  @override
  String get workOffline => 'Work offline';

  @override
  String get couldNotReach => 'We couldn\'t reach the agent';

  @override
  String get couldNotReachBody =>
      'Your queue is saved locally — nothing is lost. We\'ll sync the moment you\'re back online.';

  @override
  String get somethingWentWrong => 'Something went wrong';

  @override
  String get noConnection => 'No connection';

  @override
  String get cancel => 'Cancel';

  @override
  String get save => 'Save';

  @override
  String get saved => 'Saved';

  @override
  String get close => 'Close';

  @override
  String get loading => 'Loading';
}
