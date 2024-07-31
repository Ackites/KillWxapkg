package restore

import (
	"log"
	"sync"

	"github.com/Ackites/KillWxapkg/internal/config"
)

type CommandExecutor struct {
	manager *config.WxapkgManager
}

func NewCommandExecutor(manager *config.WxapkgManager) *CommandExecutor {
	return &CommandExecutor{manager: manager}
}

func (executor *CommandExecutor) ExecuteAll() {
	var wg sync.WaitGroup
	errCh := make(chan error, len(executor.manager.Packages))

	for _, wxapkg := range executor.manager.Packages {
		for _, parser := range wxapkg.Parsers {
			wg.Add(1)
			go func(wxapkg *config.WxapkgInfo, parser config.Parser) {
				defer wg.Done()
				if err := parser.Parse(*wxapkg); err != nil {
					errCh <- err
				}
			}(wxapkg, parser)
		}
	}

	// 等待所有任务完成, 关闭错误通道
	go func() {
		wg.Wait()
		close(errCh)
	}()

	for err := range errCh {
		if err != nil {
			log.Printf("%v", err)
		}
	}
}
