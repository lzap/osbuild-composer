package pipeline

import (
	"github.com/osbuild/osbuild-composer/internal/osbuild2"
)

// A QCOW2Pipeline turns a raw image file into qcow2 image.
type QCOW2Pipeline struct {
	Pipeline
	Compat string

	imgPipeline *LiveImgPipeline
	filename    string
}

// NewQCOW2Pipeline createsa new QCOW2 pipeline. imgPipeline is the pipeline producing the
// raw image. The pipeline name is the name of the new pipeline. Filename is the name
// of the produced qcow2 image.
func NewQCOW2Pipeline(buildPipeline *BuildPipeline, imgPipeline *LiveImgPipeline, filename string) QCOW2Pipeline {
	return QCOW2Pipeline{
		Pipeline:    New("qcow2", buildPipeline, nil),
		imgPipeline: imgPipeline,
		filename:    filename,
	}
}

func (p QCOW2Pipeline) Serialize() osbuild2.Pipeline {
	pipeline := p.Pipeline.Serialize()

	pipeline.AddStage(osbuild2.NewQEMUStage(
		osbuild2.NewQEMUStageOptions(p.filename,
			osbuild2.QEMUFormatQCOW2,
			osbuild2.QCOW2Options{
				Compat: p.Compat,
			}),
		osbuild2.NewQemuStagePipelineFilesInputs(p.imgPipeline.Name(), p.imgPipeline.Filename()),
	))

	return pipeline
}
