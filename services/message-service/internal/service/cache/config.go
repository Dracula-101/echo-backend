package cache

import "time"

// ── Key prefixes ──────────────────────────────────────────────────────────────

var (
	ConvListPrefix    = "conv_list:"
	ConvPrefix        = "conv:"
	UnreadPrefix      = "unread:"
	TotalUnreadPrefix = "total_unread:"
	MsgsPrefix        = "msgs:"
	TypingPrefix      = "typing:"
	TypingListPrefix  = "typing_list:"
	DeliveryPrefix    = "delivery:"
	ParticipantPrefix = "participant:"
	PinnedPrefix      = "pinned:"
	DMPairPrefix      = "dm_pair:"

	LockDMPrefix  = "lock:dm:"
	LockMsgPrefix = "lock:msg:"
)

// ── TTLs ──────────────────────────────────────────────────────────────────────

var (
	ConvListTTL    = 300 * time.Second
	ConvTTL        = 120 * time.Second
	UnreadTTL      = 3600 * time.Second
	TotalUnreadTTL = 60 * time.Second
	MsgsTTL        = 60 * time.Second
	TypingTTL      = 8 * time.Second
	TypingListTTL  = 10 * time.Second
	DeliveryTTL    = 3600 * time.Second
	ParticipantTTL = 600 * time.Second
	PinnedTTL      = 600 * time.Second
	DMPairTTL      = 86400 * time.Second

	LockDMTTL  = 10 * time.Second
	LockMsgTTL = 300 * time.Second
)
