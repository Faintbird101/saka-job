// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Swahili (`sw`).
class L10nSw extends L10n {
  L10nSw([String locale = 'sw']) : super(locale);

  @override
  String get appName => 'Saka Job';

  @override
  String get tagline => 'Wakala wako anatuma maombi. Wewe unakubali tu.';

  @override
  String get signIn => 'Ingia';

  @override
  String get signOut => 'Toka';

  @override
  String get email => 'Barua pepe';

  @override
  String get password => 'Nenosiri';

  @override
  String get show => 'Onyesha';

  @override
  String get hide => 'Ficha';

  @override
  String get welcomeTitle => 'Karibu Saka Job';

  @override
  String get welcomeBody =>
      'Ingia na wakala wako ataanza kusoma matangazo usiku wa leo.';

  @override
  String get createAccount => 'Fungua akaunti yako';

  @override
  String get displayName => 'Jina lako';

  @override
  String get emailRequired => 'Weka barua pepe yako';

  @override
  String get emailInvalid => 'Hii haionekani kama barua pepe';

  @override
  String get passwordRequired => 'Weka nenosiri lako';

  @override
  String passwordTooShort(int count) {
    return 'Angalau herufi $count';
  }

  @override
  String get home => 'Nyumbani';

  @override
  String get jobs => 'Kazi';

  @override
  String get pipeline => 'Mtiririko';

  @override
  String get profile => 'Wasifu';

  @override
  String get goodMorning => 'Habari za asubuhi';

  @override
  String get goodAfternoon => 'Habari za mchana';

  @override
  String get goodEvening => 'Habari za jioni';

  @override
  String get needsYourYes => 'Zinasubiri idhini yako';

  @override
  String get latestMatches => 'Zinazolingana hivi karibuni';

  @override
  String get seeAll => 'Ona zote';

  @override
  String get review => 'Kagua';

  @override
  String get nothingToReview => 'Hakuna kinachokusubiri';

  @override
  String get nothingToReviewBody =>
      'Wakala bado anasoma. Nafasi mpya zitafika baada ya uchambuzi ujao.';

  @override
  String get searchHint => 'Tafuta nafasi, kampuni...';

  @override
  String get approve => 'Kubali';

  @override
  String get approveAndApply => 'Kubali na tuma';

  @override
  String get pass => 'Ruka';

  @override
  String get askForEdits => 'Omba marekebisho';

  @override
  String jobsLeft(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count zimebaki',
      one: '1 imebaki',
      zero: 'Zimekwisha',
    );
    return '$_temp0';
  }

  @override
  String aboutMinutes(int count) {
    return 'takribani dakika $count';
  }

  @override
  String get aiScore => 'Alama ya AI';

  @override
  String get theRole => 'Nafasi';

  @override
  String get strongMatch => 'Inalingana sana';

  @override
  String get goodMatch => 'Inalingana vizuri';

  @override
  String get partialMatch => 'Inalingana kiasi';

  @override
  String get weakMatch => 'Hailingani sana';

  @override
  String aboveThreshold(int threshold) {
    return 'Juu ya kiwango chako cha $threshold';
  }

  @override
  String belowThreshold(int threshold) {
    return 'Chini ya kiwango chako cha $threshold';
  }

  @override
  String get axisSkills => 'Ujuzi na teknolojia';

  @override
  String get axisSeniority => 'Kiwango cha uzoefu';

  @override
  String get axisDomain => 'Sekta';

  @override
  String get axisLocation => 'Mahali na mtindo';

  @override
  String get axisPay => 'Malipo dhidi ya kiwango chako';

  @override
  String get axisUnknown => 'Haijatajwa';

  @override
  String weakestAxis(String axis) {
    return 'Dhaifu zaidi: $axis';
  }

  @override
  String get matchedSkills => 'Zinazolingana';

  @override
  String get missingSkills => 'Hazijathibitishwa';

  @override
  String get tailoredCv => 'CV iliyoboreshwa';

  @override
  String get coverLetter => 'Barua ya maombi';

  @override
  String get changes => 'Mabadiliko';

  @override
  String get fullCv => 'CV kamili';

  @override
  String get noDiffYet =>
      'CV hii iliundwa kabla ya ufuatiliaji wa mabadiliko. Iunde upya ili kuona mabadiliko.';

  @override
  String get looksGoodSendIt => 'Inafaa — ituma';

  @override
  String get viewPosting => 'Ona tangazo';

  @override
  String get statusNew => 'Mpya';

  @override
  String get statusScored => 'Imepimwa';

  @override
  String get statusLowMatch => 'Hailingani';

  @override
  String get statusScoreFailed => 'Upimaji ulishindikana';

  @override
  String get statusCVGenerated => 'CV tayari';

  @override
  String get statusAwaitingApproval => 'Inasubiri idhini';

  @override
  String get statusApproved => 'Imekubaliwa';

  @override
  String get statusRejected => 'Umeruka';

  @override
  String get statusApplied => 'Umetuma';

  @override
  String get statusManualApply => 'Tuma mwenyewe';

  @override
  String get statusFollowUpSent => 'Umefuatilia';

  @override
  String get statusClosed => 'Imefungwa';

  @override
  String get statusAcknowledged => 'Imepokelewa';

  @override
  String get statusInterviewing => 'Usaili';

  @override
  String get statusOfferReceived => 'Ofa';

  @override
  String get statusEmployerRejected => 'Hukuchaguliwa';

  @override
  String get inYourQueue => 'Kwenye foleni yako';

  @override
  String get waitingOnYou => 'zinasubiri idhini yako';

  @override
  String get applied => 'Umetuma';

  @override
  String get sentNoReply => 'imetumwa, hakuna jibu bado';

  @override
  String get interview => 'Usaili';

  @override
  String get offer => 'Ofa';

  @override
  String liveApplications(int count) {
    return 'Maombi $count yanaendelea';
  }

  @override
  String get masterCv => 'CV kuu';

  @override
  String get replace => 'Badilisha';

  @override
  String get upload => 'Pakia';

  @override
  String get cvStrength => 'Ubora wa CV';

  @override
  String get targetKeywords => 'Maneno muhimu';

  @override
  String get agentSettings => 'Mipangilio ya wakala';

  @override
  String get scoreThreshold => 'Kiwango cha alama';

  @override
  String get thresholdHelp =>
      'Onyesha tu nafasi zenye angalau alama hii. Juu zaidi maana yake chache, bora zaidi.';

  @override
  String get widerNet => 'Nyingi zaidi';

  @override
  String get onlyTheBest => 'Bora pekee';

  @override
  String get notifications => 'Arifa';

  @override
  String get theme => 'Muonekano';

  @override
  String get themeSystem => 'Ya mfumo';

  @override
  String get themeLight => 'Nuru';

  @override
  String get themeDark => 'Giza';

  @override
  String get language => 'Lugha';

  @override
  String get english => 'Kiingereza';

  @override
  String get swahili => 'Kiswahili';

  @override
  String get preferredLocations => 'Maeneo unayopendelea';

  @override
  String get remotePreference => 'Mtindo wa kazi';

  @override
  String get salaryFloor => 'Kiwango cha chini cha malipo';

  @override
  String get notifyOnApproval => 'Kazi zinazosubiri idhini yako';

  @override
  String get notifyOnReply => 'Majibu ya waajiri';

  @override
  String get notifyOnFollowUp => 'Ufuatiliaji unaohitajika';

  @override
  String get notifyOnFailure => 'Matatizo ya mfumo';

  @override
  String get repliesNeedingYou => 'Majibu yanayokusubiri';

  @override
  String get confirmSuggestion => 'Je, hii ni sahihi?';

  @override
  String get yesConfirm => 'Ndiyo, ni sahihi';

  @override
  String get noDismiss => 'Hapana, ondoa';

  @override
  String get retry => 'Jaribu tena';

  @override
  String get workOffline => 'Tumia bila mtandao';

  @override
  String get couldNotReach => 'Hatukuweza kumfikia wakala';

  @override
  String get couldNotReachBody =>
      'Foleni yako imehifadhiwa — hakuna kilichopotea. Tutasawazisha ukirudi mtandaoni.';

  @override
  String get somethingWentWrong => 'Kuna hitilafu imetokea';

  @override
  String get noConnection => 'Hakuna mtandao';

  @override
  String get cancel => 'Ghairi';

  @override
  String get save => 'Hifadhi';

  @override
  String get saved => 'Imehifadhiwa';

  @override
  String get close => 'Funga';

  @override
  String get loading => 'Inapakia';

  @override
  String get onboardCvTitle => 'Weka CV yako';

  @override
  String get onboardCvBody =>
      'Faili moja ya PDF au Word. Wakala ataisoma na kujenga wasifu wako.';

  @override
  String get onboardBarTitle => 'Weka kiwango chako';

  @override
  String get onboardBarBody =>
      'Kiwango cha alama, malipo ya chini, kazi ya mbali au la — badilisha wakati wowote.';

  @override
  String get onboardYesTitle => 'Kubali kila asubuhi';

  @override
  String get onboardYesBody =>
      'Dakika chache kwa siku. Mengine yote yatashughulikiwa.';
}
