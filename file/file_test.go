package file

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/desabuh/convergo/cvrdt"
	"github.com/stretchr/testify/assert"
)

func TestNewFileContextOperation(t *testing.T) {
	const SITE_ID = "1"
	const START_POS = 0
	const CONTENT = "Hi"

	tmpFile, err := os.CreateTemp("./", "tmpfile-*.txt")
	if err != nil {
		log.Fatalf("Failed to create temp file: %v", err)
	}

	fileCtx := GetNewFileContext(SITE_ID, "TEST_DOMAIN", "path", cvrdt.NewWootCvrdtWithView(SITE_ID), tmpFile, 100*time.Millisecond)

	fileCtx.TrackState()

	t.Cleanup(func() {
		fileCtx.UntrackState()
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	})

	fileCtx.UpdateState(
		cvrdt.GetNewStateFromOp(
			cvrdt.CreateNewLocalOp(
				cvrdt.Insertion,
				START_POS,
				CONTENT,
				SITE_ID,
			),
		),
	)

	var state cvrdt.CvRDTState = fileCtx.GetStateCopy().readOnlyState

	time.Sleep(300 * time.Millisecond)

	operations := make([]cvrdt.CRDTOperation, 0, len(state))
	for _, c := range state {
		operations = append(operations, c)
	}

	assert.Equal(t, 1, len(operations))

	assert.Equal(t, operations[0].Char(), CONTENT)

	tmpFile.Seek(0, 0)
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		log.Fatalf("Failed to read temp file: %v", err)
	}

	assert.Equal(t, CONTENT, string(content))
}

func TestNewFileRegistry(t *testing.T) {
	const SITE_ID = "1"
	const DOMAIN = "TEST_DOMAIN"
	const DOMAIN_PATH = "./test_domain/"
	const FILE_NAME = "file_test.txt"
	const POLLING_INTERVAL = 100 * time.Millisecond

	var registry *FileRegistry[cvrdt.CvRDTState] = CreateNewWootFileRegistry(SITE_ID, DOMAIN, DOMAIN_PATH)

	t.Cleanup(func() {
		registry.RemoveFileCtx(FILE_NAME)
		time.Sleep(50 * time.Millisecond)
		os.RemoveAll(DOMAIN_PATH)
	})

	err := registry.CreateFileCtx(FILE_NAME, POLLING_INTERVAL)

	assert.Nil(t, err, "Path error for "+DOMAIN_PATH+FILE_NAME)

	fullPath := DOMAIN_PATH + FILE_NAME
	_, err = os.Stat(fullPath)
	assert.Nil(t, err, "File should exist at "+fullPath)

	info, err := registry.GetFileCtxInfo(FILE_NAME)
	assert.Nil(t, err, "Should be able to get file context info")
	assert.Equal(t, DOMAIN, info.domain)
	assert.Equal(t, FILE_NAME, info.localPath)
}

func TestFileRegistryCreateDuplicateFile(t *testing.T) {
	const SITE_ID = "1"
	const DOMAIN = "TEST_DOMAIN"
	const DOMAIN_PATH = "./test_domain_duplicate/"
	const FILE_NAME = "duplicate.txt"
	const POLLING_INTERVAL = 100 * time.Millisecond

	registry := CreateNewWootFileRegistry(SITE_ID, DOMAIN, DOMAIN_PATH)

	t.Cleanup(func() {
		registry.RemoveFileCtx(FILE_NAME)
		time.Sleep(50 * time.Millisecond)
		os.RemoveAll(DOMAIN_PATH)
	})

	err := registry.CreateFileCtx(FILE_NAME, POLLING_INTERVAL)
	assert.Nil(t, err, "First creation should succeed")

	err = registry.CreateFileCtx(FILE_NAME, POLLING_INTERVAL)
	assert.NotNil(t, err, "Creating duplicate file should fail")
}

func TestFileRegistryUpdateFileCtx(t *testing.T) {
	const SITE_ID_1 = "1"
	const SITE_ID_2 = "2"
	const DOMAIN = "TEST_DOMAIN"
	const DOMAIN_PATH = "./test_domain_update/"
	const FILE_NAME = "update_test.txt"
	const POLLING_INTERVAL = 100 * time.Millisecond
	const CONTENT = "Hello"

	registry := CreateNewWootFileRegistry(SITE_ID_1, DOMAIN, DOMAIN_PATH)

	t.Cleanup(func() {
		registry.RemoveFileCtx(FILE_NAME)
		time.Sleep(50 * time.Millisecond)
		os.RemoveAll(DOMAIN_PATH)
	})

	err := registry.CreateFileCtx(FILE_NAME, POLLING_INTERVAL)
	assert.Nil(t, err)

	info, err := registry.GetFileCtxInfo(FILE_NAME)
	assert.Nil(t, err)

	newState := cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 0, CONTENT, SITE_ID_2),
	)

	info.readOnlyState = newState
	err = registry.UpdateFileCtx(info)
	assert.Nil(t, err)

	time.Sleep(200 * time.Millisecond)

	updatedInfo, err := registry.GetFileCtxInfo(FILE_NAME)
	assert.Nil(t, err)
	assert.True(t, len(updatedInfo.readOnlyState) > 0, "State should have operations")
}

func TestFileRegistryUpdateNonExistentFile(t *testing.T) {
	const SITE_ID = "1"
	const DOMAIN = "TEST_DOMAIN"
	const DOMAIN_PATH = "./test_domain_update_fail/"

	t.Cleanup(func() {
		os.RemoveAll(DOMAIN_PATH)
	})

	registry := CreateNewWootFileRegistry(SITE_ID, DOMAIN, DOMAIN_PATH)

	info := FileContextInfo[cvrdt.CvRDTState]{
		domain:        DOMAIN,
		localPath:     "non_existent.txt",
		readOnlyState: cvrdt.GetNewStateFromOp(),
	}

	err := registry.UpdateFileCtx(info)
	assert.NotNil(t, err, "Updating non-existent file should fail")
}

func TestFileRegistryGetFileCtxInfo(t *testing.T) {
	const SITE_ID = "1"
	const DOMAIN = "TEST_DOMAIN"
	const DOMAIN_PATH = "./test_domain_get/"
	const FILE_NAME = "get_test.txt"
	const POLLING_INTERVAL = 100 * time.Millisecond

	registry := CreateNewWootFileRegistry(SITE_ID, DOMAIN, DOMAIN_PATH)

	t.Cleanup(func() {
		registry.RemoveFileCtx(FILE_NAME)
		time.Sleep(50 * time.Millisecond)
		os.RemoveAll(DOMAIN_PATH)
	})

	err := registry.CreateFileCtx(FILE_NAME, POLLING_INTERVAL)
	assert.Nil(t, err)

	info, err := registry.GetFileCtxInfo(FILE_NAME)
	assert.Nil(t, err)
	assert.Equal(t, DOMAIN, info.domain)
	assert.Equal(t, FILE_NAME, info.localPath)
	assert.NotNil(t, info.readOnlyState)
}

func TestFileRegistryGetNonExistentFileCtxInfo(t *testing.T) {
	const SITE_ID = "1"
	const DOMAIN = "TEST_DOMAIN"
	const DOMAIN_PATH = "./test_domain_get_fail/"

	t.Cleanup(func() {
		os.RemoveAll(DOMAIN_PATH)
	})

	registry := CreateNewWootFileRegistry(SITE_ID, DOMAIN, DOMAIN_PATH)

	_, err := registry.GetFileCtxInfo("non_existent.txt")
	assert.NotNil(t, err, "Getting non-existent file should fail")
}

func TestFileRegistryRemoveFileCtx(t *testing.T) {
	const SITE_ID = "1"
	const DOMAIN = "TEST_DOMAIN"
	const DOMAIN_PATH = "./test_domain_remove/"
	const FILE_NAME = "remove_test.txt"
	const POLLING_INTERVAL = 100 * time.Millisecond

	registry := CreateNewWootFileRegistry(SITE_ID, DOMAIN, DOMAIN_PATH)

	t.Cleanup(func() {
		time.Sleep(50 * time.Millisecond)
		os.RemoveAll(DOMAIN_PATH)
	})

	err := registry.CreateFileCtx(FILE_NAME, POLLING_INTERVAL)
	assert.Nil(t, err)

	_, err = registry.GetFileCtxInfo(FILE_NAME)
	assert.Nil(t, err, "File context should exist")

	err = registry.RemoveFileCtx(FILE_NAME)
	assert.Nil(t, err)

	_, err = registry.GetFileCtxInfo(FILE_NAME)
	assert.NotNil(t, err, "File context should not exist after removal")

	fullPath := DOMAIN_PATH + FILE_NAME
	_, err = os.Stat(fullPath)
	assert.Nil(t, err, "Physical file should still exist")
}

func TestFileRegistryRemoveNonExistentFileCtx(t *testing.T) {
	const SITE_ID = "1"
	const DOMAIN = "TEST_DOMAIN"
	const DOMAIN_PATH = "./test_domain_remove_fail/"

	t.Cleanup(func() {
		os.RemoveAll(DOMAIN_PATH)
	})

	registry := CreateNewWootFileRegistry(SITE_ID, DOMAIN, DOMAIN_PATH)

	err := registry.RemoveFileCtx("non_existent.txt")
	assert.Nil(t, err, "Removing non-existent file should not error")
}

func TestFileRegistryGetAllFileCtxInfo(t *testing.T) {
	const SITE_ID = "1"
	const DOMAIN = "TEST_DOMAIN"
	const DOMAIN_PATH = "./test_domain_getall/"
	const POLLING_INTERVAL = 100 * time.Millisecond

	registry := CreateNewWootFileRegistry(SITE_ID, DOMAIN, DOMAIN_PATH)

	t.Cleanup(func() {
		fileNames := []string{"file1.txt", "file2.txt", "file3.txt"}
		for _, fileName := range fileNames {
			registry.RemoveFileCtx(fileName)
		}
		time.Sleep(50 * time.Millisecond)
		os.RemoveAll(DOMAIN_PATH)
	})

	allInfo := registry.GetAllFileCtxInfo()
	assert.Equal(t, 0, len(allInfo), "Should have no file contexts initially")

	fileNames := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, fileName := range fileNames {
		err := registry.CreateFileCtx(fileName, POLLING_INTERVAL)
		assert.Nil(t, err)
	}

	allInfo = registry.GetAllFileCtxInfo()
	assert.Equal(t, len(fileNames), len(allInfo), "Should have all file contexts")

	for _, fileName := range fileNames {
		info, exists := allInfo[fileName]
		assert.True(t, exists, "File "+fileName+" should be in map")
		assert.Equal(t, DOMAIN, info.domain)
		assert.Equal(t, fileName, info.localPath)
	}
}

func TestFileRegistryMultipleOperations(t *testing.T) {
	const SITE_ID = "1"
	const DOMAIN = "TEST_DOMAIN"
	const DOMAIN_PATH = "./test_domain_multi/"
	const FILE_NAME = "multi_test.txt"
	const POLLING_INTERVAL = 100 * time.Millisecond

	registry := CreateNewWootFileRegistry(SITE_ID, DOMAIN, DOMAIN_PATH)

	t.Cleanup(func() {
		registry.RemoveFileCtx(FILE_NAME)
		time.Sleep(50 * time.Millisecond)
		os.RemoveAll(DOMAIN_PATH)
	})

	err := registry.CreateFileCtx(FILE_NAME, POLLING_INTERVAL)
	assert.Nil(t, err)

	operations := []string{"Hello", " ", "World", "!"}
	for i, op := range operations {
		newState := cvrdt.GetNewStateFromOp(
			cvrdt.CreateNewLocalOp(cvrdt.Insertion, i, op, SITE_ID),
		)

		currentInfo, _ := registry.GetFileCtxInfo(FILE_NAME)
		mergedState := currentInfo.readOnlyState.Merge(newState)

		currentInfo.readOnlyState = mergedState
		err = registry.UpdateFileCtx(currentInfo)
		assert.Nil(t, err)
	}

	time.Sleep(200 * time.Millisecond)

	finalInfo, err := registry.GetFileCtxInfo(FILE_NAME)
	assert.Nil(t, err)
	assert.Equal(t, len(operations), len(finalInfo.readOnlyState), "Should have all operations")

	fullPath := DOMAIN_PATH + FILE_NAME
	content, err := os.ReadFile(fullPath)
	assert.Nil(t, err)

	expectedContent := "Hello World!"
	assert.Equal(t, expectedContent, string(content), "File content should match expected")
}

func TestFileRegistryConcurrentAccess(t *testing.T) {
	const SITE_ID = "1"
	const DOMAIN = "TEST_DOMAIN"
	const DOMAIN_PATH = "./test_domain_concurrent/"
	const FILE_NAME = "concurrent_test.txt"
	const POLLING_INTERVAL = 100 * time.Millisecond
	const NUM_GOROUTINES = 10

	registry := CreateNewWootFileRegistry(SITE_ID, DOMAIN, DOMAIN_PATH)

	t.Cleanup(func() {
		registry.RemoveFileCtx(FILE_NAME)
		time.Sleep(50 * time.Millisecond)
		os.RemoveAll(DOMAIN_PATH)
	})

	err := registry.CreateFileCtx(FILE_NAME, POLLING_INTERVAL)
	assert.Nil(t, err)

	done := make(chan bool, NUM_GOROUTINES)

	for i := 0; i < NUM_GOROUTINES; i++ {
		go func() {
			info, err := registry.GetFileCtxInfo(FILE_NAME)
			assert.Nil(t, err)
			assert.Equal(t, DOMAIN, info.domain)
			done <- true
		}()
	}

	for i := 0; i < NUM_GOROUTINES; i++ {
		<-done
	}
}
