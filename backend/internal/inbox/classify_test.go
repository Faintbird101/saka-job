package inbox

import "testing"

func cls(subject, body string) Classification {
	return ClassifyKeywords(Message{Subject: subject, Body: body})
}

func TestClassifyAcknowledgement(t *testing.T) {
	for _, body := range []string{
		"Thank you for applying to the Flutter Engineer role.",
		"We have received your application and will be in touch.",
		"Your application has been received.",
		"We are currently reviewing your application.",
	} {
		got := cls("Application received", body)
		if got.Kind != KindAcknowledgement {
			t.Errorf("%q -> %s (want acknowledgement)", body, got.Kind)
		}
	}
}

func TestClassifyRejection(t *testing.T) {
	for _, body := range []string{
		"We regret to inform you that we will not be moving forward.",
		"Unfortunately, we have decided to proceed with other candidates.",
		"You were unsuccessful on this occasion.",
		"We have chosen to move forward with another applicant.",
		"We will not be progressing your application further.",
	} {
		got := cls("Update on your application", body)
		if got.Kind != KindRejection {
			t.Errorf("%q -> %s (want rejection)", body, got.Kind)
		}
	}
}

func TestClassifyInterview(t *testing.T) {
	for _, body := range []string{
		"We would like to invite you to an interview next week.",
		"Can we schedule a call to discuss the role?",
		"We'd like to move you to the next stage.",
		"Please share your availability for a call.",
		"The next step is a technical assessment.",
	} {
		got := cls("Next steps", body)
		if got.Kind != KindInterview {
			t.Errorf("%q -> %s (want interview)", body, got.Kind)
		}
	}
}

func TestClassifyOffer(t *testing.T) {
	for _, body := range []string{
		"We are pleased to offer you the position.",
		"Please find attached your offer letter.",
		"We are delighted to extend an offer of employment.",
	} {
		if got := cls("Offer", body); got.Kind != KindOffer {
			t.Errorf("%q -> %s (want offer)", body, got.Kind)
		}
	}
}

// THE case that motivated doing this carefully. Rejection wording inside an
// email that is actually good news must not be read as a rejection — that is
// the error that makes you abandon a live opportunity.
func TestRejectionWordingInsideGoodNewsIsNotARejection(t *testing.T) {
	cases := []string{
		"Unfortunately we cannot match your salary expectations, but we would like to invite you to an interview.",
		"Unfortunately the original role was filled. However, we'd like to speak with you about a similar position.",
		"Unfortunately there was a delay. We are pleased to offer you the role.",
	}
	for _, body := range cases {
		got := cls("Your application", body)
		if got.Kind == KindRejection {
			t.Errorf("read as a rejection, which would abandon a live opportunity:\n  %q", body)
		}
		if got.Kind != KindInterview && got.Kind != KindOffer {
			t.Errorf("%q -> %s (want interview or offer)", body, got.Kind)
		}
	}
}

// An offer outranks interview language in the same message.
func TestOfferOutranksInterview(t *testing.T) {
	got := cls("Good news", "Following your interview we are pleased to offer you the position. We can schedule a call to discuss.")
	if got.Kind != KindOffer {
		t.Errorf("got %s, want offer", got.Kind)
	}
}

func TestUnrelatedMailIsOther(t *testing.T) {
	for _, body := range []string{
		"Your Amazon order has shipped.",
		"Reminder: your dentist appointment is on Tuesday.",
		"",
	} {
		got := cls("Random", body)
		if got.Kind != KindOther {
			t.Errorf("%q -> %s (want other)", body, got.Kind)
		}
		if !NeedsLLM(got) {
			t.Error("an 'other' verdict must be escalated, not trusted")
		}
	}
}

// Only the safe transition is auto-applied; the rest need a human.
func TestConsequentialKinds(t *testing.T) {
	if KindAcknowledgement.Consequential() {
		t.Error("acknowledgement should be safe to apply automatically")
	}
	if KindOther.Consequential() {
		t.Error("other implies no status change at all")
	}
	for _, k := range []Kind{KindRejection, KindInterview, KindOffer} {
		if !k.Consequential() {
			t.Errorf("%s must require confirmation", k)
		}
	}
}

func TestNeedsLLMOnLowConfidence(t *testing.T) {
	// "keep your CV on file" is rated below the threshold on purpose: it is
	// usually a rejection but reads as a courtesy, so it should be escalated.
	got := cls("Thanks", "We will keep your CV on file for future roles.")
	if !NeedsLLM(got) {
		t.Errorf("confidence %d should have been escalated to the model", got.Confidence)
	}
}

func TestSuggestedStatusMapping(t *testing.T) {
	want := map[Kind]string{
		KindAcknowledgement: "Acknowledged",
		KindInterview:       "Interviewing",
		KindOffer:           "OfferReceived",
		KindRejection:       "EmployerRejected",
		KindOther:           "",
	}
	for k, v := range want {
		if got := SuggestedStatus(k); got != v {
			t.Errorf("SuggestedStatus(%s) = %q, want %q", k, got, v)
		}
	}
}

// An employer rejection must never map onto 'Rejected', which means YOU
// declined the job. Conflating them makes the pipeline unreadable.
func TestEmployerRejectionIsDistinctFromSelfRejection(t *testing.T) {
	if SuggestedStatus(KindRejection) == "Rejected" {
		t.Fatal("employer rejection must not reuse 'Rejected' — that means you declined the job")
	}
}

func TestClassificationCarriesEvidence(t *testing.T) {
	got := cls("Update", "We regret to inform you that we are not proceeding.")
	if got.Evidence == "" {
		t.Error("no evidence recorded; a classification you cannot check is not auditable")
	}
	if got.Classifier != "keyword" {
		t.Errorf("classifier = %q", got.Classifier)
	}
}
