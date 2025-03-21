Below is a README content tailored for your project based on the provided directory structure and evaluation results:

---

# Shivaay_COLD

Shivaay_COLD is a project containing multiple Go-based benchmarks for evaluating various tasks. This repository includes benchmarks for activities such as riding a bus, baking a cake, grocery shopping, going on a train, and planting a tree. Each benchmark folder contains its own dataset, Go source code, and evaluation results, with final aggregated results computed in the `results` folder.

## Directory Structure

```
.
├── bus_go/                  # Benchmark for riding a bus
├── cake_go/                 # Benchmark for baking a cake
├── results/                 # Aggregated results for all benchmarks
├── shopping_go/             # Benchmark for grocery shopping
├── train_go/                # Benchmark for going on a train
└── tree_go/                 # Benchmark for planting a tree
```

Each benchmark folder contains:

- `main.go`: The Go source file to run the benchmark.
- `.csv`: Dataset used for the benchmark (e.g., `riding_on_a_bus.csv`).
- `complete_response.json`: Output of the benchmark.
- `go.mod` and `go.sum`: Go module files for dependency management.

The `results` folder contains:

- `main.go`: Aggregates and computes final results across all benchmarks.
- Accuracy metadata files (e.g., `bus_accuracy_metadata.txt`).

## Evaluation Results

Below are the evaluation results for each benchmark database as of March 21, 2025:

### Bus Evaluation

- **Database**: `bus_evaluation_db`
- **Total Samples**: 834,044
- **Correct Answers**: 686,328
- **Invalid Answers**: 0
- **Overall Accuracy**: 82.29%
- **Accuracy (excluding invalids)**: 82.29%

### Cake Evaluation

- **Database**: `cake_evaluation_db`
- **Total Samples**: 2,887,948
- **Correct Answers**: 2,610,379
- **Invalid Answers**: 0
- **Overall Accuracy**: 90.39%
- **Accuracy (excluding invalids)**: 90.39%

### Shopping Evaluation

- **Database**: `evaluation_db`
- **Total Samples**: 3,739,182
- **Correct Answers**: 3,215,338
- **Invalid Answers**: 757
- **Overall Accuracy**: 85.99%
- **Accuracy (excluding invalids)**: 86.01%

### Train Evaluation

- **Database**: `train_evaluation_db`
- **Total Samples**: 1,213,112
- **Correct Answers**: 1,091,352
- **Invalid Answers**: 0
- **Overall Accuracy**: 89.96%
- **Accuracy (excluding invalids)**: 89.96%

### Tree Evaluation

- **Database**: `tree_evaluation_db`
- **Total Samples**: 846,044
- **Correct Answers**: 786,270
- **Invalid Answers**: 0
- **Overall Accuracy**: 92.93%
- **Accuracy (excluding invalids)**: 92.93%

## Prerequisites

- **Go**: Install Go (version 1.16 or higher recommended).
- **Git LFS**: Required to clone large dataset files (e.g., `.csv` files).

## How to Clone the Repository

This repository uses Git LFS to manage large files. Follow these steps to clone it:

1. Install Git LFS:
   ```bash
   git lfs install
   ```
2. Clone the repository:

   ```bash
   git clone https://github.com/FuturixAI-and-Quantum-Works/Shivaay_COLD_Benchmark
   cd Shivaay_COLD
   ```

   Replace `<repository-url>` with the actual URL of your repository.

3. Verify that Git LFS is tracking the large files:
   ```bash
   git lfs ls-files
   ```
   You should see the `.csv` files listed.

## How to Run the Benchmarks

Each benchmark can be run independently by executing the `main.go` file in its respective folder. Follow these steps:

1. Navigate to a benchmark folder (e.g., `bus_go`):
   ```bash
   cd bus_go
   ```
2. Run the benchmark:
   ```bash
   go run main.go
   ```
3. Repeat this process for each folder:
   - `bus_go`
   - `cake_go`
   - `shopping_go`
   - `train_go`
   - `tree_go`

Each run will generate a `complete_response.json` file in the respective folder and contribute to the accuracy metadata in the `results` folder.

## How to Compute Final Results

After running all benchmarks, aggregate the results by running the `main.go` file in the `results` folder:

1. Navigate to the `results` folder:
   ```bash
   cd results
   ```
2. Run the aggregation script:
   ```bash
   go run main.go
   ```
   This will process the accuracy metadata files (e.g., `bus_accuracy_metadata.txt`) and output the final results.

## Notes

- Ensure all dependencies are installed by running `go mod tidy` in each folder if you encounter issues.
- The `.csv` files are large and managed by Git LFS; ensure you have sufficient disk space.
- Results are based on the latest run of the benchmarks as of March 21, 2025.

## Contributing

Feel free to submit pull requests or open issues for improvements or bug reports.

---

Save this content as `README.md` in the root of your repository. Let me know if you'd like any adjustments!
