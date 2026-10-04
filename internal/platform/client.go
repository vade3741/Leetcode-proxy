package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/vade3741/Leetcode-proxy/internal/model"
)

var (
	ErrResourceNotFound = errors.New("requested resource does not exist on upstream host")
	ErrEmptyDataset     = errors.New("upstream host returned an empty payload")
	ErrGatewayFailure   = errors.New("upstream gateway returned an error response")
)

const (
	userProfileGraphQLQuery = `query getUserProfile($username: String!) {
		matchedUser(username: $username) {
			username
			profile {
				realName
				ranking
				reputation
			}
			submitStatsGlobal {
				acSubmissionNum {
					difficulty
					count
					submissions
				}
			}
		}
	}`

	dailyChallengeGraphQLQuery = `query getDailyChallenge {
		activeDailyCodingChallengeQuestion {
			date
			link
			question {
				questionFrontendId
				title
				titleSlug
				difficulty
			}
		}
	}`
)

type GatewayClient struct {
	GatewayEndpointURL string
	HTTPTransportPool  *http.Client
}

func NewGatewayClient(gatewayEndpointURL string, connectionTimeout time.Duration) *GatewayClient {
	return &GatewayClient{
		GatewayEndpointURL: gatewayEndpointURL,
		HTTPTransportPool: &http.Client{
			Timeout: connectionTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 25,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression: false,
			},
		},
	}
}

func (client *GatewayClient) executeGraphQLTransaction(ctx context.Context, queryString string, inputVariables map[string]any, destinationPointer any) error {
	serializedPayload, serializationError := json.Marshal(model.GraphQLPayload{
		Query:     queryString,
		Variables: inputVariables,
	})
	if serializationError != nil {
		return fmt.Errorf("payload serialization error: %w", serializationError)
	}

	outboundRequest, creationError := http.NewRequestWithContext(ctx, http.MethodPost, client.GatewayEndpointURL, bytes.NewReader(serializedPayload))
	if creationError != nil {
		return fmt.Errorf("request building error: %w", creationError)
	}

	outboundRequest.Header.Set("Content-Type", "application/json")
	outboundRequest.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko)")
	outboundRequest.Header.Set("Referer", "https://leetcode.com")
	outboundRequest.Header.Set("Origin", "https://leetcode.com")

	httpResponse, dispatchError := client.HTTPTransportPool.Do(outboundRequest)
	if dispatchError != nil {
		return fmt.Errorf("network dispatch error: %w", dispatchError)
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%w: upstream status %d", ErrGatewayFailure, httpResponse.StatusCode)
	}

	var rawEnvelope model.GraphQLResponseEnvelope
	if decodeError := json.NewDecoder(httpResponse.Body).Decode(&rawEnvelope); decodeError != nil {
		return fmt.Errorf("json decoding error: %w", decodeError)
	}

	if len(rawEnvelope.Errors) > 0 {
		return fmt.Errorf("%w: %s", ErrGatewayFailure, rawEnvelope.Errors[0].Message)
	}

	if unmarshalError := json.Unmarshal(rawEnvelope.Data, destinationPointer); unmarshalError != nil {
		return fmt.Errorf("nested unmarshaling error: %w", unmarshalError)
	}

	return nil
}

func (client *GatewayClient) FetchCandidateProfile(ctx context.Context, targetUsername string) (*model.CandidateProfile, error) {
	var queryResult model.ProfileQueryResult
	if executionError := client.executeGraphQLTransaction(ctx, userProfileGraphQLQuery, map[string]any{"username": targetUsername}, &queryResult); executionError != nil {
		return nil, executionError
	}

	if queryResult.MatchedUser == nil {
		return nil, ErrResourceNotFound
	}

	submissionIndex := make(map[string]model.SubmissionMetric, len(queryResult.MatchedUser.SubmitStatsGlobal.AcSubmissionNum))
	for _, submissionEntry := range queryResult.MatchedUser.SubmitStatsGlobal.AcSubmissionNum {
		categoryKey := strings.ToLower(submissionEntry.Difficulty)
		submissionIndex[categoryKey] = model.SubmissionMetric{
			Difficulty:         submissionEntry.Difficulty,
			SolvedQuantity:     submissionEntry.Count,
			SubmissionAttempts: submissionEntry.Submissions,
		}
	}

	return &model.CandidateProfile{
		Username:            queryResult.MatchedUser.Username,
		FullName:            queryResult.MatchedUser.Profile.RealName,
		GlobalRanking:       queryResult.MatchedUser.Profile.Ranking,
		ReputationScore:     queryResult.MatchedUser.Profile.Reputation,
		SubmissionBreakdown: submissionIndex,
	}, nil
}

func (client *GatewayClient) FetchDailyChallenge(ctx context.Context) (*model.DailyChallengeItem, error) {
	var queryResult model.DailyChallengeQueryResult
	if executionError := client.executeGraphQLTransaction(ctx, dailyChallengeGraphQLQuery, map[string]any{}, &queryResult); executionError != nil {
		return nil, executionError
	}

	challengeNode := queryResult.ActiveDailyCodingChallengeQuestion
	if challengeNode == nil {
		return nil, ErrEmptyDataset
	}

	return &model.DailyChallengeItem{
		CalendarDate:       challengeNode.Date,
		FrontendIdentifier: challengeNode.Question.QuestionFrontendID,
		ChallengeTitle:     challengeNode.Question.Title,
		SlugIdentifier:     challengeNode.Question.TitleSlug,
		DifficultyLevel:    challengeNode.Question.Difficulty,
		DirectProblemURL:   fmt.Sprintf("https://leetcode.com%s", challengeNode.Link),
	}, nil
}
