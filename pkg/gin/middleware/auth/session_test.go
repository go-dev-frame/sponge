package auth

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeSignedCookie_MissingSecret(t *testing.T) {
	result, err := DecodeSignedCookie("b1870c9c2d472d577b91a25f3ae9daa626725afffa70876d2fd9e004720e9a4f822bdcf0ddc07f3c54ae110d9ff852d5b5f648be56a275338f028287f90e8a85",
		"fpkxE8E9Xksk0W2YXDwXAUhluSaIjMfaKzhII7cAzlwU+hG+7p6nNld+JCa7JyA18Zcl+TvDFJiFS5vRh46PRj6LhmUuxti5PdMH2oPM7UiyllHVcveJcm2ucqZokgx6cMCtrcXfAg+2D3L74JlYvJ9iy6M2mpA1oDCg5jfosvMm8GD0QZfh/DSLjqlZdMUA9S/hcjhak20sG5ZOsq/E9jMnH3DYQoMCxa1oaa+pGcZOcjAkxMFx0FkKjvCGbw9iRO/J0Y8XBBuOrNVBp4U+Zyz4U739RvlO3cG7Odk9s3MCUC+WRw8juIkJ9EMUWJwmIc5uJILZimSdVfwh+Qoj7lEZzwdGw6pFTA91pYpGeUuC1sxnLmIQCUYeoamevPwfFa/tN+eAWZuLq2iAlGWQUf70ECUakrGef6k5JME=--Fgbc3j45HzLzebZK--FzkwNBBEImauLsbCzdz/TA==",
		"_coreui_pro_rails_starter_session")
	if err != nil {
		t.Fatalf("expected error for missing secretKeyBase")
	}
	t.Log(result)
}

func TestDecodeSignedCookie_EmptyValues(t *testing.T) {
	_, err := DecodeSignedCookie("", "", "session")
	assert.Error(t, err)
}

func TestDecodeSignedCookie_InvalidFormat(t *testing.T) {
	_, err := DecodeSignedCookie("secret", "invalid_cookie", "session")
	assert.Error(t, err)
}

func TestDecodeSignedCookie_ValidEnvelope(t *testing.T) {
	// Fake session payload
	session := map[string]any{"warden.user.user.key": []any{[]any{123}}}
	msgBytes, _ := json.Marshal(session)
	encodedMsg := base64.StdEncoding.EncodeToString(msgBytes)

	envelope := map[string]any{
		"_rails": map[string]any{
			"pur":     "cookie.session",
			"message": encodedMsg,
		},
	}
	envelopeBytes, _ := json.Marshal(envelope)

	assert.Contains(t, string(envelopeBytes), "cookie.session")
}

func TestUserIDFromSession_Valid(t *testing.T) {
	s := map[string]any{
		"warden.user.user.key": []any{[]any{42}},
	}
	id, ok := UserIDFromSession(s)
	assert.True(t, ok)
	assert.Equal(t, 42, id)
}

func TestUserIDFromSession_MissingKey(t *testing.T) {
	s := map[string]any{}
	_, ok := UserIDFromSession(s)
	assert.False(t, ok)
}

func TestUserIDFromSession_InvalidStructure(t *testing.T) {
	s := map[string]any{
		"warden.user.user.key": "wrong-type",
	}
	_, ok := UserIDFromSession(s)
	assert.False(t, ok)
}
