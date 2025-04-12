package citadel

const sessionCookieName = "citadel_session"

// Describes a user's session that's persisted to their cookie.
type session struct {
	State  string // For SSO
	UserID string
}
