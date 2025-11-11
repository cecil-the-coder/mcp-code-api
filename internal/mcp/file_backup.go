package mcp

import "sync"

type FileBackupStore struct {
	mutex   sync.RWMutex
	backups map[string]string
}

func NewFileBackupStore() *FileBackupStore {
	return &FileBackupStore{
		backups: make(map[string]string),
	}
}

func (f *FileBackupStore) StoreBackup(filePath, content string) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.backups[filePath] = content
}

func (f *FileBackupStore) GetBackup(filePath string) (string, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	content, exists := f.backups[filePath]
	if !exists {
		return "", &BackupNotFoundError{Path: filePath}
	}
	return content, nil
}

func (f *FileBackupStore) HasBackup(filePath string) bool {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	_, exists := f.backups[filePath]
	return exists
}

func (f *FileBackupStore) ClearBackup(filePath string) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	delete(f.backups, filePath)
}

type BackupNotFoundError struct {
	Path string
}

func (e *BackupNotFoundError) Error() string {
	return "backup not found for path: " + e.Path
}

var globalBackupStore *FileBackupStore

func init() {
	globalBackupStore = NewFileBackupStore()
}