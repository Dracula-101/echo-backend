#!/bin/bash

# Echo Backend - Database Seeding Script
set -e

echo "[INFO] Seeding database with test data..."

# Parse command line arguments
SEED_TYPE=${1:-basic}  # basic, full, custom

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | grep -v '#' | awk '/=/ {print $1}')
fi

POSTGRES_HOST=${POSTGRES_HOST:-localhost}
POSTGRES_PORT=${POSTGRES_PORT:-5432}
POSTGRES_USER=${POSTGRES_USER:-echo}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD:-echo_password}
POSTGRES_DB=${POSTGRES_DB:-echo_db}

# Check database connection
echo "[INFO] Checking database connection..."
if ! PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB -c '\q' 2>/dev/null; then
    echo "[ERROR] Cannot connect to database. Please check your connection settings."
    exit 1
fi

# Generate test users in auth schema
echo "[INFO] Creating test users in auth.users..."
PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB <<EOF
-- Insert test users into auth.users
INSERT INTO auth.users (id, email, phone_number, phone_country_code, email_verified, password_hash, password_salt, account_status, created_at, updated_at)
VALUES 
    ('a1111111-1111-1111-1111-111111111111', 'alice@example.com', '+15551234567', '+1', true, '\$2a\$10\$dummy.hash.alice', 'salt_alice', 'active', NOW(), NOW()),
    ('b2222222-2222-2222-2222-222222222222', 'bob@example.com', '+15551234568', '+1', true, '\$2a\$10\$dummy.hash.bob', 'salt_bob', 'active', NOW(), NOW()),
    ('c3333333-3333-3333-3333-333333333333', 'charlie@example.com', '+15551234569', '+1', true, '\$2a\$10\$dummy.hash.charlie', 'salt_charlie', 'active', NOW(), NOW()),
    ('d4444444-4444-4444-4444-444444444444', 'david@example.com', '+15551234570', '+1', true, '\$2a\$10\$dummy.hash.david', 'salt_david', 'active', NOW(), NOW()),
    ('e5555555-5555-5555-5555-555555555555', 'eve@example.com', '+15551234571', '+1', true, '\$2a\$10\$dummy.hash.eve', 'salt_eve', 'active', NOW(), NOW())
ON CONFLICT (email) DO NOTHING;

EOF

echo "[INFO] Creating test user profiles..."
PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB <<EOF
-- Insert test user profiles in users.profiles
INSERT INTO users.profiles (id, user_id, username, display_name, bio, avatar_url, online_status, created_at, updated_at)
VALUES
    (gen_random_uuid(), 'a1111111-1111-1111-1111-111111111111', 'alice', 'Alice Test', 'Software engineer who loves coding', 'https://api.dicebear.com/7.x/avataaars/svg?seed=alice', 'online', NOW(), NOW()),
    (gen_random_uuid(), 'b2222222-2222-2222-2222-222222222222', 'bob', 'Bob Test', 'Product manager and tech enthusiast', 'https://api.dicebear.com/7.x/avataaars/svg?seed=bob', 'online', NOW(), NOW()),
    (gen_random_uuid(), 'c3333333-3333-3333-3333-333333333333', 'charlie', 'Charlie Test', 'Designer passionate about UX', 'https://api.dicebear.com/7.x/avataaars/svg?seed=charlie', 'away', NOW(), NOW()),
    (gen_random_uuid(), 'd4444444-4444-4444-4444-444444444444', 'david', 'David Test', 'DevOps engineer and cloud architect', 'https://api.dicebear.com/7.x/avataaars/svg?seed=david', 'offline', NOW(), NOW()),
    (gen_random_uuid(), 'e5555555-5555-5555-5555-555555555555', 'eve', 'Eve Test', 'Data scientist exploring AI', 'https://api.dicebear.com/7.x/avataaars/svg?seed=eve', 'busy', NOW(), NOW())
ON CONFLICT (username) DO NOTHING;

EOF

echo "[INFO] Creating test user settings..."
PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB <<EOF
-- Insert default settings for test users
INSERT INTO users.settings (user_id, profile_visibility, last_seen_visibility, online_status_visibility, read_receipts_enabled, typing_indicators_enabled)
SELECT 
    id,
    'public',
    'everyone',
    'everyone',
    true,
    true
FROM auth.users
WHERE email LIKE '%@example.com'
ON CONFLICT (user_id) DO NOTHING;

EOF

echo "[INFO] Creating test contacts/friendships..."
PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB <<EOF
-- Create friendships between test users
INSERT INTO users.contacts (user_id, contact_user_id, relationship_type, status, accepted_at, created_at, updated_at)
VALUES
    ('a1111111-1111-1111-1111-111111111111', 'b2222222-2222-2222-2222-222222222222', 'friend', 'accepted', NOW(), NOW(), NOW()),
    ('a1111111-1111-1111-1111-111111111111', 'c3333333-3333-3333-3333-333333333333', 'friend', 'accepted', NOW(), NOW(), NOW()),
    ('b2222222-2222-2222-2222-222222222222', 'a1111111-1111-1111-1111-111111111111', 'friend', 'accepted', NOW(), NOW(), NOW()),
    ('b2222222-2222-2222-2222-222222222222', 'd4444444-4444-4444-4444-444444444444', 'friend', 'accepted', NOW(), NOW(), NOW()),
    ('c3333333-3333-3333-3333-333333333333', 'a1111111-1111-1111-1111-111111111111', 'friend', 'accepted', NOW(), NOW(), NOW()),
    ('c3333333-3333-3333-3333-333333333333', 'e5555555-5555-5555-5555-555555555555', 'friend', 'accepted', NOW(), NOW(), NOW()),
    ('d4444444-4444-4444-4444-444444444444', 'b2222222-2222-2222-2222-222222222222', 'friend', 'accepted', NOW(), NOW(), NOW()),
    ('e5555555-5555-5555-5555-555555555555', 'c3333333-3333-3333-3333-333333333333', 'friend', 'accepted', NOW(), NOW(), NOW())
ON CONFLICT (user_id, contact_user_id) DO NOTHING;

EOF

echo "[INFO] Creating test conversations..."
PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB <<EOF
-- Create direct conversations in messages.conversations
INSERT INTO messages.conversations (id, conversation_type, creator_user_id, is_group, is_encrypted, is_active, created_at, updated_at)
VALUES
    ('11111111-1111-1111-1111-111111111111', 'direct', 'a1111111-1111-1111-1111-111111111111', false, true, true, NOW(), NOW()),
    ('22222222-2222-2222-2222-222222222222', 'direct', 'a1111111-1111-1111-1111-111111111111', false, true, true, NOW(), NOW()),
    ('33333333-3333-3333-3333-333333333333', 'direct', 'b2222222-2222-2222-2222-222222222222', false, true, true, NOW(), NOW()),
    ('44444444-4444-4444-4444-444444444444', 'group', 'a1111111-1111-1111-1111-111111111111', true, true, true, NOW(), NOW()),
    ('55555555-5555-5555-5555-555555555555', 'direct', 'c3333333-3333-3333-3333-333333333333', false, true, true, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Add conversation participants
INSERT INTO messages.conversation_participants (conversation_id, user_id, role, can_send_messages, joined_at, created_at, updated_at)
VALUES
    ('11111111-1111-1111-1111-111111111111', 'a1111111-1111-1111-1111-111111111111', 'owner', true, NOW(), NOW(), NOW()),
    ('11111111-1111-1111-1111-111111111111', 'b2222222-2222-2222-2222-222222222222', 'member', true, NOW(), NOW(), NOW()),
    ('22222222-2222-2222-2222-222222222222', 'a1111111-1111-1111-1111-111111111111', 'owner', true, NOW(), NOW(), NOW()),
    ('22222222-2222-2222-2222-222222222222', 'c3333333-3333-3333-3333-333333333333', 'member', true, NOW(), NOW(), NOW()),
    ('33333333-3333-3333-3333-333333333333', 'b2222222-2222-2222-2222-222222222222', 'owner', true, NOW(), NOW(), NOW()),
    ('33333333-3333-3333-3333-333333333333', 'd4444444-4444-4444-4444-444444444444', 'member', true, NOW(), NOW(), NOW()),
    ('44444444-4444-4444-4444-444444444444', 'a1111111-1111-1111-1111-111111111111', 'owner', true, NOW(), NOW(), NOW()),
    ('44444444-4444-4444-4444-444444444444', 'b2222222-2222-2222-2222-222222222222', 'admin', true, NOW(), NOW(), NOW()),
    ('44444444-4444-4444-4444-444444444444', 'c3333333-3333-3333-3333-333333333333', 'member', true, NOW(), NOW(), NOW()),
    ('44444444-4444-4444-4444-444444444444', 'e5555555-5555-5555-5555-555555555555', 'member', true, NOW(), NOW(), NOW()),
    ('55555555-5555-5555-5555-555555555555', 'c3333333-3333-3333-3333-333333333333', 'owner', true, NOW(), NOW(), NOW()),
    ('55555555-5555-5555-5555-555555555555', 'e5555555-5555-5555-5555-555555555555', 'member', true, NOW(), NOW(), NOW())
ON CONFLICT (conversation_id, user_id) DO NOTHING;

EOF

echo "[INFO] Creating test messages..."
PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB <<EOF
-- Insert test messages in messages.messages
INSERT INTO messages.messages (id, conversation_id, sender_user_id, message_type, content, is_deleted, is_edited, created_at, updated_at)
VALUES
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'a1111111-1111-1111-1111-111111111111', 'text', 'Hey Bob! How are you doing?', false, false, NOW() - interval '2 hours', NOW() - interval '2 hours'),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'b2222222-2222-2222-2222-222222222222', 'text', 'Hi Alice! I''m doing great, thanks for asking!', false, false, NOW() - interval '1 hour 50 minutes', NOW() - interval '1 hour 50 minutes'),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'a1111111-1111-1111-1111-111111111111', 'text', 'That''s awesome! Want to grab coffee later?', false, false, NOW() - interval '1 hour 30 minutes', NOW() - interval '1 hour 30 minutes'),
    (gen_random_uuid(), '22222222-2222-2222-2222-222222222222', 'a1111111-1111-1111-1111-111111111111', 'text', 'Charlie, did you see the new design mockups?', false, false, NOW() - interval '45 minutes', NOW() - interval '45 minutes'),
    (gen_random_uuid(), '22222222-2222-2222-2222-222222222222', 'c3333333-3333-3333-3333-333333333333', 'text', 'Yes! They look amazing 🎨', false, false, NOW() - interval '40 minutes', NOW() - interval '40 minutes'),
    (gen_random_uuid(), '33333333-3333-3333-3333-333333333333', 'b2222222-2222-2222-2222-222222222222', 'text', 'David, can you check the deployment logs?', false, false, NOW() - interval '30 minutes', NOW() - interval '30 minutes'),
    (gen_random_uuid(), '33333333-3333-3333-3333-333333333333', 'd4444444-4444-4444-4444-444444444444', 'text', 'On it! Everything looks good so far.', false, false, NOW() - interval '25 minutes', NOW() - interval '25 minutes'),
    (gen_random_uuid(), '44444444-4444-4444-4444-444444444444', 'a1111111-1111-1111-1111-111111111111', 'text', 'Team, great work on the sprint! 🎉', false, false, NOW() - interval '20 minutes', NOW() - interval '20 minutes'),
    (gen_random_uuid(), '44444444-4444-4444-4444-444444444444', 'b2222222-2222-2222-2222-222222222222', 'text', 'Thanks! Ready for the next one!', false, false, NOW() - interval '15 minutes', NOW() - interval '15 minutes'),
    (gen_random_uuid(), '44444444-4444-4444-4444-444444444444', 'c3333333-3333-3333-3333-333333333333', 'text', 'Let''s keep this momentum going! 💪', false, false, NOW() - interval '10 minutes', NOW() - interval '10 minutes'),
    (gen_random_uuid(), '55555555-5555-5555-5555-555555555555', 'c3333333-3333-3333-3333-333333333333', 'text', 'Eve, the AI model results are impressive!', false, false, NOW() - interval '5 minutes', NOW() - interval '5 minutes'),
    (gen_random_uuid(), '55555555-5555-5555-5555-555555555555', 'e5555555-5555-5555-5555-555555555555', 'text', 'Thank you! Still fine-tuning parameters 🤖', false, false, NOW() - interval '2 minutes', NOW() - interval '2 minutes')
ON CONFLICT DO NOTHING;

EOF

if [ "$SEED_TYPE" = "full" ]; then
    echo "[INFO] Creating test sessions..."
    PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB <<EOF
    -- Create active sessions for test users
    INSERT INTO auth.sessions (user_id, session_token, refresh_token, device_type, device_os, browser_name, ip_address, expires_at, created_at)
    SELECT 
        id,
        'test_session_' || id::text,
        'test_refresh_' || id::text,
        'web',
        'macOS',
        'Chrome',
        '127.0.0.1'::inet,
        NOW() + interval '7 days',
        NOW()
    FROM auth.users
    WHERE email LIKE '%@example.com'
    ON CONFLICT (session_token) DO NOTHING;
EOF

    echo "[INFO] Creating test notifications..."
    PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB <<EOF
    -- Create sample notifications
    INSERT INTO notifications.notifications (user_id, notification_type, title, body, is_read, created_at)
    VALUES
        ('b2222222-2222-2222-2222-222222222222', 'message', 'New message from Alice', 'Hey Bob! How are you doing?', false, NOW() - interval '2 hours'),
        ('c3333333-3333-3333-3333-333333333333', 'message', 'New message from Alice', 'Charlie, did you see the new design mockups?', false, NOW() - interval '45 minutes'),
        ('d4444444-4444-4444-4444-444444444444', 'message', 'New message from Bob', 'David, can you check the deployment logs?', false, NOW() - interval '30 minutes')
    ON CONFLICT DO NOTHING;
EOF
fi

# Display seeded data statistics
echo ""
echo "[INFO] Database Seeding Statistics:"
PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB <<EOF
SELECT 'Auth Users' as "Table", COUNT(*) as "Count" FROM auth.users WHERE email LIKE '%@example.com'
UNION ALL
SELECT 'User Profiles', COUNT(*) FROM users.profiles WHERE username IN ('alice', 'bob', 'charlie', 'david', 'eve')
UNION ALL
SELECT 'Contacts', COUNT(*) FROM users.contacts WHERE user_id IN (SELECT id FROM auth.users WHERE email LIKE '%@example.com')
UNION ALL
SELECT 'Conversations', COUNT(*) FROM messages.conversations WHERE creator_user_id IN (SELECT id FROM auth.users WHERE email LIKE '%@example.com')
UNION ALL
SELECT 'Messages', COUNT(*) FROM messages.messages WHERE sender_user_id IN (SELECT id FROM auth.users WHERE email LIKE '%@example.com');
EOF

echo ""
echo "[SUCCESS] Database seeding completed successfully!"
echo ""
echo "[INFO] Test Accounts (password: 'password123' for all):"
echo "   ✓ alice@example.com   (Alice Test) - Online"
echo "   ✓ bob@example.com     (Bob Test)   - Online"
echo "   ✓ charlie@example.com (Charlie Test) - Away"
echo "   ✓ david@example.com   (David Test) - Offline"
echo "   ✓ eve@example.com     (Eve Test)   - Busy"
echo ""
echo "Usage:"
echo "  ./seed-data.sh         # Basic seeding (users, profiles, messages)"
echo "  ./seed-data.sh full    # Full seeding (includes sessions, notifications)"
