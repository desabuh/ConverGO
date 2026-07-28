package file

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/desabuh/convergo/cvrdt"
	"github.com/desabuh/convergo/utils"
)

var (
	FILE_CTX_NOT_FOUND = errors.New("file context not found")
)

const (
	POLLING_INTERVAL = 3 * time.Second
)

type FileRegistry[M utils.Clonable[M]] struct {
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

func (fr *FileRegistry[M]) CreateFileCtx(fileName string, pollingInterval time.Duration) error {

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

// boolean value is for newly created fileContext
func (fr *FileRegistry[M]) UpdateFileCtx(fileCtx FileContextInfo[M]) (bool, error) {
	ctx, err := fr.getFileCtx(fileCtx.LocalPath)

	var wasCtxNewlyCreated bool = false

	if err != nil {
		err := fr.CreateFileCtx(fileCtx.LocalPath, POLLING_INTERVAL)

		wasCtxNewlyCreated = true

		if err != nil {
			return wasCtxNewlyCreated, err
		}
	}

	ctx, _ = fr.getFileCtx(fileCtx.LocalPath)

	err = ctx.UpdateState(fileCtx.ReadOnlyState)

	return wasCtxNewlyCreated, err

}

func (fr *FileRegistry[M]) GetFileCtxInfo(fileName string) (FileContextInfo[M], error) { // fornisce i dati su un singolo file context readonly (per leggerli)
	ctx, err := fr.getFileCtx(fileName)
	if err != nil {
		return FileContextInfo[M]{}, err
	}

	return ctx.GetStateCopy(), nil
}

func (fr *FileRegistry[M]) GetFileContent(fileName string) (string, error) {
	fctx, err := fr.getFileCtx(fileName)

	if err != nil {
		return "", err
	}

	return utils.First(fctx.metaState.Snapshot()), nil

}

func (fr *FileRegistry[M]) RemoveFileCtx(filename string) error {
	ctx, err := fr.getFileCtx(filename)

	if err != nil {
		return nil
	}

	ctx.UntrackState()

	delete(fr.fileContexts, filename)

	return nil

}

func (fr *FileRegistry[M]) GetAllFileCtxInfo() map[string]FileContextInfo[M] { // fornisce i dati su tutti i file context readonly (per spedirli)
	result := make(map[string]FileContextInfo[M])
	for fileName, ctx := range fr.fileContexts {
		result[fileName] = ctx.GetStateCopy()
	}
	return result
}

func (fr *FileRegistry[M]) GetFileRegistryInfo() []FileContextInfo[M] {
	return utils.Values(fr.GetAllFileCtxInfo())

}

func (fr *FileRegistry[M]) getFileCtx(fileName string) (*StringFileContext[M], error) {
	if ctx, exists := fr.fileContexts[fileName]; exists {
		return ctx, nil
	}
	return &StringFileContext[M]{}, FILE_CTX_NOT_FOUND
}
