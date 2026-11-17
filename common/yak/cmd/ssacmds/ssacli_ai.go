//go:build !irify_exclude && !ssa_only

package ssacmds

import (
	"io/fs"
	"math/rand"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yaklang/yaklang/common/ai/aispec"
	"github.com/yaklang/yaklang/common/log"
	"github.com/yaklang/yaklang/common/syntaxflow/sfcompletion"
	"github.com/yaklang/yaklang/common/syntaxflow/sfvm"
	cli "github.com/yaklang/yaklang/common/urfavecli"
	"github.com/yaklang/yaklang/common/utils"
	"github.com/yaklang/yaklang/common/utils/filesys"
)

var extraSSACommands = []*cli.Command{
	syntaxflowCompletion,
	syntaxflowTestCasesCompletion,
}

var syntaxflowCompletion = &cli.Command{
	Name:    "syntaxflowCompletion",
	Aliases: []string{"sf-complete", "sf-completions"},
	Usage:   "SyntaxFlow Rule Description Completion By AI",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "ai-type,type",
			Usage: "type of AI type",
		},
		cli.StringFlag{
			Name:  "target,t",
			Usage: "the file or directory to process, if it is a directory, all .sf files will be processed",
		},
		cli.StringFlag{
			Name:  "api-key,key,k",
			Usage: "api key of AI",
		},
		cli.StringFlag{
			Name:  "ai-model,model,m",
			Usage: "model of AI",
		},
		cli.StringFlag{
			Name:  "proxy,p",
			Usage: "proxy of AI",
		},
		cli.StringFlag{
			Name:  "domain,ai-domain",
			Usage: "domain of ai",
		},
		cli.StringFlag{
			Name:  "baseUrl,url",
			Usage: "baseUrl of ai",
		},
		cli.StringSliceFlag{
			Name:  "files,fs",
			Usage: "files to process, if it is a directory, all .sf files will be processed",
		},
		cli.IntFlag{
			Name:  "concurrency,c",
			Usage: "concurrency of AI completion, default is 5",
		},
		cli.DurationFlag{
			Name:  "skip-recent,recent",
			Usage: "skip files modified within this duration (e.g., 30m, 1h, 2h30m), default is 0 (no skip)",
		},
	},
	Action: func(c *cli.Context) error {
		target := c.String("target")
		typ := c.String("ai-type")
		key := c.String("api-key")
		model := c.String("ai-model")
		proxy := c.String("proxy")
		concurrency := c.Int("concurrency")
		domain := c.String("domain")
		baseUrl := c.String("url")
		files := c.StringSlice("fs")
		skipRecent := c.Duration("skip-recent")
		if concurrency == 0 {
			concurrency = 5 // default concurrency
		}

		var aiOpts []aispec.AIConfigOption
		if model != "" {
			aiOpts = append(aiOpts, aispec.WithModel(model))
		}
		if domain != "" {
			aiOpts = append(aiOpts, aispec.WithDomain(domain))
		}
		if typ != "" {
			aiOpts = append(aiOpts, aispec.WithType(typ))
		}
		if key != "" {
			aiOpts = append(aiOpts, aispec.WithAPIKey(key))
		}
		if proxy != "" {
			aiOpts = append(aiOpts, aispec.WithProxy(proxy))
		}
		if baseUrl != "" {
			aiOpts = append(aiOpts, aispec.WithBaseURL(baseUrl))
		}

		swg := new(sync.WaitGroup)
		errChan := make(chan error, 1)
		taskChannel := make(chan string, 1)
		var errors error
		errorDone := make(chan struct{}, 1)
		go func() {
			for err := range errChan {
				errors = utils.JoinErrors(errors, err)
			}
			errorDone <- struct{}{}
			close(errorDone)
		}()
		var taskCount atomic.Int64
		var processedCount atomic.Int64 // 成功处理的文件数
		var skippedCount atomic.Int64   // 跳过的文件数
		var errorCount atomic.Int64     // 处理失败的文件数
		for i := 0; i < concurrency; i++ {
			swg.Add(1)
			go func() {
				defer swg.Done()
				for fileName := range taskChannel {
					if !strings.HasSuffix(fileName, ".sf") {
						log.Infof("syntaxflow-completion: skipping file %s (not a .sf file)", fileName)
						continue
					}
					raw, err := os.ReadFile(fileName)
					if err != nil {
						errorCount.Add(1)
						log.Errorf("failed to read file %s: %v", fileName, err)
						continue
					}
					rule, err := sfcompletion.CompleteRule(fileName, string(raw), aiOpts...)
					if err != nil {
						errorCount.Add(1)
						err = utils.Errorf("failed parse complete file %s: %v", fileName, err)
						errChan <- utils.JoinErrors(err, err)
						log.Errorf("%v", err)
						continue
					}
					// check format rule
					if _, err := sfvm.CompileRule(rule); err != nil {
						errorCount.Add(1)
						err = utils.Errorf("failed completion sf rule %s: %v\nsf rule: \n%s", fileName, err, rule)
						errChan <- utils.JoinErrors(err, err)
						log.Errorf("%v", err)
						continue
					}
					err = os.WriteFile(fileName, []byte(rule), 0o666)
					if err != nil {
						errorCount.Add(1)
						log.Errorf("failed to write file %s: %v", fileName, err)
						errChan <- utils.Errorf("failed to write file %s: %v", fileName, err)
						continue
					}
					processedCount.Add(1)
					sleepTime := rand.Intn(5)
					log.Infof("syntaxflow-completion: completed file %s, sleep for %d seconds", fileName, sleepTime)
					time.Sleep(time.Second * time.Duration(sleepTime))
				}
			}()
		}
		// 检查文件是否应该被跳过（最近修改过）
		shouldSkipFile := func(filePath string) bool {
			if skipRecent == 0 {
				return false // 如果没有设置跳过时间，不跳过任何文件
			}

			fileInfo, err := os.Stat(filePath)
			if err != nil {
				log.Errorf("syntaxflow-completion: failed to get file info for %s: %v", filePath, err)
				return false // 如果无法获取文件信息，不跳过
			}

			modTime := fileInfo.ModTime()
			timeSinceModification := time.Since(modTime)

			if timeSinceModification < skipRecent {
				skippedCount.Add(1)
				log.Infof("syntaxflow-completion: skipping file %s (modified %v ago, within %v threshold)",
					filePath, timeSinceModification.Round(time.Second), skipRecent)
				return true
			}

			return false
		}

		addTask := func(target string) {
			if utils.IsFile(target) {
				if shouldSkipFile(target) {
					return
				}
				log.Infof("syntaxflow-completion: processing file %s", target)
				taskChannel <- target
				taskCount.Add(1)
			} else if utils.IsDir(target) {
				log.Infof("syntaxflow-completion: processing directory %s", target)
				filesys.Recursive(target, filesys.WithFileSystem(filesys.NewLocalFs()), filesys.WithFileStat(func(s string, info fs.FileInfo) error {
					if shouldSkipFile(s) {
						return nil
					}
					log.Infof("syntaxflow-completion: processing file %s", s)
					taskChannel <- s
					taskCount.Add(1)
					return nil
				}))
			} else {
				log.Errorf("syntaxflow-completion: file %s not found", target)
			}
		}

		if target != "" {
			addTask(target)
		}
		for _, f := range files {
			addTask(f)
		}
		close(taskChannel)
		swg.Wait()
		close(errChan)
		<-errorDone

		// 打印统计报告
		totalFiles := processedCount.Load() + skippedCount.Load() + errorCount.Load()
		log.Infof("==================== SyntaxFlow Completion Report ====================")
		log.Infof("Total files found: %d", totalFiles)
		log.Infof("Successfully processed: %d", processedCount.Load())
		log.Infof("Skipped (recently modified): %d", skippedCount.Load())
		log.Infof("Failed with errors: %d", errorCount.Load())
		if skipRecent > 0 {
			log.Infof("Skip threshold: files modified within %v", skipRecent)
		}
		log.Infof("=====================================================================")

		return errors
	},
}

var syntaxflowTestCasesCompletion = &cli.Command{
	Name:    "syntaxflow-test-cases-completion",
	Aliases: []string{"sf-test-cases", "sf-tc"},
	Usage:   "SyntaxFlow Rule Test Cases Completion By AI (Both Positive and Negative)",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "ai-type,type",
			Usage: "type of AI type",
		},
		cli.StringFlag{
			Name:  "target,t",
			Usage: "the file or directory to process, if it is a directory, all .sf files will be processed",
		},
		cli.StringFlag{
			Name:  "api-key,key,k",
			Usage: "api key of AI",
		},
		cli.StringFlag{
			Name:  "ai-model,model,m",
			Usage: "model of AI",
		},
		cli.StringFlag{
			Name:  "proxy,p",
			Usage: "proxy of AI",
		},
		cli.StringFlag{
			Name:  "domain,ai-domain",
			Usage: "domain of ai",
		},
		cli.StringFlag{
			Name:  "baseUrl,url",
			Usage: "baseUrl of ai",
		},
		cli.StringSliceFlag{
			Name:  "files,fs",
			Usage: "files to process, if it is a directory, all .sf files will be processed",
		},
		cli.IntFlag{
			Name:  "concurrency,c",
			Usage: "concurrency of AI completion, default is 3",
		},
	},
	Action: func(c *cli.Context) error {
		target := c.String("target")
		typ := c.String("ai-type")
		key := c.String("api-key")
		model := c.String("ai-model")
		proxy := c.String("proxy")
		concurrency := c.Int("concurrency")
		domain := c.String("domain")
		baseUrl := c.String("url")
		files := c.StringSlice("fs")
		if concurrency == 0 {
			concurrency = 1 // default concurrency, lower than desc completion
		}

		var aiOpts []aispec.AIConfigOption
		if model != "" {
			aiOpts = append(aiOpts, aispec.WithModel(model))
		}
		if domain != "" {
			aiOpts = append(aiOpts, aispec.WithDomain(domain))
		}
		if typ != "" {
			aiOpts = append(aiOpts, aispec.WithType(typ))
		}
		if key != "" {
			aiOpts = append(aiOpts, aispec.WithAPIKey(key))
		}
		if proxy != "" {
			aiOpts = append(aiOpts, aispec.WithProxy(proxy))
		}
		if baseUrl != "" {
			aiOpts = append(aiOpts, aispec.WithBaseURL(baseUrl))
		}

		swg := new(sync.WaitGroup)
		errChan := make(chan error, 1)
		taskChannel := make(chan string, 1)
		var errors error
		errorDone := make(chan struct{}, 1)
		go func() {
			for err := range errChan {
				errors = utils.JoinErrors(errors, err)
			}
			errorDone <- struct{}{}
			close(errorDone)
		}()
		var taskCount atomic.Int64
		for i := 0; i < concurrency; i++ {
			swg.Add(1)
			go func() {
				defer swg.Done()
				for fileName := range taskChannel {
					if !strings.HasSuffix(fileName, ".sf") {
						log.Infof("syntaxflow-test-cases-completion: skipping file %s (not a .sf file)", fileName)
						continue
					}
					raw, err := os.ReadFile(fileName)
					if err != nil {
						log.Errorf("failed to read file %s: %v", fileName, err)
						continue
					}
					rule, err := sfcompletion.CompleteTestCases(fileName, string(raw), aiOpts...)
					if err != nil {
						err = utils.Errorf("failed to complete test cases for file %s: %v", fileName, err)
						errChan <- err
						log.Errorf("%v", err)
						continue
					}
					// check format rule
					if _, err := sfvm.CompileRule(rule); err != nil {
						err = utils.Errorf("failed to validate completed rule for file %s: %v\nrule content: \n%s", fileName, err, rule)
						errChan <- err
						log.Errorf("%v", err)
						continue
					}

					err = os.WriteFile(fileName, []byte(rule), 0o666)
					if err != nil {
						log.Errorf("failed to write file %s: %v", fileName, err)
						errChan <- utils.Errorf("failed to write file %s: %v", fileName, err)
						continue
					}
					sleepTime := rand.Intn(3) + 2 // 2-4 seconds sleep
					log.Infof("syntaxflow-test-cases-completion: completed file %s, sleep for %d seconds", fileName, sleepTime)
					time.Sleep(time.Second * time.Duration(sleepTime))
				}
			}()
		}
		addTask := func(target string) {
			if utils.IsFile(target) {
				log.Infof("syntaxflow-test-cases-completion: processing file %s", target)
				taskChannel <- target
				taskCount.Add(1)
			} else if utils.IsDir(target) {
				log.Infof("syntaxflow-test-cases-completion: processing directory %s", target)
				filesys.Recursive(target, filesys.WithFileSystem(filesys.NewLocalFs()), filesys.WithFileStat(func(s string, info fs.FileInfo) error {
					log.Infof("syntaxflow-test-cases-completion: processing file %s", s)
					taskChannel <- s
					taskCount.Add(1)
					return nil
				}))
			} else {
				log.Errorf("syntaxflow-test-cases-completion: file %s not found", target)
			}
		}

		if target != "" {
			addTask(target)
		}
		for _, f := range files {
			addTask(f)
		}
		close(taskChannel)
		swg.Wait()
		close(errChan)
		<-errorDone
		return errors
	},
}
