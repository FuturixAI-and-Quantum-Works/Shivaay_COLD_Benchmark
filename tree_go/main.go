package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Constants
const (
	MongoURI      = "mongodb://localhost:27017"
	APIKey        = "d2dd3849-bb28-42b5-8bb6-a550f84a999e" // Replace with env var in prod
	NShot         = 4
	BatchSize     = 100
	WorkerCount   = 50
	Activity      = "shopping"
	ActivityName  = "going grocery shopping"
	AnswerTrigger = "Answer:"
	InvalidAns    = "[invalid]"
)

// Sample represents a row in your dataset
type Sample struct {
	Premise  string `json:"premise"`
	Choice1  string `json:"choice1"`
	Choice2  string `json:"choice2"`
	Question string `json:"question"`
	Label    string `json:"label"` // "0" or "1"
}

// Response represents the result of processing a sample
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

// LogEntry for real-time accuracy logs
type LogEntry struct {
	Timestamp      string `bson:"timestamp"`
	TotalQuestions int    `bson:"total_questions"`
	CorrectNum     int    `bson:"correct_num"`
	Accuracy       string `bson:"accuracy"`
	InvalidAnswers int    `bson:"invalid_answers"`
	ETA            string `bson:"eta"`
}

// Stats for tracking progress
type Stats struct {
	total     int32
	correct   int32
	invalid   int32
	totalTime float64
}

// Demo examples for few-shot prompting
var demoExamples = []struct {
	activityName string
	premise      string
	choices      []string
	question     string
	answer       string
}{
	{"going grocery shopping", "select items from the shelf", []string{"pay at the counter", "leave the store without paying"}, "effect", "A"},
	{"baking a cake", "mix the batter", []string{"burn the kitchen", "pour batter into a pan"}, "effect", "B"},
	{"riding on a bus", "board the bus", []string{"buy a ticket", "fly to another city"}, "cause", "A"},
	{"planting a tree", "dig a hole", []string{"water the plant", "cut down a tree"}, "effect", "A"},
}

var answerRegex = regexp.MustCompile(`\b(A|B)\b`)

// MongoDB collections
var (
	db          *mongo.Database
	resultsColl *mongo.Collection
	logsColl    *mongo.Collection
)

func init() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Connect to MongoDB
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(MongoURI))
	if err != nil {
		log.Fatal(err)
	}
	db = client.Database("tree_evaluation_db")
	resultsColl = db.Collection("results")
	logsColl = db.Collection("logs")

	// Clear collections (optional)
	resultsColl.Drop(context.Background())
	logsColl.Drop(context.Background())
}

// getQuestionText generates the prompt for a sample
func getQuestionText(activityName, premise string, choices []string, causalQuestion string) string {
	return fmt.Sprintf(
		"The following are multiple choice questions about '%s'. You should directly answer the question by choosing the correct option.\n"+
			"Which of the following events (given as options A or B) is a plausible %s of the event '%s'?\n"+
			"A. %s\nB. %s\nAnswer:",
		activityName, causalQuestion, premise, choices[0], choices[1],
	)
}

// createDemoText generates few-shot examples
func createDemoText(nShot int) string {
	randExamples := make([]struct {
		activityName string
		premise      string
		choices      []string
		question     string
		answer       string
	}, len(demoExamples))
	copy(randExamples, demoExamples)
	rand.Shuffle(len(randExamples), func(i, j int) {
		randExamples[i], randExamples[j] = randExamples[j], randExamples[i]
	})

	var demo strings.Builder
	for i := 0; i < nShot && i < len(randExamples); i++ {
		ex := randExamples[i]
		demo.WriteString(getQuestionText(ex.activityName, ex.premise, ex.choices, ex.question))
		demo.WriteString(" ")
		demo.WriteString(ex.answer)
		demo.WriteString("\n\n")
	}
	return demo.String()
}

// buildPrompt constructs the full prompt
func buildPrompt(sample Sample, nShot int) string {
	demo := createDemoText(nShot)
	choices := []string{sample.Choice1, sample.Choice2}
	questionText := getQuestionText(ActivityName, sample.Premise, choices, sample.Question)
	return demo + questionText
}

// cleanAnswer extracts A or B from the model response
func cleanAnswer(modelPred string) string {
	modelPred = strings.TrimSpace(modelPred)
	matches := answerRegex.FindStringSubmatch(modelPred)
	if len(matches) > 0 {
		return matches[0]
	}
	return InvalidAns
}

// askQuestion makes an API call to get the model’s response
func askQuestion(inputText string) (string, error) {
	request := struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages" binding:"required"`
		Temperature float64 `json:"temperature" binding:"required"`
		TopP        float64 `json:"top_p" binding:"required"`
	}{
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{Role: "system", Content: "You are an expert assistant. Provide the correct answer (A or B) directly without explanation."},
			{Role: "user", Content: inputText},
		},
		Temperature: 0.7,
		TopP:        0.9,
	}

	apiResponse, err := CallExternalAPI(request)
	if err != nil {
		return "", err
	}

	var result map[string]string
	json.Unmarshal(apiResponse.Body(), &result)
	return result["answer"], nil
}

func CallExternalAPI(request struct {
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages" binding:"required"`
	Temperature float64 `json:"temperature" binding:"required"`
	TopP        float64 `json:"top_p" binding:"required"`
}) (*resty.Response, error) {
	client := resty.New()
	apiResponse, err := client.R().
		SetBody(map[string]interface{}{
			"messages": []map[string]string{
				{"role": "system", "content": request.Messages[0].Content},
				{"role": "user", "content": request.Messages[1].Content},
			},
			"temperature": request.Temperature,
			"top_p":       request.TopP,
			"stream":      false,
		}).
		Post("https://shivaay_model_go.futurixai.com/v1/chat/completions")

	return apiResponse, err
}

// processSample processes a single sample
func processSample(sample Sample, stats *Stats) Response {
	startTime := time.Now()
	inputText := buildPrompt(sample, NShot)
	modelCompletion, err := askQuestion(inputText)
	if err != nil {
		log.Printf("API error: %v", err)
		modelCompletion = InvalidAns
	}

	modelAnswer := cleanAnswer(modelCompletion)
	correctAnswer := "A"
	if sample.Label == "1" {
		correctAnswer = "B"
	}

	isCorrect := modelAnswer == correctAnswer
	isInvalid := modelAnswer == InvalidAns

	if isCorrect {
		atomic.AddInt32(&stats.correct, 1)
	}
	if isInvalid {
		atomic.AddInt32(&stats.invalid, 1)
	}
	atomic.AddInt32(&stats.total, 1)
	stats.totalTime += time.Since(startTime).Seconds()

	return Response{
		Premise:         sample.Premise,
		Choice1:         sample.Choice1,
		Choice2:         sample.Choice2,
		CausalQuestion:  sample.Question,
		CorrectAnswer:   correctAnswer,
		ModelAnswer:     modelAnswer,
		ModelCompletion: modelCompletion,
		IsCorrect:       isCorrect,
		IsInvalid:       isInvalid,
		ProcessingTime:  time.Since(startTime).Seconds(),
	}
}

// formatTime converts seconds to a human-readable string
func formatTime(seconds float64) string {
	hours := int(seconds / 3600)
	minutes := int(math.Mod(seconds, 3600) / 60)
	secs := int(math.Mod(seconds, 60))
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, secs)
	}
	return fmt.Sprintf("%dm %ds", minutes, secs)
}

func main() {
	startTime := time.Now()
	stats := &Stats{}
	var wg sync.WaitGroup

	// Channels for samples and results
	samplesChan := make(chan Sample, WorkerCount)
	resultsChan := make(chan Response, BatchSize)

	// Start workers
	for i := 0; i < WorkerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for sample := range samplesChan {
				result := processSample(sample, stats)
				resultsChan <- result
			}
		}()
	}

	// Load dataset from CSV
	go func() {
		file, err := os.Open("planting_a_tree.csv")
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		reader := csv.NewReader(bufio.NewReader(file))
		header, err := reader.Read() // Skip header
		if err != nil {
			log.Fatal(err)
		}

		// Map header to indices (adjust based on actual CSV columns)
		colMap := make(map[string]int)
		for i, col := range header {
			colMap[col] = i
		}

		for {
			record, err := reader.Read()
			if err != nil {
				break // EOF or error
			}
			sample := Sample{
				Premise:  record[colMap["premise"]],
				Choice1:  record[colMap["choice1"]],
				Choice2:  record[colMap["choice2"]],
				Question: record[colMap["question"]],
				Label:    record[colMap["label"]],
			}
			samplesChan <- sample
		}
		close(samplesChan)
	}()

	// Collect results and write to MongoDB
	go func() {
		var batch []interface{}
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case result, ok := <-resultsChan:
				if !ok {
					if len(batch) > 0 {
						resultsColl.InsertMany(context.Background(), batch)
					}
					return
				}
				batch = append(batch, result)
				if len(batch) >= BatchSize {
					resultsColl.InsertMany(context.Background(), batch)
					batch = nil
				}
			case <-ticker.C:
				total := atomic.LoadInt32(&stats.total)
				correct := atomic.LoadInt32(&stats.correct)
				invalid := atomic.LoadInt32(&stats.invalid)
				accuracy := float64(correct) / float64(total) * 100
				avgTime := stats.totalTime / float64(total)
				remaining := float64(3500000-total) * avgTime // 3.5M rows
				eta := formatTime(remaining)

				logEntry := LogEntry{
					Timestamp:      time.Now().Format(time.RFC3339),
					TotalQuestions: int(total),
					CorrectNum:     int(correct),
					Accuracy:       fmt.Sprintf("%.2f%%", accuracy),
					InvalidAnswers: int(invalid),
					ETA:            eta,
				}
				logsColl.InsertOne(context.Background(), logEntry)
				log.Printf("Processed: %d, Correct: %d, Accuracy: %.2f%%, Invalid: %d, ETA: %s",
					total, correct, accuracy, invalid, eta)
			}
		}
	}()

	wg.Wait()
	close(resultsChan)

	// Final stats
	total := atomic.LoadInt32(&stats.total)
	correct := atomic.LoadInt32(&stats.correct)
	invalid := atomic.LoadInt32(&stats.invalid)
	accuracy := float64(correct) / float64(total) * 100
	totalTime := time.Since(startTime).Seconds()

	finalLog := LogEntry{
		Timestamp:      time.Now().Format(time.RFC3339),
		TotalQuestions: int(total),
		CorrectNum:     int(correct),
		Accuracy:       fmt.Sprintf("%.2f%%", accuracy),
		InvalidAnswers: int(invalid),
		ETA:            formatTime(totalTime),
	}
	logsColl.InsertOne(context.Background(), finalLog)
	log.Printf("Evaluation completed! Total time: %s, Accuracy: %.2f%%, Invalid: %d",
		formatTime(totalTime), accuracy, invalid)

	// Export to JSON
	results, _ := resultsColl.Find(context.Background(), bson.D{})
	defer results.Close(context.Background())
	var allResults []Response
	for results.Next(context.Background()) {
		var res Response
		results.Decode(&res)
		allResults = append(allResults, res)
	}
	jsonData, _ := json.MarshalIndent(allResults, "", "  ")
	os.WriteFile("complete_response.json", jsonData, 0644)

	logs, _ := logsColl.Find(context.Background(), bson.D{})
	defer logs.Close(context.Background())
	var allLogs []LogEntry
	for logs.Next(context.Background()) {
		var logEntry LogEntry
		logs.Decode(&logEntry)
		allLogs = append(allLogs, logEntry)
	}
	logsData, _ := json.MarshalIndent(allLogs, "", "  ")
	os.WriteFile("logs.json", logsData, 0644)

	log.Println("Data exported to JSON files.")
	// while true to catch all
	for {
		time.Sleep(10 * time.Second)
	}

}
