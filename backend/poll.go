package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type PollOption struct {
	ID       string `bson:"id" json:"id"`
	Text     string `bson:"text" json:"text"`
	ImageURL string `bson:"image_url,omitempty" json:"imageUrl,omitempty"`
}
type Poll struct {
	ID                 string       `bson:"_id" json:"id"`
	Question           string       `bson:"question" json:"question"`
	Options            []PollOption `bson:"options" json:"options"`
	CreatorID          string       `bson:"creator_id" json:"-"`
	CreatedAt          time.Time    `bson:"created_at" json:"createdAt"`
	Open               bool         `bson:"open" json:"open"`
	QuestionType       string       `bson:"question_type,omitempty" json:"questionType,omitempty"`
	AllowMultiple      bool         `bson:"allow_multiple,omitempty" json:"allowMultiple,omitempty"`
	Theme              string       `bson:"theme,omitempty" json:"theme,omitempty"`
	ThemeBackground    string       `bson:"theme_background,omitempty" json:"themeBackground,omitempty"`
	ThemeAccent        string       `bson:"theme_accent,omitempty" json:"themeAccent,omitempty"`
	RatingScale        int          `bson:"rating_scale,omitempty" json:"ratingScale,omitempty"`
	GridRows           []PollOption `bson:"grid_rows,omitempty" json:"gridRows,omitempty"`
	OpensAt            *time.Time   `bson:"opens_at,omitempty" json:"opensAt,omitempty"`
	EndsAt              *time.Time   `bson:"ends_at,omitempty" json:"endsAt,omitempty"`
	CloseAfterVotes    int64        `bson:"close_after_votes,omitempty" json:"closeAfterVotes,omitempty"`
	ClosedMessage      string       `bson:"closed_message,omitempty" json:"closedMessage,omitempty"`
	PasscodeHash       string       `bson:"passcode_hash,omitempty" json:"-"`
	PasscodeRequired   bool         `bson:"-" json:"passcodeRequired,omitempty"`
	FollowersEnabled   bool         `bson:"followers_enabled,omitempty" json:"followersEnabled,omitempty"`
	FollowAfterResults bool         `bson:"follow_after_results,omitempty" json:"followAfterResults,omitempty"`
	FollowerHeadline   string       `bson:"follower_headline,omitempty" json:"followerHeadline,omitempty"`
	CorrectAnswer      string       `bson:"correct_answer,omitempty" json:"-"`
}
type Vote struct {
	ID            bson.ObjectID     `bson:"_id,omitempty"`
	PollID        string            `bson:"poll_id"`
	OptionID      string            `bson:"option_id,omitempty"`
	OptionIDs     []string          `bson:"option_ids,omitempty"`
	AnswerText    string            `bson:"answer_text,omitempty"`
	Correct       bool              `bson:"correct,omitempty"`
	Ranking       []string          `bson:"ranking,omitempty"`
	GridResponses map[string]string `bson:"grid_responses,omitempty"`
	VoterID       string            `bson:"voter_id"`
	CreatedAt     time.Time         `bson:"created_at"`
}
type PollHandler struct {
	Polls, Votes *mongo.Collection
	Redis        *redis.Client
}
type CreatePollRequest struct {
	Question           string   `json:"question"`
	Options            []string `json:"options"`
	OptionImages       []string `json:"optionImages"`
	QuestionType       string   `json:"questionType"`
	AllowMultiple      bool     `json:"allowMultiple"`
	RatingScale        int      `json:"ratingScale"`
	GridRows           []string `json:"gridRows"`
	Theme              string   `json:"theme"`
	ThemeBackground    string   `json:"themeBackground"`
	ThemeAccent        string   `json:"themeAccent"`
	Passcode           string   `json:"passcode"`
	OpensAt            string   `json:"opensAt"`
	EndsAt              string   `json:"endsAt"`
	CloseAfterVotes    int64    `json:"closeAfterVotes"`
	ClosedMessage      string   `json:"closedMessage"`
	FollowersEnabled   bool     `json:"followersEnabled"`
	FollowAfterResults bool     `json:"followAfterResults"`
	FollowerHeadline   string   `json:"followerHeadline"`
	CorrectAnswer      string   `json:"correctAnswer"`
}
type VoteRequest struct {
	OptionID   string            `json:"optionId"`
	OptionIDs  []string          `json:"optionIds"`
	AnswerText string            `json:"answerText"`
	Ranking    []string          `json:"ranking"`
	Grid       map[string]string `json:"grid"`
	Passcode   string            `json:"passcode"`
	Honeypot   string            `json:"website"`
	VoterID    string            `json:"voterId"`
}
type EditPollRequest struct {
	Question string `json:"question"`
}
type PollSnapshot struct {
	Poll           Poll                        `json:"poll"`
	Counts         map[string]int64            `json:"counts"`
	Total          int64                       `json:"total"`
	GridCounts     map[string]map[string]int64 `json:"gridCounts,omitempty"`
	RankingScores  map[string]int64            `json:"rankingScores,omitempty"`
	TextAnswers    []TextAnswerMetric          `json:"textAnswers,omitempty"`
	CorrectAnswers int64                       `json:"correctAnswers,omitempty"`
}
type TextAnswerMetric struct {
	Text  string `json:"text"`
	Votes int64  `json:"votes"`
}
type liveEvent struct {
	OptionID       string                      `json:"optionId"`
	Counts         map[string]int64            `json:"counts"`
	Total          int64                       `json:"total"`
	Open           bool                        `json:"open"`
	GridCounts     map[string]map[string]int64 `json:"gridCounts,omitempty"`
	RankingScores  map[string]int64            `json:"rankingScores,omitempty"`
	TextAnswers    []TextAnswerMetric          `json:"textAnswers,omitempty"`
	CorrectAnswers int64                       `json:"correctAnswers,omitempty"`
}

func (h *PollHandler) Create(c *gin.Context) {
	var req CreatePollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}
	q, optionsList, validationErr := validatePoll(req)
	if validationErr != "" {
		c.JSON(400, gin.H{"error": validationErr})
		return
	}
	owner, _ := c.Get("userID")
	pollType := normalizeQuestionType(req.QuestionType)
	ratingScale := req.RatingScale
	if (pollType == "rating_scale" || pollType == "rating_grid") && ratingScale == 0 {
		ratingScale = 5
	}
	poll := Poll{
		ID: newID(), Question: q, Options: optionsList, CreatorID: owner.(string),
		CreatedAt: time.Now().UTC(), Open: true, QuestionType: pollType,
		AllowMultiple: req.AllowMultiple && pollType == "multiple_choice", RatingScale: ratingScale, Theme: normalizeTheme(req.Theme),
		ThemeBackground: normalizeHexColor(req.ThemeBackground, "#f2f4ec"), ThemeAccent: normalizeHexColor(req.ThemeAccent, "#316b4d"),
		CloseAfterVotes: req.CloseAfterVotes, ClosedMessage: strings.TrimSpace(req.ClosedMessage),
		FollowersEnabled: req.FollowersEnabled, FollowAfterResults: req.FollowAfterResults,
		FollowerHeadline: strings.TrimSpace(req.FollowerHeadline), CorrectAnswer: strings.TrimSpace(req.CorrectAnswer),
	}
	if req.OpensAt != "" {
		opensAt, parseErr := time.Parse(time.RFC3339, req.OpensAt)
		if parseErr != nil || opensAt.Before(time.Now().Add(-time.Minute)) {
			c.JSON(400, gin.H{"error": "Scheduled start must be a future date and time"})
			return
		}
		poll.OpensAt = &opensAt
	}
	if req.EndsAt != "" {
		endsAt, parseErr := time.Parse(time.RFC3339, req.EndsAt)
		if parseErr != nil || !endsAt.After(time.Now()) {
			c.JSON(400, gin.H{"error": "Scheduled end must be a future date and time"})
			return
		}
		if poll.OpensAt != nil && !endsAt.After(*poll.OpensAt) {
			c.JSON(400, gin.H{"error": "Poll end must be after its start time"})
			return
		}
		poll.EndsAt = &endsAt
	}
	if passcode := strings.TrimSpace(req.Passcode); passcode != "" {
		if len([]rune(passcode)) < 4 || len([]rune(passcode)) > 32 {
			c.JSON(400, gin.H{"error": "Voting passcode must be 4-32 characters"})
			return
		}
		digest := sha256.Sum256([]byte(passcode))
		poll.PasscodeHash = hex.EncodeToString(digest[:])
		poll.PasscodeRequired = true
	}
	for _, text := range req.GridRows {
		poll.GridRows = append(poll.GridRows, PollOption{ID: newID(), Text: strings.TrimSpace(text)})
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	if _, err := h.Polls.InsertOne(ctx, poll); err != nil {
		c.JSON(500, gin.H{"error": "Could not create poll"})
		return
	}
	for _, o := range poll.Options {
		_ = h.Redis.HSet(ctx, countKey(poll.ID), o.ID, 0).Err()
	}
	_ = h.Redis.HSet(ctx, countKey(poll.ID), "__total__", 0).Err()
	c.JSON(201, poll)
}

func validatePoll(req CreatePollRequest) (string, []PollOption, string) {
	question := strings.TrimSpace(req.Question)
	if len([]rune(question)) < 5 || len([]rune(question)) > 180 {
		return "", nil, "Question must be 5-180 characters"
	}
	typeName := normalizeQuestionType(req.QuestionType)
	if req.QuestionType != "" && !isValidQuestionType(req.QuestionType) {
		return "", nil, "Choose a supported question type"
	}
	switch typeName {
	case "yes_no":
		req.Options = []string{"Yes", "No"}
	case "rating_scale", "rating_grid":
		if req.RatingScale == 0 {
			req.RatingScale = 5
		}
		if req.RatingScale != 5 && req.RatingScale != 7 && req.RatingScale != 10 {
			return "", nil, "Rating scale must be 5, 7, or 10"
		}
		req.Options = make([]string, req.RatingScale)
		for i := range req.Options {
			req.Options[i] = strconv.Itoa(i + 1)
		}
	default:
		if typeName == "open_ended" || typeName == "type_answer" || typeName == "word_cloud" || typeName == "brainstorm" {
			req.Options = nil
		}
	}
	if typeName == "rating_grid" {
		if len(req.GridRows) < 2 || len(req.GridRows) > 10 {
			return "", nil, "A rating grid needs 2-10 statements"
		}
		for _, row := range req.GridRows {
			if len([]rune(strings.TrimSpace(row))) < 2 || len([]rune(row)) > 100 {
				return "", nil, "Each grid statement must be 2-100 characters"
			}
		}
	}
	if typeName == "type_answer" && len([]rune(strings.TrimSpace(req.CorrectAnswer))) == 0 {
		return "", nil, "Add the expected answer for this quiz question"
	}
	if req.CloseAfterVotes < 0 || req.CloseAfterVotes > 1000000 || len([]rune(req.ClosedMessage)) > 240 || len([]rune(req.FollowerHeadline)) > 120 {
		return "", nil, "Check the poll settings and try again"
	}
	if typeName == "multiple_choice" && req.AllowMultiple == false {
		// Multiple choice remains single-select unless the host explicitly enables several answers.
	}
	if typeName == "open_ended" || typeName == "type_answer" || typeName == "word_cloud" || typeName == "brainstorm" {
		return question, []PollOption{}, ""
	}
	if len(req.Options) < 2 || (len(req.Options) > 8 && typeName != "rating_scale" && typeName != "rating_grid") {
		return "", nil, "A poll needs at least 2 options; standard polls support up to 8"
	}
	options := make([]PollOption, 0, len(req.Options))
	seen := map[string]bool{}
	for _, raw := range req.Options {
		text := strings.TrimSpace(raw)
		key := strings.ToLower(text)
		if len([]rune(text)) == 0 || len([]rune(text)) > 80 || seen[key] {
			return "", nil, "Options must be 1-80 characters and unique"
		}
		seen[key] = true
		options = append(options, PollOption{ID: newID(), Text: text})
	}
	if typeName == "image_choice" && len(req.OptionImages) != len(options) {
		return "", nil, "Add one image URL for every image-choice option"
	}
	if len(req.OptionImages) > 0 {
		if typeName != "image_choice" || len(req.OptionImages) != len(options) {
			return "", nil, "Add one image URL for each image-choice option"
		}
		for i, raw := range req.OptionImages {
			imageURL := strings.TrimSpace(raw)
			if imageURL == "" {
				return "", nil, "Add an image URL for every image-choice option"
			}
			parsed, err := url.ParseRequestURI(imageURL)
			if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
				return "", nil, "Image choices must use valid http or https image URLs"
			}
			options[i].ImageURL = imageURL
		}
	}
	return question, options, ""
}

func normalizeQuestionType(value string) string {
	if value == "" {
		return "multiple_choice"
	}
	return value
}

func pollCanVote(p Poll, now time.Time) bool {
	return p.Open && (p.OpensAt == nil || !now.Before(*p.OpensAt)) && (p.EndsAt == nil || now.Before(*p.EndsAt))
}

func isValidQuestionType(value string) bool {
	switch value {
	case "multiple_choice", "yes_no", "rating_scale", "image_choice", "open_ended", "type_answer", "rating_grid", "ranking", "word_cloud", "brainstorm":
		return true
	default:
		return false
	}
}

func normalizeTheme(value string) string {
	switch value {
	case "engage", "trust", "warmth", "calm", "navy", "serendipity", "custom":
		return value
	default:
		return "engage"
	}
}

func normalizeHexColor(value, fallback string) string {
	value = strings.TrimSpace(value)
	if len(value) != 7 || value[0] != '#' {
		return fallback
	}
	for _, char := range value[1:] {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return fallback
		}
	}
	return value
}

func (h *PollHandler) ListMine(c *gin.Context) {
	owner, _ := c.Get("userID")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	cur, err := h.Polls.Find(ctx, bson.D{{Key: "creator_id", Value: owner.(string)}}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(100))
	if err != nil {
		c.JSON(500, gin.H{"error": "Could not load polls"})
		return
	}
	defer cur.Close(ctx)
	var polls []Poll
	if err = cur.All(ctx, &polls); err != nil {
		c.JSON(500, gin.H{"error": "Could not load polls"})
		return
	}
	if polls == nil {
		polls = []Poll{}
	}
	c.JSON(200, polls)
}
func (h *PollHandler) Close(c *gin.Context) {
	owner, _ := c.Get("userID")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	r, err := h.Polls.UpdateOne(ctx, bson.D{{Key: "_id", Value: c.Param("id")}, {Key: "creator_id", Value: owner.(string)}}, bson.D{{Key: "$set", Value: bson.D{{Key: "open", Value: false}}}})
	if err != nil {
		c.JSON(500, gin.H{"error": "Could not close poll"})
		return
	}
	if r.MatchedCount == 0 {
		c.JSON(404, gin.H{"error": "Poll not found"})
		return
	}
	c.JSON(200, gin.H{"message": "Poll closed"})
}

func (h *PollHandler) Edit(c *gin.Context) {
	var req EditPollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}
	question := strings.TrimSpace(req.Question)
	if len([]rune(question)) < 5 || len([]rune(question)) > 180 {
		c.JSON(400, gin.H{"error": "Question must be 5-180 characters"})
		return
	}
	owner, _ := c.Get("userID")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	result, err := h.Polls.UpdateOne(ctx, bson.D{{Key: "_id", Value: c.Param("id")}, {Key: "creator_id", Value: owner.(string)}, {Key: "open", Value: true}}, bson.D{{Key: "$set", Value: bson.D{{Key: "question", Value: question}}}})
	if err != nil {
		c.JSON(500, gin.H{"error": "Could not update poll"})
		return
	}
	if result.MatchedCount == 0 {
		c.JSON(404, gin.H{"error": "Open poll not found"})
		return
	}
	c.JSON(200, gin.H{"message": "Poll updated"})
}

func (h *PollHandler) Delete(c *gin.Context) {
	owner, _ := c.Get("userID")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	result, err := h.Polls.DeleteOne(ctx, bson.D{{Key: "_id", Value: c.Param("id")}, {Key: "creator_id", Value: owner.(string)}})
	if err != nil {
		c.JSON(500, gin.H{"error": "Could not delete poll"})
		return
	}
	if result.DeletedCount == 0 {
		c.JSON(404, gin.H{"error": "Poll not found"})
		return
	}
	if _, err = h.Votes.DeleteMany(ctx, bson.D{{Key: "poll_id", Value: c.Param("id")}}); err != nil {
		c.JSON(500, gin.H{"error": "Poll deleted, but vote cleanup is pending"})
		return
	}
	if err = h.Redis.Del(ctx, countKey(c.Param("id")), eventKey(c.Param("id"))).Err(); err != nil {
		c.JSON(500, gin.H{"error": "Poll deleted, but live cache cleanup is pending"})
		return
	}
	c.JSON(200, gin.H{"message": "Poll deleted"})
}
func (h *PollHandler) loadPoll(ctx context.Context, id string) (Poll, error) {
	var p Poll
	err := h.Polls.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&p)
	return p, err
}
func (h *PollHandler) GetPublic(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	p, err := h.loadPoll(ctx, c.Param("id"))
	if err != nil {
		c.JSON(404, gin.H{"error": "Poll not found"})
		return
	}
	snap, err := h.snapshot(ctx, p)
	if err != nil {
		c.JSON(503, gin.H{"error": "Results temporarily unavailable"})
		return
	}
	c.JSON(200, snap)
}
func (h *PollHandler) snapshot(ctx context.Context, p Poll) (PollSnapshot, error) {
	values, err := h.Redis.HGetAll(ctx, countKey(p.ID)).Result()
	if err != nil {
		return PollSnapshot{}, err
	}
	if len(values) == 0 { // Redis may have lost its cache. Rebuild counts from durable votes.
		pipe := h.Redis.Pipeline()
		for _, o := range p.Options {
			pipe.HSet(ctx, countKey(p.ID), o.ID, 0)
		}
		pipe.HSet(ctx, countKey(p.ID), "__total__", 0)
		_, err = pipe.Exec(ctx)
		if err != nil {
			return PollSnapshot{}, err
		}
		cur, e := h.Votes.Find(ctx, bson.D{{Key: "poll_id", Value: p.ID}})
		if e != nil {
			return PollSnapshot{}, e
		}
		var votes []Vote
		if e = cur.All(ctx, &votes); e != nil {
			cur.Close(ctx)
			return PollSnapshot{}, e
		}
		cur.Close(ctx)
		for _, vote := range votes {
			for _, optionID := range voteOptionIDs(vote) {
				if e = h.Redis.HIncrBy(ctx, countKey(p.ID), optionID, 1).Err(); e != nil {
					return PollSnapshot{}, e
				}
			}
			for rowID, optionID := range vote.GridResponses {
				if e = h.Redis.HIncrBy(ctx, countKey(p.ID), gridCountKey(rowID, optionID), 1).Err(); e != nil {
					return PollSnapshot{}, e
				}
			}
			if len(vote.Ranking) > 0 {
				for rank, optionID := range vote.Ranking {
					points := int64(len(vote.Ranking) - rank)
					if e = h.Redis.HIncrBy(ctx, countKey(p.ID), rankCountKey(optionID), points).Err(); e != nil {
						return PollSnapshot{}, e
					}
				}
			}
			if e = h.Redis.HIncrBy(ctx, countKey(p.ID), "__total__", 1).Err(); e != nil {
				return PollSnapshot{}, e
			}
		}
		values, err = h.Redis.HGetAll(ctx, countKey(p.ID)).Result()
		if err != nil {
			return PollSnapshot{}, err
		}
	}
	counts := map[string]int64{}
	for _, o := range p.Options {
		n, _ := parseInt(values[o.ID])
		counts[o.ID] = n
	}
	total, _ := parseInt(values["__total__"])
	if _, exists := values["__total__"]; !exists {
		// Backfill the respondent total for counters created before this schema.
		total, err = h.Votes.CountDocuments(ctx, bson.D{{Key: "poll_id", Value: p.ID}})
		if err != nil {
			return PollSnapshot{}, err
		}
		if err = h.Redis.HSet(ctx, countKey(p.ID), "__total__", total).Err(); err != nil {
			return PollSnapshot{}, err
		}
	}
	p.PasscodeRequired = p.PasscodeHash != ""
	p.Open = pollCanVote(p, time.Now())
	snapshot := PollSnapshot{Poll: p, Counts: counts, Total: total}
	if p.QuestionType == "rating_grid" {
		snapshot.GridCounts = map[string]map[string]int64{}
		for _, row := range p.GridRows {
			snapshot.GridCounts[row.ID] = map[string]int64{}
			for _, option := range p.Options {
				value, _ := parseInt(values[gridCountKey(row.ID, option.ID)])
				snapshot.GridCounts[row.ID][option.ID] = value
			}
		}
	}
	if p.QuestionType == "ranking" {
		snapshot.RankingScores = map[string]int64{}
		for _, option := range p.Options {
			value, _ := parseInt(values[rankCountKey(option.ID)])
			snapshot.RankingScores[option.ID] = value
		}
	}
	if p.QuestionType == "open_ended" || p.QuestionType == "word_cloud" || p.QuestionType == "brainstorm" {
		pipeline := mongo.Pipeline{
			bson.D{{Key: "$match", Value: bson.D{{Key: "poll_id", Value: p.ID}, {Key: "answer_text", Value: bson.D{{Key: "$exists", Value: true}, {Key: "$ne", Value: ""}}}}}},
			bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$answer_text"}, {Key: "votes", Value: bson.D{{Key: "$sum", Value: 1}}}}}},
			bson.D{{Key: "$sort", Value: bson.D{{Key: "votes", Value: -1}}}}, bson.D{{Key: "$limit", Value: 50}},
		}
		textCursor, queryErr := h.Votes.Aggregate(ctx, pipeline)
		if queryErr != nil {
			return PollSnapshot{}, queryErr
		}
		defer textCursor.Close(ctx)
		var rows []struct {
			Text  string `bson:"_id"`
			Votes int64  `bson:"votes"`
		}
		if queryErr = textCursor.All(ctx, &rows); queryErr != nil {
			return PollSnapshot{}, queryErr
		}
		for _, row := range rows {
			snapshot.TextAnswers = append(snapshot.TextAnswers, TextAnswerMetric{Text: row.Text, Votes: row.Votes})
		}
	}
	if p.QuestionType == "type_answer" {
		snapshot.CorrectAnswers, err = h.Votes.CountDocuments(ctx, bson.D{{Key: "poll_id", Value: p.ID}, {Key: "correct", Value: true}})
		if err != nil {
			return PollSnapshot{}, err
		}
	}
	return snapshot, nil
}

func voteOptionIDs(vote Vote) []string {
	if len(vote.OptionIDs) > 0 {
		return vote.OptionIDs
	}
	if vote.OptionID != "" {
		return []string{vote.OptionID}
	}
	return nil
}
func gridCountKey(rowID, optionID string) string { return "grid:" + rowID + ":" + optionID }
func rankCountKey(optionID string) string        { return "rank:" + optionID }
func (h *PollHandler) Vote(c *gin.Context) {
	var req VoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}
	if len(req.VoterID) < 16 || len(req.VoterID) > 100 {
		c.JSON(400, gin.H{"error": "A valid participant ID is required"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	p, err := h.loadPoll(ctx, c.Param("id"))
	if err != nil {
		c.JSON(404, gin.H{"error": "Poll not found"})
		return
	}
	if !pollCanVote(p, time.Now()) {
		message := strings.TrimSpace(p.ClosedMessage)
		if message == "" {
			message = "This poll is not accepting votes right now"
		}
		c.JSON(409, gin.H{"error": message})
		return
	}
	if req.Honeypot != "" {
		c.JSON(400, gin.H{"error": "Invalid vote"})
		return
	}
	if p.PasscodeHash != "" {
		digest := sha256.Sum256([]byte(strings.TrimSpace(req.Passcode)))
		if hex.EncodeToString(digest[:]) != p.PasscodeHash {
			c.JSON(403, gin.H{"error": "That voting passcode is not correct"})
			return
		}
	}
	pollType := normalizeQuestionType(p.QuestionType)
	optionIDs := append([]string(nil), req.OptionIDs...)
	if len(optionIDs) == 0 && req.OptionID != "" {
		optionIDs = []string{req.OptionID}
	}
	validOption := func(id string) bool {
		for _, option := range p.Options {
			if option.ID == id {
				return true
			}
		}
		return false
	}
	vote := Vote{PollID: p.ID, VoterID: req.VoterID, CreatedAt: time.Now().UTC()}
	switch pollType {
	case "open_ended", "word_cloud", "brainstorm", "type_answer":
		answer := strings.TrimSpace(req.AnswerText)
		if len([]rune(answer)) < 1 || len([]rune(answer)) > 500 {
			c.JSON(400, gin.H{"error": "Your answer must be 1-500 characters"})
			return
		}
		vote.AnswerText = answer
		if pollType == "type_answer" {
			vote.Correct = strings.EqualFold(answer, strings.TrimSpace(p.CorrectAnswer))
		}
	case "rating_grid":
		if len(req.Grid) != len(p.GridRows) {
			c.JSON(400, gin.H{"error": "Rate every statement before submitting"})
			return
		}
		for _, row := range p.GridRows {
			choice := req.Grid[row.ID]
			if choice == "" || !validOption(choice) {
				c.JSON(400, gin.H{"error": "Choose a rating for every statement"})
				return
			}
		}
		vote.GridResponses = req.Grid
	case "ranking":
		if len(req.Ranking) != len(p.Options) {
			c.JSON(400, gin.H{"error": "Rank every option"})
			return
		}
		seen := map[string]bool{}
		for _, id := range req.Ranking {
			if !validOption(id) || seen[id] {
				c.JSON(400, gin.H{"error": "Ranking must include each option exactly once"})
				return
			}
			seen[id] = true
		}
		vote.Ranking = req.Ranking
	default:
		if len(optionIDs) == 0 || (!p.AllowMultiple && len(optionIDs) != 1) {
			c.JSON(400, gin.H{"error": "Choose an answer before submitting"})
			return
		}
		seen := map[string]bool{}
		for _, id := range optionIDs {
			if !validOption(id) || seen[id] {
				c.JSON(400, gin.H{"error": "Choose valid, unique answers"})
				return
			}
			seen[id] = true
		}
		vote.OptionIDs = optionIDs
		vote.OptionID = optionIDs[0]
	}
	_, err = h.Votes.InsertOne(ctx, vote)
	if mongo.IsDuplicateKeyError(err) {
		c.JSON(409, gin.H{"error": "You have already voted in this poll"})
		return
	}
	if err != nil {
		c.JSON(500, gin.H{"error": "Could not save vote"})
		return
	}
	pipe := h.Redis.Pipeline()
	for _, id := range voteOptionIDs(vote) {
		pipe.HIncrBy(ctx, countKey(p.ID), id, 1)
	}
	for rowID, id := range vote.GridResponses {
		pipe.HIncrBy(ctx, countKey(p.ID), gridCountKey(rowID, id), 1)
	}
	for rank, id := range vote.Ranking {
		pipe.HIncrBy(ctx, countKey(p.ID), rankCountKey(id), int64(len(vote.Ranking)-rank))
	}
	totalCmd := pipe.HIncrBy(ctx, countKey(p.ID), "__total__", 1)
	if _, err = pipe.Exec(ctx); err != nil {
		_ = h.Redis.Del(ctx, countKey(p.ID)).Err()
		c.JSON(503, gin.H{"error": "Vote saved; live results are rebuilding. Refresh the poll shortly."})
		return
	}
	total := totalCmd.Val()
	if p.CloseAfterVotes > 0 && total >= p.CloseAfterVotes {
		_, _ = h.Polls.UpdateOne(ctx, bson.D{{Key: "_id", Value: p.ID}, {Key: "open", Value: true}}, bson.D{{Key: "$set", Value: bson.D{{Key: "open", Value: false}}}})
		p.Open = false
	}
	snapshot, snapErr := h.snapshot(ctx, p)
	if snapErr != nil {
		c.JSON(201, gin.H{"message": "Vote recorded", "total": total})
		return
	}
	payload, _ := json.Marshal(liveEvent{OptionID: req.OptionID, Counts: snapshot.Counts, Total: snapshot.Total, Open: snapshot.Poll.Open, GridCounts: snapshot.GridCounts, RankingScores: snapshot.RankingScores, TextAnswers: snapshot.TextAnswers, CorrectAnswers: snapshot.CorrectAnswers})
	_ = h.Redis.Publish(ctx, eventKey(p.ID), payload).Err()
	c.JSON(201, gin.H{"message": "Vote recorded", "counts": snapshot.Counts, "total": snapshot.Total, "open": snapshot.Poll.Open, "gridCounts": snapshot.GridCounts, "rankingScores": snapshot.RankingScores, "textAnswers": snapshot.TextAnswers, "correctAnswers": snapshot.CorrectAnswers})
	return
}
func (h *PollHandler) Events(c *gin.Context) {
	ctx := c.Request.Context()
	p, err := h.loadPoll(ctx, c.Param("id"))
	if err != nil {
		c.JSON(404, gin.H{"error": "Poll not found"})
		return
	}
	pub := h.Redis.Subscribe(ctx, eventKey(p.ID))
	defer pub.Close()
	if _, err = pub.Receive(ctx); err != nil {
		c.JSON(503, gin.H{"error": "Live results unavailable"})
		return
	}
	snap, err := h.snapshot(ctx, p)
	if err != nil {
		c.JSON(503, gin.H{"error": "Live results unavailable"})
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	w := c.Writer
	_, _ = fmt.Fprintf(w, "event: snapshot\ndata: %s\n\n", mustJSON(snap))
	w.Flush()
	ch := pub.Channel()
	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			_, _ = fmt.Fprintf(w, "event: vote\ndata: %s\n\n", msg.Payload)
			w.Flush()
		case <-heartbeat.C:
			_, _ = fmt.Fprint(w, ": keepalive\n\n")
			w.Flush()
		}
	}
}
func ensurePollIndexes(ctx context.Context, polls, votes *mongo.Collection) error {
	_, err := polls.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "creator_id", Value: 1}, {Key: "created_at", Value: -1}}})
	if err != nil {
		return err
	}
	_, err = votes.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "poll_id", Value: 1}, {Key: "voter_id", Value: 1}}, Options: options.Index().SetUnique(true)})
	if err != nil {
		return err
	}
	_, err = votes.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "poll_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "poll_id", Value: 1}, {Key: "option_id", Value: 1}}},
	})
	return err
}
func countKey(id string) string        { return "livepoll:counts:" + id }
func eventKey(id string) string        { return "livepoll:events:" + id }
func newID() string                    { return bson.NewObjectID().Hex() }
func mustJSON(v any) string            { b, _ := json.Marshal(v); return string(b) }
func parseInt(s string) (int64, error) { var n int64; _, err := fmt.Sscan(s, &n); return n, err }
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		configured := os.Getenv("FRONTEND_ORIGINS")
		if configured == "" {
			configured = "http://localhost:5173"
		}
		allowed := strings.Split(configured, ",")
		if origin != "" {
			ok := false
			for _, a := range allowed {
				if strings.TrimSpace(a) == origin {
					ok = true
				}
			}
			if ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			}
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
