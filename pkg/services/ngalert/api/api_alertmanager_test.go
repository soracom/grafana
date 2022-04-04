package api

import (
	"context"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	amv2 "github.com/prometheus/alertmanager/api/v2/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	acMock "github.com/grafana/grafana/pkg/services/accesscontrol/mock"
	apimodels "github.com/grafana/grafana/pkg/services/ngalert/api/tooling/definitions"
	"github.com/grafana/grafana/pkg/services/ngalert/metrics"
	ngmodels "github.com/grafana/grafana/pkg/services/ngalert/models"
	"github.com/grafana/grafana/pkg/services/ngalert/notifier"
	"github.com/grafana/grafana/pkg/services/secrets/fakes"
	secretsManager "github.com/grafana/grafana/pkg/services/secrets/manager"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/util"
	"github.com/grafana/grafana/pkg/web"
)

func TestContextWithTimeoutFromRequest(t *testing.T) {
	t.Run("assert context has default timeout when header is absent", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "https://grafana.net", nil)
		require.NoError(t, err)

		now := time.Now()
		ctx := context.Background()
		ctx, cancelFunc, err := contextWithTimeoutFromRequest(
			ctx,
			req,
			15*time.Second,
			30*time.Second)
		require.NoError(t, err)
		require.NotNil(t, cancelFunc)
		require.NotNil(t, ctx)

		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.True(t, deadline.After(now))
		require.Less(t, deadline.Sub(now).Seconds(), 30.0)
		require.GreaterOrEqual(t, deadline.Sub(now).Seconds(), 15.0)
	})

	t.Run("assert context has timeout in request header", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "https://grafana.net", nil)
		require.NoError(t, err)
		req.Header.Set("Request-Timeout", "5")

		now := time.Now()
		ctx := context.Background()
		ctx, cancelFunc, err := contextWithTimeoutFromRequest(
			ctx,
			req,
			15*time.Second,
			30*time.Second)
		require.NoError(t, err)
		require.NotNil(t, cancelFunc)
		require.NotNil(t, ctx)

		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.True(t, deadline.After(now))
		require.Less(t, deadline.Sub(now).Seconds(), 15.0)
		require.GreaterOrEqual(t, deadline.Sub(now).Seconds(), 5.0)
	})

	t.Run("assert timeout in request header cannot exceed max timeout", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "https://grafana.net", nil)
		require.NoError(t, err)
		req.Header.Set("Request-Timeout", "60")

		ctx := context.Background()
		ctx, cancelFunc, err := contextWithTimeoutFromRequest(
			ctx,
			req,
			15*time.Second,
			30*time.Second)
		require.Error(t, err, "exceeded maximum timeout")
		require.Nil(t, cancelFunc)
		require.Nil(t, ctx)
	})
}

func TestStatusForTestReceivers(t *testing.T) {
	t.Run("assert HTTP 400 Status Bad Request for no receivers", func(t *testing.T) {
		require.Equal(t, http.StatusBadRequest, statusForTestReceivers([]notifier.TestReceiverResult{}))
	})

	t.Run("assert HTTP 400 Bad Request when all invalid receivers", func(t *testing.T) {
		require.Equal(t, http.StatusBadRequest, statusForTestReceivers([]notifier.TestReceiverResult{{
			Name: "test1",
			Configs: []notifier.TestReceiverConfigResult{{
				Name:   "test1",
				UID:    "uid1",
				Status: "failed",
				Error:  notifier.InvalidReceiverError{},
			}},
		}, {
			Name: "test2",
			Configs: []notifier.TestReceiverConfigResult{{
				Name:   "test2",
				UID:    "uid2",
				Status: "failed",
				Error:  notifier.InvalidReceiverError{},
			}},
		}}))
	})

	t.Run("assert HTTP 408 Request Timeout when all receivers timed out", func(t *testing.T) {
		require.Equal(t, http.StatusRequestTimeout, statusForTestReceivers([]notifier.TestReceiverResult{{
			Name: "test1",
			Configs: []notifier.TestReceiverConfigResult{{
				Name:   "test1",
				UID:    "uid1",
				Status: "failed",
				Error:  notifier.ReceiverTimeoutError{},
			}},
		}, {
			Name: "test2",
			Configs: []notifier.TestReceiverConfigResult{{
				Name:   "test2",
				UID:    "uid2",
				Status: "failed",
				Error:  notifier.ReceiverTimeoutError{},
			}},
		}}))
	})

	t.Run("assert 207 Multi Status for different errors", func(t *testing.T) {
		require.Equal(t, http.StatusMultiStatus, statusForTestReceivers([]notifier.TestReceiverResult{{
			Name: "test1",
			Configs: []notifier.TestReceiverConfigResult{{
				Name:   "test1",
				UID:    "uid1",
				Status: "failed",
				Error:  notifier.InvalidReceiverError{},
			}},
		}, {
			Name: "test2",
			Configs: []notifier.TestReceiverConfigResult{{
				Name:   "test2",
				UID:    "uid2",
				Status: "failed",
				Error:  notifier.ReceiverTimeoutError{},
			}},
		}}))
	})
}

func TestAlertmanagerConfig(t *testing.T) {
	sut := createSut(t, nil)

	t.Run("assert 404 Not Found when applying config to nonexistent org", func(t *testing.T) {
		rc := models.ReqContext{
			Context: &web.Context{
				Req: &http.Request{},
			},
			SignedInUser: &models.SignedInUser{
				OrgId: 12,
			},
		}
		request := createAmConfigRequest(t)

		response := sut.RoutePostAlertingConfig(&rc, request)

		require.Equal(t, 404, response.Status())
		require.Contains(t, string(response.Body()), "Alertmanager does not exist for this organization")
	})

	t.Run("assert 202 when config successfully applied", func(t *testing.T) {
		rc := models.ReqContext{
			Context: &web.Context{
				Req: &http.Request{},
			},
			SignedInUser: &models.SignedInUser{
				OrgId: 1,
			},
		}
		request := createAmConfigRequest(t)

		response := sut.RoutePostAlertingConfig(&rc, request)

		require.Equal(t, 202, response.Status())
	})

	t.Run("assert 202 when alertmanager to configure is not ready", func(t *testing.T) {
		sut := createSut(t, nil)
		rc := models.ReqContext{
			Context: &web.Context{
				Req: &http.Request{},
			},
			SignedInUser: &models.SignedInUser{
				OrgId: 3, // Org 3 was initialized with broken config.
			},
		}
		request := createAmConfigRequest(t)

		response := sut.RoutePostAlertingConfig(&rc, request)

		require.Equal(t, 202, response.Status())
	})
}

func TestRouteCreateSilence(t *testing.T) {
	tesCases := []struct {
		name           string
		silence        func() apimodels.PostableSilence
		accessControl  func() accesscontrol.AccessControl
		role           models.RoleType
		expectedStatus int
	}{
		{
			name:    "new silence, fine-grained access control is enabled, not authorized",
			silence: silenceGen(withEmptyID),
			accessControl: func() accesscontrol.AccessControl {
				return acMock.New()
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:    "new silence, fine-grained access control is enabled, authorized",
			silence: silenceGen(withEmptyID),
			accessControl: func() accesscontrol.AccessControl {
				return acMock.New().WithPermissions([]*accesscontrol.Permission{
					{Action: accesscontrol.ActionAlertingInstanceCreate},
				})
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:    "new silence, fine-grained access control is disabled, Viewer",
			silence: silenceGen(withEmptyID),
			accessControl: func() accesscontrol.AccessControl {
				return acMock.New().WithDisabled()
			},
			role:           models.ROLE_VIEWER,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:    "new silence, fine-grained access control is disabled, Editor",
			silence: silenceGen(withEmptyID),
			accessControl: func() accesscontrol.AccessControl {
				return acMock.New().WithDisabled()
			},
			role:           models.ROLE_EDITOR,
			expectedStatus: http.StatusAccepted,
		},
		{
			name:    "new silence, fine-grained access control is disabled, Admin",
			silence: silenceGen(withEmptyID),
			accessControl: func() accesscontrol.AccessControl {
				return acMock.New().WithDisabled()
			},
			role:           models.ROLE_ADMIN,
			expectedStatus: http.StatusAccepted,
		},
		{
			name:    "update silence, fine-grained access control is enabled, not authorized",
			silence: silenceGen(),
			accessControl: func() accesscontrol.AccessControl {
				return acMock.New()
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:    "update silence, fine-grained access control is enabled, authorized",
			silence: silenceGen(),
			accessControl: func() accesscontrol.AccessControl {
				return acMock.New().WithPermissions([]*accesscontrol.Permission{
					{Action: accesscontrol.ActionAlertingInstanceUpdate},
				})
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:    "update silence, fine-grained access control is disabled, Viewer",
			silence: silenceGen(),
			accessControl: func() accesscontrol.AccessControl {
				return acMock.New().WithDisabled()
			},
			role:           models.ROLE_VIEWER,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:    "update silence, fine-grained access control is disabled, Editor",
			silence: silenceGen(),
			accessControl: func() accesscontrol.AccessControl {
				return acMock.New().WithDisabled()
			},
			role:           models.ROLE_EDITOR,
			expectedStatus: http.StatusAccepted,
		},
		{
			name:    "update silence, fine-grained access control is disabled, Admin",
			silence: silenceGen(),
			accessControl: func() accesscontrol.AccessControl {
				return acMock.New().WithDisabled()
			},
			role:           models.ROLE_ADMIN,
			expectedStatus: http.StatusAccepted,
		},
	}

	for _, tesCase := range tesCases {
		t.Run(tesCase.name, func(t *testing.T) {
			ac := tesCase.accessControl()
			sut := createSut(t, ac)

			rc := models.ReqContext{
				Context: &web.Context{
					Req: &http.Request{},
				},
				SignedInUser: &models.SignedInUser{
					OrgRole: tesCase.role,
					OrgId:   1,
				},
			}

			silence := tesCase.silence()

			if silence.ID != "" {
				alertmanagerFor, err := sut.mam.AlertmanagerFor(1)
				require.NoError(t, err)
				silence.ID = ""
				newID, err := alertmanagerFor.CreateSilence(&silence)
				require.NoError(t, err)
				silence.ID = newID
			}

			response := sut.RouteCreateSilence(&rc, silence)
			require.Equal(t, tesCase.expectedStatus, response.Status())
		})
	}
}

func createSut(t *testing.T, accessControl accesscontrol.AccessControl) AlertmanagerSrv {
	t.Helper()

	mam := createMultiOrgAlertmanager(t)
	store := newFakeAlertingStore(t)
	store.Setup(1)
	store.Setup(2)
	store.Setup(3)
	secrets := fakes.NewFakeSecretsService()
	if accessControl == nil {
		accessControl = acMock.New().WithDisabled()
	}
	return AlertmanagerSrv{mam: mam, store: store, secrets: secrets, ac: accessControl}
}

func createAmConfigRequest(t *testing.T) apimodels.PostableUserConfig {
	t.Helper()

	request := apimodels.PostableUserConfig{}
	err := request.UnmarshalJSON([]byte(validConfig))
	require.NoError(t, err)

	return request
}

func createMultiOrgAlertmanager(t *testing.T) *notifier.MultiOrgAlertmanager {
	t.Helper()

	configs := map[int64]*ngmodels.AlertConfiguration{
		1: {AlertmanagerConfiguration: validConfig, OrgID: 1},
		2: {AlertmanagerConfiguration: validConfig, OrgID: 2},
		3: {AlertmanagerConfiguration: brokenConfig, OrgID: 3},
	}
	configStore := notifier.NewFakeConfigStore(t, configs)
	orgStore := notifier.NewFakeOrgStore(t, []int64{1, 2, 3})
	tmpDir := t.TempDir()
	kvStore := notifier.NewFakeKVStore(t)
	secretsService := secretsManager.SetupTestService(t, fakes.NewFakeSecretsStore())
	reg := prometheus.NewPedanticRegistry()
	m := metrics.NewNGAlert(reg)
	decryptFn := secretsService.GetDecryptedValue
	cfg := &setting.Cfg{
		DataPath: tmpDir,
		UnifiedAlerting: setting.UnifiedAlertingSettings{
			AlertmanagerConfigPollInterval: 3 * time.Minute,
			DefaultConfiguration:           setting.GetAlertmanagerDefaultConfiguration(),
			DisabledOrgs:                   map[int64]struct{}{5: {}},
		}, // do not poll in tests.
	}

	mam, err := notifier.NewMultiOrgAlertmanager(cfg, &configStore, &orgStore, kvStore, decryptFn, m.GetMultiOrgAlertmanagerMetrics(), nil, log.New("testlogger"))
	require.NoError(t, err)
	err = mam.LoadAndSyncAlertmanagersForOrgs(context.Background())
	require.NoError(t, err)
	return mam
}

var validConfig = setting.GetAlertmanagerDefaultConfiguration()

var brokenConfig = `
	"alertmanager_config": {
		"route": {
			"receiver": "grafana-default-email"
		},
		"receivers": [{
			"name": "grafana-default-email",
			"grafana_managed_receiver_configs": [{
				"uid": "abc",
				"name": "default-email",
				"type": "email",
				"isDefault": true,
				"settings": {}
			}]
		}]
	}
}`

func silenceGen(mutatorFuncs ...func(*apimodels.PostableSilence)) func() apimodels.PostableSilence {
	return func() apimodels.PostableSilence {
		testString := util.GenerateShortUID()
		isEqual := rand.Int()%2 == 0
		isRegex := rand.Int()%2 == 0
		value := util.GenerateShortUID()
		if isRegex {
			value = ".*" + util.GenerateShortUID()
		}

		matchers := amv2.Matchers{&amv2.Matcher{Name: &testString, IsEqual: &isEqual, IsRegex: &isRegex, Value: &value}}
		comment := util.GenerateShortUID()
		starts := strfmt.DateTime(timeNow().Add(-time.Duration(rand.Int63n(9)+1) * time.Second))
		ends := strfmt.DateTime(timeNow().Add(time.Duration(rand.Int63n(9)+1) * time.Second))
		createdBy := "User-" + util.GenerateShortUID()
		s := apimodels.PostableSilence{
			ID: util.GenerateShortUID(),
			Silence: amv2.Silence{
				Comment:   &comment,
				CreatedBy: &createdBy,
				EndsAt:    &ends,
				Matchers:  matchers,
				StartsAt:  &starts,
			},
		}

		for _, mutator := range mutatorFuncs {
			mutator(&s)
		}

		return s
	}
}

func withEmptyID(silence *apimodels.PostableSilence) {
	silence.ID = ""
}
