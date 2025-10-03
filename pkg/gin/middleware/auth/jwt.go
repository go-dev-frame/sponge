// Package auth provides JWT authentication middleware for gin.
package auth

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/go-dev-frame/sponge/pkg/errcode"
	"github.com/go-dev-frame/sponge/pkg/gin/response"
	"github.com/go-dev-frame/sponge/pkg/jwt"
)

type SigningMethodHMAC = jwt.SigningMethodHMAC
type Claims = jwt.Claims

var (
	HS256 = jwt.HS256
	HS384 = jwt.HS384
	HS512 = jwt.HS512
)

var (
	customSigningKey    []byte
	customSigningMethod *jwt.SigningMethodHMAC
	customExpire        time.Duration
	customIssuer        string

	errOption = errors.New("jwt option is nil, please initialize first, call middleware.InitAuth()")
)

type initAuthOptions struct {
	issuer        string
	signingMethod *SigningMethodHMAC
}

func defaultInitAuthOptions() *initAuthOptions {
	return &initAuthOptions{
		signingMethod: HS256,
	}
}

// InitAuthOption set the jwt initAuthOptions.
type InitAuthOption func(*initAuthOptions)

func (o *initAuthOptions) apply(opts ...InitAuthOption) {
	for _, opt := range opts {
		opt(o)
	}
}

// WithInitAuthSigningMethod set signing method value
func WithInitAuthSigningMethod(sm *jwt.SigningMethodHMAC) InitAuthOption {
	return func(o *initAuthOptions) {
		o.signingMethod = sm
	}
}

// WithInitAuthIssuer set issuer value
func WithInitAuthIssuer(issuer string) InitAuthOption {
	return func(o *initAuthOptions) {
		o.issuer = issuer
	}
}

// InitAuth initializes jwt options.
func InitAuth(signingKey []byte, expire time.Duration, opts ...InitAuthOption) {
	o := defaultInitAuthOptions()
	o.apply(opts...)

	customSigningKey = signingKey
	customExpire = expire
	customSigningMethod = o.signingMethod
	customIssuer = o.issuer
}

// GenerateTokenOption set the jwt options.
type GenerateTokenOption func(*generateTokenOptions)

type generateTokenOptions struct {
	fields map[string]interface{}
}

func (o *generateTokenOptions) apply(opts ...GenerateTokenOption) {
	for _, opt := range opts {
		opt(o)
	}
}

// WithGenerateTokenFields set custom fields value
func WithGenerateTokenFields(fields map[string]interface{}) GenerateTokenOption {
	return func(o *generateTokenOptions) {
		o.fields = fields
	}
}

// GenerateToken generates a jwt token with the given uid and options.
func GenerateToken(uid string, opts ...GenerateTokenOption) (string, error) {
	if customSigningMethod == nil || len(customSigningKey) == 0 {
		panic(errOption)
	}

	genOpts := []jwt.GenerateTokenOption{
		jwt.WithGenerateTokenSignKey(customSigningKey),
		jwt.WithGenerateTokenSignMethod(customSigningMethod),
	}
	o := &generateTokenOptions{}
	o.apply(opts...)
	if len(o.fields) > 0 {
		genOpts = append(genOpts, jwt.WithGenerateTokenFields(o.fields))
	}

	claimsOpts := []jwt.RegisteredClaimsOption{
		jwt.WithExpires(customExpire),
	}
	if customIssuer != "" {
		claimsOpts = append(claimsOpts, jwt.WithIssuer(customIssuer))
	}
	genOpts = append(genOpts, jwt.WithGenerateTokenClaims(claimsOpts...))

	_, token, err := jwt.GenerateToken(uid, genOpts...)
	return token, err
}

// ParseToken parses the given token and returns the claims.
func ParseToken(token string) (*jwt.Claims, error) {
	if customSigningMethod == nil {
		panic(errOption)
	}

	return jwt.ValidateToken(token, jwt.WithValidateTokenSignKey(customSigningKey))
}

// RefreshToken create a new token with the given claims.
func RefreshToken(claims *jwt.Claims) (string, error) {
	return claims.NewToken(customExpire, customSigningMethod, customSigningKey)
}

// -------------------------------------------------------------------------------------------

// HeaderAuthorizationKey http header authorization key, value is "Bearer token"
const HeaderAuthorizationKey = "Authorization"

// ExtraVerifyFn extra verify function
type ExtraVerifyFn = func(claims *jwt.Claims, c *gin.Context) error

// AuthOption set the auth options.
type AuthOption func(*authOptions)

type authOptions struct {
	isReturnErrReason bool
	extraVerifyFn     ExtraVerifyFn
}

func defaultAuthOptions() *authOptions {
	return &authOptions{}
}

func (o *authOptions) apply(opts ...AuthOption) {
	for _, opt := range opts {
		opt(o)
	}
}

// WithReturnErrReason set return error reason
func WithReturnErrReason() AuthOption {
	return func(o *authOptions) {
		o.isReturnErrReason = true
	}
}

// WithExtraVerify set extra verify function
func WithExtraVerify(fn ExtraVerifyFn) AuthOption {
	return func(o *authOptions) {
		o.extraVerifyFn = fn
	}
}

func responseUnauthorized(isReturnErrReason bool, errMsg string) *errcode.Error {
	if isReturnErrReason {
		return errcode.Unauthorized.RewriteMsg("Unauthorized, " + errMsg)
	}
	return errcode.Unauthorized
}

// Auth authorization middleware, support custom extra verify.
func Auth(opts ...AuthOption) gin.HandlerFunc {
	o := defaultAuthOptions()
	o.apply(opts...)

	return func(c *gin.Context) {
		authorization := c.GetHeader(HeaderAuthorizationKey)
		if len(authorization) < 100 {
			response.Out(c, responseUnauthorized(o.isReturnErrReason, "token is illegal"))
			c.Abort()
			return
		}

		tokenString := authorization[7:] // remove Bearer prefix

		claims, err := ParseToken(tokenString)
		if err != nil {
			response.Out(c, responseUnauthorized(o.isReturnErrReason, err.Error()))
			c.Abort()
			return
		}
		// extra verify function
		if o.extraVerifyFn != nil {
			if err = o.extraVerifyFn(claims, c); err != nil {
				response.Out(c, responseUnauthorized(o.isReturnErrReason, err.Error()))
				c.Abort()
				return
			}
		}
		c.Set("claims", claims) // set claims to context
		c.Next()
	}
}

// GetClaims get jwt claims from gin context.
func GetClaims(c *gin.Context) (*jwt.Claims, bool) {
	claims, exists := c.Get("claims")
	if !exists {
		return nil, false
	}
	jwtClaims, ok := claims.(*jwt.Claims)
	return jwtClaims, ok
}
