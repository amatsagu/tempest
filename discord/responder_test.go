package discord

import (
	"errors"
	"testing"
)

type stubResponder struct {
	response  Response
	data      ResponseMessageData
	ephemeral bool
	update    bool
	calls     int
}

func (s *stubResponder) Send(r Response) error {
	s.calls++
	s.response = r
	return nil
}

func (s *stubResponder) Reply(data ResponseMessageData, ephemeral, update bool, files []File) error {
	s.calls++
	s.data = data
	s.ephemeral = ephemeral
	s.update = update
	return nil
}

func (s *stubResponder) FollowUp(ResponseMessageData, bool, []File) (*Message, error) {
	return nil, nil
}

func (s *stubResponder) EditFollowUp(MessageID, ResponseMessageData, []File) error { return nil }

func (s *stubResponder) DeleteFollowUp(MessageID) error { return nil }

func (s *stubResponder) Responded() bool { return false }

func TestDeferRequiresResponder(t *testing.T) {
	var itx CommandInteraction
	if err := itx.Defer(false); !errors.Is(err, errNoResponder) {
		t.Fatalf("err = %v, want errNoResponder", err)
	}
}

func TestDeferSendsDeferredResponse(t *testing.T) {
	stub := &stubResponder{}

	var command CommandInteraction
	command.Type = InteractionApplicationCommand
	command.SetResponder(stub)
	if err := command.Defer(true); err != nil {
		t.Fatalf("Defer: %v", err)
	}

	if stub.response.Type != ResponseDeferredChannelMessageWithSource {
		t.Fatalf("command defer type = %d, want deferred channel message", stub.response.Type)
	}

	if data, ok := stub.response.Data.(ResponseMessageData); !ok || data.Flags != MessageFlagEphemeral {
		t.Fatalf("command defer data = %#v, want an ephemeral message", stub.response.Data)
	}

	var component ComponentInteraction
	component.Type = InteractionMessageComponent
	component.SetResponder(stub)
	if err := component.Defer(false); err != nil {
		t.Fatalf("Defer: %v", err)
	}

	if stub.response.Type != ResponseDeferredUpdateMessage {
		t.Fatalf("component defer type = %d, want deferred update", stub.response.Type)
	}
}

func TestDeferRejectsAutocomplete(t *testing.T) {
	stub := &stubResponder{}

	var itx CommandInteraction
	itx.Type = InteractionApplicationCommandAutocomplete
	itx.SetResponder(stub)

	if err := itx.Defer(false); err == nil {
		t.Fatal("expected autocomplete defer to fail")
	}

	if stub.calls != 0 {
		t.Fatalf("responder calls = %d, want 0", stub.calls)
	}
}

func TestReplyDelegatesToResponder(t *testing.T) {
	stub := &stubResponder{}

	var command CommandInteraction
	command.Type = InteractionApplicationCommand
	command.SetResponder(stub)

	if err := command.Reply(ResponseMessageData{Content: "hi"}, true); err != nil {
		t.Fatalf("Reply: %v", err)
	}

	if stub.data.Content != "hi" || !stub.ephemeral || stub.update {
		t.Fatalf("reply = data %+v ephemeral %v update %v", stub.data, stub.ephemeral, stub.update)
	}

	// Update asks for the update-message first response and is never ephemeral.
	if err := command.Update(ResponseMessageData{Content: "edit"}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if stub.data.Content != "edit" || stub.ephemeral || !stub.update {
		t.Fatalf("update = data %+v ephemeral %v update %v", stub.data, stub.ephemeral, stub.update)
	}
}

func TestReplyRequiresResponder(t *testing.T) {
	var itx CommandInteraction
	if err := itx.Reply(ResponseMessageData{}, false); !errors.Is(err, errNoResponder) {
		t.Fatalf("err = %v, want errNoResponder", err)
	}

	if err := itx.Update(ResponseMessageData{}); !errors.Is(err, errNoResponder) {
		t.Fatalf("err = %v, want errNoResponder", err)
	}
}

func TestRespondSendsGivenResponse(t *testing.T) {
	stub := &stubResponder{}

	var itx ComponentInteraction
	itx.SetResponder(stub)

	if err := itx.Respond(Response{Type: ResponseUpdateMessage, Data: ResponseMessageData{Content: "hi"}}); err != nil {
		t.Fatalf("Respond: %v", err)
	}

	if stub.response.Type != ResponseUpdateMessage {
		t.Fatalf("type = %d, want update message", stub.response.Type)
	}
}

func TestRespondRequiresResponder(t *testing.T) {
	var itx ModalInteraction
	if err := itx.Respond(Response{}); !errors.Is(err, errNoResponder) {
		t.Fatalf("err = %v, want errNoResponder", err)
	}
}
