package tailscale

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"

	"github.com/gin-gonic/gin"
)

type routingHandlerCase struct {
	name       string
	action     string
	handle     func(*Service, *gin.Context)
	replaceSet func(func(context.Context, bool) error) func()
}

func routingHandlerCases() []routingHandlerCase {
	return []routingHandlerCase{
		{
			name:   "exit-node",
			action: authservice.SensitiveActionTailscaleSetExitNode,
			handle: (*Service).SetExitNode,
			replaceSet: func(replacement func(context.Context, bool) error) func() {
				previous := applyExitNodeSetting
				applyExitNodeSetting = replacement
				return func() { applyExitNodeSetting = previous }
			},
		},
		{
			name:   "subnet-route",
			action: authservice.SensitiveActionTailscaleSetSubnetRoute,
			handle: (*Service).SetSubnetRoute,
			replaceSet: func(replacement func(context.Context, bool) error) func() {
				previous := applySubnetRouteSetting
				applySubnetRouteSetting = replacement
				return func() { applySubnetRouteSetting = previous }
			},
		},
	}
}

func TestRoutingHandlersRejectMissingEnabledBeforeSensitiveAction(t *testing.T) {
	for _, testCase := range routingHandlerCases() {
		t.Run(testCase.name, func(t *testing.T) {
			isolatedMutationCoordinator(t)
			restoreSet := testCase.replaceSet(func(context.Context, bool) error {
				t.Fatal("routing setter called for an invalid request")
				return nil
			})
			t.Cleanup(restoreSet)
			restoreAuth := replaceRoutingAuthForTest(
				t,
				func(string) bool {
					t.Fatal("challenge policy checked before request validation")
					return false
				},
				func(*gin.Context, string, string) error {
					t.Fatal("challenge verified before request validation")
					return nil
				},
			)
			t.Cleanup(restoreAuth)

			response := performRoutingHandlerRequest(t, testCase, `{"challenge":"unused"}`)
			if response.Code != -1 || response.Msg != "invalid arguments" {
				t.Fatalf("invalid request response = %+v", response)
			}
		})
	}
}

func TestRoutingHandlersRequireActionBoundChallenge(t *testing.T) {
	for _, testCase := range routingHandlerCases() {
		t.Run(testCase.name, func(t *testing.T) {
			isolatedMutationCoordinator(t)
			setterCalled := false
			restoreSet := testCase.replaceSet(func(context.Context, bool) error {
				setterCalled = true
				return nil
			})
			t.Cleanup(restoreSet)
			restoreAuth := replaceRoutingAuthForTest(
				t,
				func(action string) bool {
					if action != testCase.action {
						t.Fatalf("challenge policy action = %q, want %q", action, testCase.action)
					}
					return true
				},
				func(_ *gin.Context, challenge, action string) error {
					if challenge != "routing-challenge" || action != testCase.action {
						t.Fatalf("challenge verification = (%q, %q)", challenge, action)
					}
					return errors.New("challenge rejected")
				},
			)
			t.Cleanup(restoreAuth)

			response := performRoutingHandlerRequest(
				t,
				testCase,
				`{"enabled":true,"challenge":"routing-challenge"}`,
			)
			if response.Code != -3 || response.Msg != "sensitive action verification required" {
				t.Fatalf("challenge rejection response = %+v", response)
			}
			if setterCalled {
				t.Fatal("routing setter called after challenge rejection")
			}
		})
	}
}

func TestRoutingHandlersApplyValidatedEnabledValue(t *testing.T) {
	for _, testCase := range routingHandlerCases() {
		t.Run(testCase.name, func(t *testing.T) {
			for _, expectedEnabled := range []bool{false, true} {
				t.Run(strconv.FormatBool(expectedEnabled), func(t *testing.T) {
					isolatedMutationCoordinator(t)
					setterCalled := false
					restoreSet := testCase.replaceSet(func(ctx context.Context, enabled bool) error {
						if ctx == nil || enabled != expectedEnabled {
							t.Fatalf(
								"routing setter received context=%v enabled=%v, want %v",
								ctx,
								enabled,
								expectedEnabled,
							)
						}
						setterCalled = true
						return nil
					})
					t.Cleanup(restoreSet)
					restoreAuth := replaceRoutingAuthForTest(
						t,
						func(string) bool { return true },
						func(_ *gin.Context, challenge, action string) error {
							if challenge != "routing-challenge" || action != testCase.action {
								t.Fatalf("challenge verification = (%q, %q)", challenge, action)
							}
							return nil
						},
					)
					t.Cleanup(restoreAuth)

					response := performRoutingHandlerRequest(
						t,
						testCase,
						`{"enabled":`+strconv.FormatBool(expectedEnabled)+`,"challenge":"routing-challenge"}`,
					)
					if response.Code != 0 {
						t.Fatalf("successful routing response = %+v", response)
					}
					if !setterCalled {
						t.Fatal("validated routing setter was not called")
					}
				})
			}
		})
	}
}

func replaceRoutingAuthForTest(
	t *testing.T,
	requires func(string) bool,
	verify func(*gin.Context, string, string) error,
) func() {
	t.Helper()
	previousRequires := tailscaleActionRequiresChallenge
	previousVerify := verifyTailscaleChallenge
	tailscaleActionRequiresChallenge = requires
	verifyTailscaleChallenge = verify
	return func() {
		tailscaleActionRequiresChallenge = previousRequires
		verifyTailscaleChallenge = previousVerify
	}
}

func performRoutingHandlerRequest(
	t *testing.T,
	testCase routingHandlerCase,
	body string,
) proto.Response {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/extensions/tailscale/"+testCase.name,
		strings.NewReader(body),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")
	testCase.handle(NewService(), ctx)

	var response proto.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode routing handler response: %v", err)
	}
	return response
}
