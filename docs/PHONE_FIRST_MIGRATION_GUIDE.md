# Phone-First Authentication Migration Guide

**Migrating Echo Backend from Email-Based to Phone-Based Authentication (WhatsApp-Style)**

---

## Overview

This guide documents the migration from an email-first authentication system to a phone-first authentication system, similar to how WhatsApp operates. The key changes include:

- **Phone number as primary identifier** instead of email
- **SMS OTP-based authentication** as the primary method
- **Optional email and password** for web users
- **Phone contact synchronization** for automatic friend discovery
- **Simplified registration flow** without passwords for mobile users

---

## Database Changes

### 1. Auth Schema Changes (`auth.users`)

#### Before (Email-First)
```sql
CREATE TABLE auth.users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,           -- Required
    phone_number VARCHAR(20) UNIQUE,              -- Optional
    phone_country_code VARCHAR(5),
    password_hash TEXT NOT NULL,                  -- Required
    password_salt TEXT NOT NULL,
    ...
);
```

#### After (Phone-First)
```sql
CREATE TABLE auth.users (
    id UUID PRIMARY KEY,
    phone_number VARCHAR(20) UNIQUE NOT NULL,     -- Required (E.164 format)
    phone_country_code VARCHAR(5) NOT NULL,
    phone_verified BOOLEAN DEFAULT FALSE,
    email VARCHAR(255) UNIQUE,                    -- Optional
    password_hash TEXT,                           -- Optional
    password_salt TEXT,
    has_password BOOLEAN DEFAULT FALSE,           -- Track if password set
    account_status VARCHAR(50) DEFAULT 'pending_verification',
    registration_method VARCHAR(50) DEFAULT 'phone',
    ...
    CONSTRAINT check_phone_format CHECK (phone_number ~ '^\+[1-9]\d{1,14}$')
);
```

**Key Changes:**
- `phone_number` is now `NOT NULL` and the primary identifier
- `email` and `password_hash` are now optional
- Added `phone_verified`, `phone_verified_at` fields
- Added `has_password` flag
- Added `registration_method` to track how user registered
- Account status defaults to `pending_verification`
- Added E.164 format constraint for phone numbers

### 2. OTP Verification Changes (`auth.otp_verifications`)

#### Before
```sql
CREATE TABLE auth.otp_verifications (
    identifier VARCHAR(255) NOT NULL,             -- email or phone
    identifier_type VARCHAR(20) NOT NULL,         -- email, phone
    max_attempts INTEGER DEFAULT 5,
    ...
);
```

#### After
```sql
CREATE TABLE auth.otp_verifications (
    phone_number VARCHAR(20) NOT NULL,            -- Always phone
    country_code VARCHAR(5) NOT NULL,
    max_attempts INTEGER DEFAULT 3,               -- Reduced to 3 (WhatsApp-style)
    purpose VARCHAR(50) NOT NULL,                 -- registration, login, phone_verify, reauth
    device_id VARCHAR(255),
    sent_via VARCHAR(50) DEFAULT 'sms',
    expires_at TIMESTAMPTZ NOT NULL,              -- 10 minutes
    ...
    CONSTRAINT check_otp_phone_format CHECK (phone_number ~ '^\+[1-9]\d{1,14}$')
);
```

**Key Changes:**
- Always phone-based (removed `identifier_type`)
- Reduced max attempts to 3
- Added `device_id` for device tracking
- Changed default expiry to 10 minutes
- Added phone format validation

### 3. User Profile Changes (`users.profiles`)

#### Before
```sql
CREATE TABLE users.profiles (
    username VARCHAR(50) UNIQUE NOT NULL,         -- Required
    ...
);
```

#### After
```sql
CREATE TABLE users.profiles (
    phone_number VARCHAR(20) NOT NULL,            -- Denormalized from auth
    username VARCHAR(50) UNIQUE,                  -- Optional
    display_name VARCHAR(100),                    -- Defaults to phone if not set
    is_business_account BOOLEAN DEFAULT FALSE,
    business_info JSONB,
    ...
);
```

**Key Changes:**
- `username` is now optional (can be set later)
- Added `phone_number` (denormalized for quick lookup)
- Added business account support
- Display name defaults to phone number if not set

### 4. Contacts System Overhaul (`users.contacts`)

#### Before
```sql
CREATE TABLE users.contacts (
    user_id UUID NOT NULL,
    contact_user_id UUID NOT NULL,                -- Required
    ...
);
```

#### After
```sql
CREATE TABLE users.contacts (
    user_id UUID NOT NULL,
    contact_user_id UUID,                         -- NULL if not on platform
    contact_phone_number VARCHAR(20) NOT NULL,    -- From phone book
    contact_name VARCHAR(255),                    -- Name from phone book
    contact_phone_label VARCHAR(50),              -- "Mobile", "Work", etc.
    phone_contact_id VARCHAR(255),                -- ID from phone's contact DB
    last_synced_at TIMESTAMPTZ,
    contact_source VARCHAR(50) DEFAULT 'phone_sync',
    ...
    UNIQUE(user_id, contact_phone_number)
);
```

**Key Changes:**
- `contact_user_id` can be NULL (for contacts not on platform)
- Added phone book sync fields
- Stores contacts even if they're not registered yet
- When contact joins, their `contact_user_id` is populated
- Unique constraint on `(user_id, contact_phone_number)`

### 5. New Tables Added

#### Contact Sync Log
```sql
CREATE TABLE users.contact_sync_log (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    device_id VARCHAR(255),
    total_contacts_synced INTEGER DEFAULT 0,
    new_contacts_found INTEGER DEFAULT 0,
    contacts_updated INTEGER DEFAULT 0,
    sync_status VARCHAR(50) DEFAULT 'completed',
    sync_duration_ms INTEGER,
    synced_at TIMESTAMPTZ DEFAULT NOW()
);
```

#### Contact Invitations
```sql
CREATE TABLE users.contact_invitations (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    phone_number VARCHAR(20) NOT NULL,
    invitation_method VARCHAR(50),                -- sms, whatsapp, link
    invitation_code TEXT UNIQUE,
    status VARCHAR(50) DEFAULT 'pending',
    sent_at TIMESTAMPTZ,
    accepted_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ
);
```

---

## API Changes

### Authentication Endpoints

#### 1. Registration (Before → After)

**Before (Email-First):**
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePassword123!",
  "phone_number": "+1234567890",
  "accept_terms": true
}

Response:
{
  "user_id": "uuid",
  "email": "user@example.com",
  "email_verification_sent": true
}
```

**After (Phone-First):**
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "phone_number": "+1234567890",
  "phone_country_code": "US",
  "device_id": "device_abc123",
  "device_name": "iPhone 15 Pro",
  "accept_terms": true
}

Response:
{
  "user_id": "uuid",
  "phone_number": "+1234567890",
  "otp_sent": true,
  "otp_expires_in": 600,
  "verification_id": "uuid",
  "account_status": "pending_verification",
  "message": "OTP sent to your phone. Please verify to complete registration."
}
```

#### 2. OTP Flow (New)

**Send OTP:**
```http
POST /api/v1/auth/otp/send
Content-Type: application/json

{
  "phone_number": "+1234567890",
  "purpose": "registration",
  "device_id": "device_abc123",
  "method": "sms"
}

Response:
{
  "verification_id": "uuid",
  "phone_number": "+1234567890",
  "masked_phone": "+1******7890",
  "expires_in": 600,
  "method": "sms",
  "message": "OTP sent successfully"
}
```

**Verify OTP:**
```http
POST /api/v1/auth/otp/verify
Content-Type: application/json

{
  "verification_id": "uuid",
  "phone_number": "+1234567890",
  "otp_code": "123456",
  "device_id": "device_abc123"
}

Response:
{
  "verified": true,
  "phone_number": "+1234567890",
  "phone_verified": true,
  "user_id": "uuid",
  "is_new_user": false,
  "requires_setup": false,
  "access_token": "jwt_token",
  "refresh_token": "refresh_token",
  "token_type": "Bearer",
  "expires_at": 1699209600,
  "message": "Phone verified successfully"
}
```

**Resend OTP:**
```http
POST /api/v1/auth/otp/resend
Content-Type: application/json

{
  "verification_id": "uuid",
  "phone_number": "+1234567890",
  "method": "voice"
}
```

#### 3. Login (Before → After)

**Before (Email-First):**
```http
POST /api/v1/auth/login

{
  "email": "user@example.com",
  "password": "SecurePassword123!",
  "fcm_token": "firebase_token"
}
```

**After (Phone-First with OTP):**
```http
POST /api/v1/auth/login

{
  "phone_number": "+1234567890",
  "device_id": "device_abc123",
  "fcm_token": "firebase_token"
}

Response (if OTP required):
{
  "requires_otp": true,
  "verification_id": "uuid",
  "masked_phone": "+1******7890",
  "otp_expires_in": 600
}

// Then verify OTP using /api/v1/auth/otp/verify
```

**After (Phone-First with Password - for web):**
```http
POST /api/v1/auth/login

{
  "phone_number": "+1234567890",
  "password": "SecurePassword123!",
  "device_id": "device_abc123"
}

Response:
{
  "user": { ... },
  "session": {
    "access_token": "jwt_token",
    "refresh_token": "refresh_token",
    "token_type": "Bearer",
    "expires_at": 1699209600
  }
}
```

### New Contact Endpoints

#### Sync Phone Contacts
```http
POST /api/v1/users/contacts/sync
Authorization: Bearer <token>

{
  "device_id": "device_abc123",
  "contacts": [
    {
      "phone_number": "+1234567890",
      "name": "John Doe",
      "label": "Mobile",
      "phone_contact_id": "abc123"
    },
    {
      "phone_number": "+9876543210",
      "name": "Jane Smith",
      "label": "Work"
    }
  ]
}

Response:
{
  "total_synced": 2,
  "new_contacts_found": 1,
  "contacts_updated": 1,
  "registered_contacts": [
    {
      "phone_number": "+1234567890",
      "user_id": "uuid",
      "username": "johndoe",
      "display_name": "John Doe",
      "avatar_url": "...",
      "is_contact": true
    }
  ],
  "unregistered_contacts": [
    {
      "phone_number": "+9876543210",
      "name": "Jane Smith",
      "can_invite": true
    }
  ]
}
```

#### Invite Contact
```http
POST /api/v1/users/contacts/invite
Authorization: Bearer <token>

{
  "phone_number": "+9876543210",
  "method": "sms"
}
```

---

## Migration Steps

### Phase 1: Database Migration

1. **Backup existing database**
   ```bash
   pg_dump echo_db > backup_before_phone_migration.sql
   ```

2. **Run migration script**
   ```sql
   -- 1. Add new columns to auth.users
   ALTER TABLE auth.users 
       ALTER COLUMN email DROP NOT NULL,
       ALTER COLUMN password_hash DROP NOT NULL,
       ALTER COLUMN password_salt DROP NOT NULL,
       ADD COLUMN IF NOT EXISTS phone_verified BOOLEAN DEFAULT FALSE,
       ADD COLUMN IF NOT EXISTS phone_verified_at TIMESTAMPTZ,
       ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ,
       ADD COLUMN IF NOT EXISTS has_password BOOLEAN DEFAULT TRUE,
       ADD COLUMN IF NOT EXISTS registration_method VARCHAR(50) DEFAULT 'email',
       ADD CONSTRAINT check_phone_format CHECK (phone_number IS NULL OR phone_number ~ '^\+[1-9]\d{1,14}$');
   
   -- 2. Update existing users
   UPDATE auth.users SET has_password = TRUE WHERE password_hash IS NOT NULL;
   UPDATE auth.users SET registration_method = 'email' WHERE email IS NOT NULL;
   
   -- 3. For users with phone numbers, mark as verified (migration assumption)
   UPDATE auth.users SET phone_verified = TRUE, phone_verified_at = NOW() 
   WHERE phone_number IS NOT NULL;
   
   -- 4. Update indexes
   DROP INDEX IF EXISTS idx_auth_users_email;
   CREATE INDEX idx_auth_users_phone ON auth.users(phone_number) WHERE deleted_at IS NULL;
   CREATE INDEX idx_auth_users_email ON auth.users(email) WHERE deleted_at IS NULL AND email IS NOT NULL;
   CREATE INDEX idx_auth_users_phone_verified ON auth.users(phone_number, phone_verified) WHERE deleted_at IS NULL;
   
   -- 5. Modify users.profiles
   ALTER TABLE users.profiles
       ADD COLUMN IF NOT EXISTS phone_number VARCHAR(20),
       ALTER COLUMN username DROP NOT NULL;
   
   -- Populate phone_number in profiles from auth.users
   UPDATE users.profiles p
   SET phone_number = u.phone_number
   FROM auth.users u
   WHERE p.user_id = u.id AND u.phone_number IS NOT NULL;
   
   -- 6. Recreate users.contacts table
   -- First backup existing contacts
   CREATE TABLE users.contacts_backup AS SELECT * FROM users.contacts;
   
   -- Drop and recreate with new structure
   DROP TABLE users.contacts;
   -- Create new contacts table (see schema above)
   
   -- 7. Create new tables
   -- CREATE TABLE users.contact_sync_log ...
   -- CREATE TABLE users.contact_invitations ...
   
   -- 8. Update OTP verifications table
   ALTER TABLE auth.otp_verifications
       ADD COLUMN IF NOT EXISTS phone_number VARCHAR(20),
       ADD COLUMN IF NOT EXISTS country_code VARCHAR(5),
       ADD COLUMN IF NOT EXISTS device_id VARCHAR(255),
       ALTER COLUMN max_attempts SET DEFAULT 3;
   ```

### Phase 2: Code Deployment

1. **Deploy updated auth service** with phone-first authentication
2. **Deploy API gateway** with new endpoints
3. **Update mobile apps** to use new registration/login flow
4. **Update web app** to support both phone and email login

### Phase 3: Data Migration for Existing Users

For existing users without phone numbers:
```sql
-- Mark users without phone numbers for phone verification
UPDATE auth.users 
SET account_status = 'requires_phone_verification'
WHERE phone_number IS NULL;
```

On next login, prompt these users to:
1. Add and verify phone number
2. Keep their existing email/password for web access

---

## Client Implementation Changes

### iOS/Android (Swift/Kotlin)

#### Before (Email Registration)
```swift
func register(email: String, password: String) {
    let body = [
        "email": email,
        "password": password
    ]
    // Send POST /api/v1/auth/register
}
```

#### After (Phone Registration with OTP)
```swift
func register(phoneNumber: String) async throws {
    // Step 1: Send OTP
    let otpResponse = try await api.post("/api/v1/auth/register", body: [
        "phone_number": phoneNumber,
        "phone_country_code": getCurrentCountryCode(),
        "device_id": getDeviceID(),
        "device_name": UIDevice.current.name,
        "accept_terms": true
    ])
    
    // Step 2: Show OTP input screen
    let otpCode = await showOTPScreen()
    
    // Step 3: Verify OTP
    let verifyResponse = try await api.post("/api/v1/auth/otp/verify", body: [
        "verification_id": otpResponse.verificationID,
        "phone_number": phoneNumber,
        "otp_code": otpCode,
        "device_id": getDeviceID()
    ])
    
    // Step 4: Store tokens and proceed
    saveTokens(verifyResponse.accessToken, verifyResponse.refreshToken)
    navigateToHome()
}
```

### Phone Contact Sync
```swift
func syncContacts() async throws {
    // Request contact permission
    let contacts = try await CNContactStore().requestAccess()
    
    // Format contacts
    let formattedContacts = contacts.map { contact in
        [
            "phone_number": formatE164(contact.phoneNumbers.first),
            "name": "\(contact.givenName) \(contact.familyName)",
            "phone_contact_id": contact.identifier
        ]
    }
    
    // Sync with backend
    let response = try await api.post("/api/v1/users/contacts/sync", body: [
        "device_id": getDeviceID(),
        "contacts": formattedContacts
    ])
    
    // Update local contact list
    updateRegisteredContacts(response.registeredContacts)
}
```

---

## Testing Checklist

### Registration Flow
- [ ] Can register with phone number only
- [ ] Receives OTP via SMS
- [ ] Can verify OTP within 10 minutes
- [ ] OTP expires after 10 minutes
- [ ] Limited to 3 OTP attempts
- [ ] Can resend OTP (with cooldown)
- [ ] Can request voice call for OTP
- [ ] Account created in `pending_verification` status
- [ ] Account activated after OTP verification

### Login Flow
- [ ] Can login with phone + OTP
- [ ] Can login with phone + password (if set)
- [ ] Receives new session token
- [ ] Push tokens (FCM/APNS) are updated
- [ ] Multiple devices supported

### Contact Sync
- [ ] Can sync phone contacts
- [ ] Matches registered users by phone
- [ ] Stores unregistered contacts
- [ ] Updates when contact joins
- [ ] Can invite unregistered contacts

### Edge Cases
- [ ] User with email only (legacy) can add phone
- [ ] User can have both phone and email
- [ ] Phone format validation (E.164)
- [ ] Duplicate phone number prevention
- [ ] Rate limiting on OTP requests
- [ ] Blocked phone numbers handling

---

## Rollback Plan

If issues occur:

1. **Stop new registrations** using phone-first flow
2. **Re-enable email registration** endpoint
3. **Restore database** from backup
4. **Revert code** to previous version
5. **Investigate issues** before retry

---

## Monitoring & Metrics

Track these metrics post-migration:

- OTP delivery success rate
- OTP verification success rate
- Average time to verify OTP
- Contact sync adoption rate
- Registration conversion rate (phone vs email)
- Login method distribution (OTP vs password)

---

## FAQ

**Q: What happens to existing users with only email?**
A: They can continue using email/password. On next login, we'll prompt them to optionally add and verify a phone number.

**Q: Can users have both email and phone?**
A: Yes! Phone is primary for mobile, email/password can be used for web.

**Q: What if a user changes phone numbers?**
A: They can add a new phone number, verify it via OTP, and set it as primary. The old number is kept in history.

**Q: How are contacts matched?**
A: By phone number in E.164 format. When contacts sync, we match against registered phone numbers in the database.

**Q: What if SMS doesn't arrive?**
A: User can request voice call OTP or use alternative verification methods.

---

**Migration Date:** TBD  
**Version:** 2.0.0  
**Breaking Changes:** Yes (API endpoints changed)  
**Backward Compatibility:** Partial (existing email users supported)
