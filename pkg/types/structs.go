package types

type ArmadaEvent struct {
	PipelineRun string `json:"pipelineRun"`
	Namespace   string `json:"namespace"`
}
