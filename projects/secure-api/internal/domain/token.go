package domain

// Token is an immutable value object representing an issued access token.
type Token struct {
	accessToken string
	tokenType   string
	expiresIn   int // seconds
}

func NewToken(accessToken, tokenType string, expiresIn int) Token {
	return Token{accessToken: accessToken, tokenType: tokenType, expiresIn: expiresIn}
}

func (t Token) AccessToken() string { return t.accessToken }
func (t Token) TokenType() string   { return t.tokenType }
func (t Token) ExpiresIn() int      { return t.expiresIn }
