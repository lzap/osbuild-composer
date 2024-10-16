package pipeline

import (
	"github.com/osbuild/osbuild-composer/internal/osbuild2"
	"github.com/osbuild/osbuild-composer/internal/rpmmd"
)

// An AnacondaPipeline represents the installer tree as found on an ISO.
type AnacondaPipeline struct {
	Pipeline
	// Users indicate whether or not the user spoke should be enabled in
	// anaconda. If it is, users specified in a kickstart will be configured,
	// and in case no users are provided in a kickstart the user will be
	// prompted to configure them at install time. If this is set to false
	// any kickstart provided users are ignored and the user is never
	// prompted to configure users during installation.
	Users bool
	// Biosdevname indicates whether or not biosdevname should be used to
	// name network devices when booting the installer. This may affect
	// the naming of network devices on the target system.
	Biosdevname bool
	// Variant is the variant of the product being installed, if applicable.
	Variant string

	repos        []rpmmd.RepoConfig
	packageSpecs []rpmmd.PackageSpec
	kernelVer    string
	arch         string
	product      string
	version      string
}

// NewAnacondaPipeline creates an anaconda pipeline object. repos and packages
// indicate the content to build the installer from, which is distinct from the
// packages the installer will install on the target system. kernelName is the
// name of the kernel package the intsaller will use. arch is the supported
// architecture. Product and version refers to the product the installer is the
// installer for.
func NewAnacondaPipeline(buildPipeline *BuildPipeline,
	repos []rpmmd.RepoConfig,
	packages []rpmmd.PackageSpec,
	kernelName,
	arch,
	product,
	version string) AnacondaPipeline {
	kernelVer := rpmmd.GetVerStrFromPackageSpecListPanic(packages, kernelName)
	return AnacondaPipeline{
		Pipeline:     New("anaconda-tree", buildPipeline, nil),
		repos:        repos,
		packageSpecs: packages,
		kernelVer:    kernelVer,
		arch:         arch,
		product:      product,
		version:      version,
	}
}

// KernelVer returns the NEVRA of the kernel package the installer will use at
// install time.
func (p AnacondaPipeline) KernelVer() string {
	return p.kernelVer
}

// Arch returns the supported architecture.
func (p AnacondaPipeline) Arch() string {
	return p.arch
}

// Product returns the product being installed.
func (p AnacondaPipeline) Product() string {
	return p.product
}

// Version returns the version of the product being installed.
func (p AnacondaPipeline) Version() string {
	return p.version
}

func (p AnacondaPipeline) Serialize() osbuild2.Pipeline {
	pipeline := p.Pipeline.Serialize()

	pipeline.AddStage(osbuild2.NewRPMStage(osbuild2.NewRPMStageOptions(p.repos), osbuild2.NewRpmStageSourceFilesInputs(p.packageSpecs)))
	pipeline.AddStage(osbuild2.NewBuildstampStage(&osbuild2.BuildstampStageOptions{
		Arch:    p.Arch(),
		Product: p.Product(),
		Variant: p.Variant,
		Version: p.Version(),
		Final:   true,
	}))
	pipeline.AddStage(osbuild2.NewLocaleStage(&osbuild2.LocaleStageOptions{Language: "en_US.UTF-8"}))

	rootPassword := ""
	rootUser := osbuild2.UsersStageOptionsUser{
		Password: &rootPassword,
	}

	installUID := 0
	installGID := 0
	installHome := "/root"
	installShell := "/usr/libexec/anaconda/run-anaconda"
	installPassword := ""
	installUser := osbuild2.UsersStageOptionsUser{
		UID:      &installUID,
		GID:      &installGID,
		Home:     &installHome,
		Shell:    &installShell,
		Password: &installPassword,
	}
	usersStageOptions := &osbuild2.UsersStageOptions{
		Users: map[string]osbuild2.UsersStageOptionsUser{
			"root":    rootUser,
			"install": installUser,
		},
	}

	pipeline.AddStage(osbuild2.NewUsersStage(usersStageOptions))
	pipeline.AddStage(osbuild2.NewAnacondaStage(osbuild2.NewAnacondaStageOptions(p.Users)))
	pipeline.AddStage(osbuild2.NewLoraxScriptStage(&osbuild2.LoraxScriptStageOptions{
		Path:     "99-generic/runtime-postinstall.tmpl",
		BaseArch: p.Arch(),
	}))
	pipeline.AddStage(osbuild2.NewDracutStage(dracutStageOptions(p.KernelVer(), p.Biosdevname, []string{
		"anaconda",
		"rdma",
		"rngd",
		"multipath",
		"fcoe",
		"fcoe-uefi",
		"iscsi",
		"lunmask",
		"nfs",
	})))
	pipeline.AddStage(osbuild2.NewSELinuxConfigStage(&osbuild2.SELinuxConfigStageOptions{State: osbuild2.SELinuxStatePermissive}))

	return pipeline
}

func dracutStageOptions(kernelVer string, biosdevname bool, additionalModules []string) *osbuild2.DracutStageOptions {
	kernel := []string{kernelVer}
	modules := []string{
		"bash",
		"systemd",
		"fips",
		"systemd-initrd",
		"modsign",
		"nss-softokn",
		"i18n",
		"convertfs",
		"network-manager",
		"network",
		"ifcfg",
		"url-lib",
		"drm",
		"plymouth",
		"crypt",
		"dm",
		"dmsquash-live",
		"kernel-modules",
		"kernel-modules-extra",
		"kernel-network-modules",
		"livenet",
		"lvm",
		"mdraid",
		"qemu",
		"qemu-net",
		"resume",
		"rootfs-block",
		"terminfo",
		"udev-rules",
		"dracut-systemd",
		"pollcdrom",
		"usrmount",
		"base",
		"fs-lib",
		"img-lib",
		"shutdown",
		"uefi-lib",
	}

	if biosdevname {
		modules = append(modules, "biosdevname")
	}

	modules = append(modules, additionalModules...)
	return &osbuild2.DracutStageOptions{
		Kernel:  kernel,
		Modules: modules,
		Install: []string{"/.buildstamp"},
	}
}
