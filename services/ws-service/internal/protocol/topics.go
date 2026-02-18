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

func GetResourceID(topic Topic, filters MessageFilters) string {
	switch topic {
	case TopicUser:
		if filters.UserID != nil {
			return *filters.UserID
		}
	case TopicConversation:
		if filters.ConversationIDs != nil && len(*filters.ConversationIDs) > 0 {
			return (*filters.ConversationIDs)[0]
		}
	case TopicPresence:
		return "global"
	case TopicTyping:
		if filters.ConversationID != nil {
			return *filters.ConversationID
		}
	case TopicCalls:
		if filters.ConversationID != nil {
			return *filters.ConversationID
		}
	case TopicNotifications:
		if filters.UserID != nil {
			return *filters.UserID
		}
	}
	return "default"
}
