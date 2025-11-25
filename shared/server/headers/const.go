package headers

const (
	// ------------ Standard Request Headers ------------
	ContentType   = "Content-Type"
	Authorization = "Authorization"
	Accept        = "Accept"
	UserAgent     = "User-Agent"
	Host          = "Host"
	Referer       = "Referer"
	Origin        = "Origin"

	// ------------ Content Negotiation Headers ------------
	AcceptEncoding  = "Accept-Encoding"
	AcceptLanguage  = "Accept-Language"
	AcceptCharset   = "Accept-Charset"
	ContentEncoding = "Content-Encoding"
	ContentLanguage = "Content-Language"
	ContentLength   = "Content-Length"
	ContentLocation = "Content-Location"

	// ------------ Request ID & Correlation Headers ------------
	XRequestID     = "X-Request-ID"
	XCorrelationID = "X-Correlation-ID"
	XTraceID       = "X-Trace-ID"
	XSpanID        = "X-Span-ID"

	// ------------ Client IP & Forwarding Headers ------------
	XForwardedFor   = "X-Forwarded-For"
	XForwardedHost  = "X-Forwarded-Host"
	XForwardedProto = "X-Forwarded-Proto"
	XRealIP         = "X-Real-IP"
	XCFConnectingIP = "CF-Connecting-IP"
	Forwarded       = "Forwarded"

	// ------------ Cache Control Headers ------------
	CacheControl      = "Cache-Control"
	Pragma            = "Pragma"
	Expires           = "Expires"
	Age               = "Age"
	ETag              = "ETag"
	IfMatch           = "If-Match"
	IfNoneMatch       = "If-None-Match"
	IfModifiedSince   = "If-Modified-Since"
	IfUnmodifiedSince = "If-Unmodified-Since"
	LastModified      = "Last-Modified"
	Vary              = "Vary"

	// ------------ Cache Control Values ------------
	CacheNoStore        = "no-store"
	CacheNoCache        = "no-cache"
	CacheMustRevalidate = "must-revalidate"
	CachePublic         = "public"
	CachePrivate        = "private"
	CacheMaxAge         = "max-age"

	// ------------ Cookie Headers ------------
	Cookie    = "Cookie"
	SetCookie = "Set-Cookie"

	// ------------ CORS Headers ------------
	AccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	AccessControlAllowMethods     = "Access-Control-Allow-Methods"
	AccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	AccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	AccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	AccessControlMaxAge           = "Access-Control-Max-Age"
	AccessControlRequestMethod    = "Access-Control-Request-Method"
	AccessControlRequestHeaders   = "Access-Control-Request-Headers"

	// ------------ Security Headers ------------
	StrictTransportSecurity   = "Strict-Transport-Security"
	ContentSecurityPolicy     = "Content-Security-Policy"
	XContentTypeOptions       = "X-Content-Type-Options"
	XFrameOptions             = "X-Frame-Options"
	XXSSProtection            = "X-XSS-Protection"
	ReferrerPolicy            = "Referrer-Policy"
	PermissionsPolicy         = "Permissions-Policy"
	CrossOriginEmbedderPolicy = "Cross-Origin-Embedder-Policy"
	CrossOriginOpenerPolicy   = "Cross-Origin-Opener-Policy"
	CrossOriginResourcePolicy = "Cross-Origin-Resource-Policy"

	// ------------ Authentication Headers ------------
	WWWAuthenticate    = "WWW-Authenticate"
	ProxyAuthenticate  = "Proxy-Authenticate"
	ProxyAuthorization = "Proxy-Authorization"
	AuthenticationInfo = "Authentication-Info"

	// ------------ Range & Partial Content Headers ------------
	Range        = "Range"
	ContentRange = "Content-Range"
	AcceptRanges = "Accept-Ranges"
	IfRange      = "If-Range"

	// ------------ Redirection Headers ------------
	Location = "Location"
	Refresh  = "Refresh"

	// ------------ Connection Headers ------------
	Connection       = "Connection"
	KeepAlive        = "Keep-Alive"
	TE               = "TE"
	Trailer          = "Trailer"
	TransferEncoding = "Transfer-Encoding"
	Upgrade          = "Upgrade"

	// ------------ Rate Limiting Headers ------------
	XRateLimitLimit     = "X-RateLimit-Limit"
	XRateLimitRemaining = "X-RateLimit-Remaining"
	XRateLimitReset     = "X-RateLimit-Reset"
	RetryAfter          = "Retry-After"

	// ------------ Content Type Values ------------
	ApplicationJSON           = "application/json"
	ApplicationXML            = "application/xml"
	ApplicationFormURLEncoded = "application/x-www-form-urlencoded"
	MultipartFormData         = "multipart/form-data"
	TextPlain                 = "text/plain"
	TextHTML                  = "text/html"
	ApplicationOctetStream    = "application/octet-stream"
	ApplicationPDF            = "application/pdf"

	// ------------ Custom Application Headers ------------
	XAPIKey             = "X-API-Key"
	XAPIVersion         = "X-API-Version"
	XClientID           = "X-Client-ID"
	XClientVersion      = "X-Client-Version"
	XDeviceID           = "X-Device-ID"
	XDeviceName         = "X-Device-Name"
	XDeviceType         = "X-Device-Type"
	XDevicePlatform     = "X-Device-Platform"
	XDeviceOS           = "X-Device-OS"
	XDeviceOSVersion    = "X-Device-OS-Version"
	XDeviceModel        = "X-Device-Model"
	XDeviceManufacturer = "X-Device-Manufacturer"
	XBrowserName        = "X-Browser-Name"
	XBrowserVersion     = "X-Browser-Version"
	XSessionID          = "X-Session-ID"
	XTenantID           = "X-Tenant-ID"
	XUserID             = "X-User-ID"

	// ------------ Response Time & Performance Headers ------------
	XResponseTime = "X-Response-Time"
	ServerTiming  = "Server-Timing"

	// ------------ Warning & Deprecation Headers ------------
	Warning     = "Warning"
	Deprecation = "Deprecation"
	Sunset      = "Sunset"

	// ------------ Server Information Headers ------------
	Server = "Server"
	Date   = "Date"
	Allow  = "Allow"

	// ------------ WebSocket Headers ------------
	SecWebSocketKey        = "Sec-WebSocket-Key"
	SecWebSocketAccept     = "Sec-WebSocket-Accept"
	SecWebSocketVersion    = "Sec-WebSocket-Version"
	SecWebSocketProtocol   = "Sec-WebSocket-Protocol"
	SecWebSocketExtensions = "Sec-WebSocket-Extensions"

	// ------------ Link Headers ------------
	Link = "Link"

	// ------------ Content Disposition Headers ------------
	ContentDisposition = "Content-Disposition"

	// ------------ Conditional Request Headers ------------
	Expect = "Expect"

	// ------------ Request Context Headers ------------
	From = "From"
	Via  = "Via"
)
