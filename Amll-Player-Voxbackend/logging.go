package main

import (
	"log"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type appLogWriter struct{}

var (
	logStoreMu sync.RWMutex
	logStore   *appStore
)

func setLogStore(store *appStore) {
	logStoreMu.Lock()
	logStore = store
	logStoreMu.Unlock()
}

func uiLog(format string, args ...any) {
	logStoreMu.RLock()
	store := logStore
	logStoreMu.RUnlock()
	if store != nil {
		store.AppendLog(format, args...)
	}
}

func (appLogWriter) Write(p []byte) (int, error) {
	for _, line := range strings.Split(strings.TrimSpace(string(p)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			uiLog("%s", line)
		}
	}
	return len(p), nil
}

func installAppLogging() {
	writer := appLogWriter{}
	log.SetFlags(0)
	log.SetOutput(writer)
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = writer
	gin.DefaultErrorWriter = writer
}
