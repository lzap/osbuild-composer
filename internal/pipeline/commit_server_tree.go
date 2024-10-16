package pipeline

import (
	"path/filepath"

	"github.com/osbuild/osbuild-composer/internal/common"
	"github.com/osbuild/osbuild-composer/internal/osbuild2"
	"github.com/osbuild/osbuild-composer/internal/rpmmd"
)

// An OSTreeCommitServerTreePipeline contains an nginx server serving
// an embedded ostree commit.
type OSTreeCommitServerTreePipeline struct {
	Pipeline
	// TODO: should this be configurable?
	Language string

	repos           []rpmmd.RepoConfig
	packageSpecs    []rpmmd.PackageSpec
	commitPipeline  *OSTreeCommitPipeline
	nginxConfigPath string
	listenPort      string
}

// NewOSTreeCommitServerTreePipeline creates a new pipeline. The content
// is built from repos and packages, which must contain nginx. commitPipeline
// is a pipeline producing an ostree commit to be served. nginxConfigPath
// is the path to the main nginx config file and listenPort is the port
// nginx will be listening on.
func NewOSTreeCommitServerTreePipeline(buildPipeline *BuildPipeline,
	repos []rpmmd.RepoConfig,
	packageSpecs []rpmmd.PackageSpec,
	commitPipeline *OSTreeCommitPipeline,
	nginxConfigPath,
	listenPort string) OSTreeCommitServerTreePipeline {
	return OSTreeCommitServerTreePipeline{
		Pipeline:        New("container-tree", buildPipeline, nil),
		repos:           repos,
		packageSpecs:    packageSpecs,
		commitPipeline:  commitPipeline,
		nginxConfigPath: nginxConfigPath,
		listenPort:      listenPort,
		Language:        "en_US",
	}
}

func (p OSTreeCommitServerTreePipeline) Serialize() osbuild2.Pipeline {
	pipeline := p.Pipeline.Serialize()

	pipeline.AddStage(osbuild2.NewRPMStage(osbuild2.NewRPMStageOptions(p.repos), osbuild2.NewRpmStageSourceFilesInputs(p.packageSpecs)))
	pipeline.AddStage(osbuild2.NewLocaleStage(&osbuild2.LocaleStageOptions{Language: p.Language}))

	htmlRoot := "/usr/share/nginx/html"
	repoPath := filepath.Join(htmlRoot, "repo")
	pipeline.AddStage(osbuild2.NewOSTreeInitStage(&osbuild2.OSTreeInitStageOptions{Path: repoPath}))

	pipeline.AddStage(osbuild2.NewOSTreePullStage(
		&osbuild2.OSTreePullStageOptions{Repo: repoPath},
		osbuild2.NewOstreePullStageInputs("org.osbuild.pipeline", "name:"+p.commitPipeline.Name(), p.commitPipeline.Ref()),
	))

	// make nginx log and lib directories world writeable, otherwise nginx can't start in
	// an unprivileged container
	pipeline.AddStage(osbuild2.NewChmodStage(chmodStageOptions("/var/log/nginx", "a+rwX", true)))
	pipeline.AddStage(osbuild2.NewChmodStage(chmodStageOptions("/var/lib/nginx", "a+rwX", true)))

	pipeline.AddStage(osbuild2.NewNginxConfigStage(nginxConfigStageOptions(p.nginxConfigPath, htmlRoot, p.listenPort)))

	return pipeline
}

func nginxConfigStageOptions(path, htmlRoot, listen string) *osbuild2.NginxConfigStageOptions {
	// configure nginx to work in an unprivileged container
	cfg := &osbuild2.NginxConfig{
		Listen: listen,
		Root:   htmlRoot,
		Daemon: common.BoolToPtr(false),
		PID:    "/tmp/nginx.pid",
	}
	return &osbuild2.NginxConfigStageOptions{
		Path:   path,
		Config: cfg,
	}
}

func chmodStageOptions(path, mode string, recursive bool) *osbuild2.ChmodStageOptions {
	return &osbuild2.ChmodStageOptions{
		Items: map[string]osbuild2.ChmodStagePathOptions{
			path: {Mode: mode, Recursive: recursive},
		},
	}
}
