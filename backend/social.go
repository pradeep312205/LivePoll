package main

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var followerEmailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

type SocialHandler struct {
	Polls, Comments, Followers *mongo.Collection
}

type PollComment struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	PollID    string        `bson:"poll_id" json:"pollId"`
	Name      string        `bson:"name" json:"name"`
	Text      string        `bson:"text" json:"text"`
	CreatedAt time.Time     `bson:"created_at" json:"createdAt"`
}

type PollFollower struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	PollID    string        `bson:"poll_id" json:"pollId"`
	Email     string        `bson:"email" json:"email"`
	FirstName string        `bson:"first_name,omitempty" json:"firstName,omitempty"`
	CreatedAt time.Time     `bson:"created_at" json:"createdAt"`
}

func (h *SocialHandler) CommentsList(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	count, err := h.Polls.CountDocuments(ctx, bson.D{{Key: "_id", Value: c.Param("id")}})
	if err != nil {
		c.JSON(500, gin.H{"error": "Could not load discussion"})
		return
	}
	if count == 0 {
		c.JSON(404, gin.H{"error": "Poll not found"})
		return
	}
	cur, err := h.Comments.Find(ctx, bson.D{{Key: "poll_id", Value: c.Param("id")}}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(100))
	if err != nil {
		c.JSON(500, gin.H{"error": "Could not load discussion"})
		return
	}
	defer cur.Close(ctx)
	comments := []PollComment{}
	if err = cur.All(ctx, &comments); err != nil {
		c.JSON(500, gin.H{"error": "Could not load discussion"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"comments": comments})
}

func (h *SocialHandler) AddComment(c *gin.Context) {
	var request struct {
		Name string `json:"name"`
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": "Invalid comment"})
		return
	}
	request.Name = strings.TrimSpace(request.Name)
	request.Text = strings.TrimSpace(request.Text)
	if len([]rune(request.Name)) < 1 || len([]rune(request.Name)) > 60 || len([]rune(request.Text)) < 2 || len([]rune(request.Text)) > 500 {
		c.JSON(400, gin.H{"error": "Name must be 1-60 characters and comment must be 2-500 characters"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	count, err := h.Polls.CountDocuments(ctx, bson.D{{Key: "_id", Value: c.Param("id")}})
	if err != nil {
		c.JSON(500, gin.H{"error": "Could not add comment"})
		return
	}
	if count == 0 {
		c.JSON(404, gin.H{"error": "Poll not found"})
		return
	}
	comment := PollComment{PollID: c.Param("id"), Name: request.Name, Text: request.Text, CreatedAt: time.Now().UTC()}
	result, err := h.Comments.InsertOne(ctx, comment)
	if err != nil {
		c.JSON(500, gin.H{"error": "Could not add comment"})
		return
	}
	comment.ID = result.InsertedID.(bson.ObjectID)
	c.JSON(http.StatusCreated, comment)
}

func (h *SocialHandler) Follow(c *gin.Context) {
	var request struct {
		Email     string `json:"email"`
		FirstName string `json:"firstName"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": "Invalid follower details"})
		return
	}
	request.Email = strings.ToLower(strings.TrimSpace(request.Email))
	request.FirstName = strings.TrimSpace(request.FirstName)
	if len(request.Email) > 254 || !followerEmailPattern.MatchString(request.Email) || len([]rune(request.FirstName)) > 80 {
		c.JSON(400, gin.H{"error": "Enter a valid email address and a first name up to 80 characters"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var poll Poll
	if err := h.Polls.FindOne(ctx, bson.D{{Key: "_id", Value: c.Param("id")}}).Decode(&poll); err != nil {
		c.JSON(404, gin.H{"error": "Poll not found"})
		return
	}
	if !poll.FollowersEnabled {
		c.JSON(409, gin.H{"error": "Following is not enabled for this poll"})
		return
	}
	follower := PollFollower{PollID: poll.ID, Email: request.Email, FirstName: request.FirstName, CreatedAt: time.Now().UTC()}
	result, err := h.Followers.InsertOne(ctx, follower)
	if mongo.IsDuplicateKeyError(err) {
		c.JSON(409, gin.H{"error": "This email is already following the poll"})
		return
	}
	if err != nil {
		c.JSON(500, gin.H{"error": "Could not save your follow request"})
		return
	}
	follower.ID = result.InsertedID.(bson.ObjectID)
	c.JSON(http.StatusCreated, gin.H{"message": "You are on the poll's follow list", "follower": follower})
}

func (h *SocialHandler) ListFollowers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var poll Poll
	if err := h.Polls.FindOne(ctx, bson.D{{Key: "_id", Value: c.Param("id")}, {Key: "creator_id", Value: c.GetString("userID")}}).Decode(&poll); err != nil {
		c.JSON(404, gin.H{"error": "Poll not found"})
		return
	}
	cur, err := h.Followers.Find(ctx, bson.D{{Key: "poll_id", Value: poll.ID}}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(500))
	if err != nil {
		c.JSON(500, gin.H{"error": "Could not load followers"})
		return
	}
	defer cur.Close(ctx)
	followers := []PollFollower{}
	if err = cur.All(ctx, &followers); err != nil {
		c.JSON(500, gin.H{"error": "Could not load followers"})
		return
	}
	c.JSON(200, gin.H{"followers": followers})
}

func ensureSocialIndexes(ctx context.Context, comments, followers *mongo.Collection) error {
	if _, err := comments.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "poll_id", Value: 1}, {Key: "created_at", Value: -1}}}); err != nil {
		return err
	}
	_, err := followers.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "poll_id", Value: 1}, {Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)})
	return err
}
