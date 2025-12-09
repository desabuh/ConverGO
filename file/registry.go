package file

import (
	"fmt"
	"time"

	"github.com/desabuh/convergo/utils"
)

type FileRegistry[M any] struct {
	localDomainPrefix string
	fileContexts      map[string]*StringFileContext[M]
}

func CreateNewFileRegistry[M any, R any](domain string) *FileRegistry[M] {
	return &FileRegistry[M]{
		localDomainPrefix: domain,
		fileContexts:      make(map[string]*StringFileContext[M]),
	}
}

func (fr *FileRegistry[M]) CreateFileCtx(fileName string, state utils.ObservableState[M, string], pollingInterval time.Duration) error {

	fullPath := fr.localDomainPrefix + fileName

	err := CreateFile(fullPath)

	if err != nil {
		return err
	}

	file, err := OpenFile(fullPath, RW_TRUNC_MODE)

	if err != nil {
		return err
	}

	newCtx := GetNewFileContext(fr.localDomainPrefix, fileName, state, file, pollingInterval)

	newCtx.TrackState()

	return nil

}

func (fr *FileRegistry[M]) UpdateFileCtx(fileCtx FileContextInfo[M]) error {
	ctx, err := fr.getFileCtx(fileCtx.localPath)

	if err != nil {
		return err
	}

	err = ctx.UpdateState(fileCtx.readOnlyState.GetState())

	return err

}

func (fr *FileRegistry[M]) GetFileCtxInfo(fileName string) (FileContextInfo[M], error) { // fornisce i dati su un singolo file context readonly (per leggerli)
	ctx, err := fr.getFileCtx(fileName)
	if err != nil {
		return FileContextInfo[M]{}, fmt.Errorf("file context not found")
	}

	return ctx.GetStateCopy(), nil
}

func (fr *FileRegistry[M]) GetAllFileCtxInfo() map[FileContextInfo[M]]struct{} { // fornisce i dati su tutti i file context readonly (per spedirli)
	result := make(map[FileContextInfo[M]]struct{})
	for _, ctx := range fr.fileContexts {
		result[ctx.GetStateCopy()] = struct{}{}
	}
	return result
}

func (fr *FileRegistry[M]) getFileCtx(fileName string) (*StringFileContext[M], error) {
	if ctx, exists := fr.fileContexts[fileName]; exists {
		return ctx, nil
	}
	return &StringFileContext[M]{}, fmt.Errorf("file context not found")
}
