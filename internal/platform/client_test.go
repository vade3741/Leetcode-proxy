package platform

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vade3741/Leetcode-proxy/internal/model"
)

func TestMockedCandidateProfileSuccess(t *testing.T) {
	mockPayload := model.GraphQLResponseEnvelope{
		Data: json.RawMessage(`{
			"matchedUser": {
				"username": "candidate_alex",
				"profile": {
					"realName": "Alex Merchant",
					"ranking": 3500,
					"reputation": 220
				},
				"submitStatsGlobal": {
					"acSubmissionNum": [
						{"difficulty": "All", "count": 520, "submissions": 780},
						{"difficulty": "Easy", "count": 210, "submissions": 250},
						{"difficulty": "Medium", "count": 240, "submissions": 390},
						{"difficulty": "Hard", "count": 70, "submissions": 140}
					]
				}
			}
		}`),
	}

	mockHTTPServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, incomingRequest *http.Request) {
		responseWriter.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(responseWriter).Encode(mockPayload)
	}))
	defer mockHTTPServer.Close()

	clientInstance := NewGatewayClient(mockHTTPServer.URL, 2*time.Second)
	candidateRecord, lookupError := clientInstance.FetchCandidateProfile(context.Background(), "candidate_alex")

	if lookupError != nil {
		t.Fatalf("unexpected error fetching candidate profile: %v", lookupError)
	}

	if candidateRecord.Username != "candidate_alex" || candidateRecord.FullName != "Alex Merchant" {
		t.Errorf("expected candidate_alex (Alex Merchant), got %s (%s)", candidateRecord.Username, candidateRecord.FullName)
	}

	if candidateRecord.GlobalRanking != 3500 {
		t.Errorf("expected global ranking 3500, got %d", candidateRecord.GlobalRanking)
	}

	if candidateRecord.SubmissionBreakdown["all"].SolvedQuantity != 520 {
		t.Errorf("expected 520 solved problems, got %d", candidateRecord.SubmissionBreakdown["all"].SolvedQuantity)
	}
}

func TestMockedCandidateNotFound(t *testing.T) {
	mockPayload := model.GraphQLResponseEnvelope{
		Data: json.RawMessage(`{"matchedUser": null}`),
	}

	mockHTTPServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, incomingRequest *http.Request) {
		responseWriter.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(responseWriter).Encode(mockPayload)
	}))
	defer mockHTTPServer.Close()

	clientInstance := NewGatewayClient(mockHTTPServer.URL, 2*time.Second)
	_, lookupError := clientInstance.FetchCandidateProfile(context.Background(), "unknown_candidate_identifier")

	if !errors.Is(lookupError, ErrResourceNotFound) {
		t.Fatalf("expected ErrResourceNotFound, received %v", lookupError)
	}
}

func TestMockedUpstreamGraphQLError(t *testing.T) {
	mockPayload := model.GraphQLResponseEnvelope{
		Errors: []model.GraphQLErrorDetail{
			{Message: "Cannot resolve field 'nonExistentField' on Query"},
		},
	}

	mockHTTPServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, incomingRequest *http.Request) {
		responseWriter.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(responseWriter).Encode(mockPayload)
	}))
	defer mockHTTPServer.Close()

	clientInstance := NewGatewayClient(mockHTTPServer.URL, 2*time.Second)
	_, operationalError := clientInstance.FetchDailyChallenge(context.Background())

	if !errors.Is(operationalError, ErrGatewayFailure) {
		t.Fatalf("expected ErrGatewayFailure, received %v", operationalError)
	}
}

func TestMockedUpstreamBadGateway(t *testing.T) {
	mockHTTPServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, incomingRequest *http.Request) {
		http.Error(responseWriter, "Gateway Outage", http.StatusBadGateway)
	}))
	defer mockHTTPServer.Close()

	clientInstance := NewGatewayClient(mockHTTPServer.URL, 2*time.Second)
	_, operationalError := clientInstance.FetchDailyChallenge(context.Background())

	if !errors.Is(operationalError, ErrGatewayFailure) {
		t.Fatalf("expected ErrGatewayFailure on HTTP 502, received %v", operationalError)
	}
}

func TestContextCancellationEnforcement(t *testing.T) {
	mockHTTPServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, incomingRequest *http.Request) {
		time.Sleep(100 * time.Millisecond)
		responseWriter.WriteHeader(http.StatusOK)
	}))
	defer mockHTTPServer.Close()

	clientInstance := NewGatewayClient(mockHTTPServer.URL, 5*time.Second)
	cancellationContext, cancelExecution := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancelExecution()

	_, executionError := clientInstance.FetchDailyChallenge(cancellationContext)
	if executionError == nil {
		t.Fatal("expected context deadline error, received nil")
	}
}

func TestHTTPRouterDailyEndpoint(t *testing.T) {
	mockPayload := model.GraphQLResponseEnvelope{
		Data: json.RawMessage(`{
			"activeDailyCodingChallengeQuestion": {
				"date": "2026-10-04",
				"link": "/problems/invert-binary-tree/",
				"question": {
					"questionFrontendId": "226",
					"title": "Invert Binary Tree",
					"titleSlug": "invert-binary-tree",
					"difficulty": "Easy"
				}
			}
		}`),
	}

	mockHTTPServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, incomingRequest *http.Request) {
		responseWriter.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(responseWriter).Encode(mockPayload)
	}))
	defer mockHTTPServer.Close()

	clientInstance := NewGatewayClient(mockHTTPServer.URL, 2*time.Second)
	serviceRouter := NewMerchantServiceRouter(clientInstance)

	responseRecorder := httptest.NewRecorder()
	inboundRequest := httptest.NewRequest(http.MethodGet, "/api/v1/daily", nil)
	serviceRouter.ServeHTTP(responseRecorder, inboundRequest)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("expected HTTP status 200, received %d", responseRecorder.Code)
	}

	var parsedEnvelope model.StandardServerEnvelope
	if decodeError := json.Unmarshal(responseRecorder.Body.Bytes(), &parsedEnvelope); decodeError != nil {
		t.Fatalf("failed to decode response payload: %v", decodeError)
	}

	if !parsedEnvelope.Success {
		t.Errorf("expected success status true, received false")
	}
}

func TestLiveUpstreamCanary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live upstream canary execution in short test mode")
	}

	clientInstance := NewGatewayClient("https://leetcode.com/graphql", 5*time.Second)
	challengeRecord, dispatchError := clientInstance.FetchDailyChallenge(context.Background())

	if dispatchError != nil {
		t.Skipf("skipping live check due to external network availability: %v", dispatchError)
	}

	if challengeRecord.CalendarDate == "" || challengeRecord.ChallengeTitle == "" || challengeRecord.DifficultyLevel == "" {
		t.Errorf("incomplete daily challenge record received: %+v", challengeRecord)
	}
}
