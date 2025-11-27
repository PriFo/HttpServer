package pipeline_normalization

import "testing"

func TestNewNormalizationPipeline(t *testing.T) {
	config := NewDefaultConfig()
	pipeline, err := NewNormalizationPipeline(config)
	
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	
	if pipeline == nil {
		t.Fatal("Pipeline should not be nil")
	}
}

func TestNormalizationPipeline_Normalize(t *testing.T) {
	config := NewDefaultConfig()
	pipeline, err := NewNormalizationPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	
	result, err := pipeline.Normalize("тест", "тест")
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}
	
	if result == nil {
		t.Fatal("Result should not be nil")
	}
	
	if result.Similarity.OverallSimilarity < 0.9 {
		t.Errorf("Expected high similarity for identical strings, got %f", result.Similarity.OverallSimilarity)
	}
}

func TestNormalizationPipelineConfig_Validate(t *testing.T) {
	config := NewDefaultConfig()
	
	err := config.Validate()
	if err != nil {
		t.Errorf("Default config should be valid: %v", err)
	}
}

func TestSimilarityScore_CalculateOverall(t *testing.T) {
	score := NewSimilarityScore()
	score.AddAlgorithmScore("test1", 0.8)
	score.AddAlgorithmScore("test2", 0.9)
	
	weights := map[string]float64{
		"test1": 0.5,
		"test2": 0.5,
	}
	
	score.CalculateOverall(weights, "weighted", 0.85)
	
	if score.OverallSimilarity == 0 {
		t.Error("Overall similarity should be calculated")
	}
}

