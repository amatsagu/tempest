package discord

import "testing"

func TestSubmittedComponents(t *testing.T) {
	txtPayload := []byte(`{"type":4,"id":1,"custom_id":"bio","value":"Developer from Poland"}`)
	comp, err := UnmarshalSubmittedComponent(txtPayload)
	if err != nil {
		t.Fatalf("UnmarshalSubmittedComponent failed: %v", err)
	}

	txt, ok := comp.(SubmittedTextInput)
	if !ok {
		t.Fatalf("expected SubmittedTextInput, got %T", comp)
	}
	if txt.CustomID != "bio" || txt.Value != "Developer from Poland" {
		t.Errorf("got %+v", txt)
	}

	chkPayload := []byte(`{"type":23,"id":2,"custom_id":"accept_rules","value":true}`)
	comp2, err := UnmarshalSubmittedComponent(chkPayload)
	if err != nil {
		t.Fatalf("UnmarshalSubmittedComponent failed: %v", err)
	}

	chk, ok := comp2.(SubmittedCheckbox)
	if !ok {
		t.Fatalf("expected SubmittedCheckbox, got %T", comp2)
	}
	if !chk.Value || chk.CustomID != "accept_rules" {
		t.Errorf("got %+v", chk)
	}

	labelPayload := []byte(`{"type":18,"id":3,"component":{"type":4,"id":1,"custom_id":"name","value":"Amatsagu"}}`)
	comp3, err := UnmarshalSubmittedComponent(labelPayload)
	if err != nil {
		t.Fatalf("UnmarshalSubmittedComponent label failed: %v", err)
	}

	lbl, ok := comp3.(SubmittedLabel)
	if !ok {
		t.Fatalf("expected SubmittedLabel, got %T", comp3)
	}
	nestedTxt, ok := lbl.Component.(SubmittedTextInput)
	if !ok || nestedTxt.Value != "Amatsagu" {
		t.Errorf("nested component mismatch: %+v", lbl.Component)
	}
}

func BenchmarkUnmarshalSubmittedComponent(b *testing.B) {
	payload := []byte(`{"type":4,"id":1,"custom_id":"bio","value":"Developer from Poland"}`)

	b.ReportAllocs()
	for b.Loop() {
		_, _ = UnmarshalSubmittedComponent(payload)
	}
}
