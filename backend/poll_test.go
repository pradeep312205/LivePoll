package main

import "testing"

func TestValidatePoll(t *testing.T) {
	tests := []struct {
		name    string
		request CreatePollRequest
		wantErr bool
	}{
		{name: "trims question and options", request: CreatePollRequest{Question: "  Pick a color?  ", Options: []string{" Red ", "Blue"}}},
		{name: "question too short", request: CreatePollRequest{Question: "Hey?", Options: []string{"Yes", "No"}}, wantErr: true},
		{name: "too few options", request: CreatePollRequest{Question: "Pick one?", Options: []string{"Yes"}}, wantErr: true},
		{name: "too many options", request: CreatePollRequest{Question: "Pick one?", Options: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}}, wantErr: true},
		{name: "duplicate options ignoring case", request: CreatePollRequest{Question: "Pick one?", Options: []string{"Blue", " blue "}}, wantErr: true},
		{name: "empty option", request: CreatePollRequest{Question: "Pick one?", Options: []string{"   ", "No"}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			question, options, errMessage := validatePoll(tt.request)
			if gotErr := errMessage != ""; gotErr != tt.wantErr {
				t.Fatalf("error = %q, wantErr %v", errMessage, tt.wantErr)
			}
			if !tt.wantErr {
				if question != "Pick a color?" || len(options) != 2 || options[0].Text != "Red" {
					t.Fatalf("unexpected normalized poll: question=%q options=%+v", question, options)
				}
				if options[0].ID == "" || options[0].ID == options[1].ID {
					t.Fatalf("option IDs must be unique and non-empty: %+v", options)
				}
			}
		})
	}
}
