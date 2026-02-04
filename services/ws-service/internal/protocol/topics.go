package protocol

type Topic string

const (
	TopicUser          Topic = "user"
	TopicConversation  Topic = "conversation"
	TopicPresence      Topic = "presence"
	TopicTyping        Topic = "typing"
	TopicCalls         Topic = "calls"
	TopicNotifications Topic = "notifications"
)

func GetResourceID(topic Topic, filters map[string]string) string {
	switch topic {
	case TopicUser:
		if userID, ok := filters["user_id"]; ok {
			return userID
		}
	case TopicConversation:
		if convID, ok := filters["conversation_id"]; ok {
			return convID
		}
	case TopicPresence:
		return "global"
	case TopicTyping:
		if convID, ok := filters["conversation_id"]; ok {
			return convID
		}
	case TopicCalls:
		if callID, ok := filters["call_id"]; ok {
			return callID
		}
	case TopicNotifications:
		if userID, ok := filters["user_id"]; ok {
			return userID
		}
	}
	return "default"
}
