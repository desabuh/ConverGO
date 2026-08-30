package file

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/desabuh/convergo/cvrdt"
	"github.com/desabuh/convergo/utils"
)

var (
	FILE_CTX_NOT_FOUND = errors.New("file context not found")
)

const (
	POLLING_INTERVAL = 500 * time.Millisecond
)

type FileRegistry[M utils.Clonable[M]] struct {
	mu                  sync.Mutex
	localDomainPrefix   string
	newEmptyCtxSupplier func() utils.ObservableState[M, string]
	fileContexts        map[string]*StringFileContext[M]
}

func CreateNewWootFileRegistry(siteId string, domainLocalPrefix string) *FileRegistry[cvrdt.CvRDTState] {
	return CreateNewCRDTFileRegistry(
		domainLocalPrefix,
		func() utils.ObservableState[cvrdt.CvRDTState, string] {
			return cvrdt.NewWootCvrdtWithView(siteId)
		},
	)
}

func CreateNewCRDTFileRegistry(domainLocalPrefix string, newCtxSupplier func() utils.ObservableState[cvrdt.CvRDTState, string]) *FileRegistry[cvrdt.CvRDTState] {
	return &FileRegistry[cvrdt.CvRDTState]{
		localDomainPrefix:   domainLocalPrefix,
		newEmptyCtxSupplier: newCtxSupplier,
		fileContexts:        make(map[string]*StringFileContext[cvrdt.CvRDTState]),
	}
}

// boolean value is for newly created fileContext
func (fr *FileRegistry[M]) UpdateFileCtx(fileCtx FileContextInfo[M]) (FileContextInfo[M], error) {
	fr.mu.Lock()
	defer fr.mu.Unlock()

	ctx, err := fr.getFileCtx(fileCtx.LocalPath)

	var wasCtxNewlyCreated bool = false

	if err != nil {
		err := fr.createFileCtx(fileCtx.LocalPath, POLLING_INTERVAL)

		wasCtxNewlyCreated = true

		if err != nil {
			return FileContextInfo[M]{}, err
		}
	}

	ctx, _ = fr.getFileCtx(fileCtx.LocalPath)

	err = ctx.UpdateState(fileCtx.ReadOnlyState)

	if err != nil {
		return FileContextInfo[M]{}, err
	}

	info := ctx.GetStateCopy()

	if wasCtxNewlyCreated {
		info.IsNewlyCreated = true
	}

	return info, err

}

func (fr *FileRegistry[M]) GetFileCtxInfo(fileName string) (FileContextInfo[M], error) { // fornisce i dati su un singolo file context readonly (per leggerli)
	fr.mu.Lock()
	defer fr.mu.Unlock()

	ctx, err := fr.getFileCtx(fileName)
	if err != nil {
		return FileContextInfo[M]{}, err
	}

	return ctx.GetStateCopy(), nil
}

func (fr *FileRegistry[M]) GetFileContent(fileName string) (string, error) {
	fr.mu.Lock()
	defer fr.mu.Unlock()

	fctx, err := fr.getFileCtx(fileName)

	if err != nil {
		return "", err
	}

	return utils.First(fctx.metaState.Snapshot()), nil

}

func (fr *FileRegistry[M]) GetAllFileCtxInfo() map[string]FileContextInfo[M] { // fornisce i dati su tutti i file context readonly (per spedirli)
	fr.mu.Lock()
	defer fr.mu.Unlock()

	return fr.getAllFileCtxInfoLocked()
}

func (fr *FileRegistry[M]) RemoveAllCtx() error {
	fr.mu.Lock()
	defer fr.mu.Unlock()

	ctxInfos := fr.getAllFileCtxInfoLocked()

	var errs []error

	for path := range ctxInfos {
		if err := fr.removeFileCtx(path); err != nil {
			errs = append(errs, fmt.Errorf("error while trying to close file context %s: %v", path, err))
		}
	}

	return errors.Join(errs...)

}

func (fr *FileRegistry[M]) getAllFileCtxInfoLocked() map[string]FileContextInfo[M] {
	result := make(map[string]FileContextInfo[M])
	for fileName, ctx := range fr.fileContexts {
		result[fileName] = ctx.GetStateCopy()
	}
	return result
}

func (fr *FileRegistry[M]) createFileCtx(fileName string, pollingInterval time.Duration) error {

	fullPath := fr.localDomainPrefix + fileName

	err := os.MkdirAll(fr.localDomainPrefix, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", fr.localDomainPrefix, err)
	}

	err = CreateFile(fullPath)

	if err != nil {
		return err
	}

	file, err := OpenFile(fullPath, RW_TRUNC_MODE)

	if err != nil {
		return err
	}

	newCtx := GetNewFileContext(fileName, fr.newEmptyCtxSupplier(), file, pollingInterval)

	newCtx.TrackState()

	fr.fileContexts[fileName] = newCtx

	return nil

}

func (fr *FileRegistry[M]) removeFileCtx(filename string) error {
	ctx, err := fr.getFileCtx(filename)

	if err != nil {
		return nil
	}

	err = ctx.UntrackState()

	if err != nil {
		return err
	}

	delete(fr.fileContexts, filename)

	return nil

}

func (fr *FileRegistry[M]) getFileCtx(fileName string) (*StringFileContext[M], error) {
	if ctx, exists := fr.fileContexts[fileName]; exists {
		return ctx, nil
	}
	return &StringFileContext[M]{}, FILE_CTX_NOT_FOUND
}
