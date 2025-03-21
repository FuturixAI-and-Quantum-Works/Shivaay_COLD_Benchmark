package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	MongoURI = "mongodb://localhost:27017"
)

// Response represents the result structure in MongoDB
type Response struct {
	Premise         string  `bson:"premise"`
	Choice1         string  `bson:"choice1"`
	Choice2         string  `bson:"choice2"`
	CausalQuestion  string  `bson:"causal_question"`
	CorrectAnswer   string  `bson:"correct_answer"`
	ModelAnswer     string  `bson:"model_answer"`
	ModelCompletion string  `bson:"model_completion"`
	IsCorrect       bool    `bson:"is_correct"`
	IsInvalid       bool    `bson:"is_invalid"`
	ProcessingTime  float64 `bson:"processing_time"`
}

// calculateAccuracy calculates and saves accuracy for a given database
func calculateAccuracy(client *mongo.Client, dbName, outputFile string) {
	db := client.Database(dbName)
	resultsColl := db.Collection("results")

	// Fetch all results
	results, err := resultsColl.Find(context.Background(), bson.D{})
	if err != nil {
		log.Printf("Failed to fetch results from %s: %v", dbName, err)
		return
	}
	defer results.Close(context.Background())

	var total, correct, invalid int
	var allResults []Response
	for results.Next(context.Background()) {
		var res Response
		if err := results.Decode(&res); err != nil {
			log.Printf("Failed to decode result from %s: %v", dbName, err)
			continue
		}
		allResults = append(allResults, res)
		total++
		if res.IsCorrect {
			correct++
		}
		if res.IsInvalid {
			invalid++
		}
	}

	if total == 0 {
		log.Printf("No results found in %s", dbName)
		return
	}

	// Calculate accuracy
	accuracy := float64(correct) / float64(total) * 100
	validTotal := total - invalid
	var validAccuracy float64
	if validTotal > 0 {
		validAccuracy = float64(correct) / float64(validTotal) * 100
	}

	// Prepare metadata string
	metadata := fmt.Sprintf(
		"Database: %s\n"+
			"Total Samples: %d\n"+
			"Correct Answers: %d\n"+
			"Invalid Answers: %d\n"+
			"Overall Accuracy: %.2f%%\n"+
			"Accuracy (excluding invalids): %.2f%%\n",
		dbName, total, correct, invalid, accuracy, validAccuracy,
	)

	// Print to console
	fmt.Printf("Results for %s:\n%s\n", dbName, metadata)

	// Save to file
	err = os.WriteFile(outputFile, []byte(metadata), 0644)
	if err != nil {
		log.Printf("Failed to write metadata to %s: %v", outputFile, err)
		return
	}
	log.Printf("Metadata for %s saved to %s", dbName, outputFile)
}

func main() {
	// Connect to MongoDB
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(MongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())

	// List of databases and corresponding output files
	databases := []struct {
		name       string
		outputFile string
	}{
		{"train_evaluation_db", "train_accuracy_metadata.txt"},
		{"cake_evaluation_db", "cake_accuracy_metadata.txt"},
		{"bus_evaluation_db", "bus_accuracy_metadata.txt"},
		{"tree_evaluation_db", "tree_accuracy_metadata.txt"},
		{"evaluation_db", "shopping_accuracy_metadata.txt"},
	}

	// Process each database
	for _, db := range databases {
		calculateAccuracy(client, db.name, db.outputFile)
	}
}
