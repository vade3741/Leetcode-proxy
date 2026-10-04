package platform

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/vade3741/Leetcode-proxy/internal/model"
)

type MerchantServiceRouter struct {
	UpstreamClient   *GatewayClient
	RouteMultiplexer *http.ServeMux
}

func NewMerchantServiceRouter(upstreamClient *GatewayClient) *MerchantServiceRouter {
	routerInstance := &MerchantServiceRouter{
		UpstreamClient:   upstreamClient,
		RouteMultiplexer: http.NewServeMux(),
	}
	routerInstance.registerEndpoints()
	return routerInstance
}

func (router *MerchantServiceRouter) registerEndpoints() {
	router.RouteMultiplexer.HandleFunc("/healthz", router.handleServiceHealth)
	router.RouteMultiplexer.HandleFunc("/api/v1/daily", router.handleDailyChallengeRequest)
	router.RouteMultiplexer.HandleFunc("/api/v1/user/", router.handleCandidateProfileRequest)
}

func (router *MerchantServiceRouter) ServeHTTP(responseWriter http.ResponseWriter, incomingRequest *http.Request) {
	router.RouteMultiplexer.ServeHTTP(responseWriter, incomingRequest)
}

func (router *MerchantServiceRouter) writeJSONEnvelope(responseWriter http.ResponseWriter, httpStatusCode int, isSuccess bool, responsePayload any, clientErrorMessage string, durationInterval time.Duration) {
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(httpStatusCode)

	envelopePayload := model.StandardServerEnvelope{
		Success:       isSuccess,
		Data:          responsePayload,
		ErrorMessage:  clientErrorMessage,
		ExecutionTime: durationInterval.String(),
	}

	if encodingError := json.NewEncoder(responseWriter).Encode(envelopePayload); encodingError != nil {
		log.Printf("response payload encoding failure: %v", encodingError)
	}
}

func (router *MerchantServiceRouter) handleServiceHealth(responseWriter http.ResponseWriter, incomingRequest *http.Request) {
	router.writeJSONEnvelope(responseWriter, http.StatusOK, true, map[string]string{"operational_status": "ready"}, "", 0)
}

func (router *MerchantServiceRouter) handleDailyChallengeRequest(responseWriter http.ResponseWriter, incomingRequest *http.Request) {
	if incomingRequest.Method != http.MethodGet {
		router.writeJSONEnvelope(responseWriter, http.StatusMethodNotAllowed, false, nil, "http method not allowed", 0)
		return
	}

	operationStartTime := time.Now()
	challengeRecord, operationError := router.UpstreamClient.FetchDailyChallenge(incomingRequest.Context())
	elapsedDuration := time.Since(operationStartTime)

	if operationError != nil {
		if errors.Is(operationError, ErrEmptyDataset) {
			router.writeJSONEnvelope(responseWriter, http.StatusNotFound, false, nil, operationError.Error(), elapsedDuration)
			return
		}
		router.writeJSONEnvelope(responseWriter, http.StatusBadGateway, false, nil, operationError.Error(), elapsedDuration)
		return
	}

	router.writeJSONEnvelope(responseWriter, http.StatusOK, true, challengeRecord, "", elapsedDuration)
}

func (router *MerchantServiceRouter) handleCandidateProfileRequest(responseWriter http.ResponseWriter, incomingRequest *http.Request) {
	if incomingRequest.Method != http.MethodGet {
		router.writeJSONEnvelope(responseWriter, http.StatusMethodNotAllowed, false, nil, "http method not allowed", 0)
		return
	}

	candidateIdentifier := strings.TrimPrefix(incomingRequest.URL.Path, "/api/v1/user/")
	candidateIdentifier = strings.TrimSpace(candidateIdentifier)

	if candidateIdentifier == "" {
		router.writeJSONEnvelope(responseWriter, http.StatusBadRequest, false, nil, "candidate username path parameter is required", 0)
		return
	}

	operationStartTime := time.Now()
	candidateProfile, operationError := router.UpstreamClient.FetchCandidateProfile(incomingRequest.Context(), candidateIdentifier)
	elapsedDuration := time.Since(operationStartTime)

	if operationError != nil {
		if errors.Is(operationError, ErrResourceNotFound) {
			router.writeJSONEnvelope(responseWriter, http.StatusNotFound, false, nil, fmt.Sprintf("candidate profile %q not found", candidateIdentifier), elapsedDuration)
			return
		}
		router.writeJSONEnvelope(responseWriter, http.StatusBadGateway, false, nil, operationError.Error(), elapsedDuration)
		return
	}

	router.writeJSONEnvelope(responseWriter, http.StatusOK, true, candidateProfile, "", elapsedDuration)
}
