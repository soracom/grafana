package api

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/usagestats"
	"github.com/grafana/grafana/pkg/login/social"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/services/hooks"
	"github.com/grafana/grafana/pkg/services/licensing"
	"github.com/grafana/grafana/pkg/services/login/authinfoservice"
	authinfostore "github.com/grafana/grafana/pkg/services/login/authinfoservice/database"
	"github.com/grafana/grafana/pkg/services/login/loginservice"
	"github.com/grafana/grafana/pkg/services/quota/quotatest"
	"github.com/grafana/grafana/pkg/services/secrets/fakes"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/services/user/userimpl"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/web"
)

type oauthConnectorForSyncTest struct{ social.SocialConnector }

func (oauthConnectorForSyncTest) IsSignupAllowed() bool { return true }

func TestSyncUserDoesNotMatchOAuthIdentityByEmailByDefault(t *testing.T) {
	for _, allowInsecureEmailLookup := range []bool{false, true} {
		t.Run(fmt.Sprintf("allowInsecureEmailLookup=%t", allowInsecureEmailLookup), func(t *testing.T) {
			cfg := setting.NewCfg()
			cfg.OAuthAllowInsecureEmailLookup = allowInsecureEmailLookup
			hs := &HTTPServer{
				Cfg: cfg,
				Login: loginservice.LoginServiceMock{ExpectedUserFunc: func(cmd *models.UpsertUserCommand) *user.User {
					if allowInsecureEmailLookup {
						require.NotNil(t, cmd.UserLookupParams.Email)
						assert.Equal(t, "spoofable@example.com", *cmd.UserLookupParams.Email)
					} else {
						assert.Nil(t, cmd.UserLookupParams.Email)
					}
					return &user.User{ID: 1}
				}},
			}
			ctx := &models.ReqContext{Context: &web.Context{Req: httptest.NewRequest(http.MethodGet, "/login/azuread", nil)}}
			_, err := hs.SyncUser(ctx, &models.ExternalUserInfo{AuthModule: "oauth_azuread", AuthId: "tenant-identity", Email: "spoofable@example.com"}, oauthConnectorForSyncTest{})
			require.NoError(t, err)
		})
	}
}

func setupOAuthTest(t *testing.T, cfg *setting.Cfg) *web.Mux {
	t.Helper()

	if cfg == nil {
		cfg = setting.NewCfg()
	}
	cfg.ErrTemplateName = "error-template"

	sqlStore := db.InitTestDB(t)

	hs := &HTTPServer{
		Cfg:            cfg,
		License:        &licensing.OSSLicensingService{Cfg: cfg},
		SQLStore:       sqlStore,
		SocialService:  social.ProvideService(cfg, featuremgmt.WithFeatures()),
		HooksService:   hooks.ProvideService(),
		SecretsService: fakes.NewFakeSecretsService(),
	}

	m := web.New()
	m.Use(getContextHandler(t, cfg).Middleware)
	viewPath, err := filepath.Abs("../../public/views")
	require.NoError(t, err)

	m.UseMiddleware(web.Renderer(viewPath, "[[", "]]"))

	m.Get("/login/:name", hs.OAuthLogin)
	return m
}

func TestOAuthLogin_UnknownProvider(t *testing.T) {
	m := setupOAuthTest(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/login/notaprovider", nil)
	recorder := httptest.NewRecorder()

	m.ServeHTTP(recorder, req)
	// expect to be redirected to /login
	assert.Equal(t, http.StatusFound, recorder.Code)
	assert.Equal(t, "/login", recorder.Header().Get("Location"))
}

func TestOAuthLogin_Base(t *testing.T) {
	cfg := setting.NewCfg()
	sec := cfg.Raw.Section("auth.generic_oauth")
	_, err := sec.NewKey("enabled", "true")
	require.NoError(t, err)

	m := setupOAuthTest(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "/login/generic_oauth", nil)
	recorder := httptest.NewRecorder()

	m.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusFound, recorder.Code)

	location := recorder.Header().Get("Location")
	assert.NotEmpty(t, location)

	u, err := url.Parse(location)
	require.NoError(t, err)
	assert.False(t, u.Query().Has("code_challenge"))
	assert.False(t, u.Query().Has("code_challenge_method"))

	resp := recorder.Result()
	require.NoError(t, resp.Body.Close())

	cookies := resp.Cookies()
	var stateCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == OauthStateCookieName {
			stateCookie = c
		}
	}
	require.NotNil(t, stateCookie)

	req = httptest.NewRequest(
		http.MethodGet,
		(&url.URL{
			Path: "/login/generic_oauth",
			RawQuery: url.Values{
				"code":  []string{"helloworld"},
				"state": []string{u.Query().Get("state")},
			}.Encode(),
		}).String(),
		nil,
	)
	req.AddCookie(stateCookie)
	recorder = httptest.NewRecorder()

	m.ServeHTTP(recorder, req)
	// TODO: validate that 'creating a token works'
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "login.OAuthLogin(NewTransportWithCode)")
}

func TestOAuthLogin_UsePKCE(t *testing.T) {
	cfg := setting.NewCfg()
	sec := cfg.Raw.Section("auth.generic_oauth")
	_, err := sec.NewKey("enabled", "true")
	require.NoError(t, err)
	_, err = sec.NewKey("use_pkce", "true")
	require.NoError(t, err)

	m := setupOAuthTest(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "/login/generic_oauth", nil)
	recorder := httptest.NewRecorder()

	m.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusFound, recorder.Code)

	location := recorder.Header().Get("Location")
	assert.NotEmpty(t, location)

	u, err := url.Parse(location)
	require.NoError(t, err)
	assert.True(t, u.Query().Has("code_challenge"))
	assert.Equal(t, "S256", u.Query().Get("code_challenge_method"))

	resp := recorder.Result()
	require.NoError(t, resp.Body.Close())

	var oauthCookie *http.Cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == OauthPKCECookieName {
			oauthCookie = cookie
		}
	}
	require.NotNil(t, oauthCookie)

	shasum := sha256.Sum256([]byte(oauthCookie.Value))
	assert.Equal(
		t,
		u.Query().Get("code_challenge"),
		base64.RawURLEncoding.EncodeToString(shasum[:]),
	)
}

func TestGenericOAuthStableIdentityLogin(t *testing.T) {
	sqlStore := db.InitTestDB(t)
	cfg := setting.NewCfg()
	cfg.OAuthSkipOrgRoleUpdateSync = true
	cfg.Raw.Section("auth.generic_oauth").Key("enabled").SetValue("true")
	cfg.Raw.Section("auth.generic_oauth").Key("allow_sign_up").SetValue("true")
	quota := quotatest.New(false, nil)
	userService, err := userimpl.ProvideService(sqlStore, nil, cfg, nil, nil, quota, nil, featuremgmt.WithFeatures())
	require.NoError(t, err)
	authStore := authinfostore.ProvideAuthInfoStore(sqlStore, fakes.NewFakeSecretsService(), userService)
	authService := authinfoservice.ProvideAuthInfoService(&authinfoservice.OSSUserProtectionImpl{}, authStore, &usagestats.UsageStatsMock{})
	hs := &HTTPServer{Cfg: cfg, Login: loginservice.ProvideService(sqlStore, userService, quota, authService, nil, nil)}
	connector, err := social.ProvideService(cfg, featuremgmt.WithFeatures()).GetConnector("generic_oauth")
	require.NoError(t, err)
	ctx := context.Background()

	loginWithSubject := func(subject, email, username string) (*user.User, error) {
		t.Helper()
		payload, err := json.Marshal(map[string]string{"sub": subject, "email": email, "login": username})
		require.NoError(t, err)
		idToken := "eyJhbGciOiJIUzI1NiJ9." + base64.RawURLEncoding.EncodeToString(payload) + ".test"
		token := (&oauth2.Token{}).WithExtra(map[string]interface{}{"id_token": idToken})
		info, err := connector.UserInfo(http.DefaultClient, token)
		require.NoError(t, err)
		reqCtx := &models.ReqContext{Context: &web.Context{Req: httptest.NewRequest(http.MethodGet, "/login/generic_oauth", nil)}}
		return hs.SyncUser(reqCtx, hs.buildExternalUserInfo(token, info, "generic_oauth"), connector)
	}

	t.Run("new account repeats login by subject even when profile changes", func(t *testing.T) {
		first, err := loginWithSubject("stable-subject", "first@example.com", "first-user")
		require.NoError(t, err)
		second, err := loginWithSubject("stable-subject", "first@example.com", "first-user")
		require.NoError(t, err)
		require.Equal(t, first.ID, second.ID)
		third, err := loginWithSubject("stable-subject", "changed@example.com", "renamed-user")
		require.NoError(t, err)
		require.Equal(t, first.ID, third.ID)
		require.Equal(t, "changed@example.com", third.Email)

		// A different provider subject must not claim this account by email/login.
		_, err = loginWithSubject("different-subject", "changed@example.com", "renamed-user")
		require.Error(t, err)
		_, err = authService.LookupAndUpdate(ctx, &models.GetUserByAuthInfoQuery{AuthModule: "oauth_generic_oauth", AuthId: "different-subject"})
		require.ErrorIs(t, err, user.ErrUserNotFound)
	})

	t.Run("legacy blank identity requires an administrator mapping", func(t *testing.T) {
		legacy, err := sqlStore.CreateUser(ctx, user.CreateUserCommand{Login: "legacy-user", Email: "legacy@example.com"})
		require.NoError(t, err)
		require.NoError(t, authService.SetAuthInfo(ctx, &models.SetAuthInfoCommand{UserId: legacy.ID, AuthModule: "oauth_generic_oauth"}))
		_, err = loginWithSubject("verified-legacy-subject", legacy.Email, legacy.Login)
		require.Error(t, err)
		row := &models.GetAuthInfoQuery{UserId: legacy.ID, AuthModule: "oauth_generic_oauth"}
		require.NoError(t, authService.GetAuthInfo(ctx, row))
		require.Empty(t, row.Result.AuthId)

		// Reproduce the documented, explicit administrator mapping. No login
		// path infers an identity from the existing account's email address.
		require.NoError(t, sqlStore.WithTransactionalDbSession(ctx, func(sess *db.Session) error {
			result, err := sess.Exec("UPDATE user_auth SET auth_id = ? WHERE id = ? AND user_id = ? AND auth_module = ? AND (auth_id = '' OR auth_id IS NULL)", "verified-legacy-subject", row.Result.Id, legacy.ID, "oauth_generic_oauth")
			require.NoError(t, err)
			n, err := result.RowsAffected()
			require.NoError(t, err)
			require.EqualValues(t, 1, n)
			return nil
		}))
		for i := 0; i < 2; i++ {
			loggedIn, err := loginWithSubject("verified-legacy-subject", legacy.Email, legacy.Login)
			require.NoError(t, err)
			require.Equal(t, legacy.ID, loggedIn.ID)
		}
	})

	t.Run("local account is not linked by matching email", func(t *testing.T) {
		local, err := sqlStore.CreateUser(ctx, user.CreateUserCommand{Login: "local-user", Email: "local@example.com"})
		require.NoError(t, err)
		_, err = loginWithSubject("unrelated-provider-subject", local.Email, local.Login)
		require.Error(t, err)
		row := &models.GetAuthInfoQuery{UserId: local.ID, AuthModule: "oauth_generic_oauth"}
		require.ErrorIs(t, authService.GetAuthInfo(ctx, row), user.ErrUserNotFound)
	})

	t.Run("missing subject is rejected before account creation", func(t *testing.T) {
		_, err := loginWithSubject("", "no-subject@example.com", "no-subject")
		require.ErrorContains(t, err, "non-empty sub claim")
		_, err = userService.GetByLogin(ctx, &user.GetUserByLoginQuery{LoginOrEmail: "no-subject"})
		require.ErrorIs(t, err, user.ErrUserNotFound)
	})
}
