package avs

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// An interface that represents both raw Message objects and more specifically
// typed ones. Usually, values of this interface are used with a type switch:
//	switch d := typedMessage.(type) {
//	case *Speak:
//		fmt.Printf("We got a spoken response in format %s.\n", d.Payload.Format)
//	}
//
type TypedMessage interface {
	GetMessage() *Message
	Typed() TypedMessage
}

// A general structure for contexts, events and directives.
type Message struct {
	Header  map[string]string `json:"header"`
	Payload json.RawMessage   `json:"payload,omitempty"`
}

// Creates a Message suited for being used as a context value.
func NewContext(namespace, name string) *Message {
	return &Message{
		Header: map[string]string{
			"namespace": namespace,
			"name":      name,
		},
		Payload: nil,
	}
}

// Creates a Message suited for being used as an event value.
func NewEvent(namespace, name, messageId, dialogRequestId string) *Message {
	m := &Message{
		Header: map[string]string{
			"namespace": namespace,
			"name":      name,
			"messageId": messageId,
		},
		Payload: nil,
	}
	if dialogRequestId != "" {
		m.Header["dialogRequestId"] = dialogRequestId
	}
	return m
}

// Returns a pointer to the underlying Message object.
func (m *Message) GetMessage() *Message {
	return m
}

// Returns the namespace and name as a single string.
func (m *Message) String() string {
	return fmt.Sprintf("%s.%s", m.Header["namespace"], m.Header["name"])
}

// Returns a more specific type for this context, event or directive.
func (m *Message) Typed() TypedMessage {
	switch m.String() {
	case "AudioPlayer.ClearQueue":
		return fill(new(ClearQueue), m)
	case "AudioPlayer.Play":
		return fill(new(Play), m)
	case "AudioPlayer.PlaybackState":
		return fill(new(PlaybackState), m)
	case "AudioPlayer.Stop":
		return fill(new(Stop), m)
	case "SpeechRecognizer.ExpectSpeech":
		return fill(new(ExpectSpeech), m)
	case "SpeechRecognizer.ExpectSpeechTimedOut":
		return fill(new(ExpectSpeechTimedOut), m)
	case "SpeechRecognizer.Recognize":
		return fill(new(Recognize), m)
	case "SpeechSynthesizer.Speak":
		return fill(new(Speak), m)
	case "System.Exception":
		return fill(new(Exception), m)
	case "System.SynchronizeState":
		return fill(new(SynchronizeState), m)
	default:
		return m
	}
}

// The ClearQueue directive.
type ClearQueue struct {
	*Message
	Payload struct {
		ClearBehavior ClearBehavior `json:"clearBehavior"`
	} `json:"payload"`
}

// The Exception message.
type Exception struct {
	*Message
	Payload struct {
		Code        string `json:"code"`
		Description string `json:"description"`
	} `json:"payload"`
}

// The ExpectSpeech directive.
type ExpectSpeech struct {
	*Message
	Payload struct {
		TimeoutInMilliseconds float64 `json:"timeoutInMilliseconds"`
	} `json:"payload"`
}

func (m *ExpectSpeech) Timeout() time.Duration {
	return time.Duration(m.Payload.TimeoutInMilliseconds) * time.Millisecond
}

// The ExpectSpeechTimedOut event.
type ExpectSpeechTimedOut struct {
	*Message
	Payload struct{} `json:"payload"`
}

func NewExpectSpeechTimedOut(messageId string) *ExpectSpeechTimedOut {
	m := new(ExpectSpeechTimedOut)
	m.Message = NewEvent("SpeechRecognizer", "ExpectSpeechTimedOut", messageId, "")
	return m
}

// The Play directive.
type Play struct {
	*Message
	Payload struct {
		AudioItem    AudioItem    `json:"audioItem"`
		PlayBehavior PlayBehavior `json:"playBehavior"`
	} `json:"payload"`
}

func (m *Play) DialogRequestId() string {
	return m.Header["dialogRequestId"]
}

func (m *Play) MessageId() string {
	return m.Header["messageId"]
}

// The PlaybackState context.
type PlaybackState struct {
	*Message
	Payload struct {
		Token                string         `json:"token"`
		OffsetInMilliseconds float64        `json:"offsetInMilliseconds"`
		PlayerActivity       PlayerActivity `json:"playerActivity"`
	}
}

func NewPlaybackState(token string, offset time.Duration, activity PlayerActivity) *PlaybackState {
	m := new(PlaybackState)
	m.Message = NewContext("AudioPlayer", "PlaybackState")
	m.Payload.OffsetInMilliseconds = offset.Seconds() * 1000
	m.Payload.PlayerActivity = activity
	m.Payload.Token = token
	return m
}

func (m *PlaybackState) Offset() time.Duration {
	return time.Duration(m.Payload.OffsetInMilliseconds) * time.Millisecond
}

// The Recognize event.
type Recognize struct {
	*Message
	Payload struct {
		Profile string `json:"profile"`
		Format  string `json:"format"`
	} `json:"payload"`
}

func NewRecognize(messageId, dialogRequestId string) *Recognize {
	m := new(Recognize)
	m.Message = NewEvent("SpeechRecognizer", "Recognize", messageId, dialogRequestId)
	m.Payload.Format = "AUDIO_L16_RATE_16000_CHANNELS_1"
	m.Payload.Profile = "CLOSE_TALK"
	return m
}

// The Speak directive.
type Speak struct {
	*Message
	Payload struct {
		Format string
		URL    string
	} `json:"payload"`
}

func (m *Speak) ContentId() string {
	if !strings.HasPrefix(m.Payload.URL, "cid:") {
		return ""
	}
	return m.Payload.URL[4:]
}

// The Stop directive.
type Stop struct {
	*Message
	Payload struct{} `json:"payload"`
}

// The SynchronizeState event.
type SynchronizeState struct {
	*Message
	Payload struct{} `json:"payload"`
}

func NewSynchronizeState(messageId string) *SynchronizeState {
	m := new(SynchronizeState)
	m.Message = NewEvent("System", "SynchronizeState", messageId, "")
	return m
}

// Convenience method to set up an empty typed message object from a raw Message.
func fill(dst TypedMessage, src *Message) TypedMessage {
	v := reflect.ValueOf(dst).Elem()
	v.FieldByName("Message").Set(reflect.ValueOf(src))
	payload := v.FieldByName("Payload")
	if payload.Kind() != reflect.Struct {
		return dst
	}
	json.Unmarshal(src.Payload, payload.Addr().Interface())
	return dst
}
