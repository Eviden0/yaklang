//go:build ssa_only

package ssa_compile

import (
	"context"
	"strings"

	"github.com/yaklang/yaklang/common/utils"
	"github.com/yaklang/yaklang/common/utils/filesys"
	"github.com/yaklang/yaklang/common/yak/ssaapi"
	"github.com/yaklang/yaklang/common/yak/ssaapi/ssaconfig"
)

// ParseProjectWithAutoDetective is the slim runner: it compiles the project
// in-process from an explicit config (or target + language) and deliberately
// skips the Yak plugin-based auto-detect step, which would pull the script
// engine and gRPC tree into the SSA engine.
func ParseProjectWithAutoDetective(ctx context.Context, conf *SSADetectConfig) (*SSADetectResult, error) {
	if conf == nil {
		conf = &SSADetectConfig{}
	}

	info, compileConfig, err := resolveCompileConfig(ctx, conf)
	if err != nil {
		return nil, err
	}
	if !conf.CompileImmediately {
		return &SSADetectResult{Info: info}, nil
	}

	prog, err := compileProject(ctx, compileConfig)
	if err != nil {
		return &SSADetectResult{Info: info}, err
	}
	return &SSADetectResult{Info: info, Program: prog}, nil
}

func resolveCompileConfig(ctx context.Context, conf *SSADetectConfig) (*AutoDetectInfo, *ssaconfig.Config, error) {
	if conf.Config != nil {
		cfg, err := cloneConfigWithOptions(conf.Config, append(conf.Options, ssaconfig.WithContext(ctx))...)
		if err != nil {
			return nil, nil, err
		}
		if err := ensureLocalFileSystem(cfg); err != nil {
			return nil, nil, err
		}
		return &AutoDetectInfo{
			Config:             cfg,
			CompileImmediately: conf.CompileImmediately,
		}, cfg, nil
	}

	if strings.TrimSpace(conf.Target) == "" {
		return nil, nil, utils.Errorf("target is required when config is empty")
	}

	opts := []ssaconfig.Option{
		ssaconfig.WithContext(ctx),
		ssaconfig.WithCodeSourceKind(ssaconfig.CodeSourceLocal),
		ssaconfig.WithCodeSourceLocalFile(conf.Target),
		ssaapi.WithFileSystem(filesys.NewRelLocalFs(conf.Target)),
	}
	if language := strings.TrimSpace(conf.Language); language != "" {
		opts = append(opts, ssaconfig.WithProjectRawLanguage(language))
	}
	opts = append(opts, conf.Options...)

	cfg, err := ssaconfig.NewCLIScanConfig(opts...)
	if err != nil {
		return nil, nil, err
	}
	info := &AutoDetectInfo{
		Config:             cfg,
		CompileImmediately: conf.CompileImmediately,
	}
	return info, cfg, nil
}

func cloneConfigWithOptions(base *ssaconfig.Config, options ...ssaconfig.Option) (*ssaconfig.Config, error) {
	if base == nil {
		return nil, utils.Errorf("base config is nil")
	}
	raw, err := base.ToJSONRaw()
	if err != nil {
		return nil, utils.Errorf("marshal config failed: %s", err)
	}
	cfg, err := ssaconfig.NewCLIScanConfig(ssaconfig.WithJsonRawConfig(raw))
	if err != nil {
		return nil, utils.Errorf("clone config failed: %s", err)
	}
	if err := cfg.Update(options...); err != nil {
		return nil, utils.Errorf("apply config options failed: %s", err)
	}
	return cfg, nil
}

func compileProject(ctx context.Context, config *ssaconfig.Config) (*ssaapi.Program, error) {
	if config == nil {
		return nil, utils.Errorf("config is nil")
	}
	return CompileWithConfig(ctx, config)
}

// ensureLocalFileSystem fills the compile FS for a pre-resolved config. The
// full engine's auto-detect plugin does this implicitly; the slim runner has
// to make it explicit.
func ensureLocalFileSystem(cfg *ssaconfig.Config) error {
	if cfg == nil || strings.TrimSpace(cfg.GetCodeSourceLocalFile()) == "" {
		return nil
	}
	return cfg.Update(ssaapi.WithFileSystem(filesys.NewRelLocalFs(cfg.GetCodeSourceLocalFile())))
}
