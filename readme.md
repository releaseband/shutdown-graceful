
### Shutdown graceful golang application


Examples
--------

Here is an example of using the package:

```go
    
package main

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
        "error"
)

func start(f *fiber.App) error {

	logger := zap.L().Named("shutdown")
    	
	logFunc := func(err error) {
		logger.Error("shutdown failed", zap.Error(err))    
	}

	ch := ListenShutdownSignals(f, logFunc)

	if err := f.Listen(":" + strconv.Itoa(cfg.Server.Port)); err != nil {
		return fmt.Errorf("fiber.Listen failed: %w", err)
	}

	<-ch
    
    	return nil
}
