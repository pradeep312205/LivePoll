package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type DashboardHandler struct {
	Polls, Votes *mongo.Collection
	Redis        *redis.Client
}

type AssistantMessage struct {
	Role   string `json:"role"`
	Text   string `json:"text"`
	PollID string `json:"pollId,omitempty"`
}

type PollListItem struct {
	Poll
	TotalVotes int64 `json:"totalVotes"`
}

type PollListResponse struct {
	Polls []PollListItem `json:"polls"`
	Total int64          `json:"total"`
	Page  int64          `json:"page"`
	Limit int64          `json:"limit"`
}

type PollMetric struct {
	PollID     string    `json:"pollId"`
	Question   string    `json:"question"`
	CreatedAt  time.Time `json:"createdAt"`
	Open       bool      `json:"open"`
	TotalVotes int64     `json:"totalVotes"`
}

type OptionMetric struct {
	PollID   string  `json:"pollId,omitempty"`
	OptionID string  `json:"optionId"`
	Text     string  `json:"text"`
	Votes    int64   `json:"votes"`
	Percent  float64 `json:"percent"`
}

type DailyMetric struct {
	Date  string `json:"date"`
	Votes int64  `json:"votes"`
}

type ActivityMetric struct {
	PollID    string    `json:"pollId"`
	Question  string    `json:"question"`
	Option    string    `json:"option"`
	CreatedAt time.Time `json:"createdAt"`
}

type AnalyticsResponse struct {
	TotalPolls           int64            `json:"totalPolls"`
	ActivePolls          int64            `json:"activePolls"`
	ClosedPolls          int64            `json:"closedPolls"`
	TotalVotes           int64            `json:"totalVotes"`
	AverageVotesPerPoll  float64          `json:"averageVotesPerPoll"`
	HighestParticipation *PollMetric      `json:"highestParticipation,omitempty"`
	MostSelectedOption   *OptionMetric    `json:"mostSelectedOption,omitempty"`
	VotesByPoll          []PollMetric     `json:"votesByPoll"`
	VotesOverTime        []DailyMetric    `json:"votesOverTime"`
	OptionDistribution   []OptionMetric   `json:"optionDistribution"`
	RecentActivity       []ActivityMetric `json:"recentActivity"`
}

type voteCountRow struct {
	ID    string `bson:"_id"`
	Count int64  `bson:"count"`
}

type optionCountRow struct {
	ID struct {
		PollID   string `bson:"poll_id"`
		OptionID string `bson:"option_id"`
	} `bson:"_id"`
	Count int64 `bson:"count"`
}

func (h *DashboardHandler) ListMine(c *gin.Context) {
	page, err := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Page must be a positive number"})
		return
	}
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "20"), 10, 64)
	if err != nil || limit < 1 || limit > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Limit must be between 1 and 100"})
		return
	}
	sortName := c.DefaultQuery("sort", "newest")
	if sortName != "newest" && sortName != "oldest" && sortName != "votes" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sort must be newest, oldest, or votes"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()
	filter := bson.D{{Key: "creator_id", Value: c.GetString("userID")}}
	status := c.DefaultQuery("status", "all")
	if status != "all" && status != "active" && status != "closed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status must be all, active, or closed"})
		return
	}
	if status == "active" {
		filter = append(filter, bson.E{Key: "open", Value: true})
	}
	if status == "closed" {
		filter = append(filter, bson.E{Key: "open", Value: false})
	}
	query := strings.TrimSpace(c.Query("q"))
	if len([]rune(query)) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query cannot exceed 100 characters"})
		return
	}
	if query != "" {
		filter = append(filter, bson.E{Key: "question", Value: bson.D{{Key: "$regex", Value: regexp.QuoteMeta(query)}, {Key: "$options", Value: "i"}}})
	}
	total, err := h.Polls.CountDocuments(ctx, filter)
	if err != nil {
		c.JSON(500, gin.H{"error": "Could not load polls"})
		return
	}
	direction := int32(-1)
	if sortName == "oldest" {
		direction = 1
	}
	findOptions := options.Find().SetSort(bson.D{{Key: "created_at", Value: direction}})
	if sortName != "votes" {
		findOptions.SetSkip((page - 1) * limit).SetLimit(limit)
	}
	cur, err := h.Polls.Find(ctx, filter, findOptions)
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
	ids := make([]string, 0, len(polls))
	for _, poll := range polls {
		ids = append(ids, poll.ID)
	}
	counts, err := h.voteCounts(ctx, ids, nil)
	if err != nil {
		c.JSON(500, gin.H{"error": "Could not load poll totals"})
		return
	}
	items := make([]PollListItem, 0, len(polls))
	for _, poll := range polls {
		items = append(items, PollListItem{Poll: poll, TotalVotes: counts[poll.ID]})
	}
	if sortName == "votes" {
		sort.Slice(items, func(i, j int) bool { return items[i].TotalVotes > items[j].TotalVotes })
		start := (page - 1) * limit
		if start >= int64(len(items)) {
			items = []PollListItem{}
		} else {
			end := start + limit
			if end > int64(len(items)) {
				end = int64(len(items))
			}
			items = items[start:end]
		}
	}
	c.JSON(200, PollListResponse{Polls: items, Total: total, Page: page, Limit: limit})
}

func (h *DashboardHandler) Summary(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()
	analytics, recent, err := h.buildAnalytics(ctx, c.GetString("userID"), "", "", "", "")
	if err != nil {
		c.JSON(500, gin.H{"error": "Could not calculate dashboard summary"})
		return
	}
	if len(recent) > 6 {
		recent = recent[:6]
	}
	c.JSON(200, gin.H{"summary": analytics, "recentPolls": recent})
}

func (h *DashboardHandler) Analytics(c *gin.Context) {
	from, to, err := parseDateRange(c.Query("from"), c.Query("to"))
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	status := c.DefaultQuery("status", "all")
	if status != "all" && status != "active" && status != "closed" {
		c.JSON(400, gin.H{"error": "Status must be all, active, or closed"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	result, _, err := h.buildAnalytics(ctx, c.GetString("userID"), c.Query("pollId"), status, fromDateString(from), toDateString(to))
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(404, gin.H{"error": "Poll not found"})
			return
		}
		c.JSON(500, gin.H{"error": "Could not calculate analytics"})
		return
	}
	c.JSON(200, result)
}

func fromDateString(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02")
}

func toDateString(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.AddDate(0, 0, -1).Format("2006-01-02")
}

func parseDateRange(fromText, toText string) (time.Time, time.Time, error) {
	var from, to time.Time
	var err error
	if fromText != "" {
		from, err = time.Parse("2006-01-02", fromText)
		if err != nil {
			return from, to, fmt.Errorf("From date must use YYYY-MM-DD")
		}
	}
	if toText != "" {
		to, err = time.Parse("2006-01-02", toText)
		if err != nil {
			return from, to, fmt.Errorf("To date must use YYYY-MM-DD")
		}
		to = to.AddDate(0, 0, 1)
	}
	if !from.IsZero() && !to.IsZero() {
		if from.After(to.AddDate(0, 0, -1)) {
			return from, to, fmt.Errorf("From date must be on or before to date")
		}
		if to.Sub(from) > 367*24*time.Hour {
			return from, to, fmt.Errorf("Date range cannot exceed one year")
		}
	}
	return from, to, nil
}

func (h *DashboardHandler) buildAnalytics(ctx context.Context, owner, selectedPoll, status, fromText, toText string) (AnalyticsResponse, []PollMetric, error) {
	filter := bson.D{{Key: "creator_id", Value: owner}}
	if status == "active" {
		filter = append(filter, bson.E{Key: "open", Value: true})
	}
	if status == "closed" {
		filter = append(filter, bson.E{Key: "open", Value: false})
	}
	cur, err := h.Polls.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return AnalyticsResponse{}, nil, err
	}
	defer cur.Close(ctx)
	var polls []Poll
	if err = cur.All(ctx, &polls); err != nil {
		return AnalyticsResponse{}, nil, err
	}

	response := AnalyticsResponse{TotalPolls: int64(len(polls)), VotesByPoll: []PollMetric{}, VotesOverTime: []DailyMetric{}, OptionDistribution: []OptionMetric{}, RecentActivity: []ActivityMetric{}}
	pollByID := make(map[string]Poll, len(polls))
	optionText := make(map[string]string)
	ids := make([]string, 0, len(polls))
	voteIDs := make([]string, 0, len(polls))
	selectedFound := selectedPoll == ""
	for _, poll := range polls {
		pollByID[poll.ID] = poll
		ids = append(ids, poll.ID)
		if poll.Open {
			response.ActivePolls++
		} else {
			response.ClosedPolls++
		}
		if selectedPoll == "" || selectedPoll == poll.ID {
			voteIDs = append(voteIDs, poll.ID)
		}
		if selectedPoll == poll.ID {
			selectedFound = true
		}
		for _, option := range poll.Options {
			optionText[option.ID] = option.Text
		}
	}
	if !selectedFound {
		return response, nil, mongo.ErrNoDocuments
	}
	if len(voteIDs) == 0 {
		return response, nil, nil
	}
	from, to, err := parseDateRange(fromText, toText)
	if err != nil {
		return response, nil, err
	}
	match := votesMatch(voteIDs, from, to)
	counts, err := h.voteCounts(ctx, voteIDs, match)
	if err != nil {
		return response, nil, err
	}
	metricsByID := make(map[string]PollMetric, len(voteIDs))
	response.VotesByPoll = make([]PollMetric, 0, len(voteIDs))
	for _, id := range voteIDs {
		poll := pollByID[id]
		metric := PollMetric{PollID: id, Question: poll.Question, CreatedAt: poll.CreatedAt, Open: poll.Open, TotalVotes: counts[id]}
		metricsByID[id] = metric
		response.VotesByPoll = append(response.VotesByPoll, metric)
		response.TotalVotes += metric.TotalVotes
	}
	if selectedPoll != "" {
		response.TotalPolls = 1
		response.ActivePolls = 0
		response.ClosedPolls = 0
		if pollByID[selectedPoll].Open {
			response.ActivePolls = 1
		} else {
			response.ClosedPolls = 1
		}
	} else {
		response.TotalPolls = int64(len(polls))
	}
	if response.TotalPolls > 0 {
		response.AverageVotesPerPoll = float64(response.TotalVotes) / float64(response.TotalPolls)
	}
	sort.Slice(response.VotesByPoll, func(i, j int) bool { return response.VotesByPoll[i].TotalVotes > response.VotesByPoll[j].TotalVotes })
	if len(response.VotesByPoll) > 0 && response.VotesByPoll[0].TotalVotes > 0 {
		top := response.VotesByPoll[0]
		response.HighestParticipation = &top
	}

	optionIDs := bson.D{{Key: "$ifNull", Value: bson.A{"$option_ids", bson.A{}}}}
	legacyOptionIDs := bson.D{{Key: "$cond", Value: bson.A{
		bson.D{{Key: "$ne", Value: bson.A{"$option_id", nil}}},
		bson.A{"$option_id"},
		bson.A{},
	}}}
	answerIDs := bson.D{{Key: "$cond", Value: bson.A{
		bson.D{{Key: "$gt", Value: bson.A{bson.D{{Key: "$size", Value: optionIDs}}, 0}}},
		optionIDs,
		legacyOptionIDs,
	}}}
	optionPipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: match}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "_answer_option_ids", Value: answerIDs}}}},
		bson.D{{Key: "$unwind", Value: "$_answer_option_ids"}},
		bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: bson.D{{Key: "poll_id", Value: "$poll_id"}, {Key: "option_id", Value: "$_answer_option_ids"}}}, {Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}}}}},
	}
	optionCursor, err := h.Votes.Aggregate(ctx, optionPipeline)
	if err != nil {
		return response, nil, err
	}
	var optionRows []optionCountRow
	if err = optionCursor.All(ctx, &optionRows); err != nil {
		optionCursor.Close(ctx)
		return response, nil, err
	}
	optionCursor.Close(ctx)
	for _, row := range optionRows {
		metric := OptionMetric{PollID: row.ID.PollID, OptionID: row.ID.OptionID, Text: optionText[row.ID.OptionID], Votes: row.Count}
		metric.Percent = votePercent(row.Count, metricsByID[row.ID.PollID].TotalVotes)
		if response.MostSelectedOption == nil || metric.Votes > response.MostSelectedOption.Votes {
			copy := metric
			response.MostSelectedOption = &copy
		}
		if selectedPoll != "" {
			response.OptionDistribution = append(response.OptionDistribution, metric)
		}
	}
	dailyPipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: match}},
		bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: bson.D{{Key: "$dateToString", Value: bson.D{{Key: "format", Value: "%Y-%m-%d"}, {Key: "date", Value: "$created_at"}, {Key: "timezone", Value: "UTC"}}}}}, {Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}}}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	}
	dailyCursor, err := h.Votes.Aggregate(ctx, dailyPipeline)
	if err != nil {
		return response, nil, err
	}
	var days []voteCountRow
	if err = dailyCursor.All(ctx, &days); err != nil {
		dailyCursor.Close(ctx)
		return response, nil, err
	}
	dailyCursor.Close(ctx)
	for _, day := range days {
		response.VotesOverTime = append(response.VotesOverTime, DailyMetric{Date: day.ID, Votes: day.Count})
	}
	sort.Slice(response.OptionDistribution, func(i, j int) bool {
		return response.OptionDistribution[i].Votes > response.OptionDistribution[j].Votes
	})

	recentCursor, err := h.Votes.Find(ctx, match, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(10))
	if err != nil {
		return response, nil, err
	}
	var votes []Vote
	if err = recentCursor.All(ctx, &votes); err != nil {
		recentCursor.Close(ctx)
		return response, nil, err
	}
	recentCursor.Close(ctx)
	for _, vote := range votes {
		poll, ok := pollByID[vote.PollID]
		if ok {
			response.RecentActivity = append(response.RecentActivity, ActivityMetric{PollID: vote.PollID, Question: poll.Question, Option: optionText[vote.OptionID], CreatedAt: vote.CreatedAt})
		}
	}
	recentPolls := make([]PollMetric, 0, len(polls))
	for _, poll := range polls {
		recentPolls = append(recentPolls, PollMetric{PollID: poll.ID, Question: poll.Question, CreatedAt: poll.CreatedAt, Open: poll.Open, TotalVotes: counts[poll.ID]})
	}
	return response, recentPolls, nil
}

func votesMatch(ids []string, from, to time.Time) bson.D {
	match := bson.D{{Key: "poll_id", Value: bson.D{{Key: "$in", Value: ids}}}}
	created := bson.D{}
	if !from.IsZero() {
		created = append(created, bson.E{Key: "$gte", Value: from})
	}
	if !to.IsZero() {
		created = append(created, bson.E{Key: "$lt", Value: to})
	}
	if len(created) > 0 {
		match = append(match, bson.E{Key: "created_at", Value: created})
	}
	return match
}

func votePercent(count, total int64) float64 {
	if total <= 0 || count <= 0 {
		return 0
	}
	return 100 * float64(count) / float64(total)
}

func (h *DashboardHandler) voteCounts(ctx context.Context, ids []string, match bson.D) (map[string]int64, error) {
	counts := map[string]int64{}
	if len(ids) == 0 {
		return counts, nil
	}
	if match == nil {
		match = votesMatch(ids, time.Time{}, time.Time{})
	}
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: match}},
		bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$poll_id"}, {Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}}}}},
	}
	cur, err := h.Votes.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var rows []voteCountRow
	if err = cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.ID] = row.Count
	}
	return counts, nil
}

func assistantFallback(message string, analytics AnalyticsResponse) string {
	text := strings.ToLower(message)
	switch {
	case text == "hi" || text == "hello" || text == "hey" || text == "hi there" || text == "good morning" || text == "good afternoon" || text == "good evening":
		return "Hi! What are you working on today? I can help you draft a poll, improve a question, understand your results, or figure out how to share a poll."
	case strings.Contains(text, "thank"):
		return "You’re welcome! Want help with anything else in LivePoll?"
	case text == "help" || strings.Contains(text, "what can you do"):
		return "I can help you write a balanced poll, choose a question type, improve participation, share a poll, and summarize your saved results. What would you like to do?"
	case strings.Contains(text, "most") || strings.Contains(text, "highest") || strings.Contains(text, "analytics") || strings.Contains(text, "summar") || strings.Contains(text, "result") || strings.Contains(text, "vote"):
		if analytics.TotalPolls == 0 {
			return "You don't have any polls yet, so I can't summarize account results. Create a poll first, then ask me about votes and participation."
		}
		answer := fmt.Sprintf("Your account has %d polls (%d open, %d closed) and %d total votes, averaging %.1f votes per poll.", analytics.TotalPolls, analytics.ActivePolls, analytics.ClosedPolls, analytics.TotalVotes, analytics.AverageVotesPerPoll)
		if analytics.HighestParticipation != nil {
			answer += fmt.Sprintf(" Your most participated poll is “%s” with %d votes.", analytics.HighestParticipation.Question, analytics.HighestParticipation.TotalVotes)
		}
		if analytics.MostSelectedOption != nil {
			answer += fmt.Sprintf(" The most selected answer is “%s” with %d votes.", analytics.MostSelectedOption.Text, analytics.MostSelectedOption.Votes)
		}
		return answer + " These totals come from your saved votes."
	case strings.Contains(text, "neutral") || strings.Contains(text, "unbiased"):
		return "Use neutral wording, avoid suggesting a preferred answer, and keep choices mutually exclusive. For example, ask “How satisfied are you with your experience?” rather than “How great was your experience?” Include a balanced scale such as Very dissatisfied, Dissatisfied, Neither, Satisfied, Very satisfied."
	case strings.Contains(text, "participation") || strings.Contains(text, "engagement"):
		return "To improve participation, keep the question short, make every option understandable, share the link where your audience already gathers, and tell people how the results will be used. Don't pressure participants or change the poll after voting starts."
	case strings.Contains(text, "share") || strings.Contains(text, "link"):
		return "Open My Polls, use the link action beside your poll to copy its public URL, and share that URL with participants. They can vote without an account; one vote per browser participant is enforced by the server."
	default:
		if strings.Contains(text, "poll") && (strings.Contains(text, "madurai") || strings.Contains(text, "trip") || strings.Contains(text, "travel") || strings.Contains(text, "outing")) {
			destination := "your trip"
			if match := regexp.MustCompile(`(?i)\b(?:going|traveling|travelling|trip|outing)\s+to\s+([a-z][a-z ]{1,40})`).FindStringSubmatch(message); len(match) > 1 {
				destination = strings.TrimSpace(match[1])
				for _, stop := range []string{" with ", " for ", " this ", " next ", " and "} {
					if index := strings.Index(strings.ToLower(destination), stop); index >= 0 {
						destination = strings.TrimSpace(destination[:index])
					}
				}
				if destination != "" {
					destination = strings.ToUpper(destination[:1]) + destination[1:]
				}
			}
			return fmt.Sprintf("Sounds fun! Here’s a ready-to-use poll for %s:\n\nQuestion: What should we decide first for our %s trip?\nOptions: Travel, departure time, places to visit, shared budget.\n\nIf you had a different decision in mind—like choosing a date or activity—tell me and I’ll tailor the poll.", destination, destination)
		}
		if strings.Contains(text, "question") || strings.Contains(text, "create") || strings.Contains(text, "poll") {
			return "Sure. Tell me the topic and who will answer it, and I’ll help shape a clear question and answer choices. A useful pattern is: “Which option works best for [decision]?” Keep the choices short and balanced."
		}
		return "I’m here to help with your LivePoll polls and results. Tell me a little more about what you’re trying to do, and I’ll suggest a next step. You can ask me to draft a poll, improve participation, share a link, or summarize results."
	}
}

func assistantFallbackWithContext(message string, history []AssistantMessage, analytics AnalyticsResponse) string {
	current := strings.ToLower(strings.TrimSpace(message))
	if current == "hi" || current == "hello" || current == "hey" || current == "hi there" || current == "good morning" || current == "good afternoon" || current == "good evening" || strings.Contains(current, "thank") {
		return assistantFallback(message, analytics)
	}
	var previous []string
	for _, item := range history {
		if item.Role == "user" && strings.TrimSpace(item.Text) != "" {
			previous = append(previous, strings.ToLower(strings.TrimSpace(item.Text)))
		}
	}
	if len(previous) > 5 {
		previous = previous[len(previous)-5:]
	}
	contextText := strings.Join(previous, " ")
	combined := contextText + " " + current
	travelTopic := strings.Contains(combined, "tour") || strings.Contains(combined, "trip") || strings.Contains(combined, "travel") || strings.Contains(combined, "outing") || strings.Contains(combined, "madurai") || strings.Contains(combined, "friends")
	yesNoIntent := strings.Contains(current, "yes or no") || strings.Contains(current, "yes/no") || strings.Contains(current, "yes no")
	if yesNoIntent && travelTopic {
		return "Here’s the Yes/No poll for your tour:\n\nQuestion: Are you joining the tour?\nOptions: Yes · No\n\nIn LivePoll, choose Create Poll → Yes or no, enter the question, and create it. Then share the poll link with your friends."
	}
	if travelTopic && (strings.Contains(current, "how come") || strings.Contains(current, "how to come") || strings.Contains(current, "how will") || strings.Contains(current, "transport") || strings.Contains(current, "travel option")) {
		return "For the tour, you can ask: “How will you get there?” Options: Bus, car, train, other. In Create Poll, choose Multiple choice, add those options, then share the poll link with your friends."
	}
	attendanceIntent := strings.Contains(current, "who comes") || strings.Contains(current, "who is coming") || strings.Contains(current, "who's coming") || strings.Contains(current, "comes and not") || strings.Contains(current, "coming or not") || strings.Contains(current, "come or not") || strings.Contains(current, "attending") || strings.Contains(current, "joining") || strings.Contains(current, "rsvp")
	if travelTopic && attendanceIntent {
		return "Yes—make it an RSVP poll: “Are you joining us for the tour?” Options: Yes, No, Not sure. In Create Poll, choose Multiple choice, add those three options, create the poll, and share its link."
	}
	if travelTopic && (strings.Contains(current, "poll") || strings.Contains(current, "create") || strings.Contains(current, "make") || strings.Contains(current, "going")) {
		return "Sounds fun! To find out who’s joining, try this RSVP poll: “Are you joining us for the tour?” Options: Yes, No, Not sure. If you meant a poll about the destination, date, or transport, tell me which one and I’ll draft that instead."
	}
	return assistantFallback(message, analytics)
}

func wantsPollLink(message string) bool {
	text := strings.ToLower(message)
	return strings.Contains(text, "create a link") || strings.Contains(text, "make a link") || strings.Contains(text, "generate a link") || strings.Contains(text, "give me a link") || strings.Contains(text, "share link") || strings.Contains(text, "create link")
}

func (h *DashboardHandler) createTourRSVP(ctx context.Context, ownerID string) (Poll, error) {
	question, options, validationErr := validatePoll(CreatePollRequest{Question: "Are you joining us for the tour?", QuestionType: "yes_no"})
	if validationErr != "" {
		return Poll{}, fmt.Errorf("could not prepare RSVP poll: %s", validationErr)
	}
	poll := Poll{
		ID: newID(), Question: question, Options: options, CreatorID: ownerID,
		CreatedAt: time.Now().UTC(), Open: true, QuestionType: "yes_no", Theme: "engage",
		ThemeBackground: "#f2f4ec", ThemeAccent: "#316b4d",
	}
	if _, err := h.Polls.InsertOne(ctx, poll); err != nil {
		return Poll{}, err
	}
	for _, option := range poll.Options {
		if err := h.Redis.HSet(ctx, countKey(poll.ID), option.ID, 0).Err(); err != nil {
			_ = h.Redis.Del(ctx, countKey(poll.ID)).Err()
			return poll, err
		}
	}
	if err := h.Redis.HSet(ctx, countKey(poll.ID), "__total__", 0).Err(); err != nil {
		_ = h.Redis.Del(ctx, countKey(poll.ID)).Err()
		return poll, err
	}
	return poll, nil
}

func (h *DashboardHandler) Chat(c *gin.Context) {
	var request struct {
		Message  string             `json:"message"`
		Messages []AssistantMessage `json:"messages"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}
	request.Message = strings.TrimSpace(request.Message)
	if len([]rune(request.Message)) < 1 || len([]rune(request.Message)) > 1000 {
		c.JSON(400, gin.H{"error": "Message must be 1-1000 characters"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	analytics, _, err := h.buildAnalytics(ctx, c.GetString("userID"), "", "", "", "")
	if err != nil {
		c.JSON(500, gin.H{"error": "Could not read your poll analytics"})
		return
	}
	if wantsPollLink(request.Message) {
		conversation := strings.ToLower(request.Message)
		for _, item := range request.Messages {
			conversation += " " + strings.ToLower(item.Text)
		}
		if strings.Contains(conversation, "tour") && (strings.Contains(conversation, "yes/no") || strings.Contains(conversation, "yes or no") || strings.Contains(conversation, "joining the tour")) {
			poll, createErr := h.createTourRSVP(ctx, c.GetString("userID"))
			if createErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create the tour poll. Please try again."})
				return
			}
			c.JSON(http.StatusCreated, gin.H{"mode": "guidance", "label": "Poll created", "reply": "Done! Your tour RSVP poll is ready. Open it to check the result, or copy the link to invite your friends.", "pollId": poll.ID})
			return
		}
	}
	if apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")); apiKey != "" {
		reply, callErr := askOpenAI(ctx, apiKey, request.Message, request.Messages, analytics)
		if callErr != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "The AI assistant could not respond right now. Please try again."})
			return
		}
		c.JSON(200, gin.H{"mode": "ai", "label": "AI assistant", "reply": reply})
		return
	}
	c.JSON(200, gin.H{"mode": "guidance", "label": "LivePoll built-in guidance", "reply": assistantFallbackWithContext(request.Message, request.Messages, analytics)})
}

func askOpenAI(ctx context.Context, apiKey, message string, history []AssistantMessage, analytics AnalyticsResponse) (string, error) {
	model := strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	if model == "" {
		model = "gpt-5-mini"
	}
	type inputMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	input := make([]inputMessage, 0, 13)
	start := len(history) - 12
	if start < 0 {
		start = 0
	}
	for _, item := range history[start:] {
		role := item.Role
		if role != "assistant" && role != "user" {
			continue
		}
		text := strings.TrimSpace(item.Text)
		if text == "" || len([]rune(text)) > 1000 {
			continue
		}
		input = append(input, inputMessage{Role: role, Content: text})
	}
	input = append(input, inputMessage{Role: "user", Content: message})
	contextText := fmt.Sprintf("The signed-in user's LivePoll aggregates: %d polls, %d currently open, %d closed, %d total votes, average %.1f votes per poll.", analytics.TotalPolls, analytics.ActivePolls, analytics.ClosedPolls, analytics.TotalVotes, analytics.AverageVotesPerPoll)
	if analytics.HighestParticipation != nil {
		contextText += fmt.Sprintf(" Most participated poll: %q (%d votes).", analytics.HighestParticipation.Question, analytics.HighestParticipation.TotalVotes)
	}
	if analytics.MostSelectedOption != nil {
		contextText += fmt.Sprintf(" Most selected option across polls: %q (%d selections).", analytics.MostSelectedOption.Text, analytics.MostSelectedOption.Votes)
	}
	payload := struct {
		Model        string         `json:"model"`
		Instructions string         `json:"instructions"`
		Input        []inputMessage `json:"input"`
		MaxTokens    int            `json:"max_output_tokens"`
		Store        bool           `json:"store"`
	}{model, "You are LivePoll Assistant, a warm and practical chatbot inside a live polling app. Answer the user's actual message directly; greetings should receive a natural greeting, not a repeated onboarding prompt. Use prior conversation messages to understand follow-ups. Help with poll writing, poll formats, participation, sharing, and interpreting the provided account aggregates. Never invent poll data or claim you changed the app. Keep answers concise and ask one clarifying question when needed. Do not request or reveal passwords, API keys, or private voter emails. Account context: " + contextText, input, 500, false}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/responses", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 25 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	var result struct {
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("OpenAI returned status %d", response.StatusCode)
	}
	if err = json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", err
	}
	var answer strings.Builder
	for _, item := range result.Output {
		for _, part := range item.Content {
			if part.Type == "output_text" && strings.TrimSpace(part.Text) != "" {
				if answer.Len() > 0 {
					answer.WriteString("\n")
				}
				answer.WriteString(strings.TrimSpace(part.Text))
			}
		}
	}
	if answer.Len() == 0 {
		return "", fmt.Errorf("OpenAI returned no text")
	}
	return answer.String(), nil
}
