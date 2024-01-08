package psygo

import (
	"crypto/subtle"
	"encoding/base64"
	"github.com/Psychopath-H/psyweb-master/psygo/internal/bytesconv"
	"net/http"
	"strconv"
)

// AuthUserKey is the cookie name for user credential in basic auth.
const AuthUserKey = "user"

// Accounts defines a key/value for user/pass list of authorized logins.
type Accounts map[string]string

type authPair struct {
	value string
	user  string
}

type authPairs []authPair

func (a authPairs) searchCredential(authValue string) (string, bool) {
	if authValue == "" {
		return "", false
	}
	for _, pair := range a {
		//在Go中（和大多数语言一样），普通的==比较运算符一旦发现两个字符串之间有差异，就会立即返回。
		//因此，如果第一个字符是不同的，它将在只看一个字符后返回。从理论上讲，这为定时攻击提供了机会，攻击者可以向你的应用程序发出大量请求，并查看平均响应时间的差异。
		//他们收到401响应所需的时间可以有效地告诉他们有多少字符是正确的，如果有足够的请求，他们可以建立一个完整的用户名和密码的画像。
		//像网络抖动这样的事情使得这种特定的攻击很难实现，但远程定时攻击已经成为现实，而且在未来可能变得更加可行。
		//考虑到这个因素我们可以通过使用subtle.ConstantTimeCompare()很容易地防范这种风险，这样做是有意义的。
		if subtle.ConstantTimeCompare(bytesconv.StringToBytes(pair.value), bytesconv.StringToBytes(authValue)) == 1 {
			return pair.user, true
		}
	}
	return "", false
}

// BasicAuthForRealm returns a Basic HTTP Authorization middleware. It takes as arguments a map[string]string where
// the key is the user name and the value is the password, as well as the name of the Realm.
// If the realm is empty, "Authorization Required" will be used by default.
// (see http://tools.ietf.org/html/rfc2617#section-1.2)
func BasicAuthForRealm(accounts Accounts, realm string) HandlerFunc {
	if realm == "" {
		realm = "Authorization Required"
	}
	realm = "Basic realm=" + strconv.Quote(realm) // strconv.Quote()把字符串转成带双引号的字符串
	pairs := processAccounts(accounts)
	return func(c *Context) {
		// Search user in the slice of allowed credentials
		user, found := pairs.searchCredential(c.requestHeader("Authorization"))
		if !found {
			// Credentials doesn't match, we return 401 and abort handlers chain.
			c.Header("WWW-Authenticate", realm)
			c.Writer.WriteHeader(http.StatusUnauthorized)
			return
		}

		// The user credentials was found, set user's id to key AuthUserKey in this context, the user's id can be read later using
		// c.Get(psygo.AuthUserKey).
		c.Set(AuthUserKey, user)
		c.Next()
	}
}

// BasicAuth returns a Basic HTTP Authorization middleware. It takes as argument a map[string]string where
// the key is the user name and the value is the password.
func BasicAuth(accounts Accounts) HandlerFunc {
	return BasicAuthForRealm(accounts, "")
}

// processAccount 把传入的Accounts加工成authPairs
func processAccounts(accounts Accounts) authPairs {
	length := len(accounts)
	assert1(length > 0, "Empty list of authorized credentials")
	pairs := make(authPairs, 0, length)
	for user, password := range accounts {
		assert1(user != "", "User can not be empty")
		value := authorizationHeader(user, password)
		pairs = append(pairs, authPair{
			value: value,
			user:  user,
		})
	}
	return pairs
}

// authorizationHeader 把user和password组合起来用base64编码一下
func authorizationHeader(user, password string) string {
	base := user + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString(bytesconv.StringToBytes(base))
}
