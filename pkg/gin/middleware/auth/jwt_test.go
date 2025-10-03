package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/go-dev-frame/sponge/pkg/gin/response"
	"github.com/go-dev-frame/sponge/pkg/httpcli"
	"github.com/go-dev-frame/sponge/pkg/jwt"
	"github.com/go-dev-frame/sponge/pkg/utils"
)

var (
	uid    = "100"
	fields = map[string]interface{}{
		"name":   "bob",
		"age":    10,
		"is_vip": true,
	}
	jwtSignKey = []byte("your-secret-key")

	errMsg       = http.StatusText(http.StatusUnauthorized)
	compareMsgFn = func(em string) bool {
		return strings.Contains(em, errMsg)
	}
)

func extraVerifyFn(claims *jwt.Claims, c *gin.Context) error {
	// check if token is about to expire (less than 10 minutes remaining)
	if time.Now().Unix()-claims.ExpiresAt.Unix() < int64(time.Minute*10) {
		token, err := RefreshToken(claims)
		if err != nil {
			return err
		}
		c.Header("X-Renewed-Token", token)
	}

	// judge whether the user is disabled, query whether jwt id exists from the blacklist
	//if CheckBlackList(uid, claims.ID) {
	//	return errors.New("user is disabled")
	//}

	// check fields
	if claims.UID != uid {
		return fmt.Errorf("uid not match, expect %s, got %s", uid, claims.UID)
	}
	if name, _ := claims.GetString("name"); name != fields["name"] {
		return fmt.Errorf("name not match, expect %s, got %s", fields["name"], name)
	}
	if age, _ := claims.GetInt("age"); age != fields["age"] {
		return fmt.Errorf("age not match, expect %d, got %d", fields["age"], age)
	}
	if isVip, _ := claims.GetBool("is_vip"); isVip != fields["is_vip"] {
		return fmt.Errorf("is_vip not match, expect %v, got %v", fields["is_vip"], isVip)
	}

	return nil
}

func runAuthHTTPServer() string {
	serverAddr, requestAddr := utils.GetLocalHTTPAddrPairs()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	loginHandler := func(c *gin.Context) {
		token, _ := GenerateToken(uid)
		fmt.Println("token1 =", token)
		response.Success(c, token)
	}

	loginCustomFieldsHandler := func(c *gin.Context) {
		token, _ := GenerateToken(uid, WithGenerateTokenFields(fields))
		fmt.Println("token2 =", token)
		response.Success(c, token)
	}

	getUserByIDHandler := func(c *gin.Context) {
		id := c.Param("id")
		claims, ok := GetClaims(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"msg": "unauthorized"})
			return
		}
		fmt.Println("claims =", claims)
		response.Success(c, id)
	}

	r.GET("/auth/login", loginHandler)
	r.GET("/auth/loginCustomFields", loginCustomFieldsHandler)
	r.GET("/user/:id", Auth(), getUserByIDHandler)
	r.GET("/user/log/:id", Auth(WithReturnErrReason()), getUserByIDHandler)
	r.GET("/user/extra_verify/:id", Auth(WithExtraVerify(extraVerifyFn), WithReturnErrReason()), getUserByIDHandler)

	go func() {
		err := r.Run(serverAddr)
		if err != nil {
			panic(err)
		}
	}()

	time.Sleep(time.Millisecond * 200)
	return requestAddr
}

func getUser(url string, authorization string) (gin.H, error) {
	var result = gin.H{}

	client := &http.Client{}
	request, err := http.NewRequest("GET", url, nil)
	request.Header.Add("Authorization", authorization)
	if err != nil {
		return result, err
	}
	resp, _ := client.Do(request)
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(data, &result)

	return result, err
}

func TestAuth(t *testing.T) {
	requestAddr := runAuthHTTPServer()
	InitAuth(jwtSignKey, time.Minute*10)
	//InitAuth(jwtSignKey, time.Minute*10, WithInitAuthSigningMethod(HS512), WithInitAuthIssuer("foobar.com"))

	t.Run("only uid for generate token", func(t *testing.T) {
		// get token
		result := &httpcli.StdResult{}
		err := httpcli.Get(result, requestAddr+"/auth/login")
		if err != nil {
			t.Fatal(err)
		}
		token := result.Data.(string)
		authorization := fmt.Sprintf("Bearer %s", token)

		// success
		val, err := getUser(requestAddr+"/user/"+uid, authorization)
		assert.Equal(t, val["data"], uid)

		// success
		val, err = getUser(requestAddr+"/user/log/"+uid, authorization)
		assert.Equal(t, val["data"], uid)

		// return 401, the reason is token have no extra field
		val, err = getUser(requestAddr+"/user/extra_verify/"+uid, authorization)
		assert.Equal(t, true, compareMsgFn(val["msg"].(string)))

		// return 401, the reason is token value is invalid
		val, err = getUser(requestAddr+"/user/"+uid, "error-authorization")
		assert.Equal(t, val["msg"], errMsg)
	})

	t.Run("uid and fields for generate token", func(t *testing.T) {
		// get token
		result := &httpcli.StdResult{}
		err := httpcli.Get(result, requestAddr+"/auth/loginCustomFields")
		if err != nil {
			t.Fatal(err)
		}
		token := result.Data.(string)
		authorization := fmt.Sprintf("Bearer %s", token)

		// success
		val, err := getUser(requestAddr+"/user/"+uid, authorization)
		assert.Equal(t, val["data"], uid)

		// success
		val, err = getUser(requestAddr+"/user/log/"+uid, authorization)
		assert.Equal(t, val["data"], uid)

		// return 401, the reason is token expired
		token = "eyJhbGciOiJIUzM4NCIsInR5cCI6IkpXVCJ9.eyJ1aWQiOiIxMDAiLCJmaWVsZHMiOnsiYWdlIjoxMCwiaXNfdmlwIjp0cnVlLCJuYW1lIjoiYm9iIn0sImV4cCI6MTc0NjY0MTY0MCwiaWF0IjoxNzQ2NjQxMDQwLCJqdGkiOiIxODNkNTBjNWIxZTdmMTEwIn0.P11q5VPo-88Sbw4JKLtp2_Aiz8Pc1oL-jrdEAX0NwJJoxnR_Iu8W6eI7CsUCzVGW"
		authorization = fmt.Sprintf("Bearer %s", token)
		val, err = getUser(requestAddr+"/user/extra_verify/"+uid, authorization)
		assert.Equal(t, true, compareMsgFn(val["msg"].(string)))

		// return 401, the reason is token value is invalid
		val, err = getUser(requestAddr+"/user/"+uid, "error-authorization")
		assert.Equal(t, val["msg"], errMsg)
	})
}

func TestError(t *testing.T) {
	t.Run("GenerateToken error", func(t *testing.T) {
		defer func() { recover() }()
		GenerateToken("100")
	})
	t.Run("ParseToken error", func(t *testing.T) {
		defer func() { recover() }()
		ParseToken("xxx")
	})
}
