import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_en.dart';
import 'app_localizations_sw.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of L10n
/// returned by `L10n.of(context)`.
///
/// Applications need to include `L10n.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'l10n/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: L10n.localizationsDelegates,
///   supportedLocales: L10n.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the L10n.supportedLocales
/// property.
abstract class L10n {
  L10n(String locale)
    : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static L10n of(BuildContext context) {
    return Localizations.of<L10n>(context, L10n)!;
  }

  static const LocalizationsDelegate<L10n> delegate = _L10nDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates =
      <LocalizationsDelegate<dynamic>>[
        delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[
    Locale('en'),
    Locale('sw'),
  ];

  /// No description provided for @appName.
  ///
  /// In en, this message translates to:
  /// **'Saka Job'**
  String get appName;

  /// No description provided for @tagline.
  ///
  /// In en, this message translates to:
  /// **'Your agent applies. You just say yes.'**
  String get tagline;

  /// No description provided for @signIn.
  ///
  /// In en, this message translates to:
  /// **'Sign in'**
  String get signIn;

  /// No description provided for @signOut.
  ///
  /// In en, this message translates to:
  /// **'Sign out'**
  String get signOut;

  /// No description provided for @email.
  ///
  /// In en, this message translates to:
  /// **'Email'**
  String get email;

  /// No description provided for @password.
  ///
  /// In en, this message translates to:
  /// **'Password'**
  String get password;

  /// No description provided for @show.
  ///
  /// In en, this message translates to:
  /// **'Show'**
  String get show;

  /// No description provided for @hide.
  ///
  /// In en, this message translates to:
  /// **'Hide'**
  String get hide;

  /// No description provided for @welcomeTitle.
  ///
  /// In en, this message translates to:
  /// **'Welcome to Saka Job'**
  String get welcomeTitle;

  /// No description provided for @welcomeBody.
  ///
  /// In en, this message translates to:
  /// **'Sign in and your agent starts reading the boards tonight.'**
  String get welcomeBody;

  /// No description provided for @createAccount.
  ///
  /// In en, this message translates to:
  /// **'Create your account'**
  String get createAccount;

  /// No description provided for @displayName.
  ///
  /// In en, this message translates to:
  /// **'Your name'**
  String get displayName;

  /// No description provided for @emailRequired.
  ///
  /// In en, this message translates to:
  /// **'Enter your email'**
  String get emailRequired;

  /// No description provided for @emailInvalid.
  ///
  /// In en, this message translates to:
  /// **'That does not look like an email address'**
  String get emailInvalid;

  /// No description provided for @passwordRequired.
  ///
  /// In en, this message translates to:
  /// **'Enter your password'**
  String get passwordRequired;

  /// No description provided for @passwordTooShort.
  ///
  /// In en, this message translates to:
  /// **'At least {count} characters'**
  String passwordTooShort(int count);

  /// No description provided for @home.
  ///
  /// In en, this message translates to:
  /// **'Home'**
  String get home;

  /// No description provided for @jobs.
  ///
  /// In en, this message translates to:
  /// **'Jobs'**
  String get jobs;

  /// No description provided for @pipeline.
  ///
  /// In en, this message translates to:
  /// **'Pipeline'**
  String get pipeline;

  /// No description provided for @profile.
  ///
  /// In en, this message translates to:
  /// **'Profile'**
  String get profile;

  /// No description provided for @goodMorning.
  ///
  /// In en, this message translates to:
  /// **'Good morning'**
  String get goodMorning;

  /// No description provided for @goodAfternoon.
  ///
  /// In en, this message translates to:
  /// **'Good afternoon'**
  String get goodAfternoon;

  /// No description provided for @goodEvening.
  ///
  /// In en, this message translates to:
  /// **'Good evening'**
  String get goodEvening;

  /// No description provided for @needsYourYes.
  ///
  /// In en, this message translates to:
  /// **'Needs your yes'**
  String get needsYourYes;

  /// No description provided for @latestMatches.
  ///
  /// In en, this message translates to:
  /// **'Latest matches'**
  String get latestMatches;

  /// No description provided for @seeAll.
  ///
  /// In en, this message translates to:
  /// **'See all'**
  String get seeAll;

  /// No description provided for @review.
  ///
  /// In en, this message translates to:
  /// **'Review'**
  String get review;

  /// No description provided for @nothingToReview.
  ///
  /// In en, this message translates to:
  /// **'Nothing waiting on you'**
  String get nothingToReview;

  /// No description provided for @nothingToReviewBody.
  ///
  /// In en, this message translates to:
  /// **'Your agent is still reading. New matches land after the next scan.'**
  String get nothingToReviewBody;

  /// No description provided for @searchHint.
  ///
  /// In en, this message translates to:
  /// **'Search roles, companies...'**
  String get searchHint;

  /// No description provided for @approve.
  ///
  /// In en, this message translates to:
  /// **'Approve'**
  String get approve;

  /// No description provided for @approveAndApply.
  ///
  /// In en, this message translates to:
  /// **'Approve & apply'**
  String get approveAndApply;

  /// No description provided for @pass.
  ///
  /// In en, this message translates to:
  /// **'Pass'**
  String get pass;

  /// No description provided for @askForEdits.
  ///
  /// In en, this message translates to:
  /// **'Ask for edits'**
  String get askForEdits;

  /// No description provided for @jobsLeft.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, =0{All done} =1{1 left} other{{count} left}}'**
  String jobsLeft(int count);

  /// No description provided for @aboutMinutes.
  ///
  /// In en, this message translates to:
  /// **'about {count} min'**
  String aboutMinutes(int count);

  /// No description provided for @aiScore.
  ///
  /// In en, this message translates to:
  /// **'AI score'**
  String get aiScore;

  /// No description provided for @theRole.
  ///
  /// In en, this message translates to:
  /// **'The role'**
  String get theRole;

  /// No description provided for @strongMatch.
  ///
  /// In en, this message translates to:
  /// **'Strong match'**
  String get strongMatch;

  /// No description provided for @goodMatch.
  ///
  /// In en, this message translates to:
  /// **'Good match'**
  String get goodMatch;

  /// No description provided for @partialMatch.
  ///
  /// In en, this message translates to:
  /// **'Partial match'**
  String get partialMatch;

  /// No description provided for @weakMatch.
  ///
  /// In en, this message translates to:
  /// **'Weak match'**
  String get weakMatch;

  /// No description provided for @aboveThreshold.
  ///
  /// In en, this message translates to:
  /// **'Above your {threshold} threshold'**
  String aboveThreshold(int threshold);

  /// No description provided for @belowThreshold.
  ///
  /// In en, this message translates to:
  /// **'Below your {threshold} threshold'**
  String belowThreshold(int threshold);

  /// No description provided for @axisSkills.
  ///
  /// In en, this message translates to:
  /// **'Skills & stack'**
  String get axisSkills;

  /// No description provided for @axisSeniority.
  ///
  /// In en, this message translates to:
  /// **'Seniority fit'**
  String get axisSeniority;

  /// No description provided for @axisDomain.
  ///
  /// In en, this message translates to:
  /// **'Domain'**
  String get axisDomain;

  /// No description provided for @axisLocation.
  ///
  /// In en, this message translates to:
  /// **'Location & mode'**
  String get axisLocation;

  /// No description provided for @axisPay.
  ///
  /// In en, this message translates to:
  /// **'Pay vs your floor'**
  String get axisPay;

  /// No description provided for @axisUnknown.
  ///
  /// In en, this message translates to:
  /// **'Not stated'**
  String get axisUnknown;

  /// No description provided for @weakestAxis.
  ///
  /// In en, this message translates to:
  /// **'Weakest: {axis}'**
  String weakestAxis(String axis);

  /// No description provided for @matchedSkills.
  ///
  /// In en, this message translates to:
  /// **'Matched'**
  String get matchedSkills;

  /// No description provided for @missingSkills.
  ///
  /// In en, this message translates to:
  /// **'Not evidenced'**
  String get missingSkills;

  /// No description provided for @tailoredCv.
  ///
  /// In en, this message translates to:
  /// **'Tailored CV'**
  String get tailoredCv;

  /// No description provided for @coverLetter.
  ///
  /// In en, this message translates to:
  /// **'Cover letter'**
  String get coverLetter;

  /// No description provided for @changes.
  ///
  /// In en, this message translates to:
  /// **'Changes'**
  String get changes;

  /// No description provided for @fullCv.
  ///
  /// In en, this message translates to:
  /// **'Full CV'**
  String get fullCv;

  /// No description provided for @noDiffYet.
  ///
  /// In en, this message translates to:
  /// **'This CV was generated before edit tracking. Regenerate to see the changes.'**
  String get noDiffYet;

  /// No description provided for @looksGoodSendIt.
  ///
  /// In en, this message translates to:
  /// **'Looks good — send it'**
  String get looksGoodSendIt;

  /// No description provided for @viewPosting.
  ///
  /// In en, this message translates to:
  /// **'View posting'**
  String get viewPosting;

  /// No description provided for @statusNew.
  ///
  /// In en, this message translates to:
  /// **'New'**
  String get statusNew;

  /// No description provided for @statusScored.
  ///
  /// In en, this message translates to:
  /// **'Scored'**
  String get statusScored;

  /// No description provided for @statusLowMatch.
  ///
  /// In en, this message translates to:
  /// **'Low match'**
  String get statusLowMatch;

  /// No description provided for @statusScoreFailed.
  ///
  /// In en, this message translates to:
  /// **'Scoring failed'**
  String get statusScoreFailed;

  /// No description provided for @statusCVGenerated.
  ///
  /// In en, this message translates to:
  /// **'CV ready'**
  String get statusCVGenerated;

  /// No description provided for @statusAwaitingApproval.
  ///
  /// In en, this message translates to:
  /// **'Needs your yes'**
  String get statusAwaitingApproval;

  /// No description provided for @statusApproved.
  ///
  /// In en, this message translates to:
  /// **'Approved'**
  String get statusApproved;

  /// No description provided for @statusRejected.
  ///
  /// In en, this message translates to:
  /// **'Passed'**
  String get statusRejected;

  /// No description provided for @statusApplied.
  ///
  /// In en, this message translates to:
  /// **'Applied'**
  String get statusApplied;

  /// No description provided for @statusManualApply.
  ///
  /// In en, this message translates to:
  /// **'Apply by hand'**
  String get statusManualApply;

  /// No description provided for @statusFollowUpSent.
  ///
  /// In en, this message translates to:
  /// **'Followed up'**
  String get statusFollowUpSent;

  /// No description provided for @statusClosed.
  ///
  /// In en, this message translates to:
  /// **'Closed'**
  String get statusClosed;

  /// No description provided for @statusAcknowledged.
  ///
  /// In en, this message translates to:
  /// **'Acknowledged'**
  String get statusAcknowledged;

  /// No description provided for @statusInterviewing.
  ///
  /// In en, this message translates to:
  /// **'Interviewing'**
  String get statusInterviewing;

  /// No description provided for @statusOfferReceived.
  ///
  /// In en, this message translates to:
  /// **'Offer'**
  String get statusOfferReceived;

  /// No description provided for @statusEmployerRejected.
  ///
  /// In en, this message translates to:
  /// **'Not selected'**
  String get statusEmployerRejected;

  /// No description provided for @inYourQueue.
  ///
  /// In en, this message translates to:
  /// **'In your queue'**
  String get inYourQueue;

  /// No description provided for @waitingOnYou.
  ///
  /// In en, this message translates to:
  /// **'waiting on your yes'**
  String get waitingOnYou;

  /// No description provided for @applied.
  ///
  /// In en, this message translates to:
  /// **'Applied'**
  String get applied;

  /// No description provided for @sentNoReply.
  ///
  /// In en, this message translates to:
  /// **'sent, no reply yet'**
  String get sentNoReply;

  /// No description provided for @interview.
  ///
  /// In en, this message translates to:
  /// **'Interview'**
  String get interview;

  /// No description provided for @offer.
  ///
  /// In en, this message translates to:
  /// **'Offer'**
  String get offer;

  /// No description provided for @liveApplications.
  ///
  /// In en, this message translates to:
  /// **'{count} live applications'**
  String liveApplications(int count);

  /// No description provided for @masterCv.
  ///
  /// In en, this message translates to:
  /// **'Master CV'**
  String get masterCv;

  /// No description provided for @replace.
  ///
  /// In en, this message translates to:
  /// **'Replace'**
  String get replace;

  /// No description provided for @upload.
  ///
  /// In en, this message translates to:
  /// **'Upload'**
  String get upload;

  /// No description provided for @cvStrength.
  ///
  /// In en, this message translates to:
  /// **'CV strength'**
  String get cvStrength;

  /// No description provided for @targetKeywords.
  ///
  /// In en, this message translates to:
  /// **'Target keywords'**
  String get targetKeywords;

  /// No description provided for @agentSettings.
  ///
  /// In en, this message translates to:
  /// **'Agent settings'**
  String get agentSettings;

  /// No description provided for @scoreThreshold.
  ///
  /// In en, this message translates to:
  /// **'Score threshold'**
  String get scoreThreshold;

  /// No description provided for @thresholdHelp.
  ///
  /// In en, this message translates to:
  /// **'Only surface roles scoring at least this. Higher means fewer, better.'**
  String get thresholdHelp;

  /// No description provided for @widerNet.
  ///
  /// In en, this message translates to:
  /// **'Wider net'**
  String get widerNet;

  /// No description provided for @onlyTheBest.
  ///
  /// In en, this message translates to:
  /// **'Only the best'**
  String get onlyTheBest;

  /// No description provided for @notifications.
  ///
  /// In en, this message translates to:
  /// **'Notifications'**
  String get notifications;

  /// No description provided for @theme.
  ///
  /// In en, this message translates to:
  /// **'Theme'**
  String get theme;

  /// No description provided for @themeSystem.
  ///
  /// In en, this message translates to:
  /// **'System'**
  String get themeSystem;

  /// No description provided for @themeLight.
  ///
  /// In en, this message translates to:
  /// **'Light'**
  String get themeLight;

  /// No description provided for @themeDark.
  ///
  /// In en, this message translates to:
  /// **'Dark'**
  String get themeDark;

  /// No description provided for @language.
  ///
  /// In en, this message translates to:
  /// **'Language'**
  String get language;

  /// No description provided for @english.
  ///
  /// In en, this message translates to:
  /// **'English'**
  String get english;

  /// No description provided for @swahili.
  ///
  /// In en, this message translates to:
  /// **'Kiswahili'**
  String get swahili;

  /// No description provided for @preferredLocations.
  ///
  /// In en, this message translates to:
  /// **'Preferred locations'**
  String get preferredLocations;

  /// No description provided for @remotePreference.
  ///
  /// In en, this message translates to:
  /// **'Work arrangement'**
  String get remotePreference;

  /// No description provided for @salaryFloor.
  ///
  /// In en, this message translates to:
  /// **'Salary floor'**
  String get salaryFloor;

  /// No description provided for @notifyOnApproval.
  ///
  /// In en, this message translates to:
  /// **'Jobs awaiting your approval'**
  String get notifyOnApproval;

  /// No description provided for @notifyOnReply.
  ///
  /// In en, this message translates to:
  /// **'Employer replies'**
  String get notifyOnReply;

  /// No description provided for @notifyOnFollowUp.
  ///
  /// In en, this message translates to:
  /// **'Follow-ups due'**
  String get notifyOnFollowUp;

  /// No description provided for @notifyOnFailure.
  ///
  /// In en, this message translates to:
  /// **'Pipeline problems'**
  String get notifyOnFailure;

  /// No description provided for @repliesNeedingYou.
  ///
  /// In en, this message translates to:
  /// **'Replies needing you'**
  String get repliesNeedingYou;

  /// No description provided for @confirmSuggestion.
  ///
  /// In en, this message translates to:
  /// **'Is this right?'**
  String get confirmSuggestion;

  /// No description provided for @yesConfirm.
  ///
  /// In en, this message translates to:
  /// **'Yes, that\'s right'**
  String get yesConfirm;

  /// No description provided for @noDismiss.
  ///
  /// In en, this message translates to:
  /// **'No, dismiss'**
  String get noDismiss;

  /// No description provided for @retry.
  ///
  /// In en, this message translates to:
  /// **'Try again'**
  String get retry;

  /// No description provided for @workOffline.
  ///
  /// In en, this message translates to:
  /// **'Work offline'**
  String get workOffline;

  /// No description provided for @couldNotReach.
  ///
  /// In en, this message translates to:
  /// **'We couldn\'t reach the agent'**
  String get couldNotReach;

  /// No description provided for @couldNotReachBody.
  ///
  /// In en, this message translates to:
  /// **'Your queue is saved locally — nothing is lost. We\'ll sync the moment you\'re back online.'**
  String get couldNotReachBody;

  /// No description provided for @somethingWentWrong.
  ///
  /// In en, this message translates to:
  /// **'Something went wrong'**
  String get somethingWentWrong;

  /// No description provided for @noConnection.
  ///
  /// In en, this message translates to:
  /// **'No connection'**
  String get noConnection;

  /// No description provided for @cancel.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get cancel;

  /// No description provided for @save.
  ///
  /// In en, this message translates to:
  /// **'Save'**
  String get save;

  /// No description provided for @saved.
  ///
  /// In en, this message translates to:
  /// **'Saved'**
  String get saved;

  /// No description provided for @close.
  ///
  /// In en, this message translates to:
  /// **'Close'**
  String get close;

  /// No description provided for @loading.
  ///
  /// In en, this message translates to:
  /// **'Loading'**
  String get loading;

  /// No description provided for @onboardCvTitle.
  ///
  /// In en, this message translates to:
  /// **'Drop in your CV'**
  String get onboardCvTitle;

  /// No description provided for @onboardCvBody.
  ///
  /// In en, this message translates to:
  /// **'One PDF or Word file. The agent reads it and builds your profile.'**
  String get onboardCvBody;

  /// No description provided for @onboardBarTitle.
  ///
  /// In en, this message translates to:
  /// **'Set your bar'**
  String get onboardBarTitle;

  /// No description provided for @onboardBarBody.
  ///
  /// In en, this message translates to:
  /// **'Score threshold, pay floor, remote or not — change it any time.'**
  String get onboardBarBody;

  /// No description provided for @onboardYesTitle.
  ///
  /// In en, this message translates to:
  /// **'Say yes each morning'**
  String get onboardYesTitle;

  /// No description provided for @onboardYesBody.
  ///
  /// In en, this message translates to:
  /// **'A few minutes a day. It handles the rest.'**
  String get onboardYesBody;
}

class _L10nDelegate extends LocalizationsDelegate<L10n> {
  const _L10nDelegate();

  @override
  Future<L10n> load(Locale locale) {
    return SynchronousFuture<L10n>(lookupL10n(locale));
  }

  @override
  bool isSupported(Locale locale) =>
      <String>['en', 'sw'].contains(locale.languageCode);

  @override
  bool shouldReload(_L10nDelegate old) => false;
}

L10n lookupL10n(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'en':
      return L10nEn();
    case 'sw':
      return L10nSw();
  }

  throw FlutterError(
    'L10n.delegate failed to load unsupported locale "$locale". This is likely '
    'an issue with the localizations generation tool. Please file an issue '
    'on GitHub with a reproducible sample app and the gen-l10n configuration '
    'that was used.',
  );
}
