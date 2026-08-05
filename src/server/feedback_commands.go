package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"strconv"
)

func pendingFeedbackCommand(arguments []string) error {
	flags := flag.NewFlagSet("feedback pending", flag.ContinueOnError)
	dataPath := flags.String("data", defaultDatabasePath(), "SQLite database path")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("usage: jobdeck feedback pending [--data PATH]")
	}
	database, err := openDatabase(*dataPath)
	if err != nil {
		return err
	}
	defer database.Close()
	items, err := listPendingFeedback(context.Background(), database)
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}

func resolveFeedbackCommand(arguments []string) error {
	flags := flag.NewFlagSet("feedback resolve", flag.ContinueOnError)
	dataPath := flags.String("data", defaultDatabasePath(), "SQLite database path")
	note := flags.String("note", "Preference review completed", "review outcome")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return errors.New("usage: jobdeck feedback resolve [--note TEXT] ID")
	}
	feedbackID, err := strconv.ParseInt(flags.Arg(0), 10, 64)
	if err != nil || feedbackID < 1 {
		return errors.New("feedback id must be a positive integer")
	}
	database, err := openDatabase(*dataPath)
	if err != nil {
		return err
	}
	defer database.Close()
	if err := markFeedbackReviewed(context.Background(), database, feedbackID,
		*note); err != nil {
		return err
	}
	logger := newActivityLogger(privatePath("logs"))
	if err := logger.append("agent", "feedback_reviewed", "Reviewed job feedback",
		map[string]string{
			"feedback_id": strconv.FormatInt(feedbackID, 10),
			"outcome":     *note,
		}); err != nil {
		return err
	}
	fmt.Println("Resolved feedback", feedbackID)
	return nil
}
