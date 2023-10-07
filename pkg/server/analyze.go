package server

import (
	"context"
	json "encoding/json"
	"fmt"

	schemav1 "buf.build/gen/go/k8sgpt-ai/k8sgpt/protocolbuffers/go/schema/v1"
	"github.com/k8sgpt-ai/k8sgpt/pkg/analysis"
)

func (h *handler) Analyze(ctx context.Context, i *schemav1.AnalyzeRequest) (
	*schemav1.AnalyzeResponse,
	error,
) {
	if i.Output == "" {
		i.Output = "json"
	}

	if i.Backend == "" {
		i.Backend = "openai"
	}

	if int(i.MaxConcurrency) == 0 {
		i.MaxConcurrency = 10
	}
	analysis, err := analysis.NewAnalysis(
		i.Backend,
		i.Language,
		i.Filters,
		i.Namespace,
		i.Nocache,
		i.Explain,
		int(i.MaxConcurrency),
		false, // Kubernetes Doc disabled in server mode
	)
	fmt.Printf("before create analysis %s, %s \n", analysis.AnalysisAIProvider, analysis.Namespace)
	if err != nil {
		fmt.Printf("create analysis %s", err.Error())
		return &schemav1.AnalyzeResponse{}, err
	}
	fmt.Println("before run analysis")

	analysis.RunAnalysis()

	fmt.Println("after run analysis")
	for _, result := range analysis.Results {
		fmt.Printf("result: %s\n", result.Name)
	}

	for _, error := range analysis.Errors {
		fmt.Printf("error: %s\n", error)
	}

	if i.Explain {
		err := analysis.GetAIResults(i.Output, i.Anonymize)
		if err != nil {
			return &schemav1.AnalyzeResponse{}, err
		}
	}

	out, err := analysis.PrintOutput(i.Output)
	if err != nil {
		return &schemav1.AnalyzeResponse{}, err
	}
	var obj schemav1.AnalyzeResponse

	err = json.Unmarshal(out, &obj)
	if err != nil {
		return &schemav1.AnalyzeResponse{}, err
	}

	return &obj, nil
}
