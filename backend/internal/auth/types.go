package auth

type User struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	// MarketInterest is Signup.dc.html's "Thị trường bạn quan tâm"
	// choice -- stored so the control isn't decorative, not yet read by
	// anything (crypto lands in Phase I). See phase-g.md decision 7.
	MarketInterest string `json:"marketInterest,omitempty"`
}

type registerRequest struct {
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required,min=8"`
	DisplayName    string `json:"displayName" binding:"required"`
	MarketInterest string `json:"marketInterest" binding:"omitempty,oneof=stock crypto both"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type authResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int64  `json:"expiresIn"`
	User        User   `json:"user"`
}
