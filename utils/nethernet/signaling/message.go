package signaling

import (
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)




type Message struct {


	Type int `json:"Type"`




	From string `json:"From,omitempty"`



	To string `json:"To,omitempty"`



	Data string `json:"Message,omitempty"`



	ID uuid.UUID `json:"MessageId,omitempty"`
}





type Error struct {


	Code int `json:"Code"`

	Message string `json:"Message"`
}


func (err *Error) Error() string {
	b := &strings.Builder{}
	b.WriteString("service/signaling: code ")
	b.WriteString(strconv.Itoa(err.Code))
	if err.Message != "" {
		b.WriteByte(':')
		b.WriteByte(' ')
		b.WriteString(strconv.Quote(err.Message))
	}
	return b.String()
}

const (



	ErrorCodePlayerNotFound = iota + 1



	ErrorCodeDeliveryFailure
)




const MessageTypeError = 0

const (


	MessageTypePing = iota





	MessageTypeSignal






	MessageTypeCredentials







	MessageTypeAccepted






	MessageTypeDelivered
)




type MessageStatus struct {

	MessageID uuid.UUID `json:"MessageId"`


	SenderID string `json:"ToPlayerId"`



	DeliveredOn time.Time `json:"DeliveredOn"`



	AcceptedOn time.Time `json:"AcceptedOn"`
}
