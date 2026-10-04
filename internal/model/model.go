package model

import "encoding/json"

type GraphQLPayload struct {
	OperationName string         `json:"operationName,omitempty"`
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables"`
}

type GraphQLErrorDetail struct {
	Message string `json:"message"`
}

type GraphQLResponseEnvelope struct {
	Data   json.RawMessage      `json:"data"`
	Errors []GraphQLErrorDetail `json:"errors,omitempty"`
}

type SubmissionMetric struct {
	Difficulty         string `json:"difficulty"`
	SolvedQuantity     int    `json:"solved_quantity"`
	SubmissionAttempts int    `json:"submission_attempts"`
}

type CandidateProfile struct {
	Username            string                      `json:"username"`
	FullName            string                      `json:"full_name"`
	GlobalRanking       int                         `json:"global_ranking"`
	ReputationScore     int                         `json:"reputation_score"`
	SubmissionBreakdown map[string]SubmissionMetric `json:"submission_breakdown"`
}

type DailyChallengeItem struct {
	CalendarDate       string `json:"calendar_date"`
	FrontendIdentifier string `json:"frontend_identifier"`
	ChallengeTitle     string `json:"challenge_title"`
	SlugIdentifier     string `json:"slug_identifier"`
	DifficultyLevel    string `json:"difficulty_level"`
	DirectProblemURL   string `json:"direct_problem_url"`
}

type StandardServerEnvelope struct {
	Success       bool   `json:"success"`
	Data          any    `json:"data,omitempty"`
	ErrorMessage  string `json:"error_message,omitempty"`
	ExecutionTime string `json:"execution_time,omitempty"`
}

type ProfileQueryResult struct {
	MatchedUser *struct {
		Username string `json:"username"`
		Profile  struct {
			RealName   string `json:"realName"`
			Ranking    int    `json:"ranking"`
			Reputation int    `json:"reputation"`
		} `json:"profile"`
		SubmitStatsGlobal struct {
			AcSubmissionNum []struct {
				Difficulty  string `json:"difficulty"`
				Count       int    `json:"count"`
				Submissions int    `json:"submissions"`
			} `json:"acSubmissionNum"`
		} `json:"submitStatsGlobal"`
	} `json:"matchedUser"`
}

type DailyChallengeQueryResult struct {
	ActiveDailyCodingChallengeQuestion *struct {
		Date string `json:"date"`
		Link string `json:"link"`
		Question struct {
			QuestionFrontendID string `json:"questionFrontendId"`
			Title              string `json:"title"`
			TitleSlug          string `json:"titleSlug"`
			Difficulty         string `json:"difficulty"`
		} `json:"question"`
	} `json:"activeDailyCodingChallengeQuestion"`
}
