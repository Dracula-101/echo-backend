package service

// Ensure messageService implements MessageServiceInterface
var _ MessageServiceInterface = (*messageService)(nil)

// Ensure conversationService implements ConversationServiceInterface
var _ ConversationService = (*conversationService)(nil)
